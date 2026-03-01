package main

import (
	"log"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type DocInput struct {
	DocID   string `json:"doc_id"`
	Content string `json:"content"`
}

// embed is a mock - no external embedding API.
func embed(text string) []float64 {
	vec := make([]float64, 64)
	for i := range vec {
		vec[i] = 0.1
	}
	return vec
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// > Step 01 Define Ingest Task
	workflow := client.NewWorkflow("RAGPipeline")
	// !!

	ingest := workflow.NewTask("ingest", func(ctx hatchet.Context, input DocInput) (map[string]interface{}, error) {
		return map[string]interface{}{"doc_id": input.DocID, "content": input.Content}, nil
	})

	// > Step 02 Chunk Task
	chunkContent := func(content string, chunkSize int) []string {
		var chunks []string
		for i := 0; i < len(content); i += chunkSize {
			end := i + chunkSize
			if end > len(content) {
				end = len(content)
			}
			chunks = append(chunks, content[i:end])
		}
		return chunks
	}
	_ = chunkContent
	// !!

	// > Step 03 Embed Task
	chunkAndEmbed := workflow.NewTask("chunk-and-embed", func(ctx hatchet.Context, input DocInput) (map[string]interface{}, error) {
		var ingested map[string]interface{}
		if err := ctx.ParentOutput(ingest, &ingested); err != nil {
			return nil, err
		}
		content := ingested["content"].(string)
		var chunks []string
		for i := 0; i < len(content); i += 100 {
			end := i + 100
			if end > len(content) {
				end = len(content)
			}
			chunks = append(chunks, content[i:end])
		}
		vectors := make([][]float64, len(chunks))
		for i, c := range chunks {
			vectors[i] = embed(c)
		}
		return map[string]interface{}{"doc_id": ingested["doc_id"], "vectors": vectors}, nil
	}, hatchet.WithParents(ingest))
	// !!

	_ = chunkAndEmbed

	// > Step 04 Run Worker
	worker, err := client.NewWorker("rag-worker", hatchet.WithWorkflows(workflow))
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	if err := worker.StartBlocking(interruptCtx); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
	// !!
}
