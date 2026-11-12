//go:build !e2e && !load && !rampup && !integration

package e2e

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type stepOutput struct {
	Step string `json:"step"`
}

// The waiters share one event, so its push satisfies their log entries in log order. On the
// replay after the eviction, the first waiter's completion then arrives before the third waiter
// has registered again, whatever order the engine lists the steps in. Applying it right away
// would trigger the first waiter into the log position of the third waiter's registration.
func TestDagOperatorReplayAfterEviction(t *testing.T) {
	env := newTestEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	eventKey := "dag-operator-replay:" + uuid.NewString()

	var mu sync.Mutex
	// the parent outputs each step received, recorded when it runs
	received := map[string]map[string]string{}

	workflow := env.client.NewWorkflow("dag-operator-replay")

	var waiters []*hatchet.Task

	for _, name := range []string{"a", "b", "c"} {
		waiters = append(waiters, workflow.NewTask(name,
			func(ctx hatchet.Context, _ any) (stepOutput, error) {
				mu.Lock()
				defer mu.Unlock()

				received[name] = map[string]string{}

				return stepOutput{Step: name}, nil
			},
			hatchet.WithWaitFor(hatchet.UserEventCondition(eventKey, "")),
		))
	}

	workflow.NewTask("join",
		func(ctx hatchet.Context, _ any) (stepOutput, error) {
			mu.Lock()
			defer mu.Unlock()

			received["join"] = map[string]string{}

			for _, w := range waiters {
				var out stepOutput
				if err := ctx.ParentOutput(w, &out); err == nil {
					received["join"][w.GetName()] = out.Step
				}
			}

			return stepOutput{Step: "join"}, nil
		},
		hatchet.WithParents(waiters...),
	)

	env.startWorker(t, workflow)

	ref, err := workflow.RunNoWait(ctx, map[string]any{})
	require.NoError(t, err)

	env.requireOrchestratorEvicted(ctx, t, uuid.MustParse(ref.RunId))

	require.NoError(t, env.client.Events().Push(ctx, eventKey, map[string]any{}))

	_, err = ref.ResultWithContext(ctx)
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()

	assert.Equal(t, map[string]map[string]string{
		"a":    {},
		"b":    {},
		"c":    {},
		"join": {"a": "a", "b": "b", "c": "c"},
	}, received, "every step must run and receive its parents' outputs")
}
