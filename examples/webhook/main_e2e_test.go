//go:build e2e

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/internal/testutils"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

func TestWebhook(t *testing.T) {
	testutils.Prepare(t)

	tests := []struct {
		name string
		job  func(done func()) worker.WorkflowJob
	}{
		{
			name: "simple action",
			job: func(done func()) worker.WorkflowJob {
				return worker.WorkflowJob{
					Name:        "simple-webhook",
					Description: "simple webhook",
					Steps: []*worker.WorkflowStep{
						worker.WebhookStep().SetName("step-one").SetTimeout("10s"),
						worker.WebhookStep().SetName("step-two").SetTimeout("10s"),
					},
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := make(chan string, 50)

			cleanup, err := run(events, tt.job(func() {
				events <- "done"
			}))
			if err != nil {
				t.Fatalf("run() error = %s", err)
			}

			var items []string

			interruptCh := cmdutils.InterruptChan()

		outer:
			for {
				select {
				case item := <-events:
					items = append(items, item)
				case <-interruptCh:
					break outer
				}
			}

			assert.Equal(t, []string{
				"done",
			}, items)

			if err := cleanup(); err != nil {
				t.Fatalf("cleanup() error = %s", err)
			}
		})
	}
}
