package v1

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
	fmt.Printf("sendWorkerIds called with %d workers\n", len(workerIds))

	defer func() {
		if r := recover(); r != nil {
			l.conf.l.Error().Interface("recovered", r).Msg("recovered from panic")
		}
	}()

	// at this point, we have a cleanupMu lock, so it's safe to read
	if l.cleanedUp {
		fmt.Printf("LeaseManager is cleaned up, not sending workers\n")
		return
	}

	select {
	case l.workersCh <- workerIds:
		fmt.Printf("Successfully sent %d workers to channel\n", len(workerIds))
	default:
		fmt.Printf("Failed to send workers to channel (channel full)\n")
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
	fmt.Printf("acquireWorkerLeases called for tenant %s\n", l.tenantId.String())

	activeWorkers, err := l.lr.ListActiveWorkers(ctx, l.tenantId)

	if err != nil {
		fmt.Printf("Error listing active workers: %v\n", err)
		return err
	}

	fmt.Printf("ListActiveWorkers returned %d workers\n", len(activeWorkers))

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

	fmt.Printf("Sending %d successfully acquired workers\n", len(successfullyAcquiredWorkerIds))
	l.sendWorkerIds(successfullyAcquiredWorkerIds)

	if len(leasesToRelease) != 0 {
		if err := l.lr.ReleaseLeases(ctx, l.tenantId, leasesToRelease); err != nil {
			return err
		}
	}

	return nil
}

func (l *LeaseManager) acquireQueueLeases(ctx context.Context) error {
	fmt.Printf("acquireQueueLeases called for tenant %s\n", l.tenantId.String())

	queues, err := l.lr.ListQueues(ctx, l.tenantId)

	if err != nil {
		fmt.Printf("Error listing queues: %v\n", err)
		return err
	}

	fmt.Printf("ListQueues returned %d queues\n", len(queues))

	currResourceIdsToLease := make(map[string]*sqlcv1.Lease, len(l.queueLeases))

	for _, lease := range l.queueLeases {
		currResourceIdsToLease[lease.ResourceId] = lease
		fmt.Printf("Existing lease: resourceId=%s, expiresAt=%v, id=%d\n", lease.ResourceId, lease.ExpiresAt, lease.ID)
	}

	queueIdsStr := make([]string, len(queues))
	leasesToExtend := make([]*sqlcv1.Lease, 0, len(queues))
	leasesToRelease := make([]*sqlcv1.Lease, 0, len(currResourceIdsToLease))

	for i, q := range queues {
		queueIdsStr[i] = q.Name
		fmt.Printf("Queue %d: name=%s, last_active=%v\n", i, q.Name, q.LastActive)

		if lease, ok := currResourceIdsToLease[queueIdsStr[i]]; ok {
			fmt.Printf("Found existing lease for queue %s: id=%d, expiresAt=%v\n", q.Name, lease.ID, lease.ExpiresAt)
			leasesToExtend = append(leasesToExtend, lease)
			delete(currResourceIdsToLease, queueIdsStr[i])
		} else {
			fmt.Printf("No existing lease for queue %s\n", q.Name)
		}
	}

	fmt.Printf("queueIdsStr: %v\n", queueIdsStr)
	fmt.Printf("leasesToExtend: %d leases\n", len(leasesToExtend))

	for _, lease := range currResourceIdsToLease {
		leasesToRelease = append(leasesToRelease, lease)
	}

	successfullyAcquiredQueues := []string{}

	if len(queueIdsStr) != 0 {
		fmt.Printf("Attempting to acquire leases for %d queues: %v\n", len(queueIdsStr), queueIdsStr)
		fmt.Printf("leasesToExtend: %d leases, existingLeaseIds: %v\n", len(leasesToExtend), func() []int64 {
			ids := make([]int64, 0, len(leasesToExtend))
			for _, lease := range leasesToExtend {
				ids = append(ids, lease.ID)
			}
			return ids
		}())

		queueLeases, err := l.lr.AcquireOrExtendLeases(ctx, l.tenantId, sqlcv1.LeaseKindQUEUE, queueIdsStr, leasesToExtend)

		if err != nil {
			fmt.Printf("Error acquiring queue leases: %v\n", err)
			return err
		}

		fmt.Printf("AcquireOrExtendLeases returned %d leases\n", len(queueLeases))

		l.queueLeases = queueLeases

		for _, lease := range queueLeases {
			successfullyAcquiredQueues = append(successfullyAcquiredQueues, lease.ResourceId)
		}
	} else {
		fmt.Printf("No queue IDs to acquire leases for\n")
	}

	fmt.Printf("Sending %d successfully acquired queues: %v\n", len(successfullyAcquiredQueues), successfullyAcquiredQueues)
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
