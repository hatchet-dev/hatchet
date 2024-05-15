//go:build e2e

package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/internal/testutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

func TestWebhook(t *testing.T) {
	testutils.Prepare(t)

	tests := []struct {
		name string
		job  func(events chan<- string) worker.WorkflowJob
	}{
		{
			name: "simple action",
			job: func(events chan<- string) worker.WorkflowJob {
				return worker.WorkflowJob{
					Webhook:     fmt.Sprintf("http://localhost:%s/webhook", port),
					Name:        "simple-webhook",
					Description: "simple webhook",
					Steps: []*worker.WorkflowStep{
						worker.Fn(func(ctx worker.HatchetContext) (*output, error) {
							//verifyStepRuns(client, c.TenantId(), db.JobRunStatusRunning, db.StepRunStatusRunning, nil)

							events <- "step-one"

							return &output{
								Message: "hi from " + ctx.StepName(),
							}, nil
						}).SetName("step-one").SetTimeout("60s"),
						worker.Fn(func(ctx worker.HatchetContext) (*output, error) {
							var out output
							if err := ctx.StepOutput("step-one", &out); err != nil {
								panic(err)
							}
							if out.Message != "hi from step-one" {
								panic(fmt.Errorf("expected step run output to be valid, got %s", out.Message))
							}

							events <- "step-two"

							//verifyStepRuns(client, c.TenantId(), db.JobRunStatusRunning, db.StepRunStatusRunning, nil)

							return &output{
								Message: "hi from " + ctx.StepName(),
							}, nil
						}).SetName("step-two").SetTimeout("60s").AddParents("step-one"),
					},
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			events := make(chan string, 10)
			err := run(tt.job(events))
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
				"step-one",
				"step-two",
			}, items)
		})
	}
}
