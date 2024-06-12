//go:build e2e

package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/testutils"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

func TestWebhook(t *testing.T) {
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
				t.SkipNow()
				events := make(chan string, 10)

				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()

				workflow := "simple-webhook"
				wf := worker.WorkflowJob{
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

				event := "user:create:webhook"
				if err := initialize(w, wf, event); err != nil {
					t.Fatalf("error initializing webhook: %v", err)
				}
				handler := w.WebhookHttpHandler(worker.WebhookHandlerOptions{
					Secret: "secret",
				})
				err = run(handler, c, workflow, event)
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

				verifyStepRuns(event, c.TenantId(), db.JobRunStatusSucceeded, db.StepRunStatusSucceeded, func(output string) {
					if string(output) != `{"message":"hi from webhook-step-one"}` && string(output) != `{"message":"hi from webhook-step-two"}` {
						panic(fmt.Errorf("expected step run output to be valid, got %s", output))
					}
				})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.job(t)
		})
	}
}
