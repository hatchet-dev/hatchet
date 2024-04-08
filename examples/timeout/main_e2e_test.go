//go:build e2e

package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/internal/testutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

func TestTimeout(t *testing.T) {
	testutils.Prepare(t)

	tests := []struct {
		name string
		job  worker.WorkflowJob
		skip string
	}{
		{
			skip: "TODO currently broken",
			name: "worker timeout",
			job: worker.WorkflowJob{
				Timeout:     "10s",
				Name:        "timeout",
				Description: "timeout",
				Steps: []*worker.WorkflowStep{
					worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
						time.Sleep(time.Second * 60)
						return nil, nil
					}).SetName("step-one"),
				},
			},
		},
		{
			name: "step timeout",
			job: worker.WorkflowJob{
				Name:        "timeout",
				Description: "timeout",
				Steps: []*worker.WorkflowStep{
					worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
						time.Sleep(time.Second * 60)
						return nil, nil
					}).SetName("step-one").SetTimeout("10s"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip != "" {
				t.Skip(tt.skip)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			events := make(chan string, 50)

			cleanup, err := run(events, tt.job)
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
				"done",
			}, items)

			if err := cleanup(); err != nil {
				t.Fatalf("cleanup() error = %s", err)
			}
		})
	}
}
