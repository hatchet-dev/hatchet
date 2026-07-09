// Runs an in-process no-auth Hatchet (engine + REST API + dashboard) via the embed package.
// Needs a local Postgres: DATABASE_URL=... go run ./sdks/go/examples/embedded
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hatchet-dev/hatchet/embed"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type GreetInput struct {
	Name string `json:"name"`
}

type GreetOutput struct {
	Greeting string `json:"greeting"`
}

func main() {
	ctx := context.Background()

	postgresURL := os.Getenv("DATABASE_URL")
	if postgresURL == "" {
		postgresURL = "postgres://hatchet:hatchet@127.0.0.1:5431/hatchet?sslmode=disable"
	}

	opts := []embed.Option{embed.WithPostgres(postgresURL)}

	if dir := os.Getenv("HATCHET_EMBEDDED_DASHBOARD_DIR"); dir != "" {
		opts = append(opts, embed.WithDashboardDir(dir))
	}

	inst, err := embed.Start(ctx, opts...)
	if err != nil {
		log.Fatalf("failed to start embedded hatchet: %v", err)
	}
	defer inst.Shutdown(context.Background())

	client := inst.Client()

	task := client.NewStandaloneTask("greet", func(ctx hatchet.Context, input GreetInput) (GreetOutput, error) {
		return GreetOutput{Greeting: "Hello, " + input.Name + "!"}, nil
	})

	worker, err := client.NewWorker("embedded-worker", hatchet.WithWorkflows(task))
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	cleanup, err := worker.Start()
	if err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
	defer cleanup() // nolint:errcheck

	time.Sleep(2 * time.Second)

	result, err := task.Run(ctx, GreetInput{Name: "embedded"})
	if err != nil {
		log.Fatalf("failed to run task: %v", err)
	}

	var out GreetOutput
	if err := result.Into(&out); err != nil {
		log.Fatalf("failed to decode result: %v", err)
	}

	fmt.Println("task output:", out.Greeting)
	fmt.Println("dashboard:  ", inst.DashboardURL())
	fmt.Println("api:        ", inst.APIURL())
	fmt.Println("tenant:     ", inst.TenantID())
	fmt.Println()
	fmt.Println("Instance is running. Open the dashboard above to view the run. Press Ctrl+C to stop.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}
