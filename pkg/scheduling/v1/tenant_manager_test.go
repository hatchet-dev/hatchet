//go:build !e2e && !load && !rampup && !integration

package v1

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/pgoutbox"
	outboxsqlc "github.com/hatchet-dev/pgoutbox/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/shared/timeout_lock"
	"github.com/hatchet-dev/hatchet/pkg/enums"
	v1repo "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type topicsRelease struct {
	topics []string
	// holdersStopped is whether every released topic had been acquired under a context that
	// had ended by the time of the release
	holdersStopped bool
}

// leaseRecordingOutbox records the topic leases concurrency managers take and the releases made
// for them. Methods a manager never calls are left to the nil embedded interface.
type leaseRecordingOutbox struct {
	pgoutbox.Outbox

	mu          sync.Mutex
	acquireCtxs map[string]context.Context
	releases    []topicsRelease
}

func (o *leaseRecordingOutbox) AddFlusher(string, pgoutbox.Flusher) {}

func (o *leaseRecordingOutbox) AcquireTopic(ctx context.Context, topic string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.acquireCtxs[topic] = ctx

	return nil
}

func (o *leaseRecordingOutbox) ReleaseTopics(_ context.Context, topics []string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	holdersStopped := true

	for _, topic := range topics {
		acquireCtx, held := o.acquireCtxs[topic]

		if !held || acquireCtx.Err() == nil {
			holdersStopped = false
		}
	}

	o.releases = append(o.releases, topicsRelease{topics: topics, holdersStopped: holdersStopped})

	return nil
}

func (o *leaseRecordingOutbox) ProcessMessages(context.Context, string, ...pgoutbox.ProcessOpt) ([]*outboxsqlc.Message, error) {
	return nil, nil
}

// Subscribe holds the topic until its context ends, like an exclusive subscription.
func (o *leaseRecordingOutbox) Subscribe(ctx context.Context, _ string, _ ...pgoutbox.SubscribeOpt) error {
	<-ctx.Done()

	return nil
}

// emptyConcurrencyRepo serves strategies that have no slots.
type emptyConcurrencyRepo struct {
	v1repo.ConcurrencyRepository
}

func (emptyConcurrencyRepo) ReadConcurrencySlotsForIndexing(context.Context, uuid.UUID, int64, chan<- *sqlcv1.ListConcurrencySlotsForIndexingRow) error {
	return nil
}

func (emptyConcurrencyRepo) RunConcurrencyStrategy(context.Context, uuid.UUID, *sqlcv1.V1StepConcurrency) (*v1repo.RunConcurrencyResult, error) {
	return &v1repo.RunConcurrencyResult{}, nil
}

type concurrencyOnlySchedulerRepo struct {
	v1repo.SchedulerRepository
}

func (concurrencyOnlySchedulerRepo) Concurrency() v1repo.ConcurrencyRepository {
	return emptyConcurrencyRepo{}
}

// Tearing down a tenant's concurrency managers must hand back the outbox topic leases held by
// the ones running an in-memory index, in a single release made after those managers have
// stopped. A manager on the SQL path has no lease of its own to hand back.
func TestTenantManagerCleanupReleasesConcurrencyTopicLeases(t *testing.T) {
	l := zerolog.Nop()
	tenantId := uuid.New()
	outbox := &leaseRecordingOutbox{acquireCtxs: make(map[string]context.Context)}

	// the hour-long intervals leave the run on startup as each manager's only run
	conf := &sharedConfig{
		repo:                                   concurrencyOnlySchedulerRepo{},
		outbox:                                 outbox,
		l:                                      &l,
		schedulerConcurrencyPollingMinInterval: time.Hour,
		schedulerConcurrencyPollingMaxInterval: 2 * time.Hour,
		schedulerCheckActiveMinInterval:        time.Hour,
		schedulerCheckActiveMaxInterval:        2 * time.Hour,
		concurrencyInMemoryIndexEnabled:        true,
	}

	results := make(chan *ConcurrencyResults, 8)

	tm := &tenantManager{
		cf:                            conf,
		l:                             l,
		tenantId:                      tenantId,
		concurrencyResultsCh:          results,
		concurrencyAdvisoryLock:       timeout_lock.NewKeyedTimeoutLock[int64](time.Second),
		concurrencyParentAdvisoryLock: timeout_lock.NewKeyedTimeoutLock[int64](time.Second),
	}

	newStrategy := func(id int64) *sqlcv1.V1StepConcurrency {
		return &sqlcv1.V1StepConcurrency{
			ID:             id,
			TenantID:       tenantId,
			Strategy:       enums.V1ConcurrencyStrategyGROUPROUNDROBIN,
			MaxConcurrency: 1,
		}
	}

	// a strategy with a parent always runs on the SQL path
	sqlPathStrategy := newStrategy(3)
	sqlPathStrategy.ParentStrategyID = pgtype.Int8{Int64: 1, Valid: true}

	strategies := []*sqlcv1.V1StepConcurrency{newStrategy(1), newStrategy(2), sqlPathStrategy}

	for _, strategy := range strategies {
		tm.addConcurrencyStrategy(strategy)
	}

	// an in-memory manager only produces a result once its index is built, which is after it
	// has acquired its topic
	for range strategies {
		select {
		case <-results:
		case <-time.After(5 * time.Second):
			t.Fatal("a concurrency manager did not complete its first run")
		}
	}

	tm.cleanupConcurrencyStrategies(context.Background())

	outbox.mu.Lock()
	defer outbox.mu.Unlock()

	require.Len(t, outbox.releases, 1, "the tenant's topics should be released together")
	require.ElementsMatch(t, []string{
		fmt.Sprintf("concurrency.%s.1", tenantId),
		fmt.Sprintf("concurrency.%s.2", tenantId),
	}, outbox.releases[0].topics)
	require.True(t, outbox.releases[0].holdersStopped, "topics must be released only after their managers have stopped")
	require.Empty(t, tm.concurrencyStrategies)
}

// idleTestQueuers returns queuers with no queue loop running, so a wake stays buffered in
// notifyQueueCh where a test can observe it.
func idleTestQueuers(l *zerolog.Logger, n int) []*Queuer {
	queuers := make([]*Queuer, n)

	for i := range queuers {
		queuers[i] = &Queuer{l: l, notifyQueueCh: make(chan map[string]string, 1)}
	}

	return queuers
}

func wokenCount(queuers []*Queuer) int {
	woken := 0

	for _, q := range queuers {
		woken += len(q.notifyQueueCh)
	}

	return woken
}

// A spread shorter than minNewWorkerWakeSpread is not applied, and the window never exceeds
// maxNewWorkerWakeSpread.
func TestTenantManagerNewWorkerWakeSpread(t *testing.T) {
	require.Zero(t, newWorkerWakeSpread(0))
	require.Zero(t, newWorkerWakeSpread(499), "a spread under the minimum is not applied")
	require.Equal(t, minNewWorkerWakeSpread, newWorkerWakeSpread(500))
	require.Equal(t, 5500*time.Second/newWorkerWakeRate, newWorkerWakeSpread(5500))
	require.Equal(t, maxNewWorkerWakeSpread, newWorkerWakeSpread(1_000_000))
}

// A tenant with few queuers wakes all of them as soon as a worker joins.
func TestTenantManagerNewWorkerWakesFewQueuersTogether(t *testing.T) {
	l := zerolog.Nop()
	tm := &tenantManager{l: l, queuers: idleTestQueuers(&l, 10)}

	tm.wakeQueuersForNewWorker(context.Background())

	require.Equal(t, 10, wokenCount(tm.queuers))
}

// A tenant with many queuers wakes the ones holding unassigned items as soon as a worker
// joins and spreads the wakes of the rest, without dropping any.
func TestTenantManagerNewWorkerSpreadsIdleQueuerWakes(t *testing.T) {
	l := zerolog.Nop()

	idle := idleTestQueuers(&l, 2500)
	holding := idleTestQueuers(&l, 3)

	for _, q := range holding {
		q.unassignedCount.Store(1)
	}

	tm := &tenantManager{l: l, queuers: slices.Concat(idle, holding)}

	tm.wakeQueuersForNewWorker(context.Background())

	require.Equal(t, len(holding), wokenCount(holding), "queuers holding unassigned items must not wait")
	require.Less(t, wokenCount(idle), len(idle)/2, "idle queuers must not all wake together")

	require.Eventually(t, func() bool {
		return wokenCount(idle) == len(idle)
	}, 5*time.Second, 10*time.Millisecond, "every idle queuer must still be woken")
}

// A tenant whose managers hold no topic leases has nothing to release.
func TestTenantManagerCleanupSkipsReleaseWithoutLeasedTopics(t *testing.T) {
	l := zerolog.Nop()
	outbox := &leaseRecordingOutbox{acquireCtxs: make(map[string]context.Context)}

	tm := &tenantManager{
		cf: &sharedConfig{outbox: outbox, l: &l},
		l:  l,
	}

	tm.cleanupConcurrencyStrategies(context.Background())

	outbox.mu.Lock()
	defer outbox.mu.Unlock()

	require.Empty(t, outbox.releases)
}
