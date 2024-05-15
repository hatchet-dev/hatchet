package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
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

	client := db.NewClient()
	if err := client.Connect(); err != nil {
		panic(fmt.Errorf("error connecting to database: %w", err))
	}
	defer client.Disconnect()

	port := "8741"

	err = w.RegisterWebhook(worker.Events("user:create:webhook"), fmt.Sprintf("http://localhost:%s/webhook", port), &job)
	if err != nil {
		return fmt.Errorf("error registering webhook workflow: %w", err)
	}

	go func() {
		// create webserver to handle webhook requests
		http.HandleFunc("/webhook", w.WebhookHandler(func(ctx worker.HatchetContext) interface{} {
			log.Printf("webhook received with event: %+v", ctx)

			time.Sleep(time.Second * 3) // this needs to be 3

			log.Printf("step name: %s", ctx.StepName())

			if ctx.StepName() == "step-two" {
				var out interface{}
				if err := ctx.StepOutput("step-one", out); err != nil {
					// TODO THIS CURRENTLY ERRORS AND NEEDS TO BE FIXED
					log.Printf("error getting step output: %s", err)
					//panic(err)
				}
				log.Printf("this is step-two, step-one had output: %+v", out)
			}
			verifyStepRuns(client, c.TenantId(), db.JobRunStatusRunning, db.StepRunStatusRunning, nil)

			time.Sleep(time.Second * 3)

			return struct {
				MyData string `json:"myData"`
			}{
				MyData: "hi from " + ctx.StepName(),
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

	time.Sleep(2 * time.Second) // this needs to be 2

	// TODO test for assigned status before it is started
	//verifyStepRuns(client, c.TenantId(), db.JobRunStatusRunning, db.StepRunStatusAssigned, nil)

	time.Sleep(15 * time.Second)

	verifyStepRuns(client, c.TenantId(), db.JobRunStatusSucceeded, db.StepRunStatusSucceeded, func(output string) {
		if string(output) != `{"myData":"hi from step-one"}` && string(output) != `{"myData":"hi from step-two"}` {
			panic(fmt.Errorf("expected step run output to be valid, got %s", string(output)))
		}
	})

	return nil
}

func verifyStepRuns(client *db.PrismaClient, tenantId string, jobRunStatus db.JobRunStatus, stepRunStatus db.StepRunStatus, check func(string)) {
	events, err := client.Event.FindMany(
		db.Event.TenantID.Equals(tenantId),
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
				if jobRuns.Status != jobRunStatus {
					panic(fmt.Errorf("expected job run to be %s, got %s", jobRunStatus, jobRuns.Status))
				}
				for _, stepRun := range jobRuns.StepRuns() {
					if stepRun.Status != stepRunStatus {
						panic(fmt.Errorf("expected step run to be %s, got %s", stepRunStatus, stepRun.Status))
					}
					output, ok := stepRun.Output()
					if check != nil {
						if !ok {
							panic(fmt.Errorf("expected step run to have output, got %+v", stepRun))
						}
						check(string(output))
					}
				}
			}
		}
	}
}
