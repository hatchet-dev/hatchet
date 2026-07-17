package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"

	_ "github.com/hatchet-dev/hatchet/embed"
)

type GreetInput struct {
	Name string `json:"name"`
}

type GreetOutput struct {
	Worker string `json:"worker"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	name := env("WORKER_NAME")
	databaseURL := env("DATABASE_URL")

	client, err := hatchet.NewClient(hatchet.WithEmbeddedPostgres(databaseURL))
	if err != nil {
		return err
	}
	defer func() { _ = client.Close(context.Background()) }()

	task := client.NewStandaloneTask("greet", func(ctx hatchet.Context, input GreetInput) (GreetOutput, error) {
		time.Sleep(300 * time.Millisecond)
		return GreetOutput{Worker: name}, nil
	})

	worker, err := client.NewWorker(name, hatchet.WithWorkflows(task), hatchet.WithSlots(1))
	if err != nil {
		return err
	}

	cleanup, err := worker.Start()
	if err != nil {
		return err
	}
	defer func() { _ = cleanup() }()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	return nil
}

func env(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s is not set", key)
	}
	return v
}
