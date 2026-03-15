package main

import (
	"log"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type DocInput struct {
	DocID   string `json:"doc_id"`
	Content []byte `json:"content"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// > Step 01 Define DAG
	workflow := client.NewWorkflow("DocumentPipeline")
	// !!

	// > Step 02 Parse Stage
	ingest := workflow.NewTask("ingest", func(ctx hatchet.Context, input DocInput) (map[string]interface{}, error) {
		return map[string]interface{}{"doc_id": input.DocID, "content": input.Content}, nil
	})

	parse := workflow.NewTask("parse", func(ctx hatchet.Context, input DocInput) (map[string]interface{}, error) {
		var ingested map[string]interface{}
		if err := ctx.ParentOutput(ingest, &ingested); err != nil {
			return nil, err
		}
		content := ingested["content"].([]byte)
		text := parseDocument(content)
		return map[string]interface{}{"doc_id": input.DocID, "text": text}, nil
	}, hatchet.WithParents(ingest))
	// !!

	// > Step 03 Extract Stage
	extract := workflow.NewTask("extract", func(ctx hatchet.Context, input DocInput) (map[string]interface{}, error) {
		var parsed map[string]interface{}
		if err := ctx.ParentOutput(parse, &parsed); err != nil {
			return nil, err
		}
		return map[string]interface{}{"doc_id": parsed["doc_id"], "entities": []string{"entity1", "entity2"}}, nil
	}, hatchet.WithParents(parse))
	// !!

	_ = extract

	// > Step 04 Run Worker
	worker, err := client.NewWorker("document-worker", hatchet.WithWorkflows(workflow))
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	if err := worker.StartBlocking(interruptCtx); err != nil {
		cancel()
		log.Fatalf("failed to start worker: %v", err)
	}
	// !!
}
