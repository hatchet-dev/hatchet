//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	dto "github.com/prometheus/client_model/go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func holdRows(t *testing.T, ctx context.Context, pool *pgxpool.Pool) map[uuid.UUID]*sqlcv1.V1TenantBacklogHold {
	t.Helper()

	rows, err := pool.Query(ctx, "SELECT tenant_id, backlog, backlog_limit, largest_state, held_until FROM v1_tenant_backlog_hold")
	require.NoError(t, err)
	defer rows.Close()

	res := make(map[uuid.UUID]*sqlcv1.V1TenantBacklogHold)

	for rows.Next() {
		var h sqlcv1.V1TenantBacklogHold
		require.NoError(t, rows.Scan(&h.TenantID, &h.Backlog, &h.BacklogLimit, &h.LargestState, &h.HeldUntil))
		res[h.TenantID] = &h
	}

	require.NoError(t, rows.Err())

	return res
}

func insertHold(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantId uuid.UUID, heldUntil string) {
	t.Helper()

	_, err := pool.Exec(ctx, `
		INSERT INTO v1_tenant_backlog_hold (tenant_id, backlog, backlog_limit, largest_state, held_until)
		VALUES ($1, 20, 10, 'queued', NOW() + $2::interval)`, tenantId, heldUntil)
	require.NoError(t, err)
}

func TestTenantBacklogHoldUpdate(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	repo := newTenantBacklogHoldRepository(createTaskRepository(pool).sharedRepository)

	clearHolds := func() {
		_, err := pool.Exec(ctx, "DELETE FROM v1_tenant_backlog_hold")
		require.NoError(t, err)
	}

	t.Run("a tenant at or over the limit is held with its largest state", func(t *testing.T) {
		clearHolds()

		over := uuid.New()
		at := uuid.New()
		under := uuid.New()

		require.NoError(t, repo.Update(ctx, 10, map[uuid.UUID]TenantBacklog{
			over:  {Queued: 4, Paused: 7, Retry: 1},
			at:    {Concurrency: 5, RateLimited: 5},
			under: {Queued: 9},
		}))

		holds := holdRows(t, ctx, pool)
		require.Len(t, holds, 2)

		require.Equal(t, int64(12), holds[over].Backlog)
		require.Equal(t, int64(10), holds[over].BacklogLimit)
		require.Equal(t, "paused", holds[over].LargestState)
		require.WithinDuration(t, time.Now().Add(2*time.Minute), holds[over].HeldUntil.Time, 30*time.Second)

		// a tie goes to the earlier state
		require.Equal(t, "concurrency", holds[at].LargestState)
	})

	t.Run("a held tenant measured under the limit is released, and a renewed one is updated", func(t *testing.T) {
		clearHolds()

		draining := uuid.New()
		growing := uuid.New()

		require.NoError(t, repo.Update(ctx, 10, map[uuid.UUID]TenantBacklog{
			draining: {Queued: 10},
			growing:  {Queued: 10},
		}))
		require.Len(t, holdRows(t, ctx, pool), 2)

		require.NoError(t, repo.Update(ctx, 10, map[uuid.UUID]TenantBacklog{
			draining: {Queued: 9},
			growing:  {Queued: 10, Batched: 30},
		}))

		holds := holdRows(t, ctx, pool)
		require.Len(t, holds, 1)
		require.Equal(t, int64(40), holds[growing].Backlog)
		require.Equal(t, "batched", holds[growing].LargestState)
	})

	t.Run("a held tenant not measured this pass keeps its hold", func(t *testing.T) {
		clearHolds()

		otherPartition := uuid.New()
		insertHold(t, ctx, pool, otherPartition, "1 minute")

		require.NoError(t, repo.Update(ctx, 10, map[uuid.UUID]TenantBacklog{uuid.New(): {}}))
		require.Contains(t, holdRows(t, ctx, pool), otherPartition)
	})

	t.Run("an expired hold of a tenant not measured this pass is left alone", func(t *testing.T) {
		clearHolds()

		expired := uuid.New()
		insertHold(t, ctx, pool, expired, "-1 second")

		require.NoError(t, repo.Update(ctx, 10, map[uuid.UUID]TenantBacklog{uuid.New(): {}}))
		require.Contains(t, holdRows(t, ctx, pool), expired)
	})

	t.Run("an expired hold is overwritten by the tenant's next hold", func(t *testing.T) {
		clearHolds()

		tenantId := uuid.New()
		insertHold(t, ctx, pool, tenantId, "-1 second")

		require.NoError(t, repo.Update(ctx, 10, map[uuid.UUID]TenantBacklog{tenantId: {Retry: 11}}))

		holds := holdRows(t, ctx, pool)
		require.Len(t, holds, 1)
		require.Equal(t, "retry", holds[tenantId].LargestState)
		require.True(t, holds[tenantId].HeldUntil.Time.After(time.Now()))
	})

	t.Run("a hold is logged when it starts and when it ends, not while it is renewed", func(t *testing.T) {
		clearHolds()

		var buf bytes.Buffer
		l := zerolog.New(&buf)
		logged := newTenantBacklogHoldRepository(&sharedRepository{pool: fairpool.Ungated(pool), l: &l, queries: sqlcv1.New()})

		lines := func() []string {
			defer buf.Reset()

			var res []string

			for _, line := range bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n")) {
				if len(line) == 0 {
					continue
				}

				var entry map[string]any
				require.NoError(t, json.Unmarshal(line, &entry))
				res = append(res, fmt.Sprintf("%s %s %v", entry["level"], entry["tenant_id"], entry["largest_state"]))
			}

			return res
		}

		tenantId := uuid.New()

		require.NoError(t, logged.Update(ctx, 10, map[uuid.UUID]TenantBacklog{tenantId: {Queued: 11}}))
		require.Equal(t, []string{"warn " + tenantId.String() + " queued"}, lines())

		require.NoError(t, logged.Update(ctx, 10, map[uuid.UUID]TenantBacklog{tenantId: {Queued: 12}}))
		require.Empty(t, lines())

		require.NoError(t, logged.Update(ctx, 10, map[uuid.UUID]TenantBacklog{tenantId: {Queued: 2}}))
		require.Equal(t, []string{"info " + tenantId.String() + " <nil>"}, lines())

		require.NoError(t, logged.Update(ctx, 10, map[uuid.UUID]TenantBacklog{tenantId: {Queued: 2}}))
		require.Empty(t, lines())
	})

	t.Run("a limit above the default measurement ceiling holds a tenant measured with it", func(t *testing.T) {
		clearHolds()

		tasks := createTaskRepository(pool)
		tenantId := uuid.New()
		seedBacklogRows(t, ctx, pool, "v1_queue_item", tenantId, 30)

		// measured with the default ceiling below the limit, the tenant never reaches it
		const defaultCeiling, limit = 20, 25

		backlogs, err := tasks.GetTenantBacklogs(ctx, []uuid.UUID{tenantId}, defaultCeiling)
		require.NoError(t, err)
		require.NoError(t, repo.Update(ctx, limit, backlogs))
		require.Empty(t, holdRows(t, ctx, pool))

		backlogs, err = tasks.GetTenantBacklogs(ctx, []uuid.UUID{tenantId}, max(defaultCeiling, limit))
		require.NoError(t, err)
		require.NoError(t, repo.Update(ctx, limit, backlogs))
		require.Contains(t, holdRows(t, ctx, pool), tenantId)
	})
}

func TestTenantBacklogHoldAdmit(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	repo := newTenantBacklogHoldRepository(createTaskRepository(pool).sharedRepository)

	held := uuid.New()
	expired := uuid.New()
	free := uuid.New()

	require.NoError(t, repo.Update(ctx, 10, map[uuid.UUID]TenantBacklog{held: {Paused: 12}, free: {Paused: 1}}))
	insertHold(t, ctx, pool, expired, "-1 second")

	// nothing is refused before the first refresh
	require.NoError(t, repo.Admit(held, 1))

	require.NoError(t, repo.refresh(ctx))

	t.Run("a held tenant is refused with its hold", func(t *testing.T) {
		err := repo.Admit(held, 1)

		var refusal *TenantBacklogOverLimitError
		require.ErrorAs(t, err, &refusal)
		require.Equal(t, TenantBacklogOverLimitError{Backlog: 12, Limit: 10, LargestState: "paused"}, *refusal)
	})

	t.Run("tenants without an unexpired hold are admitted", func(t *testing.T) {
		require.NoError(t, repo.Admit(free, 1))
		require.NoError(t, repo.Admit(expired, 1))
	})

	t.Run("a hold that expires after the refresh no longer refuses", func(t *testing.T) {
		lapsed := uuid.New()
		snapshot := map[uuid.UUID]*sqlcv1.V1TenantBacklogHold{
			lapsed: {TenantID: lapsed, Backlog: 12, BacklogLimit: 10, LargestState: "queued", HeldUntil: pgtype.Timestamptz{Time: time.Now().Add(-time.Millisecond), Valid: true}},
		}
		prev := repo.snapshot.Swap(&snapshot)
		defer repo.snapshot.Store(prev)

		require.NoError(t, repo.Admit(lapsed, 1))
	})

	t.Run("refused runs are counted per item", func(t *testing.T) {
		refused := func() float64 {
			m := &dto.Metric{}
			require.NoError(t, prometheus.TenantBacklogRefused.WithLabelValues(held.String()).Write(m))

			return m.GetCounter().GetValue()
		}
		before := refused()

		require.Error(t, repo.Admit(held, 3))
		require.Error(t, repo.Admit(held, 1))
		require.NoError(t, repo.Admit(free, 5))

		require.Equal(t, before+4, refused())
	})

	t.Run("RefreshLoop reads at once and returns when its context ends", func(t *testing.T) {
		looped := newTenantBacklogHoldRepository(createTaskRepository(pool).sharedRepository)

		loopCtx, stop := context.WithCancel(ctx)
		defer stop()

		done := make(chan struct{})

		go func() {
			looped.RefreshLoop(loopCtx)
			close(done)
		}()

		// well before the first tick
		require.Eventually(t, func() bool { return looped.Admit(held, 1) != nil }, tenantBacklogHoldRefreshInterval/2, 10*time.Millisecond)

		stop()

		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("RefreshLoop did not return after its context ended")
		}
	})

	t.Run("a failed refresh keeps the last snapshot", func(t *testing.T) {
		pool.Close()

		require.Error(t, repo.refresh(ctx))
		require.Error(t, repo.Admit(held, 1))
		require.NoError(t, repo.Admit(free, 1))
	})
}
