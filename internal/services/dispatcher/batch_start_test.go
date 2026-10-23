package dispatcher

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type recordingMQ struct {
	mu   sync.Mutex
	sent []*msgqueue.Message
}

func (m *recordingMQ) SendMessage(_ context.Context, _ msgqueue.Queue, msg *msgqueue.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, msg)
	return nil
}

func (m *recordingMQ) Clone() (func() error, msgqueue.MessageQueue, error) { return nil, nil, nil }
func (m *recordingMQ) SetQOS(int)                                         {}
func (m *recordingMQ) Subscribe(msgqueue.Queue, msgqueue.MsgHandler, msgqueue.MsgHandler) (func() error, error) {
	return nil, nil
}
func (m *recordingMQ) IsReady() bool { return true }

type fakeListTasks struct {
	v1.Repository
	v1.TaskRepository
	tasks []*sqlcv1.V1Task
}

func (f fakeListTasks) Tasks() v1.TaskRepository { return f }
func (f fakeListTasks) ListTasks(context.Context, uuid.UUID, []int64) ([]*sqlcv1.V1Task, error) {
	return f.tasks, nil
}

func TestBatchStartForDisconnectedWorkerRetriesMembers(t *testing.T) {
	tenantId := uuid.New()
	insertedAt := pgtype.Timestamptz{Time: time.Now(), Valid: true}

	tasks := []*sqlcv1.V1Task{
		{ID: 1, InsertedAt: insertedAt, ExternalID: uuid.New(), WorkflowRunID: uuid.New()},
		{ID: 2, InsertedAt: insertedAt, ExternalID: uuid.New(), WorkflowRunID: uuid.New()},
	}

	mq := &recordingMQ{}
	d := &DispatcherImpl{
		l:       zerologNop(),
		workers: &workers{},
		mqv1:    mq,
		pubsub:  &fakePubSub{published: make(chan published, 16)},
		repov1:  fakeListTasks{tasks: tasks},
	}

	items := make([]tasktypes.BatchTaskItem, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, tasktypes.BatchTaskItem{TaskID: task.ID, ExternalID: task.ExternalID, InsertedAt: insertedAt.Time})
	}

	msg, err := tasktypes.StartBatchMessage(tenantId, tasktypes.StartBatchTaskPayload{
		TenantId:     tenantId,
		WorkerId:     uuid.New(),
		ActionId:     "default:batch-step",
		BatchId:      uuid.NewString(),
		ExpectedSize: len(items),
		TriggerTime:  time.Now(),
		Items:        items,
	})
	require.NoError(t, err)

	require.NoError(t, d.handleBatchStartTask(context.Background(), msg))

	failedTaskIds := map[int64]bool{}
	for _, sent := range mq.sent {
		require.Equal(t, msgqueue.MsgIDTaskFailed, sent.ID)

		for _, p := range msgqueue.JSONConvert[tasktypes.FailedTaskPayload](sent.Payloads) {
			require.False(t, p.IsAppError, "worker loss must be an internal retry, not an app failure")
			failedTaskIds[p.TaskId] = true
		}
	}

	require.Equal(t, map[int64]bool{1: true, 2: true}, failedTaskIds, "every batch member should be requeued instead of left to time out")
}
