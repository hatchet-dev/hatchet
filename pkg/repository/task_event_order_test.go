//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Worker task events reach the task controller over two paths, the task queue and direct writes from
// the gRPC process, so one task's events can be applied out of order or more than once. These tests pin
// the guards that make that safe: every change is keyed on (task, retry count), and an event for a
// retry that is no longer current, or for a task already released, changes no task state.

type eventOrderFixture struct {
	t        *testing.T
	ctx      context.Context
	pool     *pgxpool.Pool
	repo     *TaskRepositoryImpl
	workflow string
}

func newEventOrderFixture(t *testing.T) *eventOrderFixture {
	t.Helper()

	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(func() { cleanup() })

	f := &eventOrderFixture{t: t, ctx: context.Background(), pool: pool, repo: newBatchTestRepository(pool), workflow: "event-order"}

	// More retries than any test uses, so only the retry-count guards stop a duplicate failure from
	// retrying again, not the retry limit.
	retries := 5
	opts := minimalWorkflowOpts(f.workflow, "v1", nil)
	opts.Tasks[0].Retries = &retries

	_, err := (&workflowRepository{sharedRepository: f.repo.sharedRepository}).PutWorkflowVersion(f.ctx, internalTenantId, opts)
	require.NoError(t, err)

	return f
}

type orderedTask struct {
	id         int64
	insertedAt pgtype.Timestamptz
}

func (f *eventOrderFixture) trigger() orderedTask {
	f.t.Helper()

	tasks, _, _, _, err := newTriggerRepository(f.repo.sharedRepository).TriggerFromWorkflowNames(f.ctx, internalTenantId, []*WorkflowNameTriggerOpts{
		{TriggerTaskData: &TriggerTaskData{WorkflowName: f.workflow, Data: []byte(`{}`)}, ExternalId: uuid.New()},
	})
	require.NoError(f.t, err)
	require.Len(f.t, tasks, 1)

	return orderedTask{id: tasks[0].ID, insertedAt: tasks[0].InsertedAt}
}

// assign writes the part of an assignment the event guards read: the retry's runtime row. The
// scheduler also deletes the queue item and takes a worker slot, which these tests don't cover.
func (f *eventOrderFixture) assign(task orderedTask, retry int32) {
	f.t.Helper()

	_, err := f.pool.Exec(f.ctx, `
		INSERT INTO v1_task_runtime (task_id, task_inserted_at, retry_count, tenant_id, timeout_at, assigned_at)
		VALUES ($1, $2, $3, $4, NOW() + INTERVAL '1 hour', NOW())`,
		task.id, task.insertedAt, retry, internalTenantId)
	require.NoError(f.t, err)
}

func (f *eventOrderFixture) key(task orderedTask, retry int32) *TaskIdInsertedAtRetryCount {
	return &TaskIdInsertedAtRetryCount{Id: task.id, InsertedAt: task.insertedAt, RetryCount: retry}
}

type taskEvent struct {
	name  string
	apply func(f *eventOrderFixture, task orderedTask, retry int32) error
}

var (
	ackEvent = taskEvent{"ack", func(f *eventOrderFixture, task orderedTask, retry int32) error {
		return f.repo.MarkTaskRuntimesAcked(f.ctx, internalTenantId, []TaskIdInsertedAtRetryCount{*f.key(task, retry)})
	}}
	completeEvent = taskEvent{"complete", func(f *eventOrderFixture, task orderedTask, retry int32) error {
		_, err := f.repo.CompleteTasks(f.ctx, internalTenantId, []CompleteTaskOpts{{TaskIdInsertedAtRetryCount: f.key(task, retry), Output: []byte(`{}`)}})
		return err
	}}
	failEvent = taskEvent{"fail", func(f *eventOrderFixture, task orderedTask, retry int32) error {
		_, err := f.repo.FailTasks(f.ctx, internalTenantId, []FailTaskOpts{{TaskIdInsertedAtRetryCount: f.key(task, retry), IsAppError: true, ErrorMessage: "boom"}})
		return err
	}}
	cancelEvent = taskEvent{"cancel", func(f *eventOrderFixture, task orderedTask, retry int32) error {
		_, err := f.repo.CancelTasks(f.ctx, internalTenantId, []TaskIdInsertedAtRetryCount{*f.key(task, retry)})
		return err
	}}
)

// events counts the task events written for one retry, by type.
func (f *eventOrderFixture) events(task orderedTask, retry int32) map[string]int {
	f.t.Helper()

	rows, err := f.pool.Query(f.ctx, `SELECT event_type::text, COUNT(*) FROM v1_task_event WHERE task_id = $1 AND task_inserted_at = $2 AND retry_count = $3 GROUP BY 1`, task.id, task.insertedAt, retry)
	require.NoError(f.t, err)
	defer rows.Close()

	res := map[string]int{}

	for rows.Next() {
		var eventType string
		var n int
		require.NoError(f.t, rows.Scan(&eventType, &n))
		res[eventType] = n
	}

	require.NoError(f.t, rows.Err())

	return res
}

func (f *eventOrderFixture) retryCount(task orderedTask) int32 {
	f.t.Helper()

	var retry int32
	require.NoError(f.t, f.pool.QueryRow(f.ctx, `SELECT retry_count FROM v1_task WHERE id = $1 AND inserted_at = $2`, task.id, task.insertedAt).Scan(&retry))

	return retry
}

// runtimes returns whether each retry still holding a runtime row has been acked.
func (f *eventOrderFixture) runtimes(task orderedTask) map[int32]bool {
	f.t.Helper()

	rows, err := f.pool.Query(f.ctx, `SELECT retry_count, acked_at IS NOT NULL FROM v1_task_runtime WHERE task_id = $1 AND task_inserted_at = $2`, task.id, task.insertedAt)
	require.NoError(f.t, err)
	defer rows.Close()

	res := map[int32]bool{}

	for rows.Next() {
		var retry int32
		var acked bool
		require.NoError(f.t, rows.Scan(&retry, &acked))
		res[retry] = acked
	}

	require.NoError(f.t, rows.Err())

	return res
}

func permutations(events []taskEvent) [][]taskEvent {
	if len(events) <= 1 {
		return [][]taskEvent{events}
	}

	var res [][]taskEvent

	for i, first := range events {
		rest := append(append([]taskEvent{}, events[:i]...), events[i+1:]...)

		for _, p := range permutations(rest) {
			res = append(res, append([]taskEvent{first}, p...))
		}
	}

	return res
}

// Every order of a retry's ack and terminal events, each delivered twice, ends in the state the first
// terminal event decides: a failure first uses exactly one retry, a completion or cancel first uses
// none, and the retry's runtime is gone either way.
func TestTaskEventsForOneRetryInAnyOrderAndDuplicated(t *testing.T) {
	f := newEventOrderFixture(t)

	for _, order := range permutations([]taskEvent{ackEvent, completeEvent, failEvent, cancelEvent}) {
		names := make([]string, 0, len(order))
		for _, e := range order {
			names = append(names, e.name)
		}

		t.Run(strings.Join(names, "-"), func(t *testing.T) {
			f.t = t
			task := f.trigger()
			f.assign(task, 0)

			for _, e := range append(order, order...) {
				require.NoError(t, e.apply(f, task, 0), e.name)
			}

			wantRetry := int32(0)

			for _, e := range order {
				if e.name == "fail" {
					wantRetry = 1
					break
				}

				if e.name == "complete" || e.name == "cancel" {
					break
				}
			}

			assert.Equal(t, wantRetry, f.retryCount(task), "retry count")
			assert.NotContains(t, f.runtimes(task), int32(0), "retry 0 still holds a runtime")
		})
	}
}

// Events for retry 0 arriving once retry 1 runs, which the queue path delivers when retry 0's failure
// was applied directly while its other events still sat in the queue.
func TestStaleRetryEventsLeaveTheNextRetryAlone(t *testing.T) {
	f := newEventOrderFixture(t)

	for _, stale := range []taskEvent{ackEvent, completeEvent, failEvent, cancelEvent} {
		t.Run(stale.name, func(t *testing.T) {
			f.t = t
			task := f.trigger()
			f.assign(task, 0)
			require.NoError(t, failEvent.apply(f, task, 0))
			require.Equal(t, int32(1), f.retryCount(task))
			f.assign(task, 1)
			before := f.events(task, 0)

			require.NoError(t, stale.apply(f, task, 0))

			assert.Equal(t, int32(1), f.retryCount(task), "retry count")
			assert.Equal(t, map[int32]bool{1: false}, f.runtimes(task), fmt.Sprintf("a stale %s changed retry 1's runtime", stale.name))
			assert.Equal(t, before, f.events(task, 0), fmt.Sprintf("a stale %s wrote a task event", stale.name))
			assert.Empty(t, f.events(task, 1), fmt.Sprintf("a stale %s wrote an event for retry 1", stale.name))
		})
	}
}

// A terminal event for the current retry is not deduplicated: once the task is released, a second
// completion, or a cancel or failure after it, writes another task event (and the handler emits
// another internal event). The task's state doesn't change. This pins that behaviour so a change
// to it is deliberate.
func TestSecondTerminalEventForTheCurrentRetryWritesAnotherEvent(t *testing.T) {
	f := newEventOrderFixture(t)

	for _, second := range []taskEvent{completeEvent, cancelEvent, failEvent} {
		t.Run("complete-then-"+second.name, func(t *testing.T) {
			f.t = t
			task := f.trigger()
			f.assign(task, 0)

			require.NoError(t, completeEvent.apply(f, task, 0))
			require.Equal(t, map[string]int{"COMPLETED": 1}, f.events(task, 0))

			require.NoError(t, second.apply(f, task, 0))

			events := f.events(task, 0)
			total := 0

			for _, n := range events {
				total += n
			}

			assert.Equal(t, 2, total, "events after a second terminal event: %v", events)
			assert.Equal(t, int32(0), f.retryCount(task), "retry count")
			assert.Empty(t, f.runtimes(task))
		})
	}
}
