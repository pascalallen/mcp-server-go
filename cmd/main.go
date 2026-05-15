package main

import (
	"log"

	inframcp "github.com/pascalallen/mcp-server-go/internal/mcp"
	"github.com/pascalallen/mcp-server-go/internal/routes"
)

func main() {
	mcpHandler := inframcp.NewServer()
	log.Println("starting mcp-server-go on :8080")
	routes.NewRouter(mcpHandler).Serve(":8080")
}
