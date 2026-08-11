package routes

import (
	"crypto/subtle"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityConfig controls the checks applied to requests before they reach
// the MCP handler.
type SecurityConfig struct {
	// AllowedOrigins lists Origin values permitted in addition to localhost
	// origins, which are always allowed.
	AllowedOrigins []string
	// AllowedHosts lists Host header values (without port) that requests
	// must carry. An empty list disables the check.
	AllowedHosts []string
	// BearerToken, when non-empty, requires every request to carry
	// "Authorization: Bearer <token>".
	BearerToken string
}

// Security validates Origin and Host headers — the defense against
// DNS-rebinding attacks the MCP spec requires of local HTTP servers — and
// optionally enforces bearer-token auth.
//
// Requests without an Origin header are allowed: non-browser MCP clients
// don't send one, and rebinding attacks originate from browsers, which do.
func Security(cfg SecurityConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if origin := c.GetHeader("Origin"); origin != "" && !originAllowed(origin, cfg.AllowedOrigins) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "origin not allowed"})
			return
		}
		if len(cfg.AllowedHosts) > 0 && !slices.Contains(cfg.AllowedHosts, stripPort(c.Request.Host)) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "host not allowed"})
			return
		}
		if cfg.BearerToken != "" {
			token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
			if !ok || subtle.ConstantTimeCompare([]byte(token), []byte(cfg.BearerToken)) != 1 {
				c.Header("WWW-Authenticate", "Bearer")
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid bearer token"})
				return
			}
		}
		c.Next()
	}
}

func originAllowed(origin string, extra []string) bool {
	if slices.Contains(extra, origin) {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return isLoopbackHost(u.Hostname())
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// stripPort returns the host portion of a Host header, tolerating values
// with no port.
func stripPort(hostport string) string {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return strings.Trim(hostport, "[]")
	}
	return host
}
