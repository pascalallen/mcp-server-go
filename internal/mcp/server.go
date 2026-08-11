package mcp

import (
	"net/http"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/pascalallen/mcp-server-go/internal/kb"
)

// NewMCPServer builds the MCP server with all tools, resources, and prompts
// registered. Callers pick a transport with NewHTTPHandler or
// server.ServeStdio.
func NewMCPServer(store *kb.Store) *server.MCPServer {
	s := server.NewMCPServer(
		"mcp-server-go",
		Version,
		server.WithToolCapabilities(true),
		// subscribe=false: per-resource subscriptions are not implemented.
		// listChanged=true: AddResource/RemoveResource notify clients as
		// knowledge base entries come and go.
		server.WithResourceCapabilities(false, true),
		server.WithPromptCapabilities(false),
	)

	echoTool := mcpgo.NewTool(
		"echo",
		mcpgo.WithDescription("Returns the input message unchanged. Useful for testing connectivity."),
		mcpgo.WithString(
			"message",
			mcpgo.Required(),
			mcpgo.Description("The message to echo back"),
		),
	)
	s.AddTool(echoTool, HandleEcho)

	serverInfoTool := mcpgo.NewTool(
		"server_info",
		mcpgo.WithDescription("Returns server name, version, and uptime in seconds."),
	)
	s.AddTool(serverInfoTool, HandleServerInfo)

	registerKB(s, store)
	registerPrompts(s)

	return s
}

// NewHTTPHandler wraps an MCP server in the streamable HTTP transport.
func NewHTTPHandler(s *server.MCPServer) http.Handler {
	return server.NewStreamableHTTPServer(s)
}
