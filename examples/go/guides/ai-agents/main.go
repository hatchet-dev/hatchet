package main

import (
	"log"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// > Step 02 Reasoning Loop
	agentReasoningLoop := func(query string) (map[string]interface{}, error) {
		messages := []map[string]interface{}{{"role": "user", "content": query}}
		for i := 0; i < 10; i++ {
			resp := CallLLM(messages)
			if resp.Done {
				return map[string]interface{}{"response": resp.Content}, nil
			}
			for _, tc := range resp.ToolCalls {
				args := make(map[string]interface{})
				for k, v := range tc.Args {
					args[k] = v
				}
				result := RunTool(tc.Name, args)
				messages = append(messages, map[string]interface{}{"role": "tool", "content": result})
			}
		}
		return map[string]interface{}{"response": "Max iterations reached"}, nil
	}

	// > Step 01 Define Agent Task
	agentTask := client.NewStandaloneDurableTask("agent-task", func(ctx hatchet.DurableContext, input map[string]interface{}) (map[string]interface{}, error) {
		query := "Hello"
		if q, ok := input["query"].(string); ok && q != "" {
			query = q
		}
		return agentReasoningLoop(query)
	})

	// > Step 03 Stream Response
	streamingTask := client.NewStandaloneDurableTask("streaming-agent-task", func(ctx hatchet.DurableContext, input map[string]interface{}) (map[string]interface{}, error) {
		tokens := []string{"Hello", " ", "world", "!"}
		for _, t := range tokens {
			ctx.PutStream(t)
		}
		return map[string]interface{}{"done": true}, nil
	})

	// > Step 04 Run Worker
	worker, err := client.NewWorker("agent-worker",
		hatchet.WithWorkflows(agentTask, streamingTask),
		hatchet.WithSlots(5),
		hatchet.WithDurableSlots(5),
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	if err := worker.StartBlocking(interruptCtx); err != nil {
		cancel()
		log.Fatalf("failed to start worker: %v", err)
	}
}
