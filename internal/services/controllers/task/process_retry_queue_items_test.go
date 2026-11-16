package task

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type publishedMsg struct {
	topic msgqueue.Topic
	msg   *msgqueue.Message
}

// recordingPubSub records every Pub call. Pub is only called from the test goroutine.
type recordingPubSub struct {
	published []publishedMsg
}

func (p *recordingPubSub) Pub(_ context.Context, topic msgqueue.Topic, msg *msgqueue.Message) error {
	p.published = append(p.published, publishedMsg{topic: topic, msg: msg})
	return nil
}

func (p *recordingPubSub) Sub(msgqueue.Topic, msgqueue.MsgHandler) (func() error, error) {
	return func() error { return nil }, nil
}

func (p *recordingPubSub) IsReady() bool { return true }

type retryQueueTaskRepo struct {
	v1.TaskRepository
	released []*sqlcv1.V1RetryQueueItem
	tasks    []*sqlcv1.V1Task
}

func (r *retryQueueTaskRepo) ProcessTaskRetryQueueItems(context.Context, uuid.UUID) ([]*sqlcv1.V1RetryQueueItem, bool, error) {
	return r.released, false, nil
}

func (r *retryQueueTaskRepo) ListTasks(_ context.Context, _ uuid.UUID, ids []int64) ([]*sqlcv1.V1Task, error) {
	requested := make(map[int64]struct{}, len(ids))

	for _, id := range ids {
		requested[id] = struct{}{}
	}

	res := make([]*sqlcv1.V1Task, 0, len(ids))

	for _, task := range r.tasks {
		if _, ok := requested[task.ID]; ok {
			res = append(res, task)
		}
	}

	return res, nil
}

type retryQueueTenantRepo struct {
	v1.TenantRepository
	tenant *sqlcv1.Tenant
}

func (r *retryQueueTenantRepo) GetTenantByID(context.Context, uuid.UUID) (*sqlcv1.Tenant, error) {
	return r.tenant, nil
}

type retryQueueRepo struct {
	v1.Repository
	tasks  *retryQueueTaskRepo
	tenant *retryQueueTenantRepo
}

func (r *retryQueueRepo) Tasks() v1.TaskRepository    { return r.tasks }
func (r *retryQueueRepo) Tenant() v1.TenantRepository { return r.tenant }

// A retry that comes due is requeued by a database trigger, so the controller has to tell the
// scheduler which queues and concurrency strategies to check, or the retry sits until the next poll.
func TestProcessTaskRetryQueueItemsNotifiesScheduler(t *testing.T) {
	const partitionId = "partition-a"

	queuedTask := func(id int64, queue string, strategyIds ...int64) *sqlcv1.V1Task {
		return &sqlcv1.V1Task{ID: id, Queue: queue, ConcurrencyStrategyIds: strategyIds}
	}

	tests := []struct {
		name            string
		tasks           []*sqlcv1.V1Task
		released        []int64
		wantQueueNames  []string
		wantStrategyIds []int64
		wantNoPublish   bool
	}{
		{
			name:           "a task without a concurrency strategy wakes its queue",
			tasks:          []*sqlcv1.V1Task{queuedTask(1, "queue-a")},
			released:       []int64{1},
			wantQueueNames: []string{"queue-a"},
		},
		{
			name:            "a task with concurrency strategies wakes only its first strategy",
			tasks:           []*sqlcv1.V1Task{queuedTask(1, "queue-a", 7, 8)},
			released:        []int64{1},
			wantStrategyIds: []int64{7},
		},
		{
			name: "queues and strategies are notified once each",
			tasks: []*sqlcv1.V1Task{
				queuedTask(1, "queue-a"),
				queuedTask(2, "queue-a"),
				queuedTask(3, "queue-b"),
				queuedTask(4, "queue-c", 7),
				queuedTask(5, "queue-c", 7),
			},
			released:        []int64{1, 2, 3, 4, 5},
			wantQueueNames:  []string{"queue-a", "queue-b"},
			wantStrategyIds: []int64{7},
		},
		{
			name:           "only released tasks are notified",
			tasks:          []*sqlcv1.V1Task{queuedTask(1, "queue-a"), queuedTask(2, "queue-b")},
			released:       []int64{2},
			wantQueueNames: []string{"queue-b"},
		},
		{
			name:          "nothing released publishes nothing",
			tasks:         []*sqlcv1.V1Task{queuedTask(1, "queue-a")},
			wantNoPublish: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenantId := uuid.New()

			released := make([]*sqlcv1.V1RetryQueueItem, 0, len(tt.released))

			for _, id := range tt.released {
				released = append(released, &sqlcv1.V1RetryQueueItem{TaskID: id, TaskRetryCount: 1, TenantID: tenantId})
			}

			pubsub := &recordingPubSub{}
			pubBuffer := msgqueue.NewMQPubBuffer(&fakeMQ{})
			t.Cleanup(pubBuffer.Stop)

			l := zerolog.Nop()

			tc := &TasksControllerImpl{
				l:         &l,
				pubsub:    pubsub,
				pubBuffer: pubBuffer,
				repov1: &retryQueueRepo{
					tasks: &retryQueueTaskRepo{released: released, tasks: tt.tasks},
					tenant: &retryQueueTenantRepo{tenant: &sqlcv1.Tenant{
						ID:                   tenantId,
						SchedulerPartitionId: sqlchelpers.TextFromStr(partitionId),
					}},
				},
			}

			_, err := tc.processTaskRetryQueueItems(context.Background(), tenantId.String())
			require.NoError(t, err)

			if tt.wantNoPublish {
				require.Empty(t, pubsub.published)
				return
			}

			require.Len(t, pubsub.published, 1)

			got := pubsub.published[0]

			assert.Equal(t, msgqueue.SchedulerPartitionTopic(partitionId), got.topic)
			assert.Equal(t, msgqueue.MsgIDCheckTenantQueue, got.msg.ID)
			assert.Equal(t, tenantId, got.msg.TenantID)

			payloads := msgqueue.JSONConvert[tasktypes.CheckTenantQueuesPayload](got.msg.Payloads)
			require.Len(t, payloads, 1)

			assert.ElementsMatch(t, tt.wantQueueNames, payloads[0].QueueNames)
			assert.ElementsMatch(t, tt.wantStrategyIds, payloads[0].StrategyIds)
			assert.False(t, payloads[0].SlotsReleased)
		})
	}
}
