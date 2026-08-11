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
| `kb_search` | Searches entries by keyword over titles, tags, and bodies. Returns scored results with snippets (default 5, max 50). |
| `kb_list` | Lists knowledge base entries (metadata only), paginated by slug via `cursor`/`limit` (default 20, max 100 per page). |

All `kb_*` tools declare output schemas and return structured content
alongside a plain-text fallback.

## Knowledge base

The server ships with a file-backed knowledge base. Each entry is a Markdown
file with YAML front matter, stored in `data/kb` (override with `-kb-dir` or
the `KB_DIR` environment variable; the path is resolved to an absolute path
and logged at startup). The filename is the entry's slug:

```markdown
---
title: Getting started with MCP
tags: [mcp, tutorial]
created: 2026-08-07T12:00:00Z
updated: 2026-08-07T12:30:00Z
---

Body markdown starts here...
```

Files on disk load at startup, and writes made through the `kb_*` tools
persist back to disk. Note that disk edits made while the server is running
are not picked up until restart.

### Resources

Every entry is also exposed as an MCP resource at `kb://{slug}`
(`text/markdown`), so clients can browse the knowledge base with
`resources/list` and read entries with `resources/read` without calling a
tool. The server announces `listChanged`, so connected clients are notified
when entries are added, updated, or deleted.

### Prompts

The `kb_answer` prompt takes a required `question` argument and returns
instructions for answering it from the knowledge base: search with
`kb_search`, read the top hits in full, then answer citing entry slugs — or
say the knowledge base doesn't cover it.

## Configuration

Flags win over environment variables; both fall back to the defaults below.

| Flag | Environment variable | Default | Description |
|------|----------------------|---------|-------------|
| `-transport` | `MCP_TRANSPORT` | `http` | Transport to serve: `http` or `stdio` |
| `-addr` | `MCP_ADDR` | `127.0.0.1:8080` | Listen address (http transport) |
| `-kb-dir` | `KB_DIR` | `data/kb` | Knowledge base directory |
| — | `MCP_ALLOWED_ORIGINS` | *(empty)* | Comma-separated extra allowed `Origin` values; localhost origins are always allowed |
| — | `MCP_ALLOWED_HOSTS` | `localhost,127.0.0.1,::1` | Comma-separated `Host` header allowlist; override it when binding beyond loopback |
| — | `MCP_AUTH_TOKEN` | *(empty)* | When set, requires `Authorization: Bearer <token>` on every request |

### Security

The HTTP transport binds to loopback (`127.0.0.1`) by default and applies
three checks to every request on `/mcp`, per the MCP spec's guidance for
local HTTP servers (DNS-rebinding protection):

1. **Origin**: requests without an `Origin` header (normal MCP clients) are
   allowed. Browser requests must come from a localhost origin or one listed
   in `MCP_ALLOWED_ORIGINS`; anything else gets `403`.
2. **Host**: the `Host` header (port ignored) must appear in
   `MCP_ALLOWED_HOSTS`; anything else gets `403`.
3. **Bearer token** (optional): with `MCP_AUTH_TOKEN` set, requests without
   a matching `Authorization: Bearer` header get `401`.

To expose the server beyond your machine deliberately, set
`-addr 0.0.0.0:8080`, add your hostname to `MCP_ALLOWED_HOSTS`, and set
`MCP_AUTH_TOKEN` — and put TLS in front of it.

## Prerequisites

- [Go](https://go.dev/dl/)

## MCP server usage

### Run the MCP server from project root

```bash
go run ./cmd                     # streamable HTTP on 127.0.0.1:8080
go run ./cmd -transport stdio    # stdio, for clients that spawn the server
```

In stdio mode all logging goes to stderr; stdout carries only JSON-RPC.

### Install the MCP server in Claude Code

```bash
# HTTP (run the server yourself first):
claude mcp add --transport http mcp-server-go http://localhost:8080/mcp

# Or stdio (Claude Code spawns the server; build it first with `go build -o mcp-server-go ./cmd`):
claude mcp add mcp-server-go -- /path/to/mcp-server-go -transport stdio
```

### Use from another application

Any MCP client that speaks streamable HTTP can consume the knowledge base by
connecting to `http://127.0.0.1:8080/mcp` — initialize a session, then call
the `kb_*` tools or read `kb://{slug}` resources. For example, with raw
JSON-RPC:

```bash
# Initialize (capture the Mcp-Session-Id response header)
curl -sD - http://127.0.0.1:8080/mcp \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"curl","version":"0"}}}'

# Then, passing -H "Mcp-Session-Id: <id>" on each call:
#   {"method":"notifications/initialized"}
#   {"id":2,"method":"tools/call","params":{"name":"kb_search","arguments":{"query":"welcome"}}}
#   {"id":3,"method":"tools/call","params":{"name":"kb_list","arguments":{"limit":2}}}
#   {"id":4,"method":"resources/read","params":{"uri":"kb://welcome"}}
#   {"id":5,"method":"prompts/get","params":{"name":"kb_answer","arguments":{"question":"What is this server?"}}}
```
