//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type countRunFixture struct {
	tenantId           uuid.UUID
	insertedAt         time.Time
	status             sqlcv1.V1ReadableStatusOlap
	workflowId         uuid.UUID
	additionalMetadata string
}

func insertCountRunFixtures(t *testing.T, ctx context.Context, pool *pgxpool.Pool, runs []countRunFixture) {
	t.Helper()

	for i, run := range runs {
		day := run.insertedAt.UTC().Format("2006-01-02")

		_, err := pool.Exec(ctx, `
			SELECT
				create_v1_range_partition('v1_runs_olap', $1::date),
				create_v1_monthly_range_partition('v1_statuses_olap', $1::date)
		`, day)
		require.NoError(t, err)

		var metadata any
		if run.additionalMetadata != "" {
			metadata = run.additionalMetadata
		}

		_, err = pool.Exec(ctx, `
			INSERT INTO v1_runs_olap (tenant_id, id, inserted_at, readable_status, kind, workflow_id, workflow_version_id, additional_metadata)
			VALUES ($1, $2, $3, $4, 'TASK', $5, $6, $7::jsonb)
		`, run.tenantId, int64(i+1), run.insertedAt, run.status, run.workflowId, uuid.New(), metadata)
		require.NoError(t, err)
	}
}

func TestCountWorkflowRunsActiveBeforeWindow(t *testing.T) {
	ctx := context.Background()

	basePool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	pool := createEnumAwarePool(t, basePool)
	repo := createOLAPRepositoryWithPayloadStore(t, pool)

	tenantId := uuid.New()
	otherTenantId := uuid.New()
	workflowA := uuid.New()
	workflowB := uuid.New()

	now := time.Now().UTC().Truncate(time.Microsecond)
	since := now.Add(-72 * time.Hour)
	windowStart := now.Add(-time.Hour)

	insertCountRunFixtures(t, ctx, pool, []countRunFixture{
		{tenantId: tenantId, insertedAt: now.Add(-48 * time.Hour), status: sqlcv1.V1ReadableStatusOlapQUEUED, workflowId: workflowA, additionalMetadata: `{"env":"prod"}`},
		{tenantId: tenantId, insertedAt: now.Add(-30 * time.Hour), status: sqlcv1.V1ReadableStatusOlapRUNNING, workflowId: workflowB},
		{tenantId: tenantId, insertedAt: now.Add(-30 * time.Hour), status: sqlcv1.V1ReadableStatusOlapEVICTED, workflowId: workflowA},
		{tenantId: tenantId, insertedAt: windowStart.Add(-time.Microsecond), status: sqlcv1.V1ReadableStatusOlapRUNNING, workflowId: workflowB},

		{tenantId: tenantId, insertedAt: now.Add(-30 * time.Hour), status: sqlcv1.V1ReadableStatusOlapCOMPLETED, workflowId: workflowA},
		{tenantId: tenantId, insertedAt: now.Add(-30 * time.Hour), status: sqlcv1.V1ReadableStatusOlapFAILED, workflowId: workflowA},
		{tenantId: tenantId, insertedAt: windowStart, status: sqlcv1.V1ReadableStatusOlapRUNNING, workflowId: workflowA},
		{tenantId: tenantId, insertedAt: now.Add(-10 * time.Minute), status: sqlcv1.V1ReadableStatusOlapQUEUED, workflowId: workflowA},
		{tenantId: tenantId, insertedAt: since.Add(-time.Hour), status: sqlcv1.V1ReadableStatusOlapRUNNING, workflowId: workflowA},
		{tenantId: otherTenantId, insertedAt: now.Add(-30 * time.Hour), status: sqlcv1.V1ReadableStatusOlapRUNNING, workflowId: workflowA},
	})

	beforeWindow := windowStart.Add(-time.Microsecond)
	active := []sqlcv1.V1ReadableStatusOlap{
		sqlcv1.V1ReadableStatusOlapQUEUED,
		sqlcv1.V1ReadableStatusOlapRUNNING,
		sqlcv1.V1ReadableStatusOlapEVICTED,
	}

	cases := []struct {
		name string
		opts ListWorkflowRunOpts
		want int64
	}{
		{
			name: "all active statuses",
			opts: ListWorkflowRunOpts{CreatedAfter: since, FinishedBefore: &beforeWindow, Statuses: active},
			want: 4,
		},
		{
			name: "running only",
			opts: ListWorkflowRunOpts{CreatedAfter: since, FinishedBefore: &beforeWindow, Statuses: []sqlcv1.V1ReadableStatusOlap{sqlcv1.V1ReadableStatusOlapRUNNING}},
			want: 2,
		},
		{
			name: "workflow filter",
			opts: ListWorkflowRunOpts{CreatedAfter: since, FinishedBefore: &beforeWindow, Statuses: active, WorkflowIds: []uuid.UUID{workflowA}},
			want: 2,
		},
		{
			name: "additional metadata filter",
			opts: ListWorkflowRunOpts{
				CreatedAfter:               since,
				FinishedBefore:             &beforeWindow,
				Statuses:                   active,
				AdditionalMetadata:         map[string]interface{}{"env": "prod"},
				AdditionalMetadataOperator: AdditionalMetadataOperatorAnd,
			},
			want: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			count, capped, err := repo.CountWorkflowRuns(ctx, tenantId, tc.opts)
			require.NoError(t, err)
			assert.Equal(t, tc.want, count)
			assert.False(t, capped)

			_, total, err := repo.ListWorkflowRuns(ctx, tenantId, ListWorkflowRunOpts{
				CreatedAfter:               tc.opts.CreatedAfter,
				FinishedBefore:             tc.opts.FinishedBefore,
				Statuses:                   tc.opts.Statuses,
				WorkflowIds:                tc.opts.WorkflowIds,
				AdditionalMetadata:         tc.opts.AdditionalMetadata,
				AdditionalMetadataOperator: tc.opts.AdditionalMetadataOperator,
				Limit:                      50,
			})
			require.NoError(t, err)
			assert.Equal(t, int(count), total, "count must match the run list total for the same filters")
		})
	}
}
