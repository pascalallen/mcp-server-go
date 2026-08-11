package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/pascalallen/mcp-server-go/internal/kb"
)

// Error convention for kb tools: problems the calling model can correct
// (missing arguments, invalid slugs, entries that don't exist) come back as
// tool results with isError set, via mcpgo.NewToolResultError. Unexpected
// infrastructure failures (disk I/O) are returned as Go errors and surface
// as JSON-RPC internal errors.

const kbURIPrefix = "kb://"

// Structured result types for the kb tools. Each doubles as the declared
// output schema for its tool, so the two can never drift apart.
type searchOutput struct {
	Results []kb.SearchResult `json:"results"`
}

type listOutput struct {
	Entries    []kb.Entry `json:"entries"`
	Total      int        `json:"total"`
	NextCursor string     `json:"next_cursor,omitempty"`
}

type deleteOutput struct {
	Slug    string `json:"slug"`
	Deleted bool   `json:"deleted"`
}

// kbAPI holds what the knowledge base handlers need: the store itself, and
// the MCP server so entries can be registered and removed as resources.
type kbAPI struct {
	store *kb.Store
	srv   *server.MCPServer
}

func registerKB(s *server.MCPServer, store *kb.Store) {
	k := &kbAPI{store: store, srv: s}

	tmpl := mcpgo.NewResourceTemplate(
		kbURIPrefix+"{slug}",
		"Knowledge base entry",
		mcpgo.WithTemplateDescription("A knowledge base entry, addressed by slug, as Markdown."),
		mcpgo.WithTemplateMIMEType("text/markdown"),
	)
	s.AddResourceTemplate(tmpl, k.readEntryResource)

	s.AddTool(mcpgo.NewTool(
		"kb_add",
		mcpgo.WithDescription("Adds a new knowledge base entry. The slug is derived from the title unless provided."),
		mcpgo.WithString("title", mcpgo.Required(), mcpgo.Description("Entry title")),
		mcpgo.WithString("body", mcpgo.Required(), mcpgo.Description("Entry body in Markdown")),
		mcpgo.WithArray("tags", mcpgo.WithStringItems(), mcpgo.Description("Tags for categorization and search")),
		mcpgo.WithString("slug", mcpgo.Description("Optional slug override (lowercase words separated by hyphens)")),
		mcpgo.WithDestructiveHintAnnotation(false),
		mcpgo.WithIdempotentHintAnnotation(false),
		mcpgo.WithOutputSchema[kb.Entry](),
	), k.handleAdd)

	s.AddTool(mcpgo.NewTool(
		"kb_get",
		mcpgo.WithDescription("Returns a knowledge base entry, including its full body."),
		mcpgo.WithString("slug", mcpgo.Required(), mcpgo.Description("Slug of the entry to fetch")),
		mcpgo.WithReadOnlyHintAnnotation(true),
		mcpgo.WithOutputSchema[kb.Entry](),
	), k.handleGet)

	s.AddTool(mcpgo.NewTool(
		"kb_update",
		mcpgo.WithDescription("Updates a knowledge base entry. Only the provided fields change; passing an empty tags array clears the tags."),
		mcpgo.WithString("slug", mcpgo.Required(), mcpgo.Description("Slug of the entry to update")),
		mcpgo.WithString("title", mcpgo.Description("New title")),
		mcpgo.WithString("body", mcpgo.Description("New body in Markdown")),
		mcpgo.WithArray("tags", mcpgo.WithStringItems(), mcpgo.Description("New tags")),
		mcpgo.WithIdempotentHintAnnotation(true),
		mcpgo.WithOutputSchema[kb.Entry](),
	), k.handleUpdate)

	s.AddTool(mcpgo.NewTool(
		"kb_delete",
		mcpgo.WithDescription("Deletes a knowledge base entry."),
		mcpgo.WithString("slug", mcpgo.Required(), mcpgo.Description("Slug of the entry to delete")),
		mcpgo.WithDestructiveHintAnnotation(true),
		mcpgo.WithIdempotentHintAnnotation(true),
		mcpgo.WithOutputSchema[deleteOutput](),
	), k.handleDelete)

	s.AddTool(mcpgo.NewTool(
		"kb_search",
		mcpgo.WithDescription("Searches knowledge base entries by keyword over titles, tags, and bodies. Returns scored results with snippets."),
		mcpgo.WithString("query", mcpgo.Required(), mcpgo.Description("Keywords to search for")),
		mcpgo.WithNumber("limit", mcpgo.Description(fmt.Sprintf("Maximum number of results (default %d, max %d)", kb.DefaultSearchLimit, kb.MaxSearchLimit))),
		mcpgo.WithReadOnlyHintAnnotation(true),
		mcpgo.WithOutputSchema[searchOutput](),
	), k.handleSearch)

	s.AddTool(mcpgo.NewTool(
		"kb_list",
		mcpgo.WithDescription("Lists knowledge base entries (metadata only, no bodies), paginated by slug."),
		mcpgo.WithString("cursor", mcpgo.Description("Opaque pagination cursor from a previous kb_list result; omit for the first page")),
		mcpgo.WithNumber("limit", mcpgo.Description(fmt.Sprintf("Maximum entries per page (default %d, max %d)", kb.DefaultListLimit, kb.MaxListLimit))),
		mcpgo.WithReadOnlyHintAnnotation(true),
		mcpgo.WithOutputSchema[listOutput](),
	), k.handleList)

	for _, e := range store.List() {
		k.addEntryResource(e)
	}
}

// addEntryResource registers (or re-registers, refreshing metadata) an entry
// as a concrete MCP resource so it shows up in resources/list. AddResource
// also notifies connected clients that the resource list changed.
func (k *kbAPI) addEntryResource(e kb.Entry) {
	k.srv.AddResource(mcpgo.NewResource(
		kbURIPrefix+e.Slug,
		e.Title,
		mcpgo.WithResourceDescription("tags: "+strings.Join(e.Tags, ", ")),
		mcpgo.WithMIMEType("text/markdown"),
	), k.readEntryResource)
}

// readEntryResource serves both template-matched reads (slug in Arguments)
// and reads of directly-registered resources (slug from the URI).
func (k *kbAPI) readEntryResource(_ context.Context, req mcpgo.ReadResourceRequest) ([]mcpgo.ResourceContents, error) {
	slug, _ := req.Params.Arguments["slug"].(string)
	if slug == "" {
		slug = strings.TrimPrefix(req.Params.URI, kbURIPrefix)
	}
	e, ok := k.store.Get(slug)
	if !ok {
		return nil, fmt.Errorf("%w: %s", kb.ErrNotFound, slug)
	}
	return []mcpgo.ResourceContents{mcpgo.TextResourceContents{
		URI:      kbURIPrefix + e.Slug,
		MIMEType: "text/markdown",
		Text:     e.Body,
	}}, nil
}

func (k *kbAPI) handleAdd(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	title, err := req.RequireString("title")
	if err != nil {
		return mcpgo.NewToolResultError(err.Error()), nil
	}
	body, err := req.RequireString("body")
	if err != nil {
		return mcpgo.NewToolResultError(err.Error()), nil
	}
	e, err := k.store.Add(req.GetString("slug", ""), title, body, req.GetStringSlice("tags", nil))
	if err != nil {
		if errors.Is(err, kb.ErrExists) || errors.Is(err, kb.ErrInvalidSlug) {
			return mcpgo.NewToolResultError(err.Error()), nil
		}
		return nil, err
	}
	k.addEntryResource(e)
	e.Body = ""
	return mcpgo.NewToolResultStructured(e, "Added "+kbURIPrefix+e.Slug), nil
}

func (k *kbAPI) handleGet(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	slug, err := req.RequireString("slug")
	if err != nil {
		return mcpgo.NewToolResultError(err.Error()), nil
	}
	e, ok := k.store.Get(slug)
	if !ok {
		return mcpgo.NewToolResultError(fmt.Sprintf("kb: entry not found: %s", slug)), nil
	}
	return mcpgo.NewToolResultStructured(e, fmt.Sprintf("%s (%s)\n\n%s", e.Title, e.Slug, e.Body)), nil
}

func (k *kbAPI) handleUpdate(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	slug, err := req.RequireString("slug")
	if err != nil {
		return mcpgo.NewToolResultError(err.Error()), nil
	}
	// Distinguish "not provided" from zero values by checking argument
	// presence, so an empty tags array clears tags while omitting the
	// field leaves them alone.
	args := req.GetArguments()
	var upd kb.Update
	if _, ok := args["title"]; ok {
		title := req.GetString("title", "")
		upd.Title = &title
	}
	if _, ok := args["body"]; ok {
		body := req.GetString("body", "")
		upd.Body = &body
	}
	if _, ok := args["tags"]; ok {
		tags := req.GetStringSlice("tags", []string{})
		upd.Tags = &tags
	}
	e, err := k.store.Update(slug, upd)
	if err != nil {
		if errors.Is(err, kb.ErrNotFound) {
			return mcpgo.NewToolResultError(err.Error()), nil
		}
		return nil, err
	}
	k.addEntryResource(e) // refresh title/tags shown in resources/list
	e.Body = ""
	return mcpgo.NewToolResultStructured(e, "Updated "+kbURIPrefix+e.Slug), nil
}

func (k *kbAPI) handleDelete(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	slug, err := req.RequireString("slug")
	if err != nil {
		return mcpgo.NewToolResultError(err.Error()), nil
	}
	if err := k.store.Delete(slug); err != nil {
		if errors.Is(err, kb.ErrNotFound) {
			return mcpgo.NewToolResultError(err.Error()), nil
		}
		return nil, err
	}
	k.srv.RemoveResource(kbURIPrefix + slug)
	return mcpgo.NewToolResultStructured(deleteOutput{Slug: slug, Deleted: true}, "Deleted "+kbURIPrefix+slug), nil
}

func (k *kbAPI) handleSearch(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return mcpgo.NewToolResultError(err.Error()), nil
	}
	results := k.store.Search(query, req.GetInt("limit", kb.DefaultSearchLimit))
	fallback := fmt.Sprintf("%d result(s) for %q", len(results), query)
	for _, r := range results {
		fallback += fmt.Sprintf("\n- %s%s (%s, score %d): %s", kbURIPrefix, r.Entry.Slug, r.Entry.Title, r.Score, r.Snippet)
	}
	return mcpgo.NewToolResultStructured(searchOutput{Results: results}, fallback), nil
}

func (k *kbAPI) handleList(_ context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	entries, total, nextCursor := k.store.ListPage(req.GetString("cursor", ""), req.GetInt("limit", 0))
	for i := range entries {
		entries[i].Body = ""
	}
	fallback := fmt.Sprintf("%d of %d entrie(s)", len(entries), total)
	for _, e := range entries {
		fallback += fmt.Sprintf("\n- %s%s: %s", kbURIPrefix, e.Slug, e.Title)
	}
	if nextCursor != "" {
		fallback += fmt.Sprintf("\nmore available: pass cursor %q", nextCursor)
	}
	return mcpgo.NewToolResultStructured(listOutput{Entries: entries, Total: total, NextCursor: nextCursor}, fallback), nil
}
