package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

// > Create an embedded client
import (
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"

	_ "github.com/hatchet-dev/hatchet-embedded"
)

type GreetInput struct {
	Name string `json:"name"`
}

type GreetOutput struct {
	Greeting string `json:"greeting"`
}

func configuredClient() (*hatchet.Client, error) {
	// > Configure the embedded engine
	client, err := hatchet.NewClient(hatchet.WithEmbedded(
		// use your own Postgres instead of the bundled one
		hatchet.WithEmbeddedDatabaseURL("postgres://..."),
		// use RabbitMQ instead of the Postgres message queue
		hatchet.WithEmbeddedRabbitMQ("amqp://..."),
		// bind the API / gRPC servers to specific ports
		hatchet.WithEmbeddedAPIPort(28243),
		hatchet.WithEmbeddedGRPCPort(7070),
		// start only the engine + gRPC, no REST API
		hatchet.WithoutEmbeddedAPI(),
		// skip running migrations on startup
		hatchet.WithoutEmbeddedMigrations(),
		// engine log level (default "warn")
		hatchet.WithEmbeddedLogLevel("info"),
	))
	return client, err
}

func fleetClient() (*hatchet.Client, error) {
	// > Fleet with a shared database
	client, err := hatchet.NewClient(hatchet.WithEmbedded(
		hatchet.WithEmbeddedDatabaseURL("postgres://user:pass@db.internal:5432/hatchet"),
	))
	return client, err
}

var _ = configuredClient
var _ = fleetClient

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()

	client, err := hatchet.NewClient(hatchet.WithEmbedded())
	if err != nil {
		return err
	}

	task := client.NewStandaloneTask("greet", func(ctx hatchet.Context, input GreetInput) (GreetOutput, error) {
		return GreetOutput{Greeting: "Hello, " + input.Name + "!"}, nil
	})

	worker, err := client.NewWorker("embedded-worker", hatchet.WithWorkflows(task))
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

	// > Stop the embedded engine
	return client.Close(ctx)
}
