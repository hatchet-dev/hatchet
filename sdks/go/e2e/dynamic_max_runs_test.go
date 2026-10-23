//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// runDynMaxRunsWorkflows triggers the dynamic workflow once per input concurrently and
// returns the run windows reported by the task functions.
func runDynMaxRunsWorkflows(t *testing.T, ctx context.Context, inputs []DynMaxRunsInput) []RunWindow {
	t.Helper()

	windows := make([]RunWindow, len(inputs))
	errs := make([]error, len(inputs))

	var wg sync.WaitGroup
	for i, input := range inputs {
		wg.Add(1)
		go func(i int, input DynMaxRunsInput) {
			defer wg.Done()

			result, err := testDynamicMaxRuns.Run(ctx, input)
			if err != nil {
				errs[i] = err
				return
			}

			var window RunWindow
			if err := result.TaskOutput("dynamic-max-runs-task").Into(&window); err != nil {
				errs[i] = fmt.Errorf("decoding run window: %w", err)
				return
			}
			windows[i] = window
		}(i, input)
	}
	wg.Wait()

	for i, err := range errs {
		require.NoError(t, err, "run %d failed", i)
	}

	return windows
}

// maxSimultaneous returns the largest number of windows that were open at once.
func maxSimultaneous(windows []RunWindow) int {
	best := 0
	for i := range windows {
		open := 0
		for j := range windows {
			if windows[j].StartMs <= windows[i].StartMs && windows[i].StartMs < windows[j].EndMs {
				open++
			}
		}
		if open > best {
			best = open
		}
	}
	return best
}

// TestDynamicMaxRunsPremiumGroupRunsWide: the premium group's evaluated limit is 3, so
// with 4 runs at least two must overlap (the static maxRuns of 1 would serialize them)
// while never more than 3 run at once (the evaluated limit still binds).
func TestDynamicMaxRunsPremiumGroupRunsWide(t *testing.T) {
	ctx := newTestContext(t)

	windows := runDynMaxRunsWorkflows(t, ctx, []DynMaxRunsInput{
		{Tier: "premium"},
		{Tier: "premium"},
		{Tier: "premium"},
		{Tier: "premium"},
	})

	sim := maxSimultaneous(windows)
	require.GreaterOrEqual(t, sim, 2, "the evaluated limit of 3 must override the static maxRuns of 1; windows: %+v", windows)
	require.LessOrEqual(t, sim, 3, "the evaluated limit of 3 must still bind; windows: %+v", windows)
}

// TestDynamicMaxRunsFreeGroupSerialized: the free group's evaluated limit is 1, so its
// runs never overlap even while premium groups run wide.
func TestDynamicMaxRunsFreeGroupSerialized(t *testing.T) {
	ctx := newTestContext(t)

	windows := runDynMaxRunsWorkflows(t, ctx, []DynMaxRunsInput{
		{Tier: "free"},
		{Tier: "free"},
	})

	requireSerialized(t, windows)
}
