package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/pascalallen/mcp-server-go/internal/kb"
)

func newTestKB(t *testing.T) (*kbAPI, string) {
	t.Helper()
	dir := t.TempDir()
	store, err := kb.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	s := server.NewMCPServer("test", "0.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(false, true),
	)
	return &kbAPI{store: store, srv: s}, dir
}

func callReq(name string, args map[string]any) mcpgo.CallToolRequest {
	return mcpgo.CallToolRequest{Params: mcpgo.CallToolParams{Name: name, Arguments: args}}
}

func TestHandleAddGetDelete(t *testing.T) {
	k, dir := newTestKB(t)

	res, err := k.handleAdd(context.Background(), callReq("kb_add", map[string]any{
		"title": "Test Entry",
		"body":  "Hello **world**.",
		"tags":  []any{"demo"},
	}))
	if err != nil || res.IsError {
		t.Fatalf("kb_add: err=%v result=%+v", err, res)
	}
	added, ok := res.StructuredContent.(kb.Entry)
	if !ok || added.Slug != "test-entry" || added.Body != "" {
		t.Fatalf("kb_add structured content = %#v", res.StructuredContent)
	}
	if _, err := os.Stat(filepath.Join(dir, "test-entry.md")); err != nil {
		t.Errorf("entry file not created: %v", err)
	}

	res, err = k.handleGet(context.Background(), callReq("kb_get", map[string]any{"slug": "test-entry"}))
	if err != nil || res.IsError {
		t.Fatalf("kb_get: err=%v result=%+v", err, res)
	}
	got := res.StructuredContent.(kb.Entry)
	if got.Body != "Hello **world**." || got.Title != "Test Entry" {
		t.Errorf("kb_get entry = %+v", got)
	}

	res, err = k.handleDelete(context.Background(), callReq("kb_delete", map[string]any{"slug": "test-entry"}))
	if err != nil || res.IsError {
		t.Fatalf("kb_delete: err=%v result=%+v", err, res)
	}
	if del, ok := res.StructuredContent.(deleteOutput); !ok || !del.Deleted || del.Slug != "test-entry" {
		t.Errorf("kb_delete structured content = %#v", res.StructuredContent)
	}
	if _, err := os.Stat(filepath.Join(dir, "test-entry.md")); !os.IsNotExist(err) {
		t.Error("entry file still exists after kb_delete")
	}
}

func TestHandleAddErrors(t *testing.T) {
	k, _ := newTestKB(t)

	res, err := k.handleAdd(context.Background(), callReq("kb_add", map[string]any{"title": "No Body"}))
	if err != nil || !res.IsError {
		t.Errorf("missing body should be a tool error, got err=%v result=%+v", err, res)
	}

	if _, err := k.handleAdd(context.Background(), callReq("kb_add", map[string]any{
		"title": "Dup", "body": "x",
	})); err != nil {
		t.Fatal(err)
	}
	res, err = k.handleAdd(context.Background(), callReq("kb_add", map[string]any{
		"title": "Dup", "body": "y",
	}))
	if err != nil || !res.IsError {
		t.Errorf("duplicate slug should be a tool error, got err=%v result=%+v", err, res)
	}

	res, err = k.handleAdd(context.Background(), callReq("kb_add", map[string]any{
		"title": "Bad", "body": "z", "slug": "Not A Slug!",
	}))
	if err != nil || !res.IsError {
		t.Errorf("invalid slug should be a tool error, got err=%v result=%+v", err, res)
	}
}

func TestHandleUpdatePartial(t *testing.T) {
	k, _ := newTestKB(t)
	if _, err := k.handleAdd(context.Background(), callReq("kb_add", map[string]any{
		"title": "Original", "body": "body", "tags": []any{"a", "b"},
	})); err != nil {
		t.Fatal(err)
	}

	// Only the title changes; tags stay because the argument is absent.
	res, err := k.handleUpdate(context.Background(), callReq("kb_update", map[string]any{
		"slug": "original", "title": "Renamed",
	}))
	if err != nil || res.IsError {
		t.Fatalf("kb_update: err=%v result=%+v", err, res)
	}
	e, _ := k.store.Get("original")
	if e.Title != "Renamed" || len(e.Tags) != 2 || e.Body != "body" {
		t.Errorf("after partial update: %+v", e)
	}

	// An explicit empty tags array clears them.
	if _, err := k.handleUpdate(context.Background(), callReq("kb_update", map[string]any{
		"slug": "original", "tags": []any{},
	})); err != nil {
		t.Fatal(err)
	}
	e, _ = k.store.Get("original")
	if len(e.Tags) != 0 {
		t.Errorf("tags not cleared: %+v", e.Tags)
	}

	res, err = k.handleUpdate(context.Background(), callReq("kb_update", map[string]any{"slug": "missing"}))
	if err != nil || !res.IsError {
		t.Errorf("update of missing entry should be a tool error, got err=%v result=%+v", err, res)
	}
}

func TestHandleSearchAndList(t *testing.T) {
	k, _ := newTestKB(t)
	for _, args := range []map[string]any{
		{"title": "Widgets Guide", "body": "All about widgets."},
		{"title": "Gadgets Guide", "body": "All about gadgets."},
	} {
		if _, err := k.handleAdd(context.Background(), callReq("kb_add", args)); err != nil {
			t.Fatal(err)
		}
	}

	res, err := k.handleSearch(context.Background(), callReq("kb_search", map[string]any{"query": "widgets"}))
	if err != nil || res.IsError {
		t.Fatalf("kb_search: err=%v result=%+v", err, res)
	}
	sr := res.StructuredContent.(searchOutput)
	if len(sr.Results) != 1 || sr.Results[0].Entry.Slug != "widgets-guide" {
		t.Errorf("kb_search results = %+v", sr.Results)
	}

	res, err = k.handleList(context.Background(), callReq("kb_list", nil))
	if err != nil || res.IsError {
		t.Fatalf("kb_list: err=%v result=%+v", err, res)
	}
	lr := res.StructuredContent.(listOutput)
	if len(lr.Entries) != 2 || lr.Entries[0].Body != "" || lr.Total != 2 || lr.NextCursor != "" {
		t.Errorf("kb_list result = %+v", lr)
	}
}

func TestHandleListPagination(t *testing.T) {
	k, _ := newTestKB(t)
	for _, slug := range []string{"alpha", "bravo", "charlie"} {
		if _, err := k.handleAdd(context.Background(), callReq("kb_add", map[string]any{
			"title": slug, "body": "body", "slug": slug,
		})); err != nil {
			t.Fatal(err)
		}
	}

	res, err := k.handleList(context.Background(), callReq("kb_list", map[string]any{"limit": 2}))
	if err != nil || res.IsError {
		t.Fatalf("kb_list page 1: err=%v result=%+v", err, res)
	}
	page1 := res.StructuredContent.(listOutput)
	if len(page1.Entries) != 2 || page1.Total != 3 || page1.NextCursor != "bravo" {
		t.Fatalf("page 1 = %+v", page1)
	}

	res, err = k.handleList(context.Background(), callReq("kb_list", map[string]any{
		"limit": 2, "cursor": page1.NextCursor,
	}))
	if err != nil || res.IsError {
		t.Fatalf("kb_list page 2: err=%v result=%+v", err, res)
	}
	page2 := res.StructuredContent.(listOutput)
	if len(page2.Entries) != 1 || page2.Entries[0].Slug != "charlie" || page2.NextCursor != "" {
		t.Errorf("page 2 = %+v", page2)
	}
}

func TestReadEntryResource(t *testing.T) {
	k, _ := newTestKB(t)
	if _, err := k.handleAdd(context.Background(), callReq("kb_add", map[string]any{
		"title": "Doc", "body": "resource body",
	})); err != nil {
		t.Fatal(err)
	}

	// Template-matched read: slug arrives in Arguments.
	req := mcpgo.ReadResourceRequest{}
	req.Params.URI = "kb://doc"
	req.Params.Arguments = map[string]any{"slug": "doc"}
	contents, err := k.readEntryResource(context.Background(), req)
	if err != nil {
		t.Fatalf("readEntryResource: %v", err)
	}
	text := contents[0].(mcpgo.TextResourceContents)
	if text.Text != "resource body" || text.MIMEType != "text/markdown" || text.URI != "kb://doc" {
		t.Errorf("resource contents = %+v", text)
	}

	// Direct read: slug derived from the URI.
	req.Params.Arguments = nil
	if _, err := k.readEntryResource(context.Background(), req); err != nil {
		t.Errorf("direct read: %v", err)
	}

	req.Params.URI = "kb://missing"
	if _, err := k.readEntryResource(context.Background(), req); err == nil {
		t.Error("missing resource should error")
	}
}
