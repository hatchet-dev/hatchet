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

type ChunkInput struct {
	Chunk string `json:"chunk"`
}

type QueryInput struct {
	Query string `json:"query"`
	TopK  int    `json:"top_k"`
}

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

	// > Step 01 Define Workflow
	workflow := client.NewWorkflow("RAGPipeline")
	// !!

	// > Step 02 Define Ingest Task
	ingest := workflow.NewTask("ingest", func(ctx hatchet.Context, input DocInput) (map[string]interface{}, error) {
		return map[string]interface{}{"doc_id": input.DocID, "content": input.Content}, nil
	})
	// !!

	// > Step 03 Chunk Task
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

	// > Step 04 Embed Task
	embedChunkTask := client.NewStandaloneTask("embed-chunk", func(ctx hatchet.Context, input ChunkInput) (map[string]interface{}, error) {
		return map[string]interface{}{"vector": embed(input.Chunk)}, nil
	})

	chunkAndEmbed := workflow.NewDurableTask("chunk-and-embed", func(ctx hatchet.DurableContext, input DocInput) (map[string]interface{}, error) {
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
		inputs := make([]hatchet.RunManyOpt, len(chunks))
		for i, c := range chunks {
			inputs[i] = hatchet.RunManyOpt{Input: ChunkInput{Chunk: c}}
		}
		runRefs, err := embedChunkTask.RunMany(ctx, inputs)
		if err != nil {
			return nil, err
		}
		vectors := make([]interface{}, len(runRefs))
		for i, ref := range runRefs {
			result, err := ref.Result()
			if err != nil {
				return nil, err
			}
			var parsed map[string]interface{}
			if err := result.TaskOutput("embed-chunk").Into(&parsed); err != nil {
				return nil, err
			}
			vectors[i] = parsed["vector"]
		}
		return map[string]interface{}{"doc_id": ingested["doc_id"], "vectors": vectors}, nil
	}, hatchet.WithParents(ingest))
	// !!

	_ = chunkAndEmbed

	// > Step 05 Query Task
	queryTask := client.NewStandaloneDurableTask("rag-query", func(ctx hatchet.DurableContext, input QueryInput) (map[string]interface{}, error) {
		res, err := embedChunkTask.Run(ctx, ChunkInput{Chunk: input.Query})
		if err != nil {
			return nil, err
		}
		var parsed map[string]interface{}
		if err := res.Into(&parsed); err != nil {
			return nil, err
		}
		// Replace with a real vector DB lookup in production
		return map[string]interface{}{"query": input.Query, "vector": parsed["vector"], "results": []interface{}{}}, nil
	})
	// !!

	// > Step 06 Run Worker
	worker, err := client.NewWorker("rag-worker", hatchet.WithWorkflows(workflow, embedChunkTask, queryTask))
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
