package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type loggerEvent struct {
	Username string            `json:"username"`
	UserID   string            `json:"user_id"`
	Data     map[string]string `json:"data"`
}

type stepOneOutput struct {
	Message string `json:"message"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	interrupt := cmdutils.InterruptChan()

	cleanup, err := run()
	if err != nil {
		panic(err)
	}

	<-interrupt

	if err := cleanup(); err != nil {
		panic(fmt.Errorf("error cleaning up: %w", err))
	}
}

func run() (func() error, error) {
	c, err := client.New()

	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}

	w, err := worker.NewWorker(
		worker.WithClient(
			c,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating worker: %w", err)
	}

	eventName := "logs:create:simple"

	err = w.RegisterWorkflow(
		&worker.WorkflowJob{
			On:          worker.Events(eventName),
			Name:        "logger",
			Description: "This runs a simple logger",
			Steps: []*worker.WorkflowStep{

				worker.Fn(step1).SetName("step-one"),
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error registering workflow: %w", err)
	}

	go func() {
		testEvent := loggerEvent{}

		// push an event
		err := c.Event().Push(
			context.Background(),
			eventName,
			testEvent,
			client.WithEventMetadata(map[string]string{
				"hello": "world",
			}),
		)
		if err != nil {
			panic(fmt.Errorf("error pushing event: %w", err))
		}
	}()

	cleanup, err := w.Start()
	if err != nil {
		panic(err)
	}

	return cleanup, nil
}

func step1(context worker.HatchetContext) error {
	zerologLogger := zerolog.New(os.Stdout)

	// Create a combined logger that forwards to Hatchet
	combinedLogger := context.NewCombinedZerologLogger(zerologLogger)

	logrusLogger := logrus.New()
	combinedLogrus := context.NewCombinedLogrusLogger(logrusLogger)

	for i := 0; i < 333; i++ {
		context.Log(fmt.Sprintf("Logging message %d", i))
		// Log messages using the combined logger
		combinedLogger.Info().Msg("This is a zerolog log message")
		combinedLogger.Error().Str("key", "value").Msg("This is an error log with structured data")

		combinedLogrus.Info("This is a logrus log message")
		combinedLogrus.WithField("key", "value").Error("This is an error log with structured data")

	}

	return nil
}
