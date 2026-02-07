package v1

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/google/uuid"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// LeaseManager is responsible for leases on multiple queues and multiplexing
// queue results to callers. It is still tenant-scoped.
type LeaseManager struct {
	lr v1.LeaseRepository

	conf *sharedConfig

	tenantId uuid.UUID

	workerLeasesMu sync.Mutex
	workerLeases   []*sqlcv1.Lease
	workersCh      notifierCh[*v1.ListActiveWorkersResult]

	queueLeasesMu sync.Mutex
	queueLeases   []*sqlcv1.Lease
	queuesCh      notifierCh[string]

	concurrencyLeasesMu sync.Mutex
	concurrencyLeases   []*sqlcv1.Lease
	concurrencyLeasesCh notifierCh[*sqlcv1.V1StepConcurrency]

	cleanedUp bool
	processMu sync.Mutex
}

func newLeaseManager(conf *sharedConfig, tenantId uuid.UUID) (*LeaseManager, notifierCh[*v1.ListActiveWorkersResult], notifierCh[string], notifierCh[*sqlcv1.V1StepConcurrency]) {
	workersCh := make(notifierCh[*v1.ListActiveWorkersResult])
	queuesCh := make(notifierCh[string])
	concurrencyLeasesCh := make(notifierCh[*sqlcv1.V1StepConcurrency])

	return &LeaseManager{
		lr:                  conf.repo.Lease(),
		conf:                conf,
		tenantId:            tenantId,
		workersCh:           workersCh,
		queuesCh:            queuesCh,
		concurrencyLeasesCh: concurrencyLeasesCh,
	}, workersCh, queuesCh, concurrencyLeasesCh
}

func (l *LeaseManager) sendWorkerIds(workerIds []*v1.ListActiveWorkersResult, isIncremental bool) {
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
	case l.workersCh <- notifierMsg[*v1.ListActiveWorkersResult]{
		items:         workerIds,
		isIncremental: isIncremental,
	}:
	default:
	}
}

func (l *LeaseManager) sendQueues(queues []string, isIncremental bool) {
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
	case l.queuesCh <- notifierMsg[string]{
		items:         queues,
		isIncremental: isIncremental,
	}:
	default:
	}
}

func (l *LeaseManager) sendConcurrencyLeases(concurrencyLeases []*sqlcv1.V1StepConcurrency, isIncremental bool) {
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
	case l.concurrencyLeasesCh <- notifierMsg[*sqlcv1.V1StepConcurrency]{
		items:         concurrencyLeases,
		isIncremental: isIncremental,
	}:
	default:
	}
}

func (l *LeaseManager) acquireWorkerLeases(ctx context.Context) error {
	l.workerLeasesMu.Lock()
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
		workerIdsStr[i] = activeWorker.ID.String()
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

	l.sendWorkerIds(successfullyAcquiredWorkerIds, false)

	if len(leasesToRelease) != 0 {
		if err := l.lr.ReleaseLeases(ctx, l.tenantId, leasesToRelease); err != nil {
			return err
		}
	}

	return nil
}

func (l *LeaseManager) notifyNewWorker(ctx context.Context, workerId uuid.UUID) error {
	if !l.workerLeasesMu.TryLock() {
		return nil
	}

	defer l.workerLeasesMu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	// look up the worker
	worker, err := l.lr.GetActiveWorker(ctx, l.tenantId, workerId)

	if err != nil {
		return err
	}

	// try to acquire a lease for the new worker
	lease, err := l.lr.AcquireOrExtendLeases(ctx, l.tenantId, sqlcv1.LeaseKindWORKER, []string{workerId.String()}, []*sqlcv1.Lease{})

	if err != nil {
		return err
	}

	l.workerLeases = append(l.workerLeases, lease...)

	// send the new worker to the channel
	l.sendWorkerIds([]*v1.ListActiveWorkersResult{worker}, true)

	return nil
}

func (l *LeaseManager) acquireQueueLeases(ctx context.Context) error {
	l.queueLeasesMu.Lock()
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

	l.sendQueues(successfullyAcquiredQueues, false)

	if len(leasesToRelease) != 0 {
		if err := l.lr.ReleaseLeases(ctx, l.tenantId, leasesToRelease); err != nil {
			return err
		}
	}

	return nil
}

func (l *LeaseManager) notifyNewQueue(ctx context.Context, queueName string) error {
	if !l.queueLeasesMu.TryLock() {
		return nil
	}

	defer l.queueLeasesMu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	// try to acquire a lease for the new queue
	lease, err := l.lr.AcquireOrExtendLeases(ctx, l.tenantId, sqlcv1.LeaseKindQUEUE, []string{queueName}, []*sqlcv1.Lease{})

	if err != nil {
		return err
	}

	l.queueLeases = append(l.queueLeases, lease...)

	// send the new queue to the channel
	l.sendQueues([]string{queueName}, true)

	return nil
}

func (l *LeaseManager) acquireConcurrencyLeases(ctx context.Context) error {
	l.concurrencyLeasesMu.Lock()
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

	l.sendConcurrencyLeases(successfullyAcquiredStrats, false)

	if len(leasesToRelease) != 0 {
		if err := l.lr.ReleaseLeases(ctx, l.tenantId, leasesToRelease); err != nil {
			return err
		}
	}

	return nil
}

func (l *LeaseManager) notifyNewConcurrencyStrategy(ctx context.Context, strategyId int64) error {
	if !l.concurrencyLeasesMu.TryLock() {
		return nil
	}

	defer l.concurrencyLeasesMu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	// look up the concurrency strategy
	strategy, err := l.lr.GetConcurrencyStrategy(ctx, l.tenantId, strategyId)

	if err != nil {
		return err
	}

	// try to acquire a lease for the new concurrency strategy
	lease, err := l.lr.AcquireOrExtendLeases(ctx, l.tenantId, sqlcv1.LeaseKindCONCURRENCYSTRATEGY, []string{fmt.Sprintf("%d", strategyId)}, []*sqlcv1.Lease{})

	if err != nil {
		return err
	}

	l.concurrencyLeases = append(l.concurrencyLeases, lease...)

	// send the new concurrency strategy to the channel
	l.sendConcurrencyLeases([]*sqlcv1.V1StepConcurrency{strategy}, true)

	return nil
}

// loopForLeases acquires new leases every 5 seconds for workers, queues, and concurrency strategies
func (l *LeaseManager) loopForLeases(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)

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
