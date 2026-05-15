# mcp-server-go

mcp-server-go is a module that is designed to give you an MCP server in Go, right out of the box. There is a publication for this repository which can be found [here](https://medium.com/@pascalallen/how-to-build-an-mcp-server-in-go-50ed8a4c9ed4).

## Available Tools

| Tool | Description |
|------|-------------|
| `echo` | Returns the input message unchanged. Useful for testing connectivity. |
| `server_info` | Returns server name, version, and uptime in seconds. |

## Prerequisites

- [Go](https://go.dev/dl/)

## MCP server usage

### Run the MCP server from project root

```bash
go run cmd/main.go
```

### Install the MCP server in Claude Code

```bash
claude mcp add --transport http mcp-server-go http://localhost:8080/mcp
```
