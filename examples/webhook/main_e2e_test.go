//go:build e2e

package main

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/internal/testutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

func TestWebhook(t *testing.T) {
	testutils.Prepare(t)

	tests := []struct {
		name string
		job  worker.WorkflowJob
	}{
		{
			name: "simple action",
			job: worker.WorkflowJob{
				Webhook:     fmt.Sprintf("http://localhost:%s/webhook", port),
				Name:        "simple-webhook",
				Description: "simple webhook",
				Steps: []*worker.WorkflowStep{ // TODO add events channel to make sure both steps were executed
					worker.Fn(func(ctx worker.HatchetContext) (*output, error) {
						log.Printf("step one received")

						time.Sleep(time.Second * 3) // this needs to be 3

						log.Printf("step name: %s", ctx.StepName())

						//verifyStepRuns(client, c.TenantId(), db.JobRunStatusRunning, db.StepRunStatusRunning, nil)

						time.Sleep(time.Second * 2)

						return &output{
							Message: "hi from " + ctx.StepName(),
						}, nil
					}).SetName("step-one").SetTimeout("60s"),
					worker.Fn(func(ctx worker.HatchetContext) (*output, error) {
						log.Printf("step two received")

						time.Sleep(time.Second * 2)

						log.Printf("step name: %s", ctx.StepName())

						var out output
						if err := ctx.StepOutput("step-one", &out); err != nil {
							panic(err)
						}
						log.Printf("this is step-two, step-one had output: %+v", out)

						if out.Message != "hi from step-one" {
							panic(fmt.Errorf("expected step run output to be valid, got %s", out.Message))
						}

						//verifyStepRuns(client, c.TenantId(), db.JobRunStatusRunning, db.StepRunStatusRunning, nil)

						time.Sleep(time.Second * 2)

						return &output{
							Message: "hi from " + ctx.StepName(),
						}, nil
					}).SetName("step-two").SetTimeout("60s").AddParents("step-one"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := run(tt.job)
			if err != nil {
				t.Fatalf("run() error = %s", err)
			}
		})
	}
}
