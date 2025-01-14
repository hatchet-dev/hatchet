package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type userCreateEvent struct {
	Username string            `json:"username"`
	UserID   string            `json:"user_id"`
	Data     map[string]string `json:"data"`
}

type stepOutput struct {
	Message string `json:"message"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	if err := run(cmdutils.InterruptChan()); err != nil {
		panic(err)
	}
}

func run(ch <-chan interface{}) error {
	c, err := client.New()

	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	w, err := worker.NewWorker(
		worker.WithClient(
			c,
		),
		worker.WithMaxRuns(500),
	)
	if err != nil {
		return fmt.Errorf("error creating worker: %w", err)
	}

	testSvc := w.NewService("test")

	stepNames := make([]string, 40) // assuming 4 steps per layer * 10 layers
	for i := range stepNames {
		stepNames[i] = generateRandomName()
	}

	steps := make([]*worker.WorkflowStep, len(stepNames))

	for i, name := range stepNames {
		steps[i] = worker.Fn(func(ctx worker.HatchetContext) (result *stepOutput, err error) {
			time.Sleep(generateRandomSleep())
			return &stepOutput{
				Message: "Completed step " + name,
			}, nil
		}).SetName(name)

		if i >= 4 {
			// setting dependencies from previous layer (4 steps back)
			steps[i].AddParents(stepNames[i-4])
		}
	}

	err = testSvc.On(
		worker.Events("crazy-dag"),
		&worker.WorkflowJob{
			Name:        "crazy-dag",
			Description: "This runs after an update to the user model with random step dependencies.",
			Steps:       steps,
		},
	)

	if err != nil {
		panic(err)
	}

	interruptCtx, cancel := cmdutils.InterruptContextFromChan(ch)
	defer cancel()

	cleanup, err := w.Start()
	if err != nil {
		return fmt.Errorf("error starting worker: %w", err)
	}

	go func() {
		for i := 0; i < 10; i++ {
			testEvent := userCreateEvent{
				Username: "echo-test",
				UserID:   "1234",
				Data: map[string]string{
					"test": "test",
				},
			}

			log.Printf("pushing event crazy-dag")

			// push an event
			err = c.Event().Push(
				context.Background(),
				"crazy-dag",
				testEvent,
			)

			if err != nil {
				log.Printf("error pushing event: %s", err.Error())
			}

			time.Sleep(5 * time.Millisecond)
		}
	}()

	for {
		select {
		case <-interruptCtx.Done():
			return cleanup()
		default:
			time.Sleep(time.Second)
		}
	}
}

func generateRandomName() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	nameLength := 20 // random length between 50 and 150
	b := make([]byte, nameLength)

	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func generateRandomSleep() time.Duration {
	return time.Duration(10+rand.Intn(30)) * time.Millisecond
}
