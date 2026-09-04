//go:build e2e

package e2e

import (
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/types"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// DynMaxRunsInput drives both expressions on the dynamic workflow: Tier is the group key
// ("input.tier") and also determines the group's limit via the max-runs expression.
type DynMaxRunsInput struct {
	Tier string `json:"tier"`
}

var testDynamicMaxRuns *hatchet.Workflow

func registerDynamicMaxRunsWorkflows(client *hatchet.Client) {
	maxRuns := int32(1)
	strategy := types.GroupRoundRobin
	// premium groups run 3 wide, everything else serializes; the static maxRuns of 1
	// must be overridden per group by this expression
	maxRunsExpr := "input.tier == 'premium' ? 3 : 1"

	testDynamicMaxRuns = client.NewWorkflow("dynamic-max-runs")
	testDynamicMaxRuns.NewTask("dynamic-max-runs-task",
		func(ctx hatchet.Context, input DynMaxRunsInput) (*RunWindow, error) {
			start := time.Now()
			time.Sleep(1500 * time.Millisecond)
			return &RunWindow{
				StartMs: start.UnixMilli(),
				EndMs:   time.Now().UnixMilli(),
			}, nil
		},
		hatchet.WithConcurrency(&types.Concurrency{
			Expression:        "input.tier",
			MaxRuns:           &maxRuns,
			LimitStrategy:     &strategy,
			MaxRunsExpression: &maxRunsExpr,
		}),
	)
}
