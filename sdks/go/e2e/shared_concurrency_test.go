//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// runSharedConcWorkflows triggers each (workflow, input) pair concurrently and returns the
// run windows reported by the task functions.
func runSharedConcWorkflows(t *testing.T, ctx context.Context, runs []struct {
	wf    *hatchet.Workflow
	task  string
	input SharedConcInput
}) []RunWindow {
	t.Helper()

	windows := make([]RunWindow, len(runs))
	errs := make([]error, len(runs))

	var wg sync.WaitGroup
	for i, r := range runs {
		wg.Add(1)
		go func(i int, wf *hatchet.Workflow, task string, input SharedConcInput) {
			defer wg.Done()

			result, err := wf.Run(ctx, input)
			if err != nil {
				errs[i] = err
				return
			}

			var window RunWindow
			if err := result.TaskOutput(task).Into(&window); err != nil {
				errs[i] = fmt.Errorf("decoding run window: %w", err)
				return
			}
			windows[i] = window
		}(i, r.wf, r.task, r.input)
	}
	wg.Wait()

	for i, err := range errs {
		require.NoError(t, err, "run %d failed", i)
	}

	return windows
}

// requireSerialized asserts that no two run windows overlapped (max concurrency of 1).
// A small tolerance absorbs clock jitter between worker-side timestamps.
func requireSerialized(t *testing.T, windows []RunWindow) {
	t.Helper()

	const toleranceMs = 100

	sorted := make([]RunWindow, len(windows))
	copy(sorted, windows)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].StartMs < sorted[j].StartMs })

	for i := 1; i < len(sorted); i++ {
		require.GreaterOrEqual(t, sorted[i].StartMs, sorted[i-1].EndMs-toleranceMs,
			"runs %d and %d overlapped: [%d, %d] vs [%d, %d]",
			i-1, i, sorted[i-1].StartMs, sorted[i-1].EndMs, sorted[i].StartMs, sorted[i].EndMs)
	}
}

// requireSomeOverlap asserts that at least one pair of windows overlapped, proving the
// worker genuinely runs these tasks concurrently when no limit binds.
func requireSomeOverlap(t *testing.T, windows []RunWindow) {
	t.Helper()

	for i := 0; i < len(windows); i++ {
		for j := i + 1; j < len(windows); j++ {
			if windows[i].StartMs < windows[j].EndMs && windows[j].StartMs < windows[i].EndMs {
				return
			}
		}
	}

	t.Fatalf("expected at least one overlapping pair, got windows: %+v", windows)
}

// TestSharedConcurrencyCrossWorkflow proves that tasks from two different workflows
// referencing the same shared strategy (max=1) with the same group key never run
// concurrently.
func TestSharedConcurrencyCrossWorkflow(t *testing.T) {
	ctx := newTestContext(t)

	windows := runSharedConcWorkflows(t, ctx, []struct {
		wf    *hatchet.Workflow
		task  string
		input SharedConcInput
	}{
		{testSharedConcA, "shared-conc-a-task", SharedConcInput{Group: "xwf-group"}},
		{testSharedConcA, "shared-conc-a-task", SharedConcInput{Group: "xwf-group"}},
		{testSharedConcB, "shared-conc-b-task", SharedConcInput{Group: "xwf-group"}},
		{testSharedConcB, "shared-conc-b-task", SharedConcInput{Group: "xwf-group"}},
	})

	requireSerialized(t, windows)
}

// TestSharedConcurrencyMixedSharedLimitBinds: the mixed task holds an inline strategy and
// the shared strategy. With distinct inline keys but one shared group key, the shared
// limit is what serializes the runs — including against another workflow's task.
func TestSharedConcurrencyMixedSharedLimitBinds(t *testing.T) {
	ctx := newTestContext(t)

	windows := runSharedConcWorkflows(t, ctx, []struct {
		wf    *hatchet.Workflow
		task  string
		input SharedConcInput
	}{
		{testSharedConcMixed, "shared-conc-mixed-task", SharedConcInput{Group: "mixed-shared-group", Inline: "inline-1"}},
		{testSharedConcMixed, "shared-conc-mixed-task", SharedConcInput{Group: "mixed-shared-group", Inline: "inline-2"}},
		{testSharedConcA, "shared-conc-a-task", SharedConcInput{Group: "mixed-shared-group"}},
	})

	requireSerialized(t, windows)
}

// TestSharedConcurrencyMixedInlineLimitBinds: with distinct shared group keys but one
// inline key, the inline (workflow-scoped) strategy serializes the runs — both limits on
// the mixed task are live at once.
func TestSharedConcurrencyMixedInlineLimitBinds(t *testing.T) {
	ctx := newTestContext(t)

	windows := runSharedConcWorkflows(t, ctx, []struct {
		wf    *hatchet.Workflow
		task  string
		input SharedConcInput
	}{
		{testSharedConcMixed, "shared-conc-mixed-task", SharedConcInput{Group: "mixed-inline-group-1", Inline: "inline-same"}},
		{testSharedConcMixed, "shared-conc-mixed-task", SharedConcInput{Group: "mixed-inline-group-2", Inline: "inline-same"}},
	})

	requireSerialized(t, windows)
}

// TestSharedConcurrencyNoLimitOverlaps is the control: when neither the shared group key
// nor the inline key collides, runs overlap freely, proving the serialization in the other
// tests comes from the concurrency strategies rather than worker capacity.
func TestSharedConcurrencyNoLimitOverlaps(t *testing.T) {
	ctx := newTestContext(t)

	windows := runSharedConcWorkflows(t, ctx, []struct {
		wf    *hatchet.Workflow
		task  string
		input SharedConcInput
	}{
		{testSharedConcMixed, "shared-conc-mixed-task", SharedConcInput{Group: "free-group-1", Inline: "free-inline-1"}},
		{testSharedConcMixed, "shared-conc-mixed-task", SharedConcInput{Group: "free-group-2", Inline: "free-inline-2"}},
	})

	requireSomeOverlap(t, windows)
}
