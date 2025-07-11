package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
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

func streamTask(ctx worker.HatchetContext, input StreamTaskInput) (*StreamTaskOutput, error) {
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
		streamTask,
		hatchet,
	)
}

// > HTTP Proxy
func main() {
	hatchet, err := v1.NewHatchetClient()
	if err != nil {
		log.Fatalf("Failed to create Hatchet client: %v", err)
	}

	streamingWorkflow := StreamingWorkflow(hatchet)

	http.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		workflowRun, err := streamingWorkflow.RunNoWait(ctx, StreamTaskInput{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		stream, err := hatchet.Runs().SubscribeToStream(ctx, workflowRun.RunId())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		flusher, _ := w.(http.Flusher)
		for content := range stream {
			fmt.Fprint(w, content)
			if flusher != nil {
				flusher.Flush()
			}
		}
	})

	log.Fatal(http.ListenAndServe(":8000", nil))
}
