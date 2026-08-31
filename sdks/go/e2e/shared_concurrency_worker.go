//go:build e2e

package e2e

import (
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/types"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

const sharedConcurrencyStrategyName = "e2e-shared-limit"

// SharedConcInput drives both concurrency expressions: Group feeds the shared strategy
// ("input.group") and Inline feeds the mixed task's inline strategy ("input.inline").
type SharedConcInput struct {
	Group  string `json:"group"`
	Inline string `json:"inline"`
}

// RunWindow reports when the task function was actually executing, so tests can assert
// whether two runs overlapped.
type RunWindow struct {
	StartMs int64 `json:"start_ms"`
	EndMs   int64 `json:"end_ms"`
}

var (
	// two separate workflows that consume the same tenant-scoped shared strategy
	testSharedConcA *hatchet.Workflow
	testSharedConcB *hatchet.Workflow
	// a workflow whose task mixes an inline (workflow-scoped) strategy with the shared one
	testSharedConcMixed *hatchet.Workflow
)

func sharedConcTaskFn(d time.Duration) func(ctx hatchet.Context, input SharedConcInput) (*RunWindow, error) {
	return func(ctx hatchet.Context, input SharedConcInput) (*RunWindow, error) {
		start := time.Now()
		time.Sleep(d)
		return &RunWindow{
			StartMs: start.UnixMilli(),
			EndMs:   time.Now().UnixMilli(),
		}, nil
	}
}

// sharedConcLimit is the shared strategy definition. It is carried in the workflow put
// requests below and upserted server-side as part of workflow registration; tasks in the
// other workflows reference it by name only, exercising both declaration styles.
func sharedConcLimit() types.SharedConcurrencyOpts {
	maxRuns := int32(1)
	strategy := types.GroupRoundRobin

	return types.SharedConcurrencyOpts{
		Name:          sharedConcurrencyStrategyName,
		Expression:    "input.group",
		MaxRuns:       &maxRuns,
		LimitStrategy: &strategy,
	}
}

func registerSharedConcurrencyWorkflows(client *hatchet.Client) {
	// workflow A carries the full definition, so registering it upserts the strategy
	testSharedConcA = client.NewWorkflow("shared-conc-a")
	testSharedConcA.NewTask("shared-conc-a-task", sharedConcTaskFn(1500*time.Millisecond),
		hatchet.WithSharedConcurrency(sharedConcLimit()),
	)

	// workflow B references the strategy by name only
	testSharedConcB = client.NewWorkflow("shared-conc-b")
	testSharedConcB.NewTask("shared-conc-b-task", sharedConcTaskFn(1500*time.Millisecond),
		hatchet.WithSharedConcurrency(types.SharedConcurrencyOpts{Name: sharedConcurrencyStrategyName}),
	)

	inlineMax := int32(1)
	inlineStrategy := types.GroupRoundRobin
	testSharedConcMixed = client.NewWorkflow("shared-conc-mixed")
	testSharedConcMixed.NewTask("shared-conc-mixed-task", sharedConcTaskFn(1500*time.Millisecond),
		hatchet.WithConcurrency(&types.Concurrency{
			Expression:    "input.inline",
			MaxRuns:       &inlineMax,
			LimitStrategy: &inlineStrategy,
		}),
		hatchet.WithSharedConcurrency(sharedConcLimit()),
	)
}
