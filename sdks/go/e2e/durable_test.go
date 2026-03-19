//go:build e2e

package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type DurableSleepOutput struct {
	Status string `json:"status"`
}

func TestDurableSleep(t *testing.T) {
	client := newClient(t)

	sleepDuration := 5 * time.Second

	task := client.NewStandaloneDurableTask("durable-sleep-e2e", func(ctx hatchet.DurableContext, input any) (DurableSleepOutput, error) {
		if _, err := ctx.SleepFor(sleepDuration); err != nil {
			return DurableSleepOutput{}, err
		}
		return DurableSleepOutput{Status: "success"}, nil
	})

	worker, err := client.NewWorker("durable-sleep-e2e-worker",
		hatchet.WithWorkflows(task),
		hatchet.WithDurableSlots(10),
	)
	require.NoError(t, err)
	cleanup := startWorker(t, worker)
	defer cleanup() //nolint:errcheck

	result, err := task.Run(bgCtx(), nil)
	require.NoError(t, err)

	var out DurableSleepOutput
	require.NoError(t, result.Into(&out))
	assert.Equal(t, "success", out.Status)
}

func TestDurableEvent(t *testing.T) {
	client := newClient(t)

	eventKey := "durable-e2e:event"

	task := client.NewStandaloneDurableTask("durable-event-e2e", func(ctx hatchet.DurableContext, input any) (DurableSleepOutput, error) {
		if _, err := ctx.WaitForEvent(eventKey, "true"); err != nil {
			return DurableSleepOutput{}, err
		}
		return DurableSleepOutput{Status: "success"}, nil
	})

	worker, err := client.NewWorker("durable-event-e2e-worker",
		hatchet.WithWorkflows(task),
		hatchet.WithDurableSlots(10),
	)
	require.NoError(t, err)
	cleanup := startWorker(t, worker)
	defer cleanup() //nolint:errcheck

	ref, err := task.RunNoWait(bgCtx(), nil)
	require.NoError(t, err)

	time.Sleep(5 * time.Second)

	require.NoError(t, client.Events().Push(bgCtx(), eventKey, map[string]any{"test": "data"}))

	result, err := ref.Result()
	require.NoError(t, err)

	var out DurableSleepOutput
	require.NoError(t, result.TaskOutput("durable-event-e2e").Into(&out))
	assert.Equal(t, "success", out.Status)
}

func TestDurableMultiSleep(t *testing.T) {
	client := newClient(t)

	sleepDuration := 5 * time.Second

	task := client.NewStandaloneDurableTask("durable-multi-sleep-e2e", func(ctx hatchet.DurableContext, input any) (map[string]int, error) {
		start := time.Now()

		for i := 0; i < 3; i++ {
			if _, err := ctx.SleepFor(sleepDuration); err != nil {
				return nil, err
			}
		}

		return map[string]int{"runtime": int(time.Since(start).Seconds())}, nil
	})

	worker, err := client.NewWorker("durable-multi-sleep-e2e-worker",
		hatchet.WithWorkflows(task),
		hatchet.WithDurableSlots(10),
	)
	require.NoError(t, err)
	cleanup := startWorker(t, worker)
	defer cleanup() //nolint:errcheck

	result, err := task.Run(bgCtx(), nil)
	require.NoError(t, err)

	var out map[string]int
	require.NoError(t, result.Into(&out))
	assert.GreaterOrEqual(t, out["runtime"], int(3*sleepDuration.Seconds()))
}
