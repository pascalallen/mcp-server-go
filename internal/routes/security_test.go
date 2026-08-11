package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestRouter(sec SecurityConfig) Router {
	gin.SetMode(gin.TestMode)
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return NewRouter(ok, sec)
}

func request(t *testing.T, r Router, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	for k, v := range headers {
		if k == "Host" {
			req.Host = v
			continue
		}
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.engine.ServeHTTP(w, req)
	return w
}

func TestSecurityOrigin(t *testing.T) {
	r := newTestRouter(SecurityConfig{AllowedOrigins: []string{"https://app.example.com"}})

	tests := []struct {
		name   string
		origin string
		want   int
	}{
		{"no origin (non-browser client)", "", http.StatusOK},
		{"localhost", "http://localhost:8080", http.StatusOK},
		{"loopback IPv4", "http://127.0.0.1:8080", http.StatusOK},
		{"loopback IPv6", "http://[::1]:8080", http.StatusOK},
		{"allowlisted", "https://app.example.com", http.StatusOK},
		{"evil", "https://evil.example", http.StatusForbidden},
		{"lookalike", "http://localhost.evil.example", http.StatusForbidden},
	}
	for _, tt := range tests {
		headers := map[string]string{}
		if tt.origin != "" {
			headers["Origin"] = tt.origin
		}
		if got := request(t, r, headers).Code; got != tt.want {
			t.Errorf("%s: status = %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestSecurityHost(t *testing.T) {
	r := newTestRouter(SecurityConfig{AllowedHosts: []string{"localhost", "127.0.0.1", "::1"}})

	tests := []struct {
		host string
		want int
	}{
		{"localhost:8080", http.StatusOK},
		{"127.0.0.1:8080", http.StatusOK},
		{"localhost", http.StatusOK},
		{"[::1]:8080", http.StatusOK},
		{"rebound.evil.example:8080", http.StatusForbidden},
	}
	for _, tt := range tests {
		if got := request(t, r, map[string]string{"Host": tt.host}).Code; got != tt.want {
			t.Errorf("Host %q: status = %d, want %d", tt.host, got, tt.want)
		}
	}

	// An empty allowlist disables the Host check.
	open := newTestRouter(SecurityConfig{})
	if got := request(t, open, map[string]string{"Host": "anything.example"}).Code; got != http.StatusOK {
		t.Errorf("empty allowlist: status = %d, want 200", got)
	}
}

func TestSecurityBearerToken(t *testing.T) {
	r := newTestRouter(SecurityConfig{BearerToken: "secret"})

	tests := []struct {
		name string
		auth string
		want int
	}{
		{"missing", "", http.StatusUnauthorized},
		{"wrong", "Bearer nope", http.StatusUnauthorized},
		{"malformed", "secret", http.StatusUnauthorized},
		{"correct", "Bearer secret", http.StatusOK},
	}
	for _, tt := range tests {
		headers := map[string]string{}
		if tt.auth != "" {
			headers["Authorization"] = tt.auth
		}
		w := request(t, r, headers)
		if w.Code != tt.want {
			t.Errorf("%s: status = %d, want %d", tt.name, w.Code, tt.want)
		}
		if tt.want == http.StatusUnauthorized && w.Header().Get("WWW-Authenticate") != "Bearer" {
			t.Errorf("%s: missing WWW-Authenticate header", tt.name)
		}
	}

	// No token configured: auth is disabled.
	open := newTestRouter(SecurityConfig{})
	if got := request(t, open, nil).Code; got != http.StatusOK {
		t.Errorf("auth disabled: status = %d, want 200", got)
	}
}
