//go:build e2e

package e2e

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type ChildInput struct {
	Value int `json:"value"`
}

type ChildOutput struct {
	Result int `json:"result"`
}

type ParentOutput struct {
	Sum int `json:"sum"`
}

func TestFanout(t *testing.T) {
	client := newClient(t)

	childTask := client.NewStandaloneTask("fanout-child-e2e", func(ctx hatchet.Context, input ChildInput) (ChildOutput, error) {
		return ChildOutput{Result: input.Value * 2}, nil
	})

	parentTask := client.NewStandaloneTask("fanout-parent-e2e", func(ctx hatchet.Context, input any) (ParentOutput, error) {
		n := 5
		sum := 0

		for i := 1; i <= n; i++ {
			result, err := childTask.Run(ctx, ChildInput{Value: i})
			if err != nil {
				return ParentOutput{}, fmt.Errorf("child %d failed: %w", i, err)
			}
			var out ChildOutput
			if err := result.Into(&out); err != nil {
				return ParentOutput{}, err
			}
			sum += out.Result
		}

		return ParentOutput{Sum: sum}, nil
	})

	worker, err := client.NewWorker("fanout-e2e-worker",
		hatchet.WithWorkflows(parentTask, childTask),
		hatchet.WithSlots(10),
	)
	require.NoError(t, err)
	cleanup := startWorker(t, worker)
	defer cleanup() //nolint:errcheck

	result, err := parentTask.Run(bgCtx(), nil)
	require.NoError(t, err)

	var out ParentOutput
	require.NoError(t, result.Into(&out))
	assert.Equal(t, 30, out.Sum) // 2+4+6+8+10
}

func TestFanoutParallel(t *testing.T) {
	client := newClient(t)

	childTask := client.NewStandaloneTask("fanout-par-child-e2e", func(ctx hatchet.Context, input ChildInput) (ChildOutput, error) {
		return ChildOutput{Result: input.Value * 2}, nil
	})

	parentTask := client.NewStandaloneTask("fanout-par-parent-e2e", func(ctx hatchet.Context, input any) (ParentOutput, error) {
		n := 5
		var mu sync.Mutex
		sum := 0
		var wg sync.WaitGroup
		var errs []error

		wg.Add(n)
		for i := 1; i <= n; i++ {
			go func(val int) {
				defer wg.Done()
				result, err := childTask.Run(ctx, ChildInput{Value: val})
				if err != nil {
					mu.Lock()
					errs = append(errs, err)
					mu.Unlock()
					return
				}
				var out ChildOutput
				if err := result.Into(&out); err != nil {
					mu.Lock()
					errs = append(errs, err)
					mu.Unlock()
					return
				}
				mu.Lock()
				sum += out.Result
				mu.Unlock()
			}(i)
		}
		wg.Wait()

		if len(errs) > 0 {
			return ParentOutput{}, errs[0]
		}
		return ParentOutput{Sum: sum}, nil
	})

	worker, err := client.NewWorker("fanout-par-e2e-worker",
		hatchet.WithWorkflows(parentTask, childTask),
		hatchet.WithSlots(10),
	)
	require.NoError(t, err)
	cleanup := startWorker(t, worker)
	defer cleanup() //nolint:errcheck

	result, err := parentTask.Run(bgCtx(), nil)
	require.NoError(t, err)

	var out ParentOutput
	require.NoError(t, result.Into(&out))
	assert.Equal(t, 30, out.Sum)
}
