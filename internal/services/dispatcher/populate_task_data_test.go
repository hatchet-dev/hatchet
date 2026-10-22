package dispatcher

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// populateTaskDataRepo is a v1.Repository whose only reachable method is Tasks; every other call
// panics through the nil embedded interface, which is the point: a failed combined read must not
// reach the payload store or the durable events repository.
type populateTaskDataRepo struct {
	v1.Repository
	tasks *populateTaskDataTasks
}

func (r *populateTaskDataRepo) Tasks() v1.TaskRepository { return r.tasks }

// populateTaskDataTasks fails ListTasksForDispatch and answers ListTasks with fixed rows (or an error),
// recording what each read was asked for.
type populateTaskDataTasks struct {
	v1.TaskRepository

	dispatchReadErr error
	listTasksRows   []*sqlcv1.V1Task
	listTasksErr    error

	listTasksCalls       int
	listTasksTenantId    uuid.UUID
	listTasksIds         []int64
	listTasksCtxErr      error
	listTasksHasDeadline bool
}

func (r *populateTaskDataTasks) ListTasksForDispatch(context.Context, uuid.UUID, []int64) (*v1.TasksForDispatch, error) {
	return nil, r.dispatchReadErr
}

func (r *populateTaskDataTasks) ListTasks(ctx context.Context, tenantId uuid.UUID, ids []int64) ([]*sqlcv1.V1Task, error) {
	r.listTasksCalls++
	r.listTasksTenantId = tenantId
	r.listTasksIds = ids
	r.listTasksCtxErr = ctx.Err()
	_, r.listTasksHasDeadline = ctx.Deadline()

	// a real ListTasks fails on a context that is already done, the way the combined read did
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return r.listTasksRows, r.listTasksErr
}

func newPopulateTaskDataDispatcher(tasks *populateTaskDataTasks) *DispatcherImpl {
	l := zerolog.Nop()

	return &DispatcherImpl{
		l:      &l,
		repov1: &populateTaskDataRepo{tasks: tasks},
	}
}

// TestPopulateTaskDataRequeuesWhenCombinedReadFails: when ListTasksForDispatch fails but the task rows
// can still be read, every requested task reaches the requeue callback and the original error is
// returned, so the task-assigned-bulk handler retries the batch instead of acknowledging it.
func TestPopulateTaskDataRequeuesWhenCombinedReadFails(t *testing.T) {
	tenantId := uuid.New()
	taskIds := []int64{11, 12, 13}
	readErr := errors.New("payload join timed out")

	rows := make([]*sqlcv1.V1Task, 0, len(taskIds))

	for _, id := range taskIds {
		rows = append(rows, &sqlcv1.V1Task{ID: id, TenantID: tenantId})
	}

	tasks := &populateTaskDataTasks{dispatchReadErr: readErr, listTasksRows: rows}
	d := newPopulateTaskDataDispatcher(tasks)

	requeued := make([]int64, 0, len(taskIds))

	got, err := d.populateTaskData(context.Background(), func(task *sqlcv1.V1Task) {
		requeued = append(requeued, task.ID)
	}, tenantId, taskIds)

	require.ErrorIs(t, err, readErr, "the combined read's error is the one returned")
	assert.Nil(t, got)

	assert.Equal(t, taskIds, requeued, "every requested task is handed to requeue")
	assert.Equal(t, 1, tasks.listTasksCalls)
	assert.Equal(t, tenantId, tasks.listTasksTenantId, "the fallback reads the requested tenant")
	assert.Equal(t, taskIds, tasks.listTasksIds, "the fallback reads the requested ids")
}

// TestPopulateTaskDataRequeuesWhenDispatchContextExpired: the combined read fails because the caller's
// dispatch deadline expired while the task rows stayed readable. The fallback must not reuse that context,
// so ListTasks receives a live, bounded context and every requested task still reaches requeue.
func TestPopulateTaskDataRequeuesWhenDispatchContextExpired(t *testing.T) {
	tenantId := uuid.New()
	taskIds := []int64{21, 22}

	rows := make([]*sqlcv1.V1Task, 0, len(taskIds))

	for _, id := range taskIds {
		rows = append(rows, &sqlcv1.V1Task{ID: id, TenantID: tenantId})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	<-ctx.Done()
	require.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)

	tasks := &populateTaskDataTasks{dispatchReadErr: ctx.Err(), listTasksRows: rows}
	d := newPopulateTaskDataDispatcher(tasks)

	requeued := make([]int64, 0, len(taskIds))

	got, err := d.populateTaskData(ctx, func(task *sqlcv1.V1Task) {
		requeued = append(requeued, task.ID)
	}, tenantId, taskIds)

	require.ErrorIs(t, err, context.DeadlineExceeded, "the combined read's error is the one returned")
	assert.Nil(t, got)

	assert.Equal(t, 1, tasks.listTasksCalls)
	assert.NoError(t, tasks.listTasksCtxErr, "the fallback runs under a context that is not done")
	assert.True(t, tasks.listTasksHasDeadline, "the fallback's context is bounded")
	assert.Equal(t, tenantId, tasks.listTasksTenantId, "the fallback reads the requested tenant")
	assert.Equal(t, taskIds, tasks.listTasksIds, "the fallback reads the requested ids")
	assert.Equal(t, taskIds, requeued, "every requested task is handed to requeue")
}

// TestPopulateTaskDataReturnsCombinedReadErrorWhenFallbackFails: when the task rows cannot be read
// either, nothing can be requeued and the combined read's error is still the one returned.
func TestPopulateTaskDataReturnsCombinedReadErrorWhenFallbackFails(t *testing.T) {
	tenantId := uuid.New()
	readErr := errors.New("payload join timed out")
	fallbackErr := errors.New("task read failed too")

	tasks := &populateTaskDataTasks{dispatchReadErr: readErr, listTasksErr: fallbackErr}
	d := newPopulateTaskDataDispatcher(tasks)

	got, err := d.populateTaskData(context.Background(), func(task *sqlcv1.V1Task) {
		t.Fatalf("unexpected requeue of task %d", task.ID)
	}, tenantId, []int64{11, 12})

	require.ErrorIs(t, err, readErr)
	assert.NotErrorIs(t, err, fallbackErr, "the fallback's error is logged, not returned")
	assert.Nil(t, got)
	assert.Equal(t, 1, tasks.listTasksCalls)
}

// TestPopulateTaskDataSkipsReadsForNoIds: an empty id list makes no read at all.
func TestPopulateTaskDataSkipsReadsForNoIds(t *testing.T) {
	tasks := &populateTaskDataTasks{dispatchReadErr: errors.New("must not be called")}
	d := newPopulateTaskDataDispatcher(tasks)

	got, err := d.populateTaskData(context.Background(), func(task *sqlcv1.V1Task) {
		t.Fatalf("unexpected requeue of task %d", task.ID)
	}, uuid.New(), nil)

	require.NoError(t, err)
	assert.Empty(t, got)
	assert.Zero(t, tasks.listTasksCalls)
}
