package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/mcp"

	"github.com/pascalallen/mcp-server-go/internal/kb"
)

// Every kb_* tool returns structured content, so each must declare the
// matching output schema.
func TestKBToolsDeclareOutputSchema(t *testing.T) {
	store, err := kb.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := NewMCPServer(store)
	resp := s.HandleMessage(context.Background(), json.RawMessage(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Result struct {
			Tools []struct {
				Name         string          `json:"name"`
				OutputSchema json.RawMessage `json:"outputSchema"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unmarshal tools/list response: %v\n%s", err, raw)
	}
	if len(parsed.Result.Tools) == 0 {
		t.Fatalf("no tools in response: %s", raw)
	}
	kbTools := 0
	for _, tool := range parsed.Result.Tools {
		if !strings.HasPrefix(tool.Name, "kb_") {
			continue
		}
		kbTools++
		if len(tool.OutputSchema) == 0 {
			t.Errorf("tool %s has no output schema", tool.Name)
		}
	}
	if kbTools != 6 {
		t.Errorf("found %d kb_* tools, want 6", kbTools)
	}
}

func TestKBAnswerPrompt(t *testing.T) {
	req := mcpgo.GetPromptRequest{}
	req.Params.Name = "kb_answer"
	req.Params.Arguments = map[string]string{"question": "How do I add an entry?"}
	res, err := handleKBAnswer(context.Background(), req)
	if err != nil {
		t.Fatalf("handleKBAnswer: %v", err)
	}
	if len(res.Messages) != 1 || res.Messages[0].Role != mcpgo.RoleUser {
		t.Fatalf("messages = %+v", res.Messages)
	}
	text, ok := res.Messages[0].Content.(mcpgo.TextContent)
	if !ok {
		t.Fatalf("content = %#v", res.Messages[0].Content)
	}
	for _, want := range []string{"kb_search", "kb_get", "How do I add an entry?"} {
		if !strings.Contains(text.Text, want) {
			t.Errorf("prompt text missing %q:\n%s", want, text.Text)
		}
	}

	req.Params.Arguments = nil
	if _, err := handleKBAnswer(context.Background(), req); err == nil {
		t.Error("missing question should error")
	}
}
