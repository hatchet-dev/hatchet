//go:build e2e

package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/internal/testutils"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

func TestWebhook(t *testing.T) {
	t.Skipf("Skipping webhook e2e test, flaky")

	testutils.Prepare(t)

	c, err := client.New()
	if err != nil {
		panic(fmt.Errorf("error creating client: %w", err))
	}

	tests := []struct {
		name string
		job  func(t *testing.T)
	}{
		{
			name: "simple action",
			job: func(t *testing.T) {
				events := make(chan string, 10)

				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()

				event := "user:webhook-simple"
				workflow := "simple-webhook"
				wf := &worker.WorkflowJob{
					On:          worker.Event(event),
					Name:        workflow,
					Description: workflow,
					Steps: []*worker.WorkflowStep{
						worker.Fn(func(ctx worker.HatchetContext) (*output, error) {
							//verifyStepRuns(client, c.TenantId(), db.JobRunStatusRunning, db.StepRunStatusRunning, nil)

							events <- "webhook-step-one"

							return &output{
								Message: "hi from " + ctx.StepName(),
							}, nil
						}).SetName("webhook-step-one").SetTimeout("60s"),
						worker.Fn(func(ctx worker.HatchetContext) (*output, error) {
							var out output
							if err := ctx.StepOutput("webhook-step-one", &out); err != nil {
								panic(err)
							}
							if out.Message != "hi from webhook-step-one" {
								panic(fmt.Errorf("expected step run output to be valid, got %s", out.Message))
							}

							events <- "webhook-step-two"

							//verifyStepRuns(client, c.TenantId(), db.JobRunStatusRunning, db.StepRunStatusRunning, nil)

							return &output{
								Message: "hi from " + ctx.StepName(),
							}, nil
						}).SetName("webhook-step-two").SetTimeout("60s").AddParents("webhook-step-one"),
					},
				}

				w, err := worker.NewWorker(
					worker.WithClient(
						c,
					),
				)
				if err != nil {
					panic(fmt.Errorf("error creating worker: %w", err))
				}

				handler := w.WebhookHttpHandler(worker.WebhookHandlerOptions{
					Secret: "secret",
				}, wf)
				err = run("simple action", w, "8742", handler, c, workflow, event)
				if err != nil {
					t.Fatalf("run() error = %s", err)
				}

				var items []string
			outer:
				for {
					select {
					case item := <-events:
						items = append(items, item)
					case <-ctx.Done():
						break outer
					}
				}

				assert.Equal(t, []string{
					"webhook-step-one",
					"webhook-step-two",
				}, items)
			},
		},
		{
			name: "mark action as failed immediately if webhook fails",
			job: func(t *testing.T) {
				workflow := "simple-webhook-failure"
				wf := &worker.WorkflowJob{
					Name:        workflow,
					Description: workflow,
					Steps: []*worker.WorkflowStep{
						worker.Fn(func(ctx worker.HatchetContext) (*output, error) {
							return &output{
								Message: "hi from " + ctx.StepName(),
							}, nil
						}).SetName("webhook-failure-step-one").SetTimeout("60s"),
					},
				}

				w, err := worker.NewWorker(
					worker.WithClient(
						c,
					),
				)
				if err != nil {
					panic(fmt.Errorf("error creating worker: %w", err))
				}

				event := "user:create-webhook-failure"
				err = w.On(worker.Events(event), wf)
				if err != nil {
					panic(fmt.Errorf("error registering webhook workflow: %w", err))
				}
				handler := func(w http.ResponseWriter, r *http.Request) {
					if r.Method == http.MethodPut {
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(fmt.Sprintf(`{"actions": ["default:%s"]}`, "webhook-failure-step-one")))
						return
					}
					w.WriteHeader(http.StatusInternalServerError) // simulate a failure
				}
				err = run("mark action as failed immediately if webhook fails", w, "8743", handler, c, workflow, event)
				if err != nil {
					t.Fatalf("run() error = %s", err)
				}
			},
		},
		{
			name: "register action",
			job: func(t *testing.T) {
				events := make(chan string, 10)

				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()

				w, err := worker.NewWorker(
					worker.WithClient(
						c,
					),
				)
				if err != nil {
					panic(fmt.Errorf("error creating worker: %w", err))
				}

				testSvc := w.NewService("test")

				err = testSvc.RegisterAction(func(ctx worker.HatchetContext) (*output, error) {
					time.Sleep(5 * time.Second)

					events <- "wha-webhook-action-1"

					return &output{
						Message: "hi from wha-webhook-action-1",
					}, nil
				}, worker.WithActionName("wha-webhook-action-1"))
				if err != nil {
					panic(err)
				}

				event := "user:wha-webhook-actions"

				err = testSvc.On(
					worker.Event(event),
					testSvc.Call("wha-webhook-action-1"),
				)

				workflow := "wha-webhook-with-actions"
				wf := &worker.WorkflowJob{
					On:          worker.Event(event),
					Name:        workflow,
					Description: workflow,
					Steps: []*worker.WorkflowStep{
						worker.Fn(func(ctx worker.HatchetContext) (*output, error) {
							//verifyStepRuns(client, c.TenantId(), db.JobRunStatusRunning, db.StepRunStatusRunning, nil)

							events <- "wha-webhook-step-one"

							return &output{
								Message: "hi from " + ctx.StepName(),
							}, nil
						}).SetName("wha-webhook-step-one").SetTimeout("60s"),
						worker.Fn(func(ctx worker.HatchetContext) (*output, error) {
							var out output
							if err := ctx.StepOutput("wha-webhook-step-one", &out); err != nil {
								panic(err)
							}
							if out.Message != "hi from wha-webhook-step-one" {
								panic(fmt.Errorf("expected step run output to be valid, got %s", out.Message))
							}

							events <- "wha-webhook-step-two"

							//verifyStepRuns(client, c.TenantId(), db.JobRunStatusRunning, db.StepRunStatusRunning, nil)

							return &output{
								Message: "hi from " + ctx.StepName(),
							}, nil
						}).SetName("wha-webhook-step-two").SetTimeout("60s").AddParents("wha-webhook-step-one"),
					},
				}

				handler := w.WebhookHttpHandler(worker.WebhookHandlerOptions{
					Secret: "secret",
				}, wf)
				err = run("register action", w, "8744", handler, c, workflow, event)
				if err != nil {
					t.Fatalf("run() error = %s", err)
				}

				var items []string
			outer:
				for {
					select {
					case item := <-events:
						items = append(items, item)
					case <-ctx.Done():
						break outer
					}
				}

				assert.Equal(t, []string{
					"wha-webhook-step-one",
					"wha-webhook-step-two",
					"wha-webhook-action-1",
				}, items)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.job(t)
		})
	}
}
