package mcp

import (
	"context"
	"fmt"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerPrompts(s *server.MCPServer) {
	s.AddPrompt(mcpgo.NewPrompt(
		"kb_answer",
		mcpgo.WithPromptDescription("Answers a question using the knowledge base: search first, read the top hits, then answer citing entry slugs."),
		mcpgo.WithArgument("question",
			mcpgo.RequiredArgument(),
			mcpgo.ArgumentDescription("The question to answer from the knowledge base"),
		),
	), handleKBAnswer)
}

func handleKBAnswer(_ context.Context, req mcpgo.GetPromptRequest) (*mcpgo.GetPromptResult, error) {
	question := req.Params.Arguments["question"]
	if question == "" {
		return nil, fmt.Errorf("missing required argument: question")
	}
	instructions := fmt.Sprintf(`Answer the following question using this server's knowledge base.

Question: %s

Follow these steps:
1. Call the kb_search tool with keywords drawn from the question.
2. Read the full text of the top 1-3 results, either with the kb_get tool or by reading the kb://{slug} resource.
3. Answer the question based on what you read, citing the slugs of the entries you used.
4. If the search returns nothing relevant, say the knowledge base has no entry covering the question instead of answering from general knowledge.`, question)

	return mcpgo.NewGetPromptResult(
		"Answer a question from the knowledge base",
		[]mcpgo.PromptMessage{
			mcpgo.NewPromptMessage(mcpgo.RoleUser, mcpgo.NewTextContent(instructions)),
		},
	), nil
}
