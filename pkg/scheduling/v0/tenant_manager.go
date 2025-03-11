package v0

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

// tenantManager manages the scheduler and queuers for a tenant and multiplexes
// messages to the relevant queuer.
type tenantManager struct {
	cf       *sharedConfig
	tenantId pgtype.UUID

	scheduler *Scheduler
	rl        *rateLimiter

	queuers   []*Queuer
	queuersMu sync.RWMutex

	leaseManager *LeaseManager

	workersCh <-chan []*repository.ListActiveWorkersResult
	queuesCh  <-chan []string
	resultsCh chan *QueueResults

	cleanup func()
}

func newTenantManager(cf *sharedConfig, tenantId string, resultsCh chan *QueueResults, exts *Extensions) *tenantManager {
	tenantIdUUID := sqlchelpers.UUIDFromStr(tenantId)

	rl := newRateLimiter(cf, tenantIdUUID)
	s := newScheduler(cf, tenantIdUUID, rl, exts)
	leaseManager, workersCh, queuesCh := newLeaseManager(cf, tenantIdUUID)

	t := &tenantManager{
		scheduler:    s,
		leaseManager: leaseManager,
		cf:           cf,
		tenantId:     tenantIdUUID,
		workersCh:    workersCh,
		queuesCh:     queuesCh,
		resultsCh:    resultsCh,
		rl:           rl,
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.cleanup = cancel

	go t.listenForWorkerLeases(ctx)
	go t.listenForQueueLeases(ctx)

	leaseManager.start(ctx)
	s.start(ctx)

	return t
}

func (t *tenantManager) Cleanup() error {
	defer t.cleanup()

	cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := t.leaseManager.cleanup(cleanupCtx)

	// clean up the other resources even if the lease manager fails to clean up
	t.queuersMu.RLock()
	defer t.queuersMu.RUnlock()

	for _, q := range t.queuers {
		q.Cleanup()
	}

	t.rl.cleanup()

	return err
}

func (t *tenantManager) listenForWorkerLeases(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case workerIds := <-t.workersCh:
			t.scheduler.setWorkers(workerIds)
		}
	}
}

func (t *tenantManager) listenForQueueLeases(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case queueNames := <-t.queuesCh:
			t.setQueuers(queueNames)
		}
	}
}

func (t *tenantManager) setQueuers(queueNames []string) {
	t.queuersMu.Lock()
	defer t.queuersMu.Unlock()

	queueNamesSet := make(map[string]struct{}, len(queueNames))

	for _, queueName := range queueNames {
		queueNamesSet[queueName] = struct{}{}
	}

	newQueueArr := make([]*Queuer, 0, len(queueNames))

	for _, q := range t.queuers {
		if _, ok := queueNamesSet[q.queueName]; ok {
			newQueueArr = append(newQueueArr, q)

			// delete from set
			delete(queueNamesSet, q.queueName)
		} else {
			// if not in new set, cleanup
			go q.Cleanup()
		}
	}

	for queueName := range queueNamesSet {
		newQueueArr = append(newQueueArr, newQueuer(t.cf, t.tenantId, queueName, t.scheduler, t.resultsCh))
	}

	t.queuers = newQueueArr
}

func (t *tenantManager) refreshAll(ctx context.Context) {
	t.queuersMu.RLock()

	eg := errgroup.Group{}

	for i := range t.queuers {
		index := i

		eg.Go(func() error {
			queuer := t.queuers[index]
			queuer.queue(ctx)
			return nil
		})
	}

	t.queuersMu.RUnlock()

	if err := eg.Wait(); err != nil {
		t.cf.l.Error().Err(err).Msg("error replenishing all queues")
	}
}

func (t *tenantManager) replenish(ctx context.Context) {
	err := t.scheduler.replenish(ctx, false)

	if err != nil {
		t.cf.l.Error().Err(err).Msg("error replenishing scheduler")
	}
}

func (t *tenantManager) queue(ctx context.Context, queueName string) {
	t.queuersMu.RLock()

	for _, q := range t.queuers {
		if q.queueName == queueName {
			t.queuersMu.RUnlock()
			q.queue(ctx)
			return
		}
	}

	t.queuersMu.RUnlock()
}
