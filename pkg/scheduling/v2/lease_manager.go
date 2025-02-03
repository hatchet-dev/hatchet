package v2

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

// LeaseManager is responsible for leases on multiple queues and multiplexing
// queue results to callers. It is still tenant-scoped.
type LeaseManager struct {
	lr repository.LeaseRepository

	conf *sharedConfig

	tenantId pgtype.UUID

	workerLeasesMu sync.Mutex
	workerLeases   []*dbsqlc.Lease
	workersCh      chan<- []*repository.ListActiveWorkersResult

	queueLeasesMu sync.Mutex
	queueLeases   []*dbsqlc.Lease
	queuesCh      chan<- []string

	cleanedUp bool
	cleanupMu sync.Mutex
}

func newLeaseManager(conf *sharedConfig, tenantId pgtype.UUID) (*LeaseManager, <-chan []*repository.ListActiveWorkersResult, <-chan []string) {
	workersCh := make(chan []*repository.ListActiveWorkersResult)
	queuesCh := make(chan []string)

	return &LeaseManager{
		lr:        conf.repo.Lease(),
		conf:      conf,
		tenantId:  tenantId,
		workersCh: workersCh,
		queuesCh:  queuesCh,
	}, workersCh, queuesCh
}

func (l *LeaseManager) sendWorkerIds(workerIds []*repository.ListActiveWorkersResult) {
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

func (l *LeaseManager) acquireWorkerLeases(ctx context.Context) error {
	if ok := l.workerLeasesMu.TryLock(); !ok {
		return nil
	}

	defer l.workerLeasesMu.Unlock()

	activeWorkers, err := l.lr.ListActiveWorkers(ctx, l.tenantId)

	if err != nil {
		return err
	}

	currResourceIdsToLease := make(map[string]*dbsqlc.Lease, len(l.workerLeases))

	for _, lease := range l.workerLeases {
		currResourceIdsToLease[lease.ResourceId] = lease
	}

	workerIdsStr := make([]string, len(activeWorkers))
	activeWorkerIdsToResults := make(map[string]*repository.ListActiveWorkersResult, len(activeWorkers))

	leasesToExtend := make([]*dbsqlc.Lease, 0, len(activeWorkers))
	leasesToRelease := make([]*dbsqlc.Lease, 0, len(currResourceIdsToLease))

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

	successfullyAcquiredWorkerIds := make([]*repository.ListActiveWorkersResult, 0)

	if len(workerIdsStr) != 0 {
		workerLeases, err := l.lr.AcquireOrExtendLeases(ctx, l.tenantId, dbsqlc.LeaseKindWORKER, workerIdsStr, leasesToExtend)

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

	currResourceIdsToLease := make(map[string]*dbsqlc.Lease, len(l.queueLeases))

	for _, lease := range l.queueLeases {
		currResourceIdsToLease[lease.ResourceId] = lease
	}

	queueIdsStr := make([]string, len(queues))
	leasesToExtend := make([]*dbsqlc.Lease, 0, len(queues))
	leasesToRelease := make([]*dbsqlc.Lease, 0, len(currResourceIdsToLease))

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

		queueLeases, err := l.lr.AcquireOrExtendLeases(ctx, l.tenantId, dbsqlc.LeaseKindQUEUE, queueIdsStr, leasesToExtend)

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

// loopForLeases acquires new leases every 1 second for workers and queues
func (l *LeaseManager) loopForLeases(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			wg := sync.WaitGroup{}

			wg.Add(2)

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

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}

func (l *LeaseManager) start(ctx context.Context) {
	go l.loopForLeases(ctx)
}
