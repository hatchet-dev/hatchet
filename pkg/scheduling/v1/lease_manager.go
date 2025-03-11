package v2

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"golang.org/x/sync/errgroup"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

// LeaseManager is responsible for leases on multiple queues and multiplexing
// queue results to callers. It is still tenant-scoped.
type LeaseManager struct {
	lr v1.LeaseRepository

	conf *sharedConfig

	tenantId pgtype.UUID

	workerLeasesMu sync.Mutex
	workerLeases   []*sqlcv1.Lease
	workersCh      chan<- []*v1.ListActiveWorkersResult

	queueLeasesMu sync.Mutex
	queueLeases   []*sqlcv1.Lease
	queuesCh      chan<- []string

	concurrencyLeasesMu sync.Mutex
	concurrencyLeases   []*sqlcv1.Lease
	concurrencyLeasesCh chan<- []*sqlcv1.V1StepConcurrency

	cleanedUp bool
	cleanupMu sync.Mutex
}

func newLeaseManager(conf *sharedConfig, tenantId pgtype.UUID) (*LeaseManager, <-chan []*v1.ListActiveWorkersResult, <-chan []string, <-chan []*sqlcv1.V1StepConcurrency) {
	workersCh := make(chan []*v1.ListActiveWorkersResult)
	queuesCh := make(chan []string)
	concurrencyLeasesCh := make(chan []*sqlcv1.V1StepConcurrency)

	return &LeaseManager{
		lr:                  conf.repo.Lease(),
		conf:                conf,
		tenantId:            tenantId,
		workersCh:           workersCh,
		queuesCh:            queuesCh,
		concurrencyLeasesCh: concurrencyLeasesCh,
	}, workersCh, queuesCh, concurrencyLeasesCh
}

func (l *LeaseManager) sendWorkerIds(workerIds []*v1.ListActiveWorkersResult) {
	defer func() {
		if r := recover(); r != nil {
			l.conf.l.Error().Interface("recovered", r).Msg("recovered from panic")
		}
	}()

	// can't cleanup while sending
	l.cleanupMu.Lock()
	defer l.cleanupMu.Unlock()

	if l.cleanedUp {
		return
	}

	select {
	case l.workersCh <- workerIds:
	default:
	}
}

func (l *LeaseManager) sendQueues(queues []string) {
	defer func() {
		if r := recover(); r != nil {
			l.conf.l.Error().Interface("recovered", r).Msg("recovered from panic")
		}
	}()

	// can't cleanup while sending
	l.cleanupMu.Lock()
	defer l.cleanupMu.Unlock()

	if l.cleanedUp {
		return
	}

	select {
	case l.queuesCh <- queues:
	default:
	}
}

func (l *LeaseManager) sendConcurrencyLeases(concurrencyLeases []*sqlcv1.V1StepConcurrency) {
	defer func() {
		if r := recover(); r != nil {
			l.conf.l.Error().Interface("recovered", r).Msg("recovered from panic")
		}
	}()

	// can't cleanup while sending
	l.cleanupMu.Lock()
	defer l.cleanupMu.Unlock()

	if l.cleanedUp {
		return
	}

	select {
	case l.concurrencyLeasesCh <- concurrencyLeases:
	default:
	}
}

func (l *LeaseManager) acquireWorkerLeases(ctx context.Context) error {
	if ok := l.workerLeasesMu.TryLock(); !ok {
		return nil
	}

	defer l.workerLeasesMu.Unlock()

	activeWorkers, err := l.lr.ListActiveWorkers(ctx, l.tenantId)

	if err != nil {
		return err
	}

	currResourceIdsToLease := make(map[string]*sqlcv1.Lease, len(l.workerLeases))

	for _, lease := range l.workerLeases {
		currResourceIdsToLease[lease.ResourceId] = lease
	}

	workerIdsStr := make([]string, len(activeWorkers))
	activeWorkerIdsToResults := make(map[string]*v1.ListActiveWorkersResult, len(activeWorkers))

	leasesToExtend := make([]*sqlcv1.Lease, 0, len(activeWorkers))
	leasesToRelease := make([]*sqlcv1.Lease, 0, len(currResourceIdsToLease))

	for i, activeWorker := range activeWorkers {
		aw := activeWorker
		workerIdsStr[i] = activeWorker.ID
		activeWorkerIdsToResults[workerIdsStr[i]] = aw

		if lease, ok := currResourceIdsToLease[workerIdsStr[i]]; ok {
			leasesToExtend = append(leasesToExtend, lease)
			delete(currResourceIdsToLease, workerIdsStr[i])
		}
	}

	for _, lease := range currResourceIdsToLease {
		leasesToRelease = append(leasesToRelease, lease)
	}

	successfullyAcquiredWorkerIds := make([]*v1.ListActiveWorkersResult, 0)

	if len(workerIdsStr) != 0 {
		workerLeases, err := l.lr.AcquireOrExtendLeases(ctx, l.tenantId, sqlcv1.LeaseKindWORKER, workerIdsStr, leasesToExtend)

		if err != nil {
			return err
		}

		l.workerLeases = workerLeases

		for _, lease := range workerLeases {
			successfullyAcquiredWorkerIds = append(successfullyAcquiredWorkerIds, activeWorkerIdsToResults[lease.ResourceId])
		}
	}

	l.sendWorkerIds(successfullyAcquiredWorkerIds)

	if len(leasesToRelease) != 0 {
		if err := l.lr.ReleaseLeases(ctx, l.tenantId, leasesToRelease); err != nil {
			return err
		}
	}

	return nil
}

func (l *LeaseManager) acquireQueueLeases(ctx context.Context) error {
	if ok := l.queueLeasesMu.TryLock(); !ok {
		return nil
	}

	defer l.queueLeasesMu.Unlock()

	queues, err := l.lr.ListQueues(ctx, l.tenantId)

	if err != nil {
		return err
	}

	currResourceIdsToLease := make(map[string]*sqlcv1.Lease, len(l.queueLeases))

	for _, lease := range l.queueLeases {
		currResourceIdsToLease[lease.ResourceId] = lease
	}

	queueIdsStr := make([]string, len(queues))
	leasesToExtend := make([]*sqlcv1.Lease, 0, len(queues))
	leasesToRelease := make([]*sqlcv1.Lease, 0, len(currResourceIdsToLease))

	for i, q := range queues {
		queueIdsStr[i] = q.Name

		if lease, ok := currResourceIdsToLease[queueIdsStr[i]]; ok {
			leasesToExtend = append(leasesToExtend, lease)
			delete(currResourceIdsToLease, queueIdsStr[i])
		}
	}

	for _, lease := range currResourceIdsToLease {
		leasesToRelease = append(leasesToRelease, lease)
	}

	successfullyAcquiredQueues := []string{}

	if len(queueIdsStr) != 0 {

		queueLeases, err := l.lr.AcquireOrExtendLeases(ctx, l.tenantId, sqlcv1.LeaseKindQUEUE, queueIdsStr, leasesToExtend)

		if err != nil {
			return err
		}

		l.queueLeases = queueLeases

		for _, lease := range queueLeases {
			successfullyAcquiredQueues = append(successfullyAcquiredQueues, lease.ResourceId)
		}
	}

	l.sendQueues(successfullyAcquiredQueues)

	if len(leasesToRelease) != 0 {
		if err := l.lr.ReleaseLeases(ctx, l.tenantId, leasesToRelease); err != nil {
			return err
		}
	}

	return nil
}

func (l *LeaseManager) acquireConcurrencyLeases(ctx context.Context) error {
	if ok := l.concurrencyLeasesMu.TryLock(); !ok {
		return nil
	}

	defer l.concurrencyLeasesMu.Unlock()

	strats, err := l.lr.ListConcurrencyStrategies(ctx, l.tenantId)

	if err != nil {
		return err
	}

	currResourceIdsToLease := make(map[string]*sqlcv1.Lease, len(l.concurrencyLeases))

	for _, lease := range l.concurrencyLeases {
		currResourceIdsToLease[lease.ResourceId] = lease
	}

	strategyIdsStr := make([]string, len(strats))
	activeStratIdsToStrategies := make(map[string]*sqlcv1.V1StepConcurrency, len(strats))

	leasesToExtend := make([]*sqlcv1.Lease, 0, len(strats))
	leasesToRelease := make([]*sqlcv1.Lease, 0, len(currResourceIdsToLease))

	for i, s := range strats {
		strategyIdsStr[i] = fmt.Sprintf("%d", s.ID)

		if lease, ok := currResourceIdsToLease[strategyIdsStr[i]]; ok {
			leasesToExtend = append(leasesToExtend, lease)
			delete(currResourceIdsToLease, strategyIdsStr[i])
		}

		activeStratIdsToStrategies[strategyIdsStr[i]] = s
	}

	for _, lease := range currResourceIdsToLease {
		leasesToRelease = append(leasesToRelease, lease)
	}

	successfullyAcquiredStrats := []*sqlcv1.V1StepConcurrency{}

	if len(strategyIdsStr) != 0 {

		concurrencyLeases, err := l.lr.AcquireOrExtendLeases(ctx, l.tenantId, sqlcv1.LeaseKindCONCURRENCYSTRATEGY, strategyIdsStr, leasesToExtend)

		if err != nil {
			return err
		}

		l.concurrencyLeases = concurrencyLeases

		for _, lease := range concurrencyLeases {
			successfullyAcquiredStrats = append(successfullyAcquiredStrats, activeStratIdsToStrategies[lease.ResourceId])
		}
	}

	l.sendConcurrencyLeases(successfullyAcquiredStrats)

	if len(leasesToRelease) != 0 {
		if err := l.lr.ReleaseLeases(ctx, l.tenantId, leasesToRelease); err != nil {
			return err
		}
	}

	return nil
}

// loopForLeases acquires new leases every 1 second for workers and queues
func (l *LeaseManager) loopForLeases(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			wg := sync.WaitGroup{}

			wg.Add(3)

			go func() {
				defer wg.Done()
				if err := l.acquireWorkerLeases(ctx); err != nil {
					l.conf.l.Error().Err(err).Msg("error acquiring worker leases")
				}
			}()

			go func() {
				defer wg.Done()
				if err := l.acquireQueueLeases(ctx); err != nil {
					l.conf.l.Error().Err(err).Msg("error acquiring queue leases")
				}
			}()

			go func() {
				defer wg.Done()
				if err := l.acquireConcurrencyLeases(ctx); err != nil {
					l.conf.l.Error().Err(err).Msg("error acquiring concurrency leases")
				}
			}()

			wg.Wait()
		}
	}
}

func (l *LeaseManager) cleanup(ctx context.Context) error {
	l.cleanupMu.Lock()
	defer l.cleanupMu.Unlock()

	if l.cleanedUp {
		return nil
	}

	l.cleanedUp = true

	// close channels
	defer close(l.workersCh)
	defer close(l.queuesCh)
	defer close(l.concurrencyLeasesCh)

	eg := errgroup.Group{}

	eg.Go(func() error {
		l.workerLeasesMu.Lock()
		defer l.workerLeasesMu.Unlock()

		return l.lr.ReleaseLeases(ctx, l.tenantId, l.workerLeases)
	})

	eg.Go(func() error {
		l.queueLeasesMu.Lock()
		defer l.queueLeasesMu.Unlock()

		return l.lr.ReleaseLeases(ctx, l.tenantId, l.queueLeases)
	})

	eg.Go(func() error {
		l.concurrencyLeasesMu.Lock()
		defer l.concurrencyLeasesMu.Unlock()

		return l.lr.ReleaseLeases(ctx, l.tenantId, l.concurrencyLeases)
	})

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}

func (l *LeaseManager) start(ctx context.Context) {
	go l.loopForLeases(ctx)
}
