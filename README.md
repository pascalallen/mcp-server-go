# mcp-server-go

mcp-server-go is a module that is designed to give you an MCP server in Go, right out of the box. There is a publication for this repository which can be found [here](https://medium.com/@pascalallen/how-to-build-an-mcp-server-in-go-50ed8a4c9ed4).

## Available Tools

| Tool | Description |
|------|-------------|
| `echo` | Returns the input message unchanged. Useful for testing connectivity. |
| `server_info` | Returns server name, version, and uptime in seconds. |
| `kb_add` | Adds a new knowledge base entry. The slug is derived from the title unless provided. |
| `kb_get` | Returns a knowledge base entry, including its full body. |
| `kb_update` | Updates a knowledge base entry. Only the provided fields change. |
| `kb_delete` | Deletes a knowledge base entry. |
| `kb_search` | Searches entries by keyword over titles, tags, and bodies. Returns scored results with snippets. |
| `kb_list` | Lists all knowledge base entries (metadata only). |

## Knowledge base

The server ships with a file-backed knowledge base. Each entry is a Markdown
file with YAML front matter, stored in `data/kb` (override with the `KB_DIR`
environment variable). The filename is the entry's slug:

```markdown
---
title: Getting started with MCP
tags: [mcp, tutorial]
created: 2026-08-07T12:00:00Z
updated: 2026-08-07T12:30:00Z
---

Body markdown starts here...
```

Entries are editable both ways: change the files on disk (they load at
startup), or call the `kb_*` tools from any connected MCP client. Writes made
through the tools persist back to disk.

### Resources

Every entry is also exposed as an MCP resource at `kb://{slug}`
(`text/markdown`), so clients can browse the knowledge base with
`resources/list` and read entries with `resources/read` without calling a
tool. The server announces `listChanged`, so connected clients are notified
when entries are added, updated, or deleted.

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

### Use from another application

Any MCP client that speaks streamable HTTP can consume the knowledge base by
connecting to `http://localhost:8080/mcp` — initialize a session, then call
the `kb_*` tools or read `kb://{slug}` resources. For example, with the
official Go SDK-style flow over raw JSON-RPC:

```bash
# Initialize (capture the Mcp-Session-Id response header)
curl -sD - http://localhost:8080/mcp \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"curl","version":"0"}}}'

# Then, passing -H "Mcp-Session-Id: <id>" on each call:
#   {"method":"notifications/initialized"}
#   {"id":2,"method":"tools/call","params":{"name":"kb_search","arguments":{"query":"welcome"}}}
#   {"id":3,"method":"resources/read","params":{"uri":"kb://welcome"}}
```
