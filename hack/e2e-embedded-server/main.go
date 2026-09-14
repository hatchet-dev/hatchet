package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	embed "github.com/hatchet-dev/hatchet-embedded"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type echoInput struct {
	Message string `json:"message"`
}

type echoOutput struct {
	Message  string `json:"message"`
	EchoedAt string `json:"echoed_at"`
}

func main() {
	tokenFile := flag.String("token-file", "", "path to write the admin API token to")
	apiPort := flag.Int("api-port", 8080, "API server port")
	grpcPort := flag.Int("grpc-port", 7077, "gRPC server port")
	logLevel := flag.String("log-level", "warn", "engine log level")
	withWorker := flag.Bool("with-worker", true, "run an echo worker with a registered workflow")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	inst, err := embed.Start(ctx,
		embed.WithAPIPort(*apiPort),
		embed.WithGRPCPort(*grpcPort),
		embed.WithLogLevel(*logLevel),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start embedded hatchet: %v\n", err)
		os.Exit(1)
	}

	if *tokenFile != "" {
		if err := os.WriteFile(*tokenFile, []byte(inst.Token()), 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write token file: %v\n", err)
			os.Exit(1)
		}
	}

	var cleanupWorker func() error
	if *withWorker {
		client := inst.Client()
		task := client.NewStandaloneTask("e2e-echo", func(ctx hatchet.Context, input echoInput) (echoOutput, error) {
			return echoOutput{Message: input.Message, EchoedAt: time.Now().Format(time.RFC3339)}, nil
		},
			hatchet.WithWorkflowEvents("e2e:echo"),
		)

		worker, err := client.NewWorker("e2e-worker", hatchet.WithWorkflows(task))
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to create worker: %v\n", err)
			os.Exit(1)
		}

		cleanupWorker, err = worker.Start()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to start worker: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("embedded hatchet ready api=%s grpc=%s tenant=%s\n", inst.APIURL(), inst.GRPCAddress(), inst.TenantID())

	<-ctx.Done()

	if cleanupWorker != nil {
		_ = cleanupWorker()
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := inst.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "shutdown error: %v\n", err)
		os.Exit(1)
	}
}
