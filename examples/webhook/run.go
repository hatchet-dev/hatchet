package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

var prisma *db.PrismaClient

func init() {
	prisma = db.NewClient()
	if err := prisma.Connect(); err != nil {
		panic(fmt.Errorf("error connecting to database: %w", err))
	}
	cancel := cmdutils.InterruptChan()
	go func(prisma *db.PrismaClient) {
		<-cancel
		_ = prisma.Disconnect()
	}(prisma)
}

func initialize(w *worker.Worker, job worker.WorkflowJob, event string) error {
	err := w.On(worker.Events(event), &job)
	if err != nil {
		return fmt.Errorf("error registering webhook workflow: %w", err)
	}

	return nil
}

func run(handler func(w http.ResponseWriter, r *http.Request), c client.Client, workflow string, event string) error {
	wf, err := prisma.Workflow.FindFirst(
		db.Workflow.Name.Equals(workflow),
	).Exec(context.Background())
	if err != nil {
		panic(fmt.Errorf("error finding webhook worker: %w", err))
	}

	// create webserver to handle webhook requests
	mux := http.NewServeMux()

	// Register the HelloHandler to the /hello route
	mux.HandleFunc("/webhook", handler)

	port := "8741"
	// Create a custom server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	defer func(server *http.Server, ctx context.Context) {
		err := server.Shutdown(ctx)
		if err != nil {
			panic(err)
		}
	}(server, context.Background())

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	if err := setup(c, wf.ID); err != nil {
		panic(fmt.Errorf("error setting up webhook: %w", err))
	}

	time.Sleep(30 * time.Second)

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
		event,
		testEvent,
	)
	if err != nil {
		panic(fmt.Errorf("error pushing event: %w", err))
	}

	// TODO test for assigned status before it is started
	//time.Sleep(2 * time.Second)
	//verifyStepRuns(client, c.TenantId(), db.JobRunStatusRunning, db.StepRunStatusAssigned, nil)

	time.Sleep(5 * time.Second)

	return nil
}

func setup(c client.Client, wfId string) error {
	tenantId := openapi_types.UUID{}
	if err := tenantId.Scan(c.TenantId()); err != nil {
		return fmt.Errorf("error getting tenant id: %w", err)
	}

	secret := "secret"
	res, err := c.API().WebhookCreate(context.Background(), tenantId, rest.WebhookCreateJSONRequestBody{
		Url: "http://localhost:8741/webhook",
		Workflows: []string{
			wfId,
		},
		Secret: &secret,
	})
	if err != nil {
		return fmt.Errorf("error creating webhook worker: %w", err)
	}

	if res.StatusCode != 200 {
		return fmt.Errorf("error creating webhook, failed with status code %d", res.StatusCode)
	}

	return nil
}

func verifyStepRuns(event string, tenantId string, jobRunStatus db.JobRunStatus, stepRunStatus db.StepRunStatus, check func(string)) {
	events, err := prisma.Event.FindMany(
		db.Event.TenantID.Equals(tenantId),
		db.Event.Key.Equals(event),
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
