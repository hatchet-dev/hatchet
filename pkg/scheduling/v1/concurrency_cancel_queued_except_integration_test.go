//go:build integration

package v1_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/config/database"
)

// countConcurrencySlots returns the number of filled (running) and unfilled (still-queued)
// v1_concurrency_slot rows for a strategy. The CANCEL_QUEUED_EXCEPT_* queries leave "spared" queued
// slots untouched (no result row), so counting the table directly is the only way to see how many
// survivors a pass actually kept.
func countConcurrencySlots(t *testing.T, ctx context.Context, conf *database.Layer, strategyID int64) (running, queued int) {
	t.Helper()

	err := conf.Pool.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE is_filled),
			count(*) FILTER (WHERE NOT is_filled)
		FROM v1_concurrency_slot
		WHERE strategy_id = $1
	`, strategyID).Scan(&running, &queued)
	require.NoError(t, err)

	return running, queued
}

// TestConcurrency_CancelQueuedExcept_StandaloneKeepsOnlyMaxRuns pins the core invariant of the
// CANCEL_QUEUED_EXCEPT_NEWEST / CANCEL_QUEUED_EXCEPT_OLDEST strategies on the no-parent
// (task-level) path, which runs RunCancelQueuedExceptNewest / RunCancelQueuedExceptOldest:
//
// after one scheduling pass over a backlog on a single key, the surviving footprint is exactly
// 2*maxRuns slots - maxRuns promoted to running, plus maxRuns still queued (kept for later
// promotion) - and everything else is cancelled.
func TestConcurrency_CancelQueuedExcept_StandaloneKeepsOnlyMaxRuns(t *testing.T) {
	const (
		maxRuns  = 2
		numTasks = 8 // 2 running + 2 queued survivors + 4 cancelled
	)

	cases := []struct {
		name     string
		strategy string
	}{
		{"except_newest", "CANCEL_QUEUED_EXCEPT_NEWEST"},
		{"except_oldest", "CANCEL_QUEUED_EXCEPT_OLDEST"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			runWithDatabase(t, func(conf *database.Layer) error {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				requireSchedulerSchema(t, ctx, conf)

				s := setupStepConcurrencyTest(t, ctx, conf, "cqe-standalone-"+tc.name, tc.strategy, maxRuns, numTasks)

				res, err := s.concurrencyRepo.RunConcurrencyStrategy(ctx, s.tenantId, s.strategy)
				require.NoError(t, err)

				require.Len(t, res.Queued, maxRuns,
					"%s must promote exactly maxRuns tasks to running", tc.strategy)
				require.Len(t, res.Cancelled, numTasks-2*maxRuns,
					"%s must cancel everything outside the maxRuns running + maxRuns queued bands", tc.strategy)

				running, queued := countConcurrencySlots(t, ctx, conf, s.strategy.ID)
				require.Equal(t, maxRuns, running, "%s: exactly maxRuns slots filled", tc.strategy)
				require.Equal(t, maxRuns, queued,
					"%s: exactly maxRuns slots kept queued (expected maxRuns, not 2*maxRuns and not 0)", tc.strategy)

				return nil
			})
		})
	}
}
