// Third-party integration - requires: go get github.com/sashabaranov/go-openai
// See: /guides/rag-and-indexing

package integrations

import (
	"context"
	"os"

	"github.com/sashabaranov/go-openai"
)

// > OpenAI embedding usage
func Embed(ctx context.Context, text string) ([]float32, error) {
	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	resp, err := client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Model: openai.AdaEmbeddingV2,
		Input: text,
	})
	if err != nil {
		return nil, err
	}
	return resp.Data[0].Embedding, nil
}

