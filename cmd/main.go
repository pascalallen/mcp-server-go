package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/server"

	"github.com/pascalallen/mcp-server-go/internal/kb"
	inframcp "github.com/pascalallen/mcp-server-go/internal/mcp"
	"github.com/pascalallen/mcp-server-go/internal/routes"
)

func main() {
	transport := flag.String("transport", envOr("MCP_TRANSPORT", "http"), "transport to serve: http or stdio")
	addr := flag.String("addr", envOr("MCP_ADDR", "127.0.0.1:8080"), "listen address for the http transport")
	kbDir := flag.String("kb-dir", envOr("KB_DIR", "data/kb"), "directory holding knowledge base entries")
	flag.Parse()

	if *transport == "stdio" {
		// Stdout carries JSON-RPC in stdio mode; keep all logging on stderr.
		log.SetOutput(os.Stderr)
	}

	dir, err := filepath.Abs(*kbDir)
	if err != nil {
		log.Fatalf("kb: resolve dir: %v", err)
	}
	store, err := kb.Open(dir)
	if err != nil {
		log.Fatalf("kb: %v", err)
	}

	mcpServer := inframcp.NewMCPServer(store)

	switch *transport {
	case "stdio":
		log.Printf("starting mcp-server-go on stdio (kb dir: %s)", dir)
		if err := server.ServeStdio(mcpServer, server.WithErrorLogger(log.New(os.Stderr, "", log.LstdFlags))); err != nil {
			log.Fatalf("stdio: %v", err)
		}
	case "http":
		sec := routes.SecurityConfig{
			AllowedOrigins: splitList(os.Getenv("MCP_ALLOWED_ORIGINS")),
			AllowedHosts:   allowedHosts(),
			BearerToken:    os.Getenv("MCP_AUTH_TOKEN"),
		}
		log.Printf("starting mcp-server-go on %s (kb dir: %s)", *addr, dir)
		routes.NewRouter(inframcp.NewHTTPHandler(mcpServer), sec).Serve(*addr)
	default:
		log.Fatalf("unknown transport %q: want http or stdio", *transport)
	}
}

// allowedHosts returns the Host-header allowlist: MCP_ALLOWED_HOSTS if set,
// otherwise the loopback names. Set MCP_ALLOWED_HOSTS when binding to a
// non-loopback address.
func allowedHosts() []string {
	hosts := splitList(envOr("MCP_ALLOWED_HOSTS", "localhost,127.0.0.1,::1"))
	for i, h := range hosts {
		hosts[i] = strings.Trim(h, "[]")
	}
	return hosts
}

func envOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func splitList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
