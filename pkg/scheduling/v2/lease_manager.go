package v2

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

type ListActiveWorkersResult struct {
	ID     pgtype.UUID
	Labels []*dbsqlc.ListManyWorkerLabelsRow
}

type leaseRepo interface {
	ListQueues(ctx context.Context, tenantId pgtype.UUID) ([]*dbsqlc.Queue, error)
	ListActiveWorkers(ctx context.Context, tenantId pgtype.UUID) ([]*ListActiveWorkersResult, error)

	AcquireLeases(ctx context.Context, kind dbsqlc.LeaseKind, resourceIds []string, existingLeases []*dbsqlc.Lease) ([]*dbsqlc.Lease, error)
	ReleaseLeases(ctx context.Context, leases []*dbsqlc.Lease) error
}

type leaseDbQueries struct {
	tenantId pgtype.UUID

	queries *dbsqlc.Queries
	pool    *pgxpool.Pool

	leaseDuration pgtype.Interval

	l *zerolog.Logger
}

func newLeaseDbQueries(tenantId pgtype.UUID, queries *dbsqlc.Queries, pool *pgxpool.Pool, l *zerolog.Logger) *leaseDbQueries {
	return &leaseDbQueries{
		tenantId: tenantId,
		queries:  queries,
		pool:     pool,
		l:        l,
	}
}

func (d *leaseDbQueries) AcquireLeases(ctx context.Context, kind dbsqlc.LeaseKind, resourceIds []string, existingLeases []*dbsqlc.Lease) ([]*dbsqlc.Lease, error) {
	leaseIds := make([]int64, len(existingLeases))

	for i, lease := range existingLeases {
		leaseIds[i] = lease.ID
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, d.pool, d.l, 5000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	err = d.queries.GetLeasesToAcquire(ctx, tx, dbsqlc.GetLeasesToAcquireParams{
		Kind:        kind,
		Resourceids: resourceIds,
		Tenantid:    d.tenantId,
	})

	if err != nil {
		return nil, err
	}

	leases, err := d.queries.AcquireLeases(ctx, tx, dbsqlc.AcquireLeasesParams{
		Kind:             kind,
		LeaseDuration:    d.leaseDuration,
		Resourceids:      resourceIds,
		Tenantid:         d.tenantId,
		Existingleaseids: leaseIds,
	})

	if err != nil {
		return nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return leases, nil
}

func (d *leaseDbQueries) ReleaseLeases(ctx context.Context, leases []*dbsqlc.Lease) error {
	leaseIds := make([]int64, len(leases))

	for i, lease := range leases {
		leaseIds[i] = lease.ID
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, d.pool, d.l, 5000)

	if err != nil {
		return err
	}

	defer rollback()

	_, err = d.queries.ReleaseLeases(ctx, tx, leaseIds)

	if err != nil {
		return err
	}

	if err := commit(ctx); err != nil {
		return err
	}

	return nil
}

func (d *leaseDbQueries) ListQueues(ctx context.Context, tenantId pgtype.UUID) ([]*dbsqlc.Queue, error) {
	return d.queries.ListQueues(ctx, d.pool, tenantId)
}

func (d *leaseDbQueries) ListActiveWorkers(ctx context.Context, tenantId pgtype.UUID) ([]*ListActiveWorkersResult, error) {
	activeWorkers, err := d.queries.ListActiveWorkers(ctx, d.pool, tenantId)

	if err != nil {
		return nil, err
	}

	workerIds := make([]pgtype.UUID, 0, len(activeWorkers))

	for _, worker := range activeWorkers {
		workerIds = append(workerIds, worker.ID)
	}

	labels, err := d.queries.ListManyWorkerLabels(ctx, d.pool, workerIds)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	workerIdsToLabels := make(map[string][]*dbsqlc.ListManyWorkerLabelsRow, len(labels))

	for _, label := range labels {
		wId := sqlchelpers.UUIDToStr(label.WorkerId)

		if _, ok := workerIdsToLabels[wId]; !ok {
			workerIdsToLabels[wId] = make([]*dbsqlc.ListManyWorkerLabelsRow, 0)
		}

		workerIdsToLabels[wId] = append(workerIdsToLabels[wId], label)
	}

	res := make([]*ListActiveWorkersResult, 0, len(activeWorkers))

	for _, worker := range activeWorkers {
		wId := sqlchelpers.UUIDToStr(worker.ID)
		res = append(res, &ListActiveWorkersResult{
			ID:     worker.ID,
			Labels: workerIdsToLabels[wId],
		})
	}

	return res, nil
}

// LeaseManager is responsible for leases on multiple queues and multiplexing
// queue results to callers. It is still tenant-scoped.
type LeaseManager struct {
	lr leaseRepo

	conf *sharedConfig

	tenantId pgtype.UUID

	workerLeasesMu sync.Mutex
	workerLeases   []*dbsqlc.Lease
	workersCh      chan<- []*ListActiveWorkersResult

	queueLeasesMu sync.Mutex
	queueLeases   []*dbsqlc.Lease
	queuesCh      chan<- []string
}

func newLeaseManager(conf *sharedConfig, tenantId pgtype.UUID) (*LeaseManager, <-chan []*ListActiveWorkersResult, <-chan []string) {
	workersCh := make(chan []*ListActiveWorkersResult)
	queuesCh := make(chan []string)

	return &LeaseManager{
		lr:        newLeaseDbQueries(tenantId, conf.queries, conf.pool, conf.l),
		conf:      conf,
		tenantId:  tenantId,
		workersCh: workersCh,
		queuesCh:  queuesCh,
	}, workersCh, queuesCh
}

func (l *LeaseManager) sendWorkerIds(workerIds []*ListActiveWorkersResult) {
	select {
	case l.workersCh <- workerIds:
	default:
	}
}

func (l *LeaseManager) sendQueues(queues []string) {
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

	workerIdsStr := make([]string, len(activeWorkers))
	activeWorkerIdsToResults := make(map[string]*ListActiveWorkersResult, len(activeWorkers))

	for i, activeWorker := range activeWorkers {
		aw := activeWorker
		workerIdsStr[i] = sqlchelpers.UUIDToStr(activeWorker.ID)
		activeWorkerIdsToResults[workerIdsStr[i]] = aw
	}

	successfullyAcquiredWorkerIds := make([]*ListActiveWorkersResult, 0)

	if len(workerIdsStr) != 0 {
		workerLeases, err := l.lr.AcquireLeases(ctx, dbsqlc.LeaseKindWORKER, workerIdsStr, l.workerLeases)

		if err != nil {
			return err
		}

		l.workerLeases = workerLeases

		for _, lease := range workerLeases {
			successfullyAcquiredWorkerIds = append(successfullyAcquiredWorkerIds, activeWorkerIdsToResults[lease.ResourceId])
		}
	}

	l.sendWorkerIds(successfullyAcquiredWorkerIds)

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

	queueIdsStr := make([]string, len(queues))

	for i, q := range queues {
		queueIdsStr[i] = q.Name
	}

	successfullyAcquiredQueues := []string{}

	if len(queueIdsStr) != 0 {
		queueLeases, err := l.lr.AcquireLeases(ctx, dbsqlc.LeaseKindQUEUE, queueIdsStr, l.queueLeases)

		if err != nil {
			return err
		}

		l.queueLeases = queueLeases

		for _, lease := range queueLeases {
			successfullyAcquiredQueues = append(successfullyAcquiredQueues, lease.ResourceId)
		}
	}

	l.sendQueues(successfullyAcquiredQueues)

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
	// close channels
	defer close(l.workersCh)
	defer close(l.queuesCh)

	eg := errgroup.Group{}

	eg.Go(func() error {
		l.workerLeasesMu.Lock()
		defer l.workerLeasesMu.Unlock()

		return l.lr.ReleaseLeases(ctx, l.workerLeases)
	})

	eg.Go(func() error {
		l.queueLeasesMu.Lock()
		defer l.queueLeasesMu.Unlock()

		return l.lr.ReleaseLeases(ctx, l.queueLeases)
	})

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}

func (l *LeaseManager) start(ctx context.Context) {
	go l.loopForLeases(ctx)
}
