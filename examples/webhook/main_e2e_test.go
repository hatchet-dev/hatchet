//go:build e2e

package main

import (
	"testing"

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
				Name:        "simple-webhook",
				Description: "simple webhook",
				Steps: []*worker.WorkflowStep{
					worker.WebhookStep().SetName("step-one").SetTimeout("60s"),
					worker.WebhookStep().SetName("step-two").SetTimeout("60s").AddParents("step-one"),
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
