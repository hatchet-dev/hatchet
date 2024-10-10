package v2

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sasha-s/go-deadlock"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

// tenantManager manages the scheduler and queuers for a tenant and multiplexes
// messages to the relevant queuer.
type tenantManager struct {
	cf       *sharedConfig
	tenantId pgtype.UUID

	scheduler *Scheduler

	queuers   []*Queuer
	queuersMu deadlock.RWMutex

	leaseManager *LeaseManager

	workersCh <-chan []pgtype.UUID
	queuesCh  <-chan []string
	resultsCh chan *QueueResults

	cleanup func()
}

func newTenantManager(cf *sharedConfig, tenantId string, resultsCh chan *QueueResults) *tenantManager {
	fmt.Println("<<<<<<>>>>>>>> INITIALIZING TENANT MANAGER")

	tenantIdUUID := sqlchelpers.UUIDFromStr(tenantId)

	s := newScheduler(cf, tenantIdUUID)
	leaseManager, workersCh, queuesCh := newLeaseManager(cf, tenantIdUUID)

	t := &tenantManager{
		scheduler:    s,
		leaseManager: leaseManager,
		cf:           cf,
		tenantId:     tenantIdUUID,
		workersCh:    workersCh,
		queuesCh:     queuesCh,
		resultsCh:    resultsCh,
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
	fmt.Println("<<<<<<>>>>>>>> CLEANING UP TENANT MANAGER")
	defer t.cleanup()

	cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := t.leaseManager.cleanup(cleanupCtx)

	if err != nil {
		return err
	}

	for _, q := range t.queuers {
		q.Cleanup()
	}

	return nil
}

func (t *tenantManager) listenForWorkerLeases(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case workerIds := <-t.workersCh:
			t.scheduler.setWorkerIds(workerIds)
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
		}
	}

	for queueName := range queueNamesSet {
		newQueueArr = append(newQueueArr, newQueuer(t.cf, t.tenantId, queueName, t.scheduler, t.resultsCh))
	}

	t.queuers = newQueueArr
}

func (t *tenantManager) refreshAll(ctx context.Context) {
	t.queuersMu.RLock()
	defer t.queuersMu.RUnlock()

	eg := errgroup.Group{}

	eg.Go(func() error {
		return t.scheduler.replenish(ctx)
	})

	for i := range t.queuers {
		index := i

		eg.Go(func() error {
			queuer := t.queuers[index]
			queuer.queue()
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		t.cf.l.Error().Err(err).Msg("error replenishing all queues")
	}
}

func (t *tenantManager) replenish(ctx context.Context, queueName string) {
	t.queuersMu.RLock()
	defer t.queuersMu.RUnlock()

	err := t.scheduler.replenish(ctx)

	if err != nil {
		t.cf.l.Error().Err(err).Msg("error replenishing scheduler")
	}
}

func (t *tenantManager) queue(ctx context.Context, queueName string) {
	t.queuersMu.RLock()
	defer t.queuersMu.RUnlock()

	for _, q := range t.queuers {
		if q.queueName == queueName {
			q.queue()
			return
		}
	}
}
