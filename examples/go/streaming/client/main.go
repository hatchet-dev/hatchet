package main

import (
	"context"
	"log"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type StreamTaskInput struct {
	Text string `json:"text"`
}

type StreamTaskOutput struct {
	Message string `json:"message"`
}

// > Streaming
const annaKarenina = `
Happy families are all alike; every unhappy family is unhappy in its own way.

Everything was in confusion in the Oblonskys' house. The wife had discovered that the husband was carrying on an intrigue with a French girl, who had been a governess in their family, and she had announced to her husband that she could not go on living in the same house with him.
`

// createChunks splits content into chunks of specified size
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

// StreamingTask demonstrates streaming data from a Hatchet workflow step
func streamingTaskFn(ctx worker.HatchetContext, input StreamTaskInput) (*StreamTaskOutput, error) {
	// Use provided text or default to Anna Karenina
	text := input.Text
	if text == "" {
		text = annaKarenina
	}
	
	// Sleep briefly to avoid race conditions (matching TypeScript example)
	time.Sleep(2 * time.Second)
	
	// Create chunks and stream them (matching TypeScript example)
	chunks := createChunks(text, 10)
	log.Printf("Starting to stream %d chunks", len(chunks))
	
	for i, chunk := range chunks {
		log.Printf("Streaming chunk %d: %q", i, chunk)
		ctx.StreamEvent([]byte(chunk))
		time.Sleep(200 * time.Millisecond)
	}
	
	log.Println("Finished streaming all chunks")
	
	return &StreamTaskOutput{
		Message: "Streaming completed",
	}, nil
}

func StreamingWorkflow(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[StreamTaskInput, StreamTaskOutput] {
	// Create a streaming task using the task factory
	streamingTask := factory.NewTask(
		create.StandaloneTask{
			Name: "stream-example",
		}, 
		streamingTaskFn,
		hatchet,
	)

	return streamingTask
}

func main() {
	// Create Hatchet client using v1 SDK
	hatchet, err := v1.NewHatchetClient()
	if err != nil {
		log.Fatalf("Failed to create Hatchet client: %v", err)
	}

	// Create the streaming workflow
	streamingWorkflow := StreamingWorkflow(hatchet)

	log.Println("Running streaming workflow...")
	
	// Run the streaming workflow
	workflowRun, err := streamingWorkflow.RunNoWait(context.Background(), StreamTaskInput{
		Text: "", // Use default Anna Karenina text
	})
	if err != nil {
		log.Fatalf("Failed to run workflow: %v", err)
	}

	log.Printf("Workflow started with run ID: %s", workflowRun.RunId())
	log.Println("Subscribing to stream events...")

	// Subscribe to the stream and log events
	err = hatchet.V0().Subscribe().Stream(context.Background(), workflowRun.RunId(), func(event client.StreamEvent) error {
		log.Printf("Received stream event: %q", string(event.Message))
		return nil
	})

	if err != nil {
		log.Fatalf("Stream error: %v", err)
	}

	log.Println("Stream completed!")
}