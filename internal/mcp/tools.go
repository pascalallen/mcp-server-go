package mcp

import (
	"context"
	"fmt"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
)

const Version = "0.1.0"

var startTime = time.Now()

func HandleEcho(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	args := req.GetArguments()
	msg, ok := args["message"].(string)
	if !ok || msg == "" {
		return nil, fmt.Errorf("message argument is required and must be a string")
	}
	return mcpgo.NewToolResultText(msg), nil
}

func HandleServerInfo(_ context.Context, _ mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	uptime := time.Since(startTime).Round(time.Second)
	info := fmt.Sprintf(`{"name":"mcp-server-go","version":%q,"uptime_seconds":%d}`, Version, int(uptime.Seconds()))
	return mcpgo.NewToolResultText(info), nil
}
