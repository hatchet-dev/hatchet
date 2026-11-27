package scheduler

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/partition"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	repov1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/scheduling"
	v1 "github.com/hatchet-dev/hatchet/pkg/scheduling/v1"
)

// brokerMQ fails a publish whose context is already cancelled, as the rabbitmq implementation does
// when it acquires a channel, and records the task ids of every bulk assignment it delivers.
type brokerMQ struct {
	mu        sync.Mutex
	delivered []int64
}

func (m *brokerMQ) SendMessage(ctx context.Context, _ msgqueue.Queue, msg *msgqueue.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if msg.ID != msgqueue.MsgIDTaskAssignedBulk {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, payload := range msgqueue.JSONConvert[tasktypes.TaskAssignedBulkTaskPayload](msg.Payloads) {
		for _, taskIds := range payload.WorkerIdToTaskIds {
			m.delivered = append(m.delivered, taskIds...)
		}
	}

	return nil
}

func (m *brokerMQ) deliveredTaskIds() []int64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	return append([]int64(nil), m.delivered...)
}

func (m *brokerMQ) Clone() (func() error, msgqueue.MessageQueue, error) { return nil, nil, nil }
func (m *brokerMQ) SetQOS(_ int)                                        {}
func (m *brokerMQ) Subscribe(_ msgqueue.Queue, _, _ msgqueue.MsgHandler) (func() error, error) {
	return func() error { return nil }, nil
}
func (m *brokerMQ) IsReady() bool { return true }

type noopPubSub struct{}

func (noopPubSub) Pub(_ context.Context, _ msgqueue.Topic, _ *msgqueue.Message) error { return nil }
func (noopPubSub) Sub(_ msgqueue.Topic, _ msgqueue.MsgHandler) (func() error, error) {
	return func() error { return nil }, nil
}
func (noopPubSub) IsReady() bool { return true }

// stoppingPool hands out the result channels and, when told to drop every tenant, emits one more
// committed assignment: a queuer flush that committed while its tenant was being stopped.
type stoppingPool struct {
	scheduling.Pool

	results      chan *v1.QueueResults
	lastResult   *v1.QueueResults
	tenantsSetTo chan []*sqlcv1.Tenant
}

func (p *stoppingPool) GetResultsCh() chan *v1.QueueResults { return p.results }
func (p *stoppingPool) GetConcurrencyResultsCh() chan *v1.ConcurrencyResults {
	return make(chan *v1.ConcurrencyResults)
}
func (p *stoppingPool) GetRateLimitPausesEndedCh() chan uuid.UUID { return make(chan uuid.UUID) }

func (p *stoppingPool) SetTenants(tenants []*sqlcv1.Tenant) {
	p.results <- p.lastResult
	p.tenantsSetTo <- tenants
}

type noTenantsRepo struct{ repov1.Repository }

func (noTenantsRepo) Tenant() repov1.TenantRepository { return noTenantsTenantRepo{} }

type noTenantsTenantRepo struct{ repov1.TenantRepository }

func (noTenantsTenantRepo) ListTenantsBySchedulerPartition(_ context.Context, _ string) ([]*sqlcv1.Tenant, error) {
	return nil, errors.New("tenant reconciliation is not exercised here")
}

func assignedResult(tenantId, workerId uuid.UUID, taskId int64) *v1.QueueResults {
	return &v1.QueueResults{
		TenantId: tenantId,
		Assigned: []*repov1.AssignedItem{{
			WorkerId: workerId,
			QueueItem: &sqlcv1.V1QueueItem{
				TaskID:         taskId,
				TaskInsertedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			},
		}},
	}
}

func TestSchedulerPublishesEveryCommittedAssignmentWhenShuttingDown(t *testing.T) {
	const inFlightTaskId, lastTaskId = int64(1), int64(2)

	tenantId, workerId, dispatcherId := uuid.New(), uuid.New(), uuid.New()

	mq := &brokerMQ{}
	pool := &stoppingPool{
		results:      make(chan *v1.QueueResults, 10),
		lastResult:   assignedResult(tenantId, workerId, lastTaskId),
		tenantsSetTo: make(chan []*sqlcv1.Tenant, 1),
	}

	// the in-flight assignment stays in its dispatcher lookup until shutdown has begun
	lookupStarted := make(chan struct{}, 2)
	releaseLookup := make(chan struct{})

	lookup := func(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (map[uuid.UUID]uuid.UUID, map[uuid.UUID]struct{}, error) {
		lookupStarted <- struct{}{}
		<-releaseLookup

		return map[uuid.UUID]uuid.UUID{workerId: dispatcherId}, map[uuid.UUID]struct{}{}, nil
	}

	cron, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))
	require.NoError(t, err)

	l := zerolog.Nop()

	s := &Scheduler{
		mq:            mq,
		pubsub:        noopPubSub{},
		pubBuffer:     msgqueue.NewMQPubBuffer(mq),
		l:             &l,
		repov1:        noTenantsRepo{},
		s:             cron,
		p:             &partition.Partition{},
		pool:          pool,
		dispatcherIds: newDispatcherIdCache(lookup),
		queueMetrics:  newQueueMetricsPoller(nil, &l),
	}

	cleanup, err := s.Start()
	require.NoError(t, err)

	pool.results <- assignedResult(tenantId, workerId, inFlightTaskId)

	select {
	case <-lookupStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("the in-flight assignment was never handled")
	}

	cleanupDone := make(chan error, 1)

	go func() { cleanupDone <- cleanup() }()

	cleanupReturned := false

	select {
	case tenants := <-pool.tenantsSetTo:
		assert.Empty(t, tenants, "shutdown drops every tenant from the pool so it stops assigning")
	case err := <-cleanupDone:
		require.NoError(t, err)
		cleanupReturned = true
		t.Error("cleanup returned without stopping the pool from assigning")
	case <-time.After(5 * time.Second):
		t.Fatal("cleanup neither stopped the pool nor returned")
	}

	close(releaseLookup)

	if !cleanupReturned {
		select {
		case err := <-cleanupDone:
			require.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("cleanup did not return")
		}
	}

	assert.ElementsMatch(t, []int64{inFlightTaskId, lastTaskId}, mq.deliveredTaskIds(),
		"every assignment the pool committed reaches its dispatcher by the time cleanup returns")
}
