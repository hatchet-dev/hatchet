package main

import (
	"log"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// > Step 01 Define Streaming Task
	task := client.NewStandaloneTask("stream-example", func(ctx hatchet.Context, input map[string]interface{}) (map[string]string, error) {
		for i := 0; i < 5; i++ {
			ctx.PutStream("chunk-" + string(rune('0'+i)))
			time.Sleep(500 * time.Millisecond)
		}
		return map[string]string{"status": "done"}, nil
	})
	// !!

	// > Step 02 Emit Chunks
	emitChunks := func(ctx hatchet.Context) {
		for i := 0; i < 5; i++ {
			ctx.PutStream("chunk-" + string(rune('0'+i)))
			time.Sleep(500 * time.Millisecond)
		}
	}
	_ = emitChunks
	// !!

	// > Step 04 Run Worker
	worker, err := client.NewWorker("streaming-worker", hatchet.WithWorkflows(task))
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
