package main

import (
	"log"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type DurableMemoInput struct{}

type DurableMemoOutput struct {
	Value string `json:"value"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	task := client.NewStandaloneDurableTask("durable-memo", func(ctx hatchet.DurableContext, input DurableMemoInput) (DurableMemoOutput, error) {
		result, err := ctx.Memo(func() (interface{}, error) {
			return "Hello, World!", nil
		}, []interface{}{
			`{"name": "John"}`,
		})
		if err != nil {
			return DurableMemoOutput{}, err
		}

		result, err = ctx.Memo(func() (interface{}, error) {
			return "Hello, World!", nil
		}, []interface{}{
			`{"name": "John"}`,
		})
		if err != nil {
			return DurableMemoOutput{}, err
		}

		return DurableMemoOutput{
			Value: result.(string),
		}, nil
	})

	worker, err := client.NewWorker("durable-memo-worker", hatchet.WithWorkflows(task))
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

	<-interruptCtx.Done()
}
