package shared

import (
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
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

func StreamTask(ctx worker.HatchetContext, input StreamTaskInput) (*StreamTaskOutput, error) {
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

func StreamingWorkflow(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[StreamTaskInput, StreamTaskOutput] {
	return factory.NewTask(
		create.StandaloneTask{
			Name: "stream-example",
		},
		StreamTask,
		hatchet,
	)
}
