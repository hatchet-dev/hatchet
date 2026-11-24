package task

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type fakeRepo struct {
	v1.Repository
	tasks *fakeTaskRepo
}

func (r *fakeRepo) Tasks() v1.TaskRepository { return r.tasks }

// fakeTaskRepo returns optedInChildren for any durable task it is asked about and records which
// durable tasks it was asked about.
type fakeTaskRepo struct {
	v1.TaskRepository
	optedInChildren []v1.TaskIdInsertedAtRetryCount
	askedFor        []int64
}

func (r *fakeTaskRepo) ListUnfinishedDurableOrchestratorChildren(_ context.Context, _ uuid.UUID, _ []uuid.UUID) ([]v1.TaskIdInsertedAtRetryCount, error) {
	return nil, nil
}

func (r *fakeTaskRepo) ListUnfinishedCancelOnParentCancelChildren(_ context.Context, _ uuid.UUID, durableTasks []v1.TaskIdInsertedAtRetryCount) ([]v1.TaskIdInsertedAtRetryCount, error) {
	for _, t := range durableTasks {
		r.askedFor = append(r.askedFor, t.Id)
	}

	return r.optedInChildren, nil
}

func durableRow(id int64, currentRetry bool, workerId uuid.UUID) *sqlcv1.ReleaseTasksRow {
	return &sqlcv1.ReleaseTasksRow{
		ID:             id,
		ExternalID:     uuid.New(),
		IsCurrentRetry: currentRetry,
		IsDurable:      true,
		WorkerID:       workerId,
	}
}

func TestCancelChildrenOfCancelledTasks_CascadesForDurableTasks(t *testing.T) {
	tenantId := uuid.New()
	children := []v1.TaskIdInsertedAtRetryCount{{Id: 100}, {Id: 101}}

	tests := []struct {
		name          string
		released      []*sqlcv1.ReleaseTasksRow
		wantAskedFor  []int64
		wantCancelled []int64
	}{
		{
			name:          "durable task with a worker",
			released:      []*sqlcv1.ReleaseTasksRow{durableRow(1, true, uuid.New())},
			wantAskedFor:  []int64{1},
			wantCancelled: []int64{100, 101},
		},
		{
			name:          "evicted durable task without a worker",
			released:      []*sqlcv1.ReleaseTasksRow{durableRow(1, true, uuid.Nil)},
			wantAskedFor:  []int64{1},
			wantCancelled: []int64{100, 101},
		},
		{
			name:     "stale retry of a durable task",
			released: []*sqlcv1.ReleaseTasksRow{durableRow(1, false, uuid.New())},
		},
		{
			name:     "non-durable task",
			released: []*sqlcv1.ReleaseTasksRow{childRow(1), nil},
		},
		{
			name: "durable DAG orchestrator with a worker",
			released: []*sqlcv1.ReleaseTasksRow{{
				ID:                1,
				ExternalID:        uuid.New(),
				IsCurrentRetry:    true,
				IsDurable:         true,
				IsDagOrchestrator: true,
				WorkerID:          uuid.New(),
			}},
		},
		{
			name: "durable DAG orchestrator without a worker",
			released: []*sqlcv1.ReleaseTasksRow{{
				ID:                1,
				ExternalID:        uuid.New(),
				IsCurrentRetry:    true,
				IsDurable:         true,
				IsDagOrchestrator: true,
				WorkerID:          uuid.Nil,
			}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mq := &fakeMQ{}
			tasks := &fakeTaskRepo{optedInChildren: children}
			c := newTestController(mq)
			c.repov1 = &fakeRepo{tasks: tasks}

			require.NoError(t, c.cancelChildrenOfCancelledTasks(context.Background(), tenantId, tc.released))
			require.Equal(t, tc.wantAskedFor, tasks.askedFor)

			if tc.wantCancelled == nil {
				require.Zero(t, mq.sends.Load())
				return
			}

			require.Equal(t, int64(1), mq.sends.Load())
			msg := mq.lastMsg.Load()
			require.Equal(t, msgqueue.MsgIDCancelTasks, msg.ID)

			payloads := msgqueue.JSONConvert[tasktypes.CancelTasksPayload](msg.Payloads)
			require.Len(t, payloads, 1)

			var cancelled []int64
			for _, task := range payloads[0].Tasks {
				cancelled = append(cancelled, task.Id)
			}

			require.Equal(t, tc.wantCancelled, cancelled)
		})
	}
}
