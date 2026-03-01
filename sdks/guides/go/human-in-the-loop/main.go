package main

import (
	"log"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type ApprovalInput struct {
	Action string `json:"action"`
	To     string `json:"to"`
}

type ApprovalOutput struct {
	Status string      `json:"status"`
	Action interface{} `json:"action,omitempty"`
	Reason string      `json:"reason,omitempty"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// > Step 01 Define Approval Task
	task := client.NewStandaloneDurableTask("approval-task", func(ctx hatchet.DurableContext, input ApprovalInput) (ApprovalOutput, error) {
		proposedAction := map[string]string{"action": "send_email", "to": "user@example.com"}
		event, err := ctx.WaitForEvent("approval:decision", "")
		if err != nil {
			return ApprovalOutput{}, err
		}
		var eventData map[string]interface{}
		if err := hatchet.EventInto(event, &eventData); err != nil {
			return ApprovalOutput{}, err
		}
		approved, _ := eventData["approved"].(bool)
		if approved {
			return ApprovalOutput{Status: "approved", Action: proposedAction}, nil
		}
		reason, _ := eventData["reason"].(string)
		return ApprovalOutput{Status: "rejected", Reason: reason}, nil
	})
	// !!

	// > Step 02 Wait For Event
	// Pause until the approval event is pushed. Worker slot is freed while waiting.
	_ = func(ctx hatchet.DurableContext, proposedAction map[string]string) (ApprovalOutput, error) {
		event, err := ctx.WaitForEvent("approval:decision", "")
		if err != nil {
			return ApprovalOutput{}, err
		}
		var eventData map[string]interface{}
		if err := hatchet.EventInto(event, &eventData); err != nil {
			return ApprovalOutput{}, err
		}
		approved, _ := eventData["approved"].(bool)
		if approved {
			return ApprovalOutput{Status: "approved", Action: proposedAction}, nil
		}
		reason, _ := eventData["reason"].(string)
		return ApprovalOutput{Status: "rejected", Reason: reason}, nil
	}
	// !!

	// > Step 04 Run Worker
	worker, err := client.NewWorker("human-in-the-loop-worker",
		hatchet.WithWorkflows(task),
		hatchet.WithDurableSlots(5),
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	go func() {
		if err := worker.StartBlocking(interruptCtx); err != nil {
			log.Fatalf("failed to start worker: %v", err)
		}
	}()
	// !!

	<-interruptCtx.Done()
}
