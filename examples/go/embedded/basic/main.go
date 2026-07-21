package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"

	_ "github.com/hatchet-dev/hatchet/embed"
)

type GreetInput struct {
	Name string `json:"name"`
}

type GreetOutput struct {
	Greeting string `json:"greeting"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is not set")
	}

	client, err := hatchet.NewClient(hatchet.WithEmbeddedPostgres(databaseURL))
	if err != nil {
		return err
	}
	defer func() { _ = client.Close(context.Background()) }()

	task := client.NewStandaloneTask("greet", func(ctx hatchet.Context, input GreetInput) (GreetOutput, error) {
		return GreetOutput{Greeting: "Hello, " + input.Name + "!"}, nil
	})

	worker, err := client.NewWorker("basic-worker", hatchet.WithWorkflows(task))
	if err != nil {
		return err
	}

	cleanup, err := worker.Start()
	if err != nil {
		return err
	}
	defer func() { _ = cleanup() }()

	time.Sleep(2 * time.Second)

	result, err := task.Run(ctx, GreetInput{Name: "embed"})
	if err != nil {
		return err
	}

	var out GreetOutput
	if err := result.Into(&out); err != nil {
		return err
	}

	fmt.Println(out.Greeting)
	return nil
}
