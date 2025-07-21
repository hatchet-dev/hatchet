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

// leaseExpiryThreshold is the time threshold for extending leases.
// Only leases that expire within this duration will be extended.
const leaseExpiryThreshold = 10 * time.Second

// shouldExtendLease returns true if the lease should be extended.
// This happens when:
// 1. ExpiresAt is not set (treat as expired), or
// 2. ExpiresAt is set and expires within the threshold
func shouldExtendLease(lease *sqlcv1.Lease) bool {
	if !lease.ExpiresAt.Valid {
		// If ExpiresAt is not set, treat it as expired and extend it
		return true
	}

	return time.Until(lease.ExpiresAt.Time) < leaseExpiryThreshold
}

// LeaseManager is responsible for leases on multiple queues and multiplexing
// queue results to callers. It is still tenant-scoped.
type LeaseManager struct {
	lr v1.LeaseRepository

	conf *sharedConfig

	tenantId pgtype.UUID

	workerLeases []*sqlcv1.Lease
	workersCh    chan<- []*v1.ListActiveWorkersResult

	queueLeases []*sqlcv1.Lease
	queuesCh    chan<- []string

	concurrencyLeases   []*sqlcv1.Lease
	concurrencyLeasesCh chan<- []*sqlcv1.V1StepConcurrency

	cleanedUp bool
	processMu sync.Mutex
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

	// at this point, we have a cleanupMu lock, so it's safe to read
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

	// at this point, we have a cleanupMu lock, so it's safe to read
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

	// at this point, we have a cleanupMu lock, so it's safe to read
	if l.cleanedUp {
		return
	}

	select {
	case l.concurrencyLeasesCh <- concurrencyLeases:
	default:
	}
}

func (l *LeaseManager) acquireWorkerLeases(ctx context.Context) error {
	activeWorkers, err := l.lr.ListActiveWorkers(ctx, l.tenantId)

	if err != nil {
		return err
	}

	currResourceIdsToLease := make(map[string]*sqlcv1.Lease, len(l.workerLeases))

	for _, lease := range l.workerLeases {
		currResourceIdsToLease[lease.ResourceId] = lease
	}

	workerIdsStr := make([]string, 0, len(activeWorkers))
	activeWorkerIdsToResults := make(map[string]*v1.ListActiveWorkersResult, len(activeWorkers))

	leasesToExtend := make([]*sqlcv1.Lease, 0, len(activeWorkers))
	leasesToRelease := make([]*sqlcv1.Lease, 0, len(currResourceIdsToLease))

	// Track existing valid leases that don't need extension
	existingValidLeases := make([]*sqlcv1.Lease, 0, len(activeWorkers))

	for _, activeWorker := range activeWorkers {
		activeWorkerIdsToResults[activeWorker.ID] = activeWorker

		if lease, ok := currResourceIdsToLease[activeWorker.ID]; ok {
			// only extend leases that are about to expire
			if shouldExtendLease(lease) {
				leasesToExtend = append(leasesToExtend, lease)
				workerIdsStr = append(workerIdsStr, activeWorker.ID)
			} else {
				// This is a valid lease that doesn't need extension yet
				existingValidLeases = append(existingValidLeases, lease)
			}
			delete(currResourceIdsToLease, activeWorker.ID)
			continue
		}

		workerIdsStr = append(workerIdsStr, activeWorker.ID)
	}

	for _, lease := range currResourceIdsToLease {
		leasesToRelease = append(leasesToRelease, lease)
	}

	successfullyAcquiredWorkerIds := make([]*v1.ListActiveWorkersResult, 0, len(activeWorkers))

	allWorkerLeases := make([]*sqlcv1.Lease, 0, len(activeWorkers))

	// First, add existing valid leases that don't need extension
	allWorkerLeases = append(allWorkerLeases, existingValidLeases...)
	for _, lease := range existingValidLeases {
		successfullyAcquiredWorkerIds = append(successfullyAcquiredWorkerIds, activeWorkerIdsToResults[lease.ResourceId])
	}

	// Then, add newly acquired/extended leases
	if len(workerIdsStr) != 0 {
		workerLeases, err := l.lr.AcquireOrExtendLeases(ctx, l.tenantId, sqlcv1.LeaseKindWORKER, workerIdsStr, leasesToExtend)

		if err != nil {
			return err
		}

		allWorkerLeases = append(allWorkerLeases, workerLeases...)

		for _, lease := range workerLeases {
			successfullyAcquiredWorkerIds = append(successfullyAcquiredWorkerIds, activeWorkerIdsToResults[lease.ResourceId])
		}
	}

	l.workerLeases = allWorkerLeases

	l.sendWorkerIds(successfullyAcquiredWorkerIds)

	if len(leasesToRelease) != 0 {
		if err := l.lr.ReleaseLeases(ctx, l.tenantId, leasesToRelease); err != nil {
			return err
		}
	}

	return nil
}

func (l *LeaseManager) acquireQueueLeases(ctx context.Context) error {
	queues, err := l.lr.ListQueues(ctx, l.tenantId)

	if err != nil {
		return err
	}

	currResourceIdsToLease := make(map[string]*sqlcv1.Lease, len(l.queueLeases))

	for _, lease := range l.queueLeases {
		currResourceIdsToLease[lease.ResourceId] = lease
	}

	queueIdsStr := make([]string, 0, len(queues))
	leasesToExtend := make([]*sqlcv1.Lease, 0, len(queues))
	leasesToRelease := make([]*sqlcv1.Lease, 0, len(currResourceIdsToLease))

	// Track existing valid leases that don't need extension
	existingValidLeases := make([]*sqlcv1.Lease, 0, len(queues))

	for _, q := range queues {
		if lease, ok := currResourceIdsToLease[q.Name]; ok {
			// only extend leases that are about to expire
			if shouldExtendLease(lease) {
				leasesToExtend = append(leasesToExtend, lease)
				queueIdsStr = append(queueIdsStr, q.Name)
			} else {
				// This is a valid lease that doesn't need extension yet
				existingValidLeases = append(existingValidLeases, lease)
			}
			delete(currResourceIdsToLease, q.Name)
			continue
		}

		queueIdsStr = append(queueIdsStr, q.Name)
	}

	for _, lease := range currResourceIdsToLease {
		leasesToRelease = append(leasesToRelease, lease)
	}

	successfullyAcquiredQueues := []string{}
	allQueueLeases := make([]*sqlcv1.Lease, 0, len(queues))

	// First, add existing valid leases that don't need extension
	allQueueLeases = append(allQueueLeases, existingValidLeases...)
	for _, lease := range existingValidLeases {
		successfullyAcquiredQueues = append(successfullyAcquiredQueues, lease.ResourceId)
	}

	// Then, add newly acquired/extended leases
	if len(queueIdsStr) != 0 {

		queueLeases, err := l.lr.AcquireOrExtendLeases(ctx, l.tenantId, sqlcv1.LeaseKindQUEUE, queueIdsStr, leasesToExtend)

		if err != nil {
			return err
		}

		allQueueLeases = append(allQueueLeases, queueLeases...)

		for _, lease := range queueLeases {
			successfullyAcquiredQueues = append(successfullyAcquiredQueues, lease.ResourceId)
		}
	}

	l.queueLeases = allQueueLeases

	l.sendQueues(successfullyAcquiredQueues)

	if len(leasesToRelease) != 0 {
		if err := l.lr.ReleaseLeases(ctx, l.tenantId, leasesToRelease); err != nil {
			return err
		}
	}

	return nil
}

func (l *LeaseManager) acquireConcurrencyLeases(ctx context.Context) error {
	strats, err := l.lr.ListConcurrencyStrategies(ctx, l.tenantId)

	if err != nil {
		return err
	}

	currResourceIdsToLease := make(map[string]*sqlcv1.Lease, len(l.concurrencyLeases))

	for _, lease := range l.concurrencyLeases {
		currResourceIdsToLease[lease.ResourceId] = lease
	}

	strategyIdsStr := make([]string, 0, len(strats))
	activeStratIdsToStrategies := make(map[string]*sqlcv1.V1StepConcurrency, len(strats))

	leasesToExtend := make([]*sqlcv1.Lease, 0, len(strats))
	leasesToRelease := make([]*sqlcv1.Lease, 0, len(currResourceIdsToLease))

	// Track existing valid leases that don't need extension
	existingValidLeases := make([]*sqlcv1.Lease, 0, len(strats))

	for _, s := range strats {
		strategyId := fmt.Sprintf("%d", s.ID)
		activeStratIdsToStrategies[strategyId] = s

		if lease, ok := currResourceIdsToLease[strategyId]; ok {
			// only extend leases that are about to expire
			if shouldExtendLease(lease) {
				leasesToExtend = append(leasesToExtend, lease)
				strategyIdsStr = append(strategyIdsStr, strategyId)
			} else {
				// This is a valid lease that doesn't need extension yet
				existingValidLeases = append(existingValidLeases, lease)
			}
			delete(currResourceIdsToLease, strategyId)
			continue
		}

		strategyIdsStr = append(strategyIdsStr, strategyId)
	}

	for _, lease := range currResourceIdsToLease {
		leasesToRelease = append(leasesToRelease, lease)
	}

	successfullyAcquiredStrats := []*sqlcv1.V1StepConcurrency{}
	allConcurrencyLeases := make([]*sqlcv1.Lease, 0, len(strats))

	// First, add existing valid leases that don't need extension
	allConcurrencyLeases = append(allConcurrencyLeases, existingValidLeases...)
	for _, lease := range existingValidLeases {
		successfullyAcquiredStrats = append(successfullyAcquiredStrats, activeStratIdsToStrategies[lease.ResourceId])
	}

	// Then, add newly acquired/extended leases
	if len(strategyIdsStr) != 0 {

		concurrencyLeases, err := l.lr.AcquireOrExtendLeases(ctx, l.tenantId, sqlcv1.LeaseKindCONCURRENCYSTRATEGY, strategyIdsStr, leasesToExtend)

		if err != nil {
			return err
		}

		allConcurrencyLeases = append(allConcurrencyLeases, concurrencyLeases...)

		for _, lease := range concurrencyLeases {
			successfullyAcquiredStrats = append(successfullyAcquiredStrats, activeStratIdsToStrategies[lease.ResourceId])
		}
	}

	l.concurrencyLeases = allConcurrencyLeases

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
			// we acquire a processMu lock here to prevent cleanup from occurring simultaneously
			l.processMu.Lock()

			// we don't want to block the cleanup process, so we use a separate context with a timeout
			loopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)

			wg := sync.WaitGroup{}

			wg.Add(3)

			go func() {
				defer wg.Done()

				if err := l.acquireWorkerLeases(loopCtx); err != nil {
					l.conf.l.Error().Err(err).Msg("error acquiring worker leases")
				}
			}()

			go func() {
				defer wg.Done()

				if err := l.acquireQueueLeases(loopCtx); err != nil {
					l.conf.l.Error().Err(err).Msg("error acquiring queue leases")
				}
			}()

			go func() {
				defer wg.Done()

				if err := l.acquireConcurrencyLeases(loopCtx); err != nil {
					l.conf.l.Error().Err(err).Msg("error acquiring concurrency leases")
				}
			}()

			wg.Wait()

			cancel()
			l.processMu.Unlock()
		}
	}
}

func (l *LeaseManager) cleanup(ctx context.Context) error {
	// we acquire a process locks here to prevent concurrent cleanup and lease acquisition
	l.processMu.Lock()
	defer l.processMu.Unlock()

	if l.cleanedUp {
		return nil
	}

	l.cleanedUp = true

	eg := errgroup.Group{}

	eg.Go(func() error {
		return l.lr.ReleaseLeases(ctx, l.tenantId, l.workerLeases)
	})

	eg.Go(func() error {
		return l.lr.ReleaseLeases(ctx, l.tenantId, l.queueLeases)
	})

	eg.Go(func() error {
		return l.lr.ReleaseLeases(ctx, l.tenantId, l.concurrencyLeases)
	})

	if err := eg.Wait(); err != nil {
		return err
	}

	// close channels: this is safe to do because each channel is guarded by l.cleanedUp + the process lock
	close(l.workersCh)
	close(l.queuesCh)
	close(l.concurrencyLeasesCh)

	return nil
}

func (l *LeaseManager) start(ctx context.Context) {
	go l.loopForLeases(ctx)
}
