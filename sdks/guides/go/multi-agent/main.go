package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type SpecialistInput struct {
	Task    string `json:"task"`
	Context string `json:"context,omitempty"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// > Step 01 Specialist Agents
	researchTask := client.NewStandaloneDurableTask("research-specialist", func(ctx hatchet.DurableContext, input SpecialistInput) (map[string]interface{}, error) {
		return map[string]interface{}{"result": MockSpecialistLLM(input.Task, "research")}, nil
	})

	writingTask := client.NewStandaloneDurableTask("writing-specialist", func(ctx hatchet.DurableContext, input SpecialistInput) (map[string]interface{}, error) {
		return map[string]interface{}{"result": MockSpecialistLLM(input.Task, "writing")}, nil
	})

	codeTask := client.NewStandaloneDurableTask("code-specialist", func(ctx hatchet.DurableContext, input SpecialistInput) (map[string]interface{}, error) {
		return map[string]interface{}{"result": MockSpecialistLLM(input.Task, "code")}, nil
	})
	// !!

	specialists := map[string]*hatchet.StandaloneTask{
		"research": researchTask,
		"writing":  writingTask,
		"code":     codeTask,
	}

	// > Step 02 Orchestrator Loop
	orchestrator := client.NewStandaloneDurableTask("multi-agent-orchestrator", func(ctx hatchet.DurableContext, input map[string]interface{}) (map[string]interface{}, error) {
		messages := []map[string]interface{}{{"role": "user", "content": input["goal"].(string)}}

		for i := 0; i < 10; i++ {
			response := MockOrchestratorLLM(messages)

			if response.Done {
				return map[string]interface{}{"result": response.Content}, nil
			}

			specialist, ok := specialists[response.ToolCall.Name]
			if !ok {
				return nil, fmt.Errorf("unknown specialist: %s", response.ToolCall.Name)
			}

			var contextParts []string
			for _, m := range messages {
				contextParts = append(contextParts, m["content"].(string))
			}

			taskResult, err := specialist.Run(ctx, SpecialistInput{
				Task:    response.ToolCall.Args["task"],
				Context: strings.Join(contextParts, "\n"),
			})
			if err != nil {
				return nil, err
			}
			var result map[string]interface{}
			if err := taskResult.Into(&result); err != nil {
				return nil, err
			}

			messages = append(messages,
				map[string]interface{}{"role": "assistant", "content": fmt.Sprintf("Called %s", response.ToolCall.Name)},
				map[string]interface{}{"role": "tool", "content": result["result"].(string)},
			)
		}

		return map[string]interface{}{"result": "Max iterations reached"}, nil
	})
	// !!

	// > Step 03 Run Worker
	worker, err := client.NewWorker("multi-agent-worker",
		hatchet.WithWorkflows(researchTask, writingTask, codeTask, orchestrator),
		hatchet.WithSlots(10),
		hatchet.WithDurableSlots(5),
	)
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
