# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

An MCP (Model Context Protocol) server in Go, built on `github.com/mark3labs/mcp-go`. It exposes a file-backed knowledge base through MCP tools (`kb_*`), resources (`kb://{slug}`), and a `kb_answer` prompt, over either streamable HTTP or stdio.

## Commands

```bash
go build ./...                      # build everything
go test ./...                       # run all tests
go test ./internal/kb -run TestName # run a single test
go vet ./...                        # static checks
go run ./cmd                        # run server (streamable HTTP on 127.0.0.1:8080, endpoint /mcp)
go run ./cmd -transport stdio       # run server on stdio
```

Configuration is flags-over-env-vars (`-transport`/`MCP_TRANSPORT`, `-addr`/`MCP_ADDR`, `-kb-dir`/`KB_DIR`, plus env-only `MCP_ALLOWED_ORIGINS`, `MCP_ALLOWED_HOSTS`, `MCP_AUTH_TOKEN`). The full table is in README.md — keep the two in sync when adding options.

## Architecture

Transport-agnostic core, wired together in `cmd/main.go`:

- **`internal/kb`** — the knowledge base, with no MCP dependency. `Store` (store.go) holds an in-memory map of entries keyed by slug, guarded by an RWMutex; every write persists to a Markdown file (`<slug>.md` with YAML front matter) in the KB directory before returning. Files load once at startup (`kb.Open`); malformed files are skipped with a log, not a startup failure. entry.go handles slug validation/derivation and front-matter parsing; search.go implements keyword scoring with snippets.
- **`internal/mcp`** — MCP layer. `NewMCPServer` (server.go) registers everything; kb_tools.go defines the `kb_*` tool handlers plus the `kb://{slug}` resource template, prompts.go the `kb_answer` prompt, tools.go the standalone `echo`/`server_info` tools and the `Version` const. Handlers mirror each KB write into MCP resource registration (`AddResource`/`RemoveResource`) so clients get `listChanged` notifications.
- **`internal/routes`** — HTTP transport only (Gin). routes.go mounts the MCP handler at `/mcp`; security.go is middleware enforcing Origin and Host allowlists (DNS-rebinding protection per the MCP spec) and optional bearer-token auth. Stdio transport bypasses this package entirely.

## Conventions

- **Tool error convention** (documented in kb_tools.go): mistakes the calling model can correct (bad arguments, invalid slug, missing entry) return a tool result with `isError` via `mcpgo.NewToolResultError`; unexpected infrastructure failures (disk I/O) return Go errors, surfacing as JSON-RPC internal errors.
- **Structured output**: each `kb_*` tool declares an output schema derived from its Go result struct (e.g. `searchOutput`, `listOutput` in kb_tools.go), and returns structured content alongside a plain-text fallback. New tools should follow suit so schema and payload can't drift.
- **Stdio discipline**: in stdio mode stdout carries only JSON-RPC; all logging must go to stderr (main.go sets `log.SetOutput(os.Stderr)`).
- **Pagination/limits**: `kb_list` paginates by slug via `cursor`/`limit` (default 20, max 100); `kb_search` clamps limits (default 5, max 50). Clamp rather than error on out-of-range values.
- The README documents the tool list, configuration, and security model in detail — update it when changing any of those.
