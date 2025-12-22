package shared

import (
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type StreamTaskInput struct{}

type StreamTaskOutput struct {
	Message string `json:"message"`
}

// > Streaming
const annaKarenina = `
Happy families are all alike; every unhappy family is unhappy in its own way.

Everything was in confusion in the Oblonskys' house. The wife had discovered that the husband was carrying on an intrigue with a French girl, who had been a governess in their family, and she had announced to her husband that she could not go on living in the same house with him.
`

func createChunks(content string, n int) []string {
	var chunks []string
	for i := 0; i < len(content); i += n {
		end := i + n
		if end > len(content) {
			end = len(content)
		}
		chunks = append(chunks, content[i:end])
	}
	return chunks
}

func StreamTask(ctx hatchet.Context, input StreamTaskInput) (*StreamTaskOutput, error) {
	time.Sleep(2 * time.Second)

	chunks := createChunks(annaKarenina, 10)

	for _, chunk := range chunks {
		ctx.PutStream(chunk)
		time.Sleep(200 * time.Millisecond)
	}

	return &StreamTaskOutput{
		Message: "Streaming completed",
	}, nil
}


func StreamingWorkflow(client *hatchet.Client) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("stream-example", StreamTask)
}
