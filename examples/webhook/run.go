package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

func run(job worker.WorkflowJob) error {
	c, err := client.New()
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	w, err := worker.NewWorker(
		worker.WithClient(
			c,
		),
	)
	if err != nil {
		return fmt.Errorf("error creating worker: %w", err)
	}

	port := "8741"

	ww, err := w.RegisterWebhook(worker.Events("user:create:webhook"), fmt.Sprintf("http://localhost:%s/webhook", port), &job)
	if err != nil {
		return fmt.Errorf("error registering webhook workflow: %w", err)
	}

	go func() {
		// create webserver to handle webhook requests
		http.HandleFunc("/webhook", ww.Middleware(func(event dispatcher.WebhookEvent) interface{} {
			log.Printf("webhook received with event: %+v", event)

			return struct {
				MyData string `json:"myData"`
			}{
				MyData: "hi from " + event.StepName,
			}
		}))

		log.Printf("starting webhook server on port %s", port)
		if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	log.Printf("pushing event")

	testEvent := userCreateEvent{
		Username: "echo-test",
		UserID:   "1234",
		Data: map[string]string{
			"test": "test",
		},
	}

	// push an event
	err = c.Event().Push(
		context.Background(),
		"user:create:webhook",
		testEvent,
	)
	if err != nil {
		panic(fmt.Errorf("error pushing event: %w", err))
	}

	time.Sleep(2 * time.Second)

	client := db.NewClient()
	if err := client.Connect(); err != nil {
		panic(fmt.Errorf("error connecting to database: %w", err))
	}
	defer client.Disconnect()

	events, err := client.Event.FindMany(
		db.Event.TenantID.Equals(c.TenantId()),
		db.Event.Key.Equals("user:create:webhook"),
	).With(
		db.Event.WorkflowRuns.Fetch().With(
			db.WorkflowRunTriggeredBy.Parent.Fetch().With(
				db.WorkflowRun.JobRuns.Fetch().With(
					db.JobRun.StepRuns.Fetch(),
				),
			),
		),
	).Exec(context.Background())
	if err != nil {
		panic(fmt.Errorf("error finding events: %w", err))
	}

	if len(events) == 0 {
		panic(fmt.Errorf("no events found"))
	}

	for _, event := range events {
		if len(event.WorkflowRuns()) == 0 {
			panic(fmt.Errorf("no workflow runs found"))
		}
		for _, workflowRun := range event.WorkflowRuns() {
			if len(workflowRun.Parent().JobRuns()) == 0 {
				panic(fmt.Errorf("no job runs found"))
			}
			for _, jobRuns := range workflowRun.Parent().JobRuns() {
				if jobRuns.Status != db.JobRunStatusSucceeded {
					panic(fmt.Errorf("expected job run to be running, got %s", jobRuns.Status))
				}
				for _, stepRun := range jobRuns.StepRuns() {
					if stepRun.Status != db.StepRunStatusSucceeded {
						panic(fmt.Errorf("expected step run to be failed, got %s", stepRun.Status))
					}
					output, ok := stepRun.Output()
					if !ok {
						panic(fmt.Errorf("expected step run to have output, got %s", stepRun.Status))
					}
					if string(output) != `{"myData":"hi from step-one"}` && string(output) != `{"myData":"hi from step-two"}` {
						panic(fmt.Errorf("expected step run output to be valid, got %s", string(output)))
					}
				}
			}
		}
	}

	return nil
}
