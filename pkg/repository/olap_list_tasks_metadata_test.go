//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/enums"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type taskMetadataFixture struct {
	tenantId           uuid.UUID
	insertedAt         pgtype.Timestamptz
	standaloneWorkflow uuid.UUID
	dagWorkflow        uuid.UUID
	prodTask           uuid.UUID
	stagingTask        uuid.UUID
	dagChildren        []uuid.UUID
	dagExternalId      uuid.UUID
}

func seedTasksWithMetadata(t *testing.T, ctx context.Context, repo *OLAPRepositoryImpl) taskMetadataFixture {
	t.Helper()

	f := taskMetadataFixture{
		tenantId:           uuid.New(),
		insertedAt:         pgtype.Timestamptz{Time: time.Now().UTC().Truncate(time.Microsecond), Valid: true},
		standaloneWorkflow: uuid.New(),
		dagWorkflow:        uuid.New(),
		prodTask:           uuid.New(),
		stagingTask:        uuid.New(),
		dagChildren:        []uuid.UUID{uuid.New(), uuid.New()},
		dagExternalId:      uuid.New(),
	}

	newTask := func(id int64, externalId uuid.UUID, workflowId uuid.UUID, metadata string) *V1TaskWithPayload {
		return &V1TaskWithPayload{
			V1Task: &sqlcv1.V1Task{
				ID:                 id,
				InsertedAt:         f.insertedAt,
				TenantID:           f.tenantId,
				Queue:              "default",
				ActionID:           "test:metadata",
				StepID:             uuid.New(),
				WorkflowID:         workflowId,
				WorkflowVersionID:  uuid.New(),
				WorkflowRunID:      externalId,
				ScheduleTimeout:    "5m",
				Sticky:             sqlcv1.V1StickyStrategyNONE,
				ExternalID:         externalId,
				DisplayName:        "metadata-task",
				Input:              []byte(`{}`),
				AdditionalMetadata: []byte(metadata),
			},
			Payload: []byte(`{}`),
		}
	}

	_, locksNotAcquired, err := repo.CreateTasks(ctx, f.tenantId, []*V1TaskWithPayload{
		newTask(1, f.prodTask, f.standaloneWorkflow, `{"env": "prod", "team": "a"}`),
		newTask(2, f.stagingTask, f.standaloneWorkflow, `{"env": "staging", "team": "b"}`),
	})
	require.NoError(t, err)
	require.Empty(t, locksNotAcquired)

	dagId := int64(10)

	locksNotAcquired, err = repo.CreateDAGs(ctx, f.tenantId, []*DAGWithData{{
		V1Dag: &sqlcv1.V1Dag{
			ID:                dagId,
			InsertedAt:        f.insertedAt,
			TenantID:          f.tenantId,
			ExternalID:        f.dagExternalId,
			DisplayName:       "metadata-dag",
			WorkflowID:        f.dagWorkflow,
			WorkflowVersionID: uuid.New(),
		},
		Input:              []byte(`{}`),
		AdditionalMetadata: []byte(`{"env": "prod", "team": "c"}`),
	}})
	require.NoError(t, err)
	require.Empty(t, locksNotAcquired)

	children := make([]*V1TaskWithPayload, 0, len(f.dagChildren))

	for i, childExternalId := range f.dagChildren {
		child := newTask(dagId+int64(i)+1, childExternalId, f.dagWorkflow, `{"env": "prod", "team": "c"}`)
		child.WorkflowRunID = f.dagExternalId
		child.DagID = pgtype.Int8{Int64: dagId, Valid: true}
		child.DagInsertedAt = f.insertedAt
		children = append(children, child)
	}

	_, locksNotAcquired, err = repo.CreateTasks(ctx, f.tenantId, children)
	require.NoError(t, err)
	require.Empty(t, locksNotAcquired)

	return f
}

func TestListTasks_AdditionalMetadataFilters(t *testing.T) {
	ctx, _, repo := setupOLAPRepository(t)
	f := seedTasksWithMetadata(t, ctx, repo)

	allTasks := append([]uuid.UUID{f.prodTask, f.stagingTask}, f.dagChildren...)
	prodTasks := append([]uuid.UUID{f.prodTask}, f.dagChildren...)

	cases := []struct {
		name     string
		opts     ListTaskRunOpts
		expected []uuid.UUID
	}{
		{
			name:     "no metadata filter",
			expected: allTasks,
		},
		{
			name: "single key matches standalone tasks and DAG children through the DAG run",
			opts: ListTaskRunOpts{
				AdditionalMetadata:         map[string]interface{}{"env": "prod"},
				AdditionalMetadataOperator: AdditionalMetadataOperatorAnd,
			},
			expected: prodTasks,
		},
		{
			name: "AND requires every pair",
			opts: ListTaskRunOpts{
				AdditionalMetadata:         map[string]interface{}{"env": "prod", "team": "a"},
				AdditionalMetadataOperator: AdditionalMetadataOperatorAnd,
			},
			expected: []uuid.UUID{f.prodTask},
		},
		{
			name: "OR matches any pair",
			opts: ListTaskRunOpts{
				AdditionalMetadata:         map[string]interface{}{"env": "staging", "team": "c"},
				AdditionalMetadataOperator: AdditionalMetadataOperatorOr,
			},
			expected: append([]uuid.UUID{f.stagingTask}, f.dagChildren...),
		},
		{
			name: "OR with no matching pair",
			opts: ListTaskRunOpts{
				AdditionalMetadata:         map[string]interface{}{"env": "dev", "team": "z"},
				AdditionalMetadataOperator: AdditionalMetadataOperatorOr,
			},
			expected: nil,
		},
		{
			name: "metadata filter combined with workflow filter",
			opts: ListTaskRunOpts{
				AdditionalMetadata:         map[string]interface{}{"env": "prod"},
				AdditionalMetadataOperator: AdditionalMetadataOperatorAnd,
				WorkflowIds:                []uuid.UUID{f.dagWorkflow},
			},
			expected: f.dagChildren,
		},
		{
			name: "workflow filter without metadata",
			opts: ListTaskRunOpts{
				WorkflowIds: []uuid.UUID{f.standaloneWorkflow},
			},
			expected: []uuid.UUID{f.prodTask, f.stagingTask},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := tc.opts
			opts.CreatedAfter = f.insertedAt.Time.Add(-time.Hour)
			opts.Limit = 100
			opts.IncludeTotalCount = true

			tasks, count, err := repo.ListTasks(ctx, f.tenantId, opts)
			require.NoError(t, err)

			externalIds := make([]uuid.UUID, 0, len(tasks))

			for _, task := range tasks {
				externalIds = append(externalIds, task.ExternalID)
			}

			require.ElementsMatch(t, tc.expected, externalIds)
			require.Equal(t, len(tc.expected), count)
		})
	}
}

func TestListTasks_DAGChildrenReadMetadataFromTheDAGRun(t *testing.T) {
	ctx, _, repo := setupOLAPRepository(t)
	f := seedTasksWithMetadata(t, ctx, repo)

	tasks, _, err := repo.ListTasks(ctx, f.tenantId, ListTaskRunOpts{
		CreatedAfter: f.insertedAt.Time.Add(-time.Hour),
		WorkflowIds:  []uuid.UUID{f.dagWorkflow},
		Limit:        100,
	})
	require.NoError(t, err)
	require.Len(t, tasks, len(f.dagChildren))

	for _, task := range tasks {
		var metadata map[string]string
		require.NoError(t, json.Unmarshal(task.AdditionalMetadata, &metadata))
		require.Equal(t, map[string]string{"env": "prod", "team": "c"}, metadata)

		queuedEvent := sqlcv1.CreateTaskEventsOLAPParams{
			TenantID:       f.tenantId,
			TaskID:         task.ID,
			TaskInsertedAt: task.InsertedAt,
			EventType:      enums.V1EventTypeOlapQUEUED,
			WorkflowID:     f.dagWorkflow,
			EventTimestamp: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
			ReadableStatus: enums.V1ReadableStatusOlapQUEUED,
			ExternalID:     uuid.New(),
		}
		_, locksNotAcquired, err := repo.CreateTaskEvents(ctx, f.tenantId, []sqlcv1.CreateTaskEventsOLAPParams{queuedEvent}, map[uuid.UUID]uuid.UUID{queuedEvent.ExternalID: f.dagExternalId}, nil, nil, nil)
		require.NoError(t, err)
		require.Empty(t, locksNotAcquired)

		singleTask, _, err := repo.ReadTaskRunData(ctx, f.tenantId, task.ID, task.InsertedAt, nil)
		require.NoError(t, err)
		require.JSONEq(t, string(task.AdditionalMetadata), string(singleTask.AdditionalMetadata))
	}
}

func TestCreateTasks_TaskCreatedAsFailedHasRunFinishedAt(t *testing.T) {
	ctx, _, repo := setupOLAPRepository(t)

	tenantId := uuid.New()
	beforeCreate := time.Now().Add(-time.Second)

	_, locksNotAcquired, err := repo.CreateTasks(ctx, tenantId, []*V1TaskWithPayload{{
		V1Task: &sqlcv1.V1Task{
			ID:                 1,
			InsertedAt:         pgtype.Timestamptz{Time: time.Now().UTC().Truncate(time.Microsecond), Valid: true},
			TenantID:           tenantId,
			Queue:              "default",
			ActionID:           "test:failed-at-creation",
			StepID:             uuid.New(),
			WorkflowID:         uuid.New(),
			WorkflowVersionID:  uuid.New(),
			WorkflowRunID:      uuid.New(),
			ScheduleTimeout:    "5m",
			Sticky:             sqlcv1.V1StickyStrategyNONE,
			ExternalID:         uuid.New(),
			DisplayName:        "failed-at-creation",
			Input:              []byte(`{}`),
			AdditionalMetadata: []byte(`{}`),
			InitialState:       sqlcv1.V1TaskInitialStateFAILED,
		},
		Payload: []byte(`{}`),
	}})
	require.NoError(t, err)
	require.Empty(t, locksNotAcquired)

	failedRuns, err := repo.ListFailedWorkflowRunsByFinishedAt(ctx, tenantId, beforeCreate, time.Now().Add(time.Minute), 10)
	require.NoError(t, err)
	require.Len(t, failedRuns, 1)
}
