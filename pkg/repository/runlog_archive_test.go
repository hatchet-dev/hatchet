//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
	"github.com/hatchet-dev/hatchet/pkg/repository/runlog"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// These tests build core rows directly with SQL at an old inserted_at and run partition maintenance with the
// real clock and 24h retention, so days today-2 and older are candidates.

type runLogFixture struct {
	t      *testing.T
	ctx    context.Context
	pool   *pgxpool.Pool
	repo   *TaskRepositoryImpl
	store  *runlog.MemoryStore
	log    *runlog.Log
	tenant uuid.UUID
	today  time.Time
}

func newRunLogFixture(t *testing.T) *runLogFixture {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	logger := zerolog.Nop()
	repo := createTaskRepository(pool)
	repo.payloadStore = NewPayloadStoreRepository(fairpool.Ungated(pool), &logger, repo.queries, PayloadStoreRepositoryOpts{
		EnablePayloadConsolidation: true,
	})

	store := runlog.NewMemoryStore()
	repo.runLog = newRunLogReader(runlog.New(store, "test"), fairpool.Ungated(pool))

	f := &runLogFixture{
		t:      t,
		ctx:    context.Background(),
		pool:   pool,
		repo:   repo,
		store:  store,
		log:    runlog.New(store, "test"),
		tenant: uuid.New(),
		today:  runlog.Day(time.Now()),
	}

	for i := 1; i <= 6; i++ {
		f.createPartitions(f.daysAgo(i))
	}

	return f
}

func (f *runLogFixture) daysAgo(n int) time.Time {
	return f.today.AddDate(0, 0, -n)
}

func (f *runLogFixture) createPartitions(day time.Time) {
	creations, err := f.repo.queries.CreatePartitions(f.ctx, f.pool, pgtype.Date{Time: day, Valid: true})
	require.NoError(f.t, err)

	if creations.V1Payload > 0 {
		require.NoError(f.t, createExternalIdUniqueConstraintsOnDailyPartitions(f.ctx, f.pool, "v1_payload", day))
	}

	if creations.V1LookupTable > 0 {
		require.NoError(f.t, createExternalIdUniqueConstraintsOnMonthlyPartitions(f.ctx, f.pool, "v1_lookup_table", day))
	}
}

func (f *runLogFixture) maintain() error {
	return f.repo.UpdateTablePartitionsAt(f.ctx, time.Now().UTC(), 24*time.Hour)
}

type fixtureTask struct {
	insertedAt         time.Time
	workflowRunId      *uuid.UUID
	dag                *sqlcv1.V1Dag
	parent             *sqlcv1.V1Task
	isDurable          bool
	isDagOrchestrator  bool
	payloadReferenceId *uuid.UUID
}

func (f *runLogFixture) insertTask(o fixtureTask) *sqlcv1.V1Task {
	externalId := uuid.New()
	runId := externalId

	if o.workflowRunId != nil {
		runId = *o.workflowRunId
	}

	var dagId pgtype.Int8
	var dagInsertedAt, parentInsertedAt pgtype.Timestamptz
	var parentId pgtype.Int8
	var parentExternalId *uuid.UUID

	if o.dag != nil {
		dagId = pgtype.Int8{Int64: o.dag.ID, Valid: true}
		dagInsertedAt = o.dag.InsertedAt
		runId = o.dag.ExternalID
	}

	if o.parent != nil {
		parentId = pgtype.Int8{Int64: o.parent.ID, Valid: true}
		parentInsertedAt = o.parent.InsertedAt
		parentExternalId = &o.parent.ExternalID
	}

	rows, err := f.pool.Query(f.ctx, `
		INSERT INTO v1_task (
			inserted_at, tenant_id, queue, action_id, step_id, step_readable_id, workflow_id, workflow_version_id,
			workflow_run_id, schedule_timeout, sticky, external_id, display_name, input, step_index, additional_metadata,
			dag_id, dag_inserted_at, parent_task_id, parent_task_inserted_at, parent_task_external_id,
			is_durable, is_dag_orchestrator, payload_reference_id
		) VALUES (
			$1, $2, 'default', 'act', gen_random_uuid(), 'step', gen_random_uuid(), gen_random_uuid(),
			$3, '5m', 'NONE', $4, 'fixture', '{}', 0, '{"k": "v"}',
			$5, $6, $7, $8, $9,
			$10, $11, $12
		) RETURNING *
	`, o.insertedAt, f.tenant, runId, externalId, dagId, dagInsertedAt, parentId, parentInsertedAt, parentExternalId,
		o.isDurable, o.isDagOrchestrator, o.payloadReferenceId)
	require.NoError(f.t, err)

	task, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[sqlcv1.V1Task])
	require.NoError(f.t, err)

	if o.payloadReferenceId == nil {
		f.storePayload(PayloadRef[TaskInputShape]{
			TenantId: f.tenant, Id: task.ID, InsertedAt: task.InsertedAt, ExternalId: task.ExternalID,
		}.StoreOpts(NewPayload[TaskInputShape]([]byte(fmt.Sprintf(`{"input": %d}`, task.ID)))))
	}

	return task
}

func (f *runLogFixture) insertDag(insertedAt time.Time, payloadReferenceId *uuid.UUID) *sqlcv1.V1Dag {
	rows, err := f.pool.Query(f.ctx, `
		INSERT INTO v1_dag (inserted_at, tenant_id, external_id, display_name, workflow_id, workflow_version_id, payload_reference_id)
		VALUES ($1, $2, gen_random_uuid(), 'fixture-dag', gen_random_uuid(), gen_random_uuid(), $3)
		RETURNING *
	`, insertedAt, f.tenant, payloadReferenceId)
	require.NoError(f.t, err)

	dag, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[sqlcv1.V1Dag])
	require.NoError(f.t, err)

	_, err = f.pool.Exec(f.ctx, `INSERT INTO v1_dag_data (dag_id, dag_inserted_at, input) VALUES ($1, $2, '{}')`, dag.ID, dag.InsertedAt)
	require.NoError(f.t, err)

	return dag
}

func (f *runLogFixture) storePayload(opts ...StorePayloadOpts) {
	require.NoError(f.t, f.repo.payloadStore.Store(f.ctx, f.pool, opts...))
}

// complete makes a task terminal: no queue item, a COMPLETED event with an output payload.
func (f *runLogFixture) complete(task *sqlcv1.V1Task) {
	_, err := f.pool.Exec(f.ctx, `DELETE FROM v1_queue_item WHERE task_id = $1 AND task_inserted_at = $2`, task.ID, task.InsertedAt)
	require.NoError(f.t, err)

	var event sqlcv1.V1TaskEvent

	rows, err := f.pool.Query(f.ctx, `
		INSERT INTO v1_task_event (inserted_at, tenant_id, task_id, task_inserted_at, retry_count, event_type)
		VALUES ($1, $2, $3, $4, $5, 'COMPLETED')
		RETURNING *
	`, task.InsertedAt.Time.Add(time.Second), f.tenant, task.ID, task.InsertedAt, task.RetryCount)
	require.NoError(f.t, err)

	ev, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[sqlcv1.V1TaskEvent])
	require.NoError(f.t, err)
	event = *ev

	f.storePayload(taskEventDataStoreOpts(f.tenant, &event, []byte(fmt.Sprintf(`{"output": %d}`, task.ID))))
}

func (f *runLogFixture) exec(sql string, args ...any) {
	_, err := f.pool.Exec(f.ctx, sql, args...)
	require.NoError(f.t, err)
}

func (f *runLogFixture) partitionExists(name string) bool {
	var exists bool
	require.NoError(f.t, f.pool.QueryRow(f.ctx, `SELECT EXISTS (SELECT 1 FROM pg_class WHERE relname = $1)`, name).Scan(&exists))
	return exists
}

func (f *runLogFixture) dayState(day time.Time) string {
	var state string
	err := f.pool.QueryRow(f.ctx, `SELECT state FROM v1_run_log_day WHERE day = $1`, pgtype.Date{Time: day, Valid: true}).Scan(&state)

	if errors.Is(err, pgx.ErrNoRows) {
		return ""
	}

	require.NoError(f.t, err)
	return state
}

func partitionName(table string, day time.Time) string {
	return table + "_" + day.Format("20060102")
}

// coreSnapshot is what core holds for a task, in the shape a bundle holds it.
func (f *runLogFixture) coreSnapshot(task *sqlcv1.V1Task) (taskJSON, eventsJSON string, payloads map[sqlcv1.V1PayloadType]string) {
	rows, err := f.pool.Query(f.ctx, `SELECT * FROM v1_task WHERE id = $1 AND inserted_at = $2`, task.ID, task.InsertedAt)
	require.NoError(f.t, err)
	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[sqlcv1.V1Task])
	require.NoError(f.t, err)

	rows, err = f.pool.Query(f.ctx, `SELECT * FROM v1_task_event WHERE task_id = $1 AND task_inserted_at = $2 ORDER BY id`, task.ID, task.InsertedAt)
	require.NoError(f.t, err)
	events, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[sqlcv1.V1TaskEvent])
	require.NoError(f.t, err)

	opts := []RetrievePayloadOpts{{
		TenantId: f.tenant, Id: row.ID, InsertedAt: row.InsertedAt, Type: sqlcv1.V1PayloadTypeTASKINPUT,
		ExternalId: row.ExternalID, PayloadReferenceId: row.PayloadReferenceID,
	}}

	for _, e := range events {
		opts = append(opts, taskEventDataStoreOpts(f.tenant, e, nil).key)
	}

	found, err := f.repo.payloadStore.Retrieve(f.ctx, f.pool, opts...)
	require.NoError(f.t, err)

	payloads = map[sqlcv1.V1PayloadType]string{}

	for _, o := range opts {
		if v, ok := found[o]; ok {
			payloads[o.Type] = string(v)
		}
	}

	return mustJSON(f.t, row), mustJSON(f.t, events), payloads
}

func (f *runLogFixture) requireBundleMatchesCore(day time.Time, task *sqlcv1.V1Task, taskJSON, eventsJSON string, payloads map[sqlcv1.V1PayloadType]string) *runlog.Bundle {
	b, err := f.log.GetTask(f.ctx, f.tenant, day, task.ExternalID)
	require.NoError(f.t, err)

	assert.JSONEq(f.t, taskJSON, mustJSON(f.t, b.Task))
	assert.JSONEq(f.t, eventsJSON, mustJSON(f.t, b.Events))

	got := map[sqlcv1.V1PayloadType]string{}

	for _, p := range b.Payloads {
		if p.Type == sqlcv1.V1PayloadTypeTASKINPUT || p.Type == sqlcv1.V1PayloadTypeTASKEVENTDATA {
			got[p.Type] = string(p.Value)
		}
	}

	require.Len(f.t, got, len(payloads))

	for typ, want := range payloads {
		assert.JSONEq(f.t, want, got[typ], "payload %s of task %d", typ, task.ID)
	}

	return b
}

func mustJSON(t *testing.T, v any) string {
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	return string(raw)
}

// uuidV7At is a version 7 uuid whose timestamp is at.
func uuidV7At(at time.Time) uuid.UUID {
	id := uuid.Must(uuid.NewV7())
	ms := uint64(at.UnixMilli()) // nolint:gosec

	for i := range 6 {
		id[i] = byte(ms >> (40 - 8*i))
	}

	return id
}

func TestRunLog_FinishedDayIsArchivedAndDropped(t *testing.T) {
	f := newRunLogFixture(t)
	day := f.daysAgo(3)
	next := f.daysAgo(2)

	// a standalone task
	standalone := f.insertTask(fixtureTask{insertedAt: day.Add(9 * time.Hour)})
	f.complete(standalone)

	// two runs triggered by one event share its payload row through a reference id
	ref := uuidV7At(day.Add(10 * time.Hour))
	f.storePayload(PayloadRef[UserEventInputShape]{
		TenantId: f.tenant, Id: 1, InsertedAt: pgtype.Timestamptz{Time: day.Add(10 * time.Hour), Valid: true},
		ExternalId: ref, PayloadReferenceId: &ref,
	}.StoreOpts(NewPayload[UserEventInputShape]([]byte(`{"event": "shared"}`))))

	eventRunA := f.insertTask(fixtureTask{insertedAt: day.Add(10 * time.Hour), payloadReferenceId: &ref})
	eventRunB := f.insertTask(fixtureTask{insertedAt: day.Add(10 * time.Hour), payloadReferenceId: &ref})
	f.complete(eventRunA)
	f.complete(eventRunB)

	// a legacy DAG with two finished steps
	dagRef := uuidV7At(day.Add(11 * time.Hour))
	dag := f.insertDag(day.Add(11*time.Hour), &dagRef)
	f.storePayload(PayloadRef[DAGInputShape]{
		TenantId: f.tenant, Id: dag.ID, InsertedAt: dag.InsertedAt, ExternalId: dag.ExternalID, PayloadReferenceId: &dagRef,
	}.StoreOpts(NewPayload[DAGInputShape]([]byte(`{"dag": "input"}`))))

	step1 := f.insertTask(fixtureTask{insertedAt: day.Add(11 * time.Hour), dag: dag, payloadReferenceId: &dagRef})
	step2 := f.insertTask(fixtureTask{insertedAt: day.Add(11*time.Hour + time.Minute), dag: dag, payloadReferenceId: &dagRef})
	f.complete(step1)
	f.complete(step2)

	tasks := []*sqlcv1.V1Task{standalone, eventRunA, eventRunB, step1, step2}

	type snapshot struct {
		task, events string
		payloads     map[sqlcv1.V1PayloadType]string
	}

	snapshots := make([]snapshot, len(tasks))

	for i, task := range tasks {
		snapshots[i].task, snapshots[i].events, snapshots[i].payloads = f.coreSnapshot(task)
	}

	require.NoError(t, f.maintain())

	// the day and the next (empty) day are archived and dropped
	assert.Equal(t, runLogStateDropped, f.dayState(day))
	assert.Equal(t, runLogStateDropped, f.dayState(next))
	assert.Empty(t, f.dayState(f.daysAgo(1)), "within retention")

	for _, table := range []string{"v1_task", "v1_task_event", "v1_dag", "v1_dag_to_task", "v1_dag_data", "v1_durable_event_log_entry"} {
		assert.False(t, f.partitionExists(partitionName(table, day)), table)
		assert.False(t, f.partitionExists(partitionName(table, next)), table)
		assert.True(t, f.partitionExists(partitionName(table, f.daysAgo(1))), table)
	}

	// payloads and events lag one day: the day's go because the next day is droppable, the next day's stay
	// because the day after it is within retention
	assert.False(t, f.partitionExists(partitionName("v1_payload", day)))
	assert.False(t, f.partitionExists(partitionName("v1_event", day)))
	assert.True(t, f.partitionExists(partitionName("v1_payload", next)))
	assert.True(t, f.partitionExists(partitionName("v1_event", next)))

	// only the tenant's day was written; the empty next day has no objects
	assert.Len(t, f.store.Keys(f.log.DayPrefix(f.tenant, day)), 3)
	assert.Empty(t, f.store.Keys(f.log.DayPrefix(f.tenant, next)))

	for i, task := range tasks {
		f.requireBundleMatchesCore(day, task, snapshots[i].task, snapshots[i].events, snapshots[i].payloads)
	}

	bundles, err := f.log.GetRun(f.ctx, f.tenant, day, dag.ExternalID)
	require.NoError(t, err)
	require.Len(t, bundles, 2)
	require.NotNil(t, bundles[0].Dag, "the DAG's first task carries the DAG")
	assert.Nil(t, bundles[1].Dag)
	assert.Equal(t, dag.ExternalID, bundles[0].Dag.Dag.ExternalID)
	assert.Len(t, bundles[0].Dag.Tasks, 2)
	require.NotNil(t, bundles[0].Dag.Data)
	dagInput := bundles[0].PayloadFor(sqlcv1.V1PayloadTypeDAGINPUT, dag.ID, dag.InsertedAt)
	require.NotNil(t, dagInput)
	assert.JSONEq(t, `{"dag": "input"}`, string(dagInput.Value))

	shared, err := f.log.GetPayloadByReference(f.ctx, f.tenant, ref)
	require.NoError(t, err)
	assert.JSONEq(t, `{"event": "shared"}`, string(shared.Value))

	// the lookup row outlives the task, so the day can still be resolved from an external id
	resolved, err := runlog.ResolveDay(f.ctx, f.pool, f.tenant, standalone.ExternalID)
	require.NoError(t, err)
	assert.Equal(t, day, resolved)

	// a second pass finds nothing left to do
	puts := f.store.Puts()
	require.NoError(t, f.maintain())
	assert.Equal(t, puts, f.store.Puts())
}

func TestRunLog_DayWithLiveWorkIsPinned(t *testing.T) {
	f := newRunLogFixture(t)
	queuedDay, matchDay, retryDay := f.daysAgo(5), f.daysAgo(4), f.daysAgo(3)

	// a queued task (its queue item comes from the insert trigger)
	queued := f.insertTask(fixtureTask{insertedAt: queuedDay.Add(time.Hour)})
	f.complete(f.insertTask(fixtureTask{insertedAt: queuedDay.Add(2 * time.Hour)}))

	// a durable task asleep: no runtime row, only the wait
	sleeping := f.insertTask(fixtureTask{insertedAt: matchDay.Add(time.Hour), isDurable: true})
	f.exec(`DELETE FROM v1_queue_item WHERE task_id = $1`, sleeping.ID)
	f.exec(`INSERT INTO v1_match (tenant_id, kind, signal_task_id, signal_task_inserted_at, signal_task_external_id) VALUES ($1, 'SIGNAL', $2, $3, $4)`,
		f.tenant, sleeping.ID, sleeping.InsertedAt, sleeping.ExternalID)

	// a task whose retry is scheduled after the drop attempt
	retrying := f.insertTask(fixtureTask{insertedAt: retryDay.Add(time.Hour)})
	f.exec(`DELETE FROM v1_queue_item WHERE task_id = $1`, retrying.ID)
	f.exec(`INSERT INTO v1_task_event (inserted_at, tenant_id, task_id, task_inserted_at, retry_count, event_type) VALUES ($1, $2, $3, $4, 0, 'FAILED')`,
		retrying.InsertedAt.Time.Add(time.Second), f.tenant, retrying.ID, retrying.InsertedAt)
	f.exec(`UPDATE v1_task SET retry_count = 1 WHERE id = $1 AND inserted_at = $2`, retrying.ID, retrying.InsertedAt)
	retrying.RetryCount = 1
	f.exec(`DELETE FROM v1_queue_item WHERE task_id = $1`, retrying.ID)
	f.exec(`INSERT INTO v1_retry_queue_item (task_id, task_inserted_at, task_retry_count, retry_after, tenant_id) VALUES ($1, $2, 1, NOW() + INTERVAL '1 hour', $3)`,
		retrying.ID, retrying.InsertedAt, f.tenant)

	require.NoError(t, f.maintain())

	for _, day := range []time.Time{queuedDay, matchDay, retryDay} {
		assert.Empty(t, f.dayState(day), day)
		assert.True(t, f.partitionExists(partitionName("v1_task", day)), day)
		assert.True(t, f.partitionExists(partitionName("v1_payload", day)), day)
		assert.Empty(t, f.store.Keys(f.log.DayPrefix(f.tenant, day)), "a pinned day is not archived")
	}

	pins, err := f.repo.runLogPins(f.ctx)
	require.NoError(t, err)
	assert.Positive(t, pins[queuedDay]["queue_item"])
	assert.Positive(t, pins[matchDay]["match"])
	assert.Positive(t, pins[retryDay]["retry_queue_item"])

	// the empty day after them still goes, but the payloads of the pinned day before it stay
	assert.Equal(t, runLogStateDropped, f.dayState(f.daysAgo(2)))
	assert.True(t, f.partitionExists(partitionName("v1_payload", retryDay)))

	// the match sweep keeps waits whose day is pinned, however old
	_, err = f.repo.Cleanup(f.ctx)
	require.NoError(t, err)

	var matches int
	require.NoError(t, f.pool.QueryRow(f.ctx, `SELECT count(*) FROM v1_match WHERE signal_task_id = $1`, sleeping.ID).Scan(&matches))
	assert.Equal(t, 1, matches)

	// once the work finishes the days go
	f.complete(queued)
	f.exec(`DELETE FROM v1_match WHERE signal_task_id = $1`, sleeping.ID)
	f.complete(sleeping)
	f.exec(`DELETE FROM v1_retry_queue_item WHERE task_id = $1`, retrying.ID)
	f.complete(retrying)

	require.NoError(t, f.maintain())

	for _, day := range []time.Time{queuedDay, matchDay, retryDay} {
		assert.Equal(t, runLogStateDropped, f.dayState(day), day)
		assert.False(t, f.partitionExists(partitionName("v1_task", day)), day)
	}

	b, err := f.log.GetTask(f.ctx, f.tenant, retryDay, retrying.ExternalID)
	require.NoError(t, err)
	assert.Equal(t, int32(1), b.Task.RetryCount)
	require.Len(t, b.Events, 2)
	assert.Equal(t, sqlcv1.V1TaskEventTypeFAILED, b.Events[0].EventType)
	assert.Equal(t, sqlcv1.V1TaskEventTypeCOMPLETED, b.Events[1].EventType)
}

func TestRunLog_RunTreeWithLiveStepIsPinned(t *testing.T) {
	f := newRunLogFixture(t)
	legacyDay, operatorDay, childDay := f.daysAgo(5), f.daysAgo(4), f.daysAgo(3)

	// legacy DAG: the first step finished, the second is not created yet (a trigger match on the DAG)
	dag := f.insertDag(legacyDay.Add(time.Hour), nil)
	f.complete(f.insertTask(fixtureTask{insertedAt: legacyDay.Add(time.Hour), dag: dag}))
	f.exec(`INSERT INTO v1_match (tenant_id, kind, trigger_dag_id, trigger_dag_inserted_at, trigger_step_id) VALUES ($1, 'TRIGGER', $2, $3, gen_random_uuid())`,
		f.tenant, dag.ID, dag.InsertedAt)

	// operator DAG: the orchestrator runs today; its first step finished days ago, the next is not started
	orchestrator := f.insertTask(fixtureTask{insertedAt: f.today.Add(time.Minute), isDurable: true, isDagOrchestrator: true})
	f.exec(`DELETE FROM v1_queue_item WHERE task_id = $1`, orchestrator.ID)
	f.exec(`INSERT INTO v1_task_runtime (task_id, task_inserted_at, retry_count, tenant_id, timeout_at) VALUES ($1, $2, 0, $3, NOW() + INTERVAL '1 day')`,
		orchestrator.ID, orchestrator.InsertedAt, f.tenant)
	step := f.insertTask(fixtureTask{insertedAt: operatorDay.Add(time.Hour), parent: orchestrator, workflowRunId: &orchestrator.ExternalID})
	f.complete(step)
	f.exec(`
		INSERT INTO v1_durable_event_log_entry (tenant_id, external_id, inserted_at, durable_task_id, durable_task_inserted_at, kind, node_id, branch_id, idempotency_key, child_task_external_id)
		VALUES ($1, gen_random_uuid(), $2, $3, $4, 'RUN', 1, 1, '\x00', $5)
	`, f.tenant, orchestrator.InsertedAt, orchestrator.ID, orchestrator.InsertedAt, step.ExternalID)

	// a child workflow still running today whose parent finished days ago
	parent := f.insertTask(fixtureTask{insertedAt: childDay.Add(time.Hour)})
	f.complete(parent)
	f.insertTask(fixtureTask{insertedAt: f.today.Add(time.Minute), parent: parent})

	require.NoError(t, f.maintain())

	pins, err := f.repo.runLogPins(f.ctx)
	require.NoError(t, err)
	assert.Positive(t, pins[legacyDay]["match"])
	assert.Positive(t, pins[legacyDay]["dag_task"])
	assert.Positive(t, pins[operatorDay]["orchestrator_step"])
	assert.Positive(t, pins[childDay]["run_tree"])

	for _, day := range []time.Time{legacyDay, operatorDay, childDay} {
		assert.Empty(t, f.dayState(day), day)
		assert.True(t, f.partitionExists(partitionName("v1_task", day)), day)
		assert.True(t, f.partitionExists(partitionName("v1_task_event", day)), day)
	}
}

// failingStore fails every Put after the first okPuts.
type failingStore struct {
	*runlog.MemoryStore
	okPuts int
}

func (s *failingStore) Put(ctx context.Context, key string, body []byte) error {
	if s.okPuts <= 0 {
		return fmt.Errorf("injected failure writing %s", key)
	}

	s.okPuts--

	return s.MemoryStore.Put(ctx, key, body)
}

func TestRunLog_CrashesThenRerun(t *testing.T) {
	f := newRunLogFixture(t)
	day := f.daysAgo(3)

	var tasks []*sqlcv1.V1Task

	for i := range 3 {
		task := f.insertTask(fixtureTask{insertedAt: day.Add(time.Duration(i+1) * time.Hour)})
		f.complete(task)
		tasks = append(tasks, task)
	}

	snapshotTask, snapshotEvents, snapshotPayloads := f.coreSnapshot(tasks[0])

	// crash while uploading: the data file is written, the index is not
	f.repo.runLog = newRunLogReader(runlog.New(&failingStore{MemoryStore: f.store, okPuts: 1}, "test"), fairpool.Ungated(f.pool))
	require.Error(t, f.maintain())
	assert.Empty(t, f.dayState(day))
	_, err := f.log.GetTask(f.ctx, f.tenant, day, tasks[0].ExternalID)
	require.ErrorIs(t, err, runlog.ErrNotFound, "a tenant-day without its marker is absent")

	// crash between upload and drop
	f.repo.runLog = newRunLogReader(runlog.New(f.store, "test"), fairpool.Ungated(f.pool))
	f.repo.runLogBeforeDrop = func(time.Time) error { return errors.New("injected crash before drop") }
	require.Error(t, f.maintain())
	assert.Equal(t, runLogStateArchived, f.dayState(day))
	assert.True(t, f.partitionExists(partitionName("v1_task", day)))

	// the rerun drops without writing the archive again
	puts := f.store.Puts()
	f.repo.runLogBeforeDrop = nil
	require.NoError(t, f.maintain())
	assert.Equal(t, puts, f.store.Puts())
	assert.Equal(t, runLogStateDropped, f.dayState(day))
	assert.False(t, f.partitionExists(partitionName("v1_task", day)))

	prefix := f.log.DayPrefix(f.tenant, day)
	assert.Equal(t, []string{prefix + "_COMPLETE", prefix + "data-00000.jsonl.gz", prefix + "index.json.gz"}, f.store.Keys(prefix))

	idx, err := f.log.Index(f.ctx, f.tenant, day)
	require.NoError(t, err)
	assert.Len(t, idx.Tasks, len(tasks))

	f.requireBundleMatchesCore(day, tasks[0], snapshotTask, snapshotEvents, snapshotPayloads)

	for _, task := range tasks {
		_, err := f.log.GetTask(f.ctx, f.tenant, day, task.ExternalID)
		require.NoError(t, err)
	}
}

func TestRunLog_InterruptedDropIsFinishedWithoutRearchiving(t *testing.T) {
	f := newRunLogFixture(t)
	day := f.daysAgo(3)

	task := f.insertTask(fixtureTask{insertedAt: day.Add(time.Hour)})
	f.complete(task)

	f.repo.runLogBeforeDrop = func(time.Time) error { return errors.New("injected crash before drop") }
	require.Error(t, f.maintain())

	// the crash happened after the first partition of the day was dropped
	require.NoError(t, f.repo.setRunLogDayState(f.ctx, day, runLogStateDropping))
	require.NoError(t, f.repo.detachAndDropPartition(f.ctx, "v1_task_event", partitionName("v1_task_event", day)))

	puts := f.store.Puts()
	f.repo.runLogBeforeDrop = nil
	require.NoError(t, f.maintain())

	assert.Equal(t, puts, f.store.Puts(), "a half-dropped day is never archived again")
	assert.Equal(t, runLogStateDropped, f.dayState(day))
	assert.False(t, f.partitionExists(partitionName("v1_task", day)))

	b, err := f.log.GetTask(f.ctx, f.tenant, day, task.ExternalID)
	require.NoError(t, err)
	require.Len(t, b.Events, 1)
}

func TestRunLog_ArchiveIsRedoneWhenTheDayChangesBeforeTheDrop(t *testing.T) {
	f := newRunLogFixture(t)
	day := f.daysAgo(3)

	task := f.insertTask(fixtureTask{insertedAt: day.Add(time.Hour)})
	f.complete(task)

	f.repo.runLogBeforeDrop = func(time.Time) error { return errors.New("injected crash before drop") }
	require.Error(t, f.maintain())
	require.Equal(t, runLogStateArchived, f.dayState(day))

	// a replay ran to completion between the archive and the drop: the archive holds the old attempt
	f.exec(`UPDATE v1_task SET retry_count = 1 WHERE id = $1 AND inserted_at = $2`, task.ID, task.InsertedAt)
	task.RetryCount = 1
	f.complete(task)

	f.repo.runLogBeforeDrop = nil
	require.NoError(t, f.maintain())
	assert.Empty(t, f.dayState(day), "the stale archive is discarded, the day kept")
	assert.True(t, f.partitionExists(partitionName("v1_task", day)))

	require.NoError(t, f.maintain())
	assert.Equal(t, runLogStateDropped, f.dayState(day))

	b, err := f.log.GetTask(f.ctx, f.tenant, day, task.ExternalID)
	require.NoError(t, err)
	assert.Equal(t, int32(1), b.Task.RetryCount)
	assert.Len(t, b.Events, 2)
}

func TestRunLog_LaterDagStepReadsSharedInputFromTheLog(t *testing.T) {
	f := newRunLogFixture(t)
	dagDay, laterDay := f.daysAgo(5), f.daysAgo(3)

	// a legacy DAG whose second step was created two days after the DAG; both steps read the DAG's shared
	// input row, which is stored on the DAG's day
	ref := uuidV7At(dagDay.Add(time.Hour))
	dag := f.insertDag(dagDay.Add(time.Hour), &ref)
	f.storePayload(PayloadRef[DAGInputShape]{
		TenantId: f.tenant, Id: dag.ID, InsertedAt: dag.InsertedAt, ExternalId: dag.ExternalID, PayloadReferenceId: &ref,
	}.StoreOpts(NewPayload[DAGInputShape]([]byte(`{"dag": "input"}`))))

	first := f.insertTask(fixtureTask{insertedAt: dagDay.Add(time.Hour), dag: dag, payloadReferenceId: &ref})
	f.complete(first)
	later := f.insertTask(fixtureTask{insertedAt: laterDay.Add(time.Hour), dag: dag, payloadReferenceId: &ref})

	// the later step is still queued: the DAG's day stays and so does the later day
	require.NoError(t, f.maintain())
	assert.Empty(t, f.dayState(dagDay))
	assert.Empty(t, f.dayState(laterDay))

	// it finishes; the DAG's day goes first (with its payloads, since the next day is empty) while the later
	// day is held back by a stale queue item pointing at it
	f.complete(later)
	f.exec(`INSERT INTO v1_retry_queue_item (task_id, task_inserted_at, task_retry_count, retry_after, tenant_id) VALUES (-1, $1, 0, NOW(), $2)`,
		laterDay.Add(time.Hour), f.tenant)

	require.NoError(t, f.maintain())
	require.Equal(t, runLogStateDropped, f.dayState(dagDay))
	require.False(t, f.partitionExists(partitionName("v1_payload", dagDay)))
	require.Empty(t, f.dayState(laterDay))

	f.exec(`DELETE FROM v1_retry_queue_item WHERE task_id = -1`)
	require.NoError(t, f.maintain())
	require.Equal(t, runLogStateDropped, f.dayState(laterDay))

	b, err := f.log.GetTask(f.ctx, f.tenant, laterDay, later.ExternalID)
	require.NoError(t, err)
	input := b.PayloadFor(sqlcv1.V1PayloadTypeTASKINPUT, later.ID, later.InsertedAt)
	require.NotNil(t, input, "the later step's input is read from the run log of the DAG's day")
	assert.JSONEq(t, `{"dag": "input"}`, string(input.Value))

	// the run spans both days
	bundles, err := f.log.GetRun(f.ctx, f.tenant, dagDay, dag.ExternalID)
	require.NoError(t, err)
	assert.Len(t, bundles, 2)
}
