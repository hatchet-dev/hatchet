//go:build e2e

package main

import (
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/internal/testutils"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
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
					worker.WebhookStep().SetName("step-two").SetTimeout("60s"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := make(chan string, 50)

			cleanup, err := run(events, tt.job)
			if err != nil {
				t.Fatalf("run() error = %s", err)
			}

			var items []string

			interruptCh := cmdutils.InterruptChan()

		outer:
			for {
				select {
				case item, ok := <-events:
					if !ok {
						break outer
					}
					items = append(items, item)
				case <-interruptCh:
					log.Printf("interrupt")
					break outer
				case <-time.After(time.Second * 60):
					log.Printf("timed out waiting for webhook")
					break outer
				}
			}

			assert.Equal(t, []string{
				"step-one",
				"step-two",
			}, items)

			if err := cleanup(); err != nil {
				t.Fatalf("cleanup() error = %s", err)
			}
		})
	}
}
