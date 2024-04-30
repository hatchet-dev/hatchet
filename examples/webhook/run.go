package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

func run(done chan<- string, job worker.WorkflowJob) (func() error, error) {
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

	port := "8741"

	err = w.RegisterWebhook(worker.Events("user:create:webhook"), fmt.Sprintf("http://localhost:%s/webhook", port), &job)
	if err != nil {
		return nil, fmt.Errorf("error registering webhook workflow: %w", err)
	}

	go func() {
		// create webserver to handle webhook requests
		http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
			data, err := io.ReadAll(r.Body)
			if err != nil {
				panic(err)
			}

			log.Printf("got webhook request!")

			var event dispatcher.WebhookEvent
			if err := json.Unmarshal(data, &event); err != nil {
				panic(err)
			}

			indent, _ := json.MarshalIndent(event, "", "  ")
			log.Printf("data: %s", string(indent))

			w.WriteHeader(http.StatusOK)
			resp := struct {
				MyData string `json:"myData"`
			}{
				MyData: "hi from " + event.StepName,
			}
			respBytes, err := json.Marshal(resp)
			_, _ = w.Write(respBytes)

			done <- event.StepName

			if event.StepName == "step-two" {
				close(done)
			}
		})

		log.Printf("starting webhook server on port %s", port)
		if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	go func() {
		log.Printf("pushing event")

		testEvent := userCreateEvent{
			Username: "echo-test",
			UserID:   "1234",
			Data: map[string]string{
				"test": "test",
			},
		}

		// push an event
		err := c.Event().Push(
			context.Background(),
			"user:create:webhook",
			testEvent,
		)
		if err != nil {
			panic(fmt.Errorf("error pushing event: %w", err))
		}
	}()

	cleanup := func() error {
		return nil
	}

	return cleanup, nil
}
