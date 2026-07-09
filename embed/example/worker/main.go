package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type GreetInput struct {
	Name string `json:"name"`
}

type GreetOutput struct {
	Worker string `json:"worker"`
}

func main() {
	name := env("WORKER_NAME")

	info := readEngine(env("ENGINE_FILE"))

	_ = os.Setenv("HATCHET_CLIENT_TOKEN", info["token"])
	_ = os.Setenv("HATCHET_CLIENT_HOST_PORT", info["grpcAddress"])
	_ = os.Setenv("HATCHET_CLIENT_SERVER_URL", info["apiURL"])
	_ = os.Setenv("HATCHET_CLIENT_TENANT_ID", info["tenantID"])
	_ = os.Setenv("HATCHET_CLIENT_TLS_STRATEGY", "none")

	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("%s: could not create client: %v", name, err)
	}

	task := client.NewStandaloneTask("greet", func(ctx hatchet.Context, input GreetInput) (GreetOutput, error) {
		time.Sleep(300 * time.Millisecond)
		return GreetOutput{Worker: name}, nil
	})

	worker, err := client.NewWorker(name, hatchet.WithWorkflows(task), hatchet.WithSlots(1))
	if err != nil {
		log.Fatalf("%s: could not create worker: %v", name, err)
	}

	cleanup, err := worker.Start()
	if err != nil {
		log.Fatalf("%s: could not start worker: %v", name, err)
	}
	defer func() { _ = cleanup() }()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}

func env(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s is not set", key)
	}
	return v
}

func readEngine(path string) map[string]string {
	b, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		log.Fatalf("could not read engine file: %v", err)
	}
	var info map[string]string
	if err := json.Unmarshal(b, &info); err != nil {
		log.Fatalf("could not parse engine file %s: %v", path, err)
	}
	return info
}
