//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestTasksForDispatchFromRows(t *testing.T) {
	tenantId := uuid.New()
	insertedAt := pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	ownerInsertedAt := pgtype.Timestamptz{Time: time.Now().UTC().Add(-time.Minute), Valid: true}
	ownExternalId := uuid.New()
	referenceId := uuid.New()
	workerId := uuid.New()
	batchId := uuid.New()
	timeoutAt := pgtype.Timestamp{Time: time.Now().UTC(), Valid: true}

	withOwnRow := &sqlcv1.ListTasksForDispatchRow{
		V1Task:               sqlcv1.V1Task{ID: 1, InsertedAt: insertedAt, TenantID: tenantId, RetryCount: 2, ExternalID: ownExternalId},
		RuntimeWorkerID:      &workerId,
		RuntimeBatchID:       &batchId,
		RuntimeBatchSize:     pgtype.Int4{Int32: 3, Valid: true},
		RuntimeBatchIndex:    pgtype.Int4{Int32: 1, Valid: true},
		RuntimeBatchKey:      pgtype.Text{String: "k", Valid: true},
		RuntimeTimeoutAt:     timeoutAt,
		PayloadID:            pgtype.Int8{Int64: 1, Valid: true},
		PayloadInsertedAt:    insertedAt,
		PayloadExternalID:    &ownExternalId,
		PayloadType:          sqlcv1.NullV1PayloadType{V1PayloadType: sqlcv1.V1PayloadTypeTASKINPUT, Valid: true},
		PayloadLocation:      sqlcv1.NullV1PayloadLocation{V1PayloadLocation: sqlcv1.V1PayloadLocationINLINE, Valid: true},
		PayloadInlineContent: []byte(`{"a":1}`),
		PayloadUpdatedAt:     insertedAt,
	}

	// a task reading the owner's row: the payload columns are the owner's, not the task's
	referencing := &sqlcv1.ListTasksForDispatchRow{
		V1Task:               sqlcv1.V1Task{ID: 2, InsertedAt: insertedAt, TenantID: tenantId, PayloadReferenceID: &referenceId},
		RuntimeTimeoutAt:     timeoutAt,
		PayloadID:            pgtype.Int8{Int64: 99, Valid: true},
		PayloadInsertedAt:    ownerInsertedAt,
		PayloadExternalID:    &referenceId,
		PayloadType:          sqlcv1.NullV1PayloadType{V1PayloadType: sqlcv1.V1PayloadTypeUSEREVENTINPUT, Valid: true},
		PayloadLocation:      sqlcv1.NullV1PayloadLocation{V1PayloadLocation: sqlcv1.V1PayloadLocationINLINE, Valid: true},
		PayloadInlineContent: []byte(`{"b":2}`),
		PayloadUpdatedAt:     ownerInsertedAt,
	}

	// the same task joined to a second row sharing the reference id: the task is kept once, both rows are kept
	referencingAgain := &sqlcv1.ListTasksForDispatchRow{
		V1Task:               referencing.V1Task,
		RuntimeTimeoutAt:     timeoutAt,
		PayloadID:            pgtype.Int8{Int64: 100, Valid: true},
		PayloadInsertedAt:    ownerInsertedAt,
		PayloadExternalID:    &referenceId,
		PayloadType:          sqlcv1.NullV1PayloadType{V1PayloadType: sqlcv1.V1PayloadTypeTASKOUTPUT, Valid: true},
		PayloadLocation:      sqlcv1.NullV1PayloadLocation{V1PayloadLocation: sqlcv1.V1PayloadLocationINLINE, Valid: true},
		PayloadInlineContent: []byte(`{"c":3}`),
		PayloadUpdatedAt:     ownerInsertedAt,
	}

	// a runtime row but no payload row: every payload column is NULL
	bare := &sqlcv1.ListTasksForDispatchRow{
		V1Task:           sqlcv1.V1Task{ID: 3, InsertedAt: insertedAt, TenantID: tenantId},
		RuntimeTimeoutAt: timeoutAt,
	}

	res := tasksForDispatchFromRows([]*sqlcv1.ListTasksForDispatchRow{withOwnRow, nil, referencing, referencingAgain, bare})

	require.Len(t, res.Tasks, 3)
	assert.Equal(t, int64(1), res.Tasks[0].ID)
	assert.Equal(t, int64(2), res.Tasks[1].ID)
	assert.Equal(t, int64(3), res.Tasks[2].ID)

	require.Len(t, res.PayloadRows, 3)

	ownRow := res.PayloadRows[0]
	assert.Equal(t, tenantId, ownRow.TenantID)
	assert.Equal(t, int64(1), ownRow.ID)
	assert.Equal(t, insertedAt, ownRow.InsertedAt)
	assert.Equal(t, sqlcv1.V1PayloadTypeTASKINPUT, ownRow.Type)
	assert.Equal(t, sqlcv1.V1PayloadLocationINLINE, ownRow.Location)
	assert.Equal(t, ownExternalId, ownRow.ExternalID)
	assert.Equal(t, []byte(`{"a":1}`), ownRow.InlineContent)

	ownerRow := res.PayloadRows[1]
	assert.Equal(t, int64(99), ownerRow.ID)
	assert.Equal(t, ownerInsertedAt, ownerRow.InsertedAt)
	assert.Equal(t, sqlcv1.V1PayloadTypeUSEREVENTINPUT, ownerRow.Type)
	assert.Equal(t, referenceId, ownerRow.ExternalID)
	assert.Equal(t, []byte(`{"b":2}`), ownerRow.InlineContent)

	assert.Equal(t, int64(100), res.PayloadRows[2].ID)
	assert.Equal(t, sqlcv1.V1PayloadTypeTASKOUTPUT, res.PayloadRows[2].Type)

	require.Len(t, res.Runtimes, 3)

	runtime := res.Runtimes[1]
	require.NotNil(t, runtime)
	assert.Equal(t, int64(1), runtime.TaskID)
	assert.Equal(t, insertedAt, runtime.TaskInsertedAt)
	assert.Equal(t, int32(2), runtime.RetryCount)
	assert.Equal(t, tenantId, runtime.TenantID)
	assert.Equal(t, &workerId, runtime.WorkerID)
	assert.Equal(t, &batchId, runtime.BatchID)
	assert.Equal(t, int32(3), runtime.BatchSize.Int32)
	assert.Equal(t, int32(1), runtime.BatchIndex.Int32)
	assert.Equal(t, "k", runtime.BatchKey.String)

	require.NotNil(t, res.Runtimes[3])
	assert.Nil(t, res.Runtimes[3].WorkerID)
}

// TestListTasksForDispatchMatchesThreeReadPath checks, for every combination of the payload store flags,
// that the dispatcher's single read plus Resolve yields the same task rows, runtime rows, inputs and
// step run data bytes as ListTasks + Retrieve + ListTaskRuntimes for:
//   - a task with its own TASK_INPUT row
//   - a task whose payload_reference_id points at a USER_EVENT_INPUT row owned by an event
//   - a task whose payload_reference_id points at a DAG_INPUT row owned by its DAG
//   - a task with an empty input
//
// and that two tasks are left out of the result: one whose runtime row belongs to another tenant, and
// one whose runtime row was deleted after assignment.
//
// The fixtures are created the way the engine creates them: the trigger repository mints the DAG's
// reference id, and the event's reference id is minted the way the ingestor mints it.
func TestListTasksForDispatchMatchesThreeReadPath(t *testing.T) {
	for _, wrapTaskInputInStepRunData := range []bool{true, false} {
		for _, enablePayloadConsolidation := range []bool{true, false} {
			t.Run(fmt.Sprintf("wrapTaskInputInStepRunData=%t/enablePayloadConsolidation=%t", wrapTaskInputInStepRunData, enablePayloadConsolidation), func(t *testing.T) {
				assertDispatchReadMatchesThreeReadPath(t, wrapTaskInputInStepRunData, enablePayloadConsolidation)
			})
		}
	}
}

func assertDispatchReadMatchesThreeReadPath(t *testing.T, wrapTaskInputInStepRunData, enablePayloadConsolidation bool) {
	t.Helper()

	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	repo := newBatchTestRepository(pool)
	shared := repo.sharedRepository
	shared.wrapTaskInputInStepRunData = wrapTaskInputInStepRunData
	shared.enablePayloadConsolidation = enablePayloadConsolidation
	shared.payloadStore = NewPayloadStoreRepository(pool, shared.l, shared.queries, PayloadStoreRepositoryOpts{
		WrapTaskInputInStepRunData: wrapTaskInputInStepRunData,
		EnablePayloadConsolidation: enablePayloadConsolidation,
	})

	ctx := context.Background()
	workflows := &workflowRepository{sharedRepository: shared}
	triggers := newTriggerRepository(shared)

	const eventKey = "dispatch-reads:event"

	eventWorkflow := minimalWorkflowOpts("dispatch-reads-event", "v1", nil)
	eventWorkflow.EventTriggers = []string{eventKey}

	dagWorkflow := minimalWorkflowOpts("dispatch-reads-dag", "v1", nil)
	dagWorkflow.Tasks = append(dagWorkflow.Tasks, CreateStepOpts{
		ReadableId: "step2",
		Action:     "integration:step2",
		Parents:    []string{"step1"},
	})

	for _, opts := range []*CreateWorkflowVersionOpts{
		minimalWorkflowOpts("dispatch-reads-single", "v1", nil),
		minimalWorkflowOpts("dispatch-reads-empty", "v1", nil),
		eventWorkflow,
		dagWorkflow,
	} {
		_, err := workflows.PutWorkflowVersion(ctx, internalTenantId, opts)
		require.NoError(t, err)
	}

	expectedInputs := make(map[int64]string)

	// the single workflow is triggered three times: one task is dispatched, one carries a runtime row of
	// another tenant, and one has its runtime row deleted before the read
	ownRowExternalId := uuid.New()
	foreignRuntimeExternalId := uuid.New()
	releasedExternalId := uuid.New()

	namedTasks, _, _, _, err := triggers.TriggerFromWorkflowNames(ctx, internalTenantId, []*WorkflowNameTriggerOpts{
		{TriggerTaskData: &TriggerTaskData{WorkflowName: "dispatch-reads-single", Data: []byte(`{"order_id": 1}`)}, ExternalId: ownRowExternalId},
		{TriggerTaskData: &TriggerTaskData{WorkflowName: "dispatch-reads-single", Data: []byte(`{"order_id": 4}`)}, ExternalId: foreignRuntimeExternalId},
		{TriggerTaskData: &TriggerTaskData{WorkflowName: "dispatch-reads-single", Data: []byte(`{"order_id": 5}`)}, ExternalId: releasedExternalId},
		{TriggerTaskData: &TriggerTaskData{WorkflowName: "dispatch-reads-empty"}, ExternalId: uuid.New()},
		{TriggerTaskData: &TriggerTaskData{WorkflowName: "dispatch-reads-dag", Data: []byte(`{"order_id": 3}`)}, ExternalId: uuid.New()},
	})
	require.NoError(t, err)
	require.Len(t, namedTasks, 5, "step2 waits on step1, so each workflow creates one task at trigger time")

	var ownRowTask, foreignRuntimeTask, releasedTask, emptyTask, dagTask *V1TaskWithPayload

	for _, task := range namedTasks {
		switch task.WorkflowID {
		case workflowIdByName(t, shared, "dispatch-reads-single"):
			switch task.ExternalID {
			case ownRowExternalId:
				ownRowTask = task
				expectedInputs[task.ID] = `{"order_id": 1}`
			case foreignRuntimeExternalId:
				foreignRuntimeTask = task
			case releasedExternalId:
				releasedTask = task
			}
		case workflowIdByName(t, shared, "dispatch-reads-empty"):
			emptyTask = task
		case workflowIdByName(t, shared, "dispatch-reads-dag"):
			dagTask = task
			expectedInputs[task.ID] = `{"order_id": 3}`
		}
	}

	require.NotNil(t, ownRowTask)
	require.NotNil(t, foreignRuntimeTask)
	require.NotNil(t, releasedTask)
	require.NotNil(t, emptyTask)
	require.NotNil(t, dagTask)

	// the ingestor mints a time-ordered reference id for an event only under payload consolidation; the
	// time in the id is the inserted_at the event's USER_EVENT_INPUT row is stored under
	var eventPayloadReferenceId *uuid.UUID

	if shared.payloadStore.PayloadConsolidationEnabled() {
		referenceId := NewPayloadReferenceId()
		eventPayloadReferenceId = &referenceId
	}

	eventRes, err := triggers.TriggerFromEvents(ctx, internalTenantId, []EventTriggerOpts{{
		ExternalId:         uuid.New(),
		SeenAt:             time.Now().UTC(),
		Key:                eventKey,
		Data:               []byte(`{"order_id": 2}`),
		PayloadReferenceId: eventPayloadReferenceId,
	}})
	require.NoError(t, err)
	require.Len(t, eventRes.Tasks, 1)

	eventTask := eventRes.Tasks[0]
	expectedInputs[eventTask.ID] = `{"order_id": 2}`

	assert.Nil(t, ownRowTask.PayloadReferenceID, "a single task triggered by name stores its own row")

	if enablePayloadConsolidation {
		require.NotNil(t, eventTask.PayloadReferenceID, "the event task must read the event's USER_EVENT_INPUT row")
		assert.Equal(t, *eventPayloadReferenceId, *eventTask.PayloadReferenceID)
		require.NotNil(t, dagTask.PayloadReferenceID, "the dag task must read the dag's DAG_INPUT row")
	} else {
		assert.Nil(t, eventTask.PayloadReferenceID)
		assert.Nil(t, dagTask.PayloadReferenceID)
	}

	// every dispatched task holds a runtime row for its current retry, as it does once the scheduler has
	// assigned it; the released task's row is deleted again, the way a release or reassignment removes it
	insertRuntime := func(task *V1TaskWithPayload, workerId, tenantId uuid.UUID) {
		_, err := pool.Exec(ctx, `
			INSERT INTO v1_task_runtime (task_id, task_inserted_at, retry_count, worker_id, tenant_id, timeout_at)
			VALUES ($1, $2, $3, $4, $5, NOW() + INTERVAL '1 minute')
		`, task.ID, task.InsertedAt, task.RetryCount, workerId, tenantId)
		require.NoError(t, err)
	}

	workerId := uuid.New()
	dispatchedTasks := []*V1TaskWithPayload{ownRowTask, emptyTask, dagTask, eventTask}

	for _, task := range dispatchedTasks {
		insertRuntime(task, workerId, internalTenantId)
	}

	insertRuntime(releasedTask, workerId, internalTenantId)

	_, err = pool.Exec(ctx, `DELETE FROM v1_task_runtime WHERE task_id = $1`, releasedTask.ID)
	require.NoError(t, err)

	// a runtime row keyed by the task's triple but carrying a different tenant_id is not the task's
	// runtime row: the runtime match applies the tenant predicate ListTaskRuntimes applies
	insertRuntime(foreignRuntimeTask, uuid.New(), uuid.New())

	allTasks := []*V1TaskWithPayload{ownRowTask, foreignRuntimeTask, releasedTask, emptyTask, dagTask, eventTask}
	taskIds := make([]int64, 0, len(allTasks))

	for _, task := range allTasks {
		taskIds = append(taskIds, task.ID)
	}

	// the three-read path
	allListedTasks, err := repo.ListTasks(ctx, internalTenantId, taskIds)
	require.NoError(t, err)
	require.Len(t, allListedTasks, len(allTasks))

	listedRuntimes, err := repo.ListTaskRuntimes(ctx, internalTenantId, allListedTasks)
	require.NoError(t, err)
	require.Len(t, listedRuntimes, len(dispatchedTasks))

	// on the three-read path a task without a runtime row is still sent; the single read leaves it out,
	// so the comparison covers the tasks that hold one
	listedTasks := make([]*sqlcv1.V1Task, 0, len(dispatchedTasks))

	for _, task := range allListedTasks {
		if _, ok := listedRuntimes[task.ID]; ok {
			listedTasks = append(listedTasks, task)
		}
	}

	require.Len(t, listedTasks, len(dispatchedTasks))

	refs := make([]PayloadRef[TaskInputShape], 0, len(listedTasks))

	for _, task := range listedTasks {
		refs = append(refs, dispatchTestPayloadRef(task))
	}

	retrieved, err := shared.payloadStore.TaskInputs().Retrieve(ctx, nil, refs...)
	require.NoError(t, err)

	// the comparison below is only meaningful if the three-read path found the stored rows
	for _, task := range []*V1TaskWithPayload{ownRowTask, dagTask, eventTask} {
		require.Contains(t, retrieved, dispatchTestPayloadRef(task.V1Task), "the three-read path must find the input of task %d", task.ID)
	}

	// the single read
	dispatchData, err := repo.ListTasksForDispatch(ctx, internalTenantId, taskIds)
	require.NoError(t, err)

	resolved, err := shared.payloadStore.TaskInputs().Resolve(ctx, nil, dispatchData.PayloadRows, refs...)
	require.NoError(t, err)

	sortTasksById(listedTasks)
	sortTasksById(dispatchData.Tasks)

	assert.Equal(t, listedTasks, dispatchData.Tasks)
	assert.Equal(t, listedRuntimes, dispatchData.Runtimes)
	assert.Equal(t, &workerId, dispatchData.Runtimes[ownRowTask.ID].WorkerID)

	returnedTaskIds := make([]int64, 0, len(dispatchData.Tasks))

	for _, task := range dispatchData.Tasks {
		returnedTaskIds = append(returnedTaskIds, task.ID)
	}

	assert.NotContains(t, returnedTaskIds, foreignRuntimeTask.ID, "a task whose only runtime row belongs to another tenant is not returned")
	assert.NotContains(t, dispatchData.Runtimes, foreignRuntimeTask.ID)
	assert.NotContains(t, returnedTaskIds, releasedTask.ID, "a task whose runtime row was deleted is not returned")
	assert.NotContains(t, dispatchData.Runtimes, releasedTask.ID)

	assert.Equal(t, len(retrieved), len(resolved))

	for ref, payload := range retrieved {
		assert.Equal(t, payload.Bytes(), resolved[ref].Bytes(), "input bytes for task %d", ref.Id)
	}

	for _, task := range listedTasks {
		ref := dispatchTestPayloadRef(task)

		oldBytes := dispatchedStepRunDataBytes(t, task, RetrievePayloadOrFallback(retrieved, ref, task.Input))
		newBytes := dispatchedStepRunDataBytes(t, task, RetrievePayloadOrFallback(resolved, ref, task.Input))

		assert.Equal(t, oldBytes, newBytes, "step run data bytes for task %d", task.ID)

		var stepRunData V1StepRunData
		require.NoError(t, json.Unmarshal(newBytes, &stepRunData))

		if expected, ok := expectedInputs[task.ID]; ok {
			assert.JSONEq(t, expected, string(stepRunData.Input), "input for task %d", task.ID)
		} else {
			assert.True(t, isEmptyJSONInput(stepRunData.Input), "input for task %d should be empty, got %s", task.ID, stepRunData.Input)
		}
	}

	// the joined payload rows are the real rows: the task's own TASK_INPUT row, or the owner's row
	payloadRowsByExternalId := make(map[uuid.UUID]*sqlcv1.V1Payload)

	for _, row := range dispatchData.PayloadRows {
		payloadRowsByExternalId[row.ExternalID] = row
	}

	ownRow, ok := payloadRowsByExternalId[ownRowTask.ExternalID]
	require.True(t, ok, "the task with its own row must join to it")
	assert.Equal(t, sqlcv1.V1PayloadTypeTASKINPUT, ownRow.Type)
	assert.Equal(t, ownRowTask.ID, ownRow.ID)

	if enablePayloadConsolidation {
		eventOwnerRow, ok := payloadRowsByExternalId[*eventTask.PayloadReferenceID]
		require.True(t, ok, "the event task must join to the event's row")
		assert.Equal(t, sqlcv1.V1PayloadTypeUSEREVENTINPUT, eventOwnerRow.Type)
		assert.NotEqual(t, eventTask.ID, eventOwnerRow.ID)

		dagOwnerRow, ok := payloadRowsByExternalId[*dagTask.PayloadReferenceID]
		require.True(t, ok, "the dag task must join to the dag's row")
		assert.Equal(t, sqlcv1.V1PayloadTypeDAGINPUT, dagOwnerRow.Type)
	}
}

func dispatchTestPayloadRef(task *sqlcv1.V1Task) PayloadRef[TaskInputShape] {
	return PayloadRef[TaskInputShape]{
		Id:                 task.ID,
		InsertedAt:         task.InsertedAt,
		TenantId:           task.TenantID,
		ExternalId:         task.ExternalID,
		PayloadReferenceId: task.PayloadReferenceID,
	}
}

// dispatchedStepRunDataBytes builds the step run data the dispatcher sends for a task from its input payload.
func dispatchedStepRunDataBytes(t *testing.T, task *sqlcv1.V1Task, input Payload[TaskInputShape]) []byte {
	t.Helper()

	stepRunData, err := NewV1StepRunDataFromTaskAndInputPayload(task, input.Bytes())
	require.NoError(t, err)

	return stepRunData.Bytes()
}

// isEmptyJSONInput accepts the ways an absent input serializes: nothing, JSON null or an empty object.
func isEmptyJSONInput(input json.RawMessage) bool {
	trimmed := string(bytes.TrimSpace(input))

	return trimmed == "" || trimmed == "null" || trimmed == "{}"
}

func sortTasksById(tasks []*sqlcv1.V1Task) {
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
}

func workflowIdByName(t *testing.T, shared *sharedRepository, name string) uuid.UUID {
	t.Helper()

	workflows, err := shared.queries.ListWorkflowsByNames(context.Background(), shared.pool, sqlcv1.ListWorkflowsByNamesParams{
		Tenantid:      internalTenantId,
		Workflownames: []string{name},
	})
	require.NoError(t, err)
	require.Len(t, workflows, 1)

	return workflows[0].WorkflowId
}
