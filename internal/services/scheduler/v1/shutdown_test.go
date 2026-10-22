package scheduler

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/partition"
	repov1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/scheduling"
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

func (m *recordingMQ) sentIDs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	ids := make([]string, 0, len(m.sent))
	for _, msg := range m.sent {
		ids = append(ids, msg.ID)
	}

	return ids
}

func (m *recordingMQ) Clone() (func() error, msgqueue.MessageQueue, error) { return nil, nil, nil }
func (m *recordingMQ) SetQOS(int)                                         {}
func (m *recordingMQ) Subscribe(msgqueue.Queue, msgqueue.MsgHandler, msgqueue.MsgHandler) (func() error, error) {
	return func() error { return nil }, nil
}
func (m *recordingMQ) IsReady() bool { return true }

type noopPubSub struct{}

func (noopPubSub) Pub(context.Context, msgqueue.Topic, *msgqueue.Message) error { return nil }
func (noopPubSub) Sub(msgqueue.Topic, msgqueue.MsgHandler) (func() error, error) {
	return func() error { return nil }, nil
}
func (noopPubSub) IsReady() bool { return true }

type fakeRepo struct {
	repov1.Repository
	repov1.WorkerRepository
	repov1.TenantRepository
	repov1.TaskRepository
	dispatcherId uuid.UUID
}

func (f fakeRepo) Workers() repov1.WorkerRepository { return f }
func (f fakeRepo) Tenant() repov1.TenantRepository  { return f }
func (f fakeRepo) Tasks() repov1.TaskRepository     { return f }

func (f fakeRepo) GetDispatcherIdsForWorkers(_ context.Context, _ uuid.UUID, workerIds []uuid.UUID) (map[uuid.UUID]uuid.UUID, map[uuid.UUID]struct{}, error) {
	res := make(map[uuid.UUID]uuid.UUID, len(workerIds))
	for _, id := range workerIds {
		res[id] = f.dispatcherId
	}

	return res, map[uuid.UUID]struct{}{}, nil
}

func (f fakeRepo) ListTenantsBySchedulerPartition(context.Context, string) ([]*sqlcv1.Tenant, error) {
	return nil, nil
}

// flushOnStopPool models a batch scheduler whose last flush commits while the pool is being torn down.
type flushOnStopPool struct {
	scheduling.Pool
	results     chan *scheduling.QueueResults
	concurrency chan *scheduling.ConcurrencyResults
	onStop      *scheduling.QueueResults
	once        sync.Once
}

func (p *flushOnStopPool) GetResultsCh() chan *scheduling.QueueResults { return p.results }
func (p *flushOnStopPool) GetConcurrencyResultsCh() chan *scheduling.ConcurrencyResults {
	return p.concurrency
}
func (p *flushOnStopPool) SetTenants([]*sqlcv1.Tenant) {}
func (p *flushOnStopPool) StopTenants() {
	p.once.Do(func() { p.results <- p.onStop })
}

func TestBatchFlushedDuringShutdownIsStillDispatched(t *testing.T) {
	tenantId := uuid.New()
	workerId := uuid.New()
	mq := &recordingMQ{}

	pool := &flushOnStopPool{
		results:     make(chan *scheduling.QueueResults, 10),
		concurrency: make(chan *scheduling.ConcurrencyResults, 10),
		onStop: &scheduling.QueueResults{
			TenantId: tenantId,
			Assigned: []*repov1.AssignedItem{{
				WorkerId: workerId,
				QueueItem: &sqlcv1.V1QueueItem{
					TaskID:         1,
					TaskInsertedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
					ExternalID:     uuid.New(),
					StepID:         uuid.New(),
					ActionID:       "default:batch-step",
				},
				Batch: &repov1.BatchAssignmentMetadata{BatchID: uuid.NewString(), Reason: repov1.FlushReasonIntervalElapsed},
			}},
		},
	}

	l := zerolog.Nop()

	s, err := New(
		WithMessageQueue(mq),
		WithPubSub(noopPubSub{}),
		WithRepository(fakeRepo{dispatcherId: uuid.New()}),
		WithPartition(&partition.Partition{}),
		WithSchedulerPool(pool),
		WithLogger(&l),
	)
	require.NoError(t, err)

	cleanup, err := s.Start()
	require.NoError(t, err)

	require.NoError(t, cleanup())

	// the engine tears the pool down after the scheduler service; a no-op once the service has stopped it
	pool.StopTenants()

	require.Eventually(t, func() bool {
		for _, id := range mq.sentIDs() {
			if id == msgqueue.MsgIDBatchStart {
				return true
			}
		}

		return false
	}, 2*time.Second, 10*time.Millisecond, "a batch committed during shutdown must still be sent to its worker")
}
