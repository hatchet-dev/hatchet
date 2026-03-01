package main

import (
	"log"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type PipelineInput struct {
	Prompt string `json:"prompt"`
}

// generate is a mock - no external LLM API.
func generate(prompt string) map[string]interface{} {
	n := 50
	if len(prompt) < n {
		n = len(prompt)
	}
	return map[string]interface{}{"content": "Generated for: " + prompt[:n] + "...", "valid": true}
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// > Step 01 Define Pipeline
	workflow := client.NewWorkflow("LLMPipeline")
	// !!

	// > Step 02 Prompt Task
	buildPrompt := func(userInput, context string) string {
		if context != "" {
			return "Process the following: " + userInput + "\nContext: " + context
		}
		return "Process the following: " + userInput
	}
	_ = buildPrompt
	// !!

	promptTask := workflow.NewTask("prompt-task", func(ctx hatchet.Context, input PipelineInput) (map[string]interface{}, error) {
		return map[string]interface{}{"prompt": input.Prompt}, nil
	})

	// > Step 03 Validate Task
	generateTask := workflow.NewTask("generate-task", func(ctx hatchet.Context, input PipelineInput) (map[string]interface{}, error) {
		var prev map[string]interface{}
		if err := ctx.ParentOutput(promptTask, &prev); err != nil {
			return nil, err
		}
		output := generate(prev["prompt"].(string))
		if !output["valid"].(bool) {
			panic("validation failed")
		}
		return output, nil
	}, hatchet.WithParents(promptTask))
	// !!

	_ = generateTask

	// > Step 04 Run Worker
	worker, err := client.NewWorker("llm-pipeline-worker", hatchet.WithWorkflows(workflow))
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
