package main

import (
	"log"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type MessageInput struct {
	Message string `json:"message"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// > Step 01 Classify Task
	classifyTask := client.NewStandaloneTask("classify-message", func(ctx hatchet.Context, input MessageInput) (map[string]interface{}, error) {
		return map[string]interface{}{"category": MockClassify(input.Message)}, nil
	})

	// > Step 02 Specialist Tasks
	supportTask := client.NewStandaloneTask("handle-support", func(ctx hatchet.Context, input MessageInput) (map[string]interface{}, error) {
		return map[string]interface{}{"response": MockReply(input.Message, "support"), "category": "support"}, nil
	})

	salesTask := client.NewStandaloneTask("handle-sales", func(ctx hatchet.Context, input MessageInput) (map[string]interface{}, error) {
		return map[string]interface{}{"response": MockReply(input.Message, "sales"), "category": "sales"}, nil
	})

	defaultTask := client.NewStandaloneTask("handle-default", func(ctx hatchet.Context, input MessageInput) (map[string]interface{}, error) {
		return map[string]interface{}{"response": MockReply(input.Message, "other"), "category": "other"}, nil
	})

	// > Step 03 Router Task
	routerTask := client.NewStandaloneDurableTask("message-router", func(ctx hatchet.DurableContext, input map[string]interface{}) (map[string]interface{}, error) {
		msg := input["message"].(string)
		classResult, err := classifyTask.Run(ctx, MessageInput{Message: msg})
		if err != nil {
			return nil, err
		}

		switch classResult["category"].(string) {
		case "support":
			return supportTask.Run(ctx, MessageInput{Message: msg})
		case "sales":
			return salesTask.Run(ctx, MessageInput{Message: msg})
		default:
			return defaultTask.Run(ctx, MessageInput{Message: msg})
		}
	})

	// > Step 04 Run Worker
	worker, err := client.NewWorker("routing-worker",
		hatchet.WithWorkflows(classifyTask, supportTask, salesTask, defaultTask, routerTask),
		hatchet.WithSlots(5),
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
}
