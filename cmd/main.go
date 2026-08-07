package main

import (
	"log"
	"os"

	"github.com/pascalallen/mcp-server-go/internal/kb"
	inframcp "github.com/pascalallen/mcp-server-go/internal/mcp"
	"github.com/pascalallen/mcp-server-go/internal/routes"
)

func main() {
	dir := os.Getenv("KB_DIR")
	if dir == "" {
		dir = "data/kb"
	}
	store, err := kb.Open(dir)
	if err != nil {
		log.Fatalf("kb: %v", err)
	}
	mcpHandler := inframcp.NewServer(store)
	log.Printf("starting mcp-server-go on :8080 (kb dir: %s)", dir)
	routes.NewRouter(mcpHandler).Serve(":8080")
}
