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
		job  func(done func()) worker.WorkflowJob
	}{
		{
			name: "step timeout",
			job: func(done func()) worker.WorkflowJob {
				return worker.WorkflowJob{
					Name:        "timeout",
					Description: "timeout",
					Steps: []*worker.WorkflowStep{
						worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
							select {
							case <-time.After(time.Second * 30):
								return &stepOneOutput{
									Message: "finished",
								}, nil
							case <-ctx.Done():
								done()
								return nil, nil
							}
						}).SetName("step-one").SetTimeout("10s"),
					},
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			events := make(chan string, 50)

			cleanup, err := run(events, tt.job(func() {
				events <- "done"
			}))
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
				"done", // cancellation signal
				"done", // test check
			}, items)

			if err := cleanup(); err != nil {
				t.Fatalf("cleanup() error = %s", err)
			}
		})
	}
}
