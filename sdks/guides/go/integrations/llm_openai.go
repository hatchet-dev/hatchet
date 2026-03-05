// Third-party integration - requires: go get github.com/sashabaranov/go-openai
// See: /guides/ai-agents

package integrations

import (
	"context"
	"encoding/json"
	"os"

	"github.com/sashabaranov/go-openai"
)

// > OpenAI usage
func Complete(ctx context.Context, messages []openai.ChatCompletionMessage) (map[string]interface{}, error) {
	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    openai.GPT4oMini,
		Messages: messages,
	})
	if err != nil {
		return nil, err
	}
	msg := resp.Choices[0].Message
	toolCalls := make([]map[string]interface{}, 0)
	for _, tc := range msg.ToolCalls {
		var args map[string]interface{}
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		toolCalls = append(toolCalls, map[string]interface{}{"name": tc.Function.Name, "args": args})
	}
	return map[string]interface{}{
		"content":    msg.Content,
		"tool_calls": toolCalls,
		"done":       len(toolCalls) == 0,
	}, nil
}

// !!
