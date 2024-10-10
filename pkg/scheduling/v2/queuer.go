package v2

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/sasha-s/go-deadlock"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

type queueRepo interface {
	ListQueues(ctx context.Context, tenantId pgtype.UUID) ([]*dbsqlc.Queue, error)
}

type queueDbQueries struct {
	queries *dbsqlc.Queries
	pool    *pgxpool.Pool
	l       *zerolog.Logger
}

func newQueueDbQueries(queries *dbsqlc.Queries, pool *pgxpool.Pool, l *zerolog.Logger) *queueDbQueries {
	return &queueDbQueries{
		queries: queries,
		pool:    pool,
		l:       l,
	}
}

func (d *queueDbQueries) ListQueues(ctx context.Context, tenantId pgtype.UUID) ([]*dbsqlc.Queue, error) {
	return d.queries.ListQueues(ctx, d.pool, tenantId)
}

type queueItemRepo interface {
	ListQueueItems(ctx context.Context) ([]*dbsqlc.QueueItem, error)
	MarkQueueItemsProcessed(ctx context.Context, r *assignResults) (succeeded []*AssignedQueueItem, failed []*AssignedQueueItem, err error)
}

type queueItemDbQueries struct {
	tenantId  pgtype.UUID
	queueName string

	queries *dbsqlc.Queries
	pool    *pgxpool.Pool
	l       *zerolog.Logger

	limit  pgtype.Int4
	gtId   pgtype.Int8
	gtIdMu deadlock.Mutex
}

func newQueueItemDbQueries(cf *sharedConfig, tenantId pgtype.UUID, queueName string, limit int32) *queueItemDbQueries {
	return &queueItemDbQueries{
		tenantId:  tenantId,
		queueName: queueName,
		queries:   cf.queries,
		pool:      cf.pool,
		l:         cf.l,
		limit: pgtype.Int4{
			Int32: limit,
			Valid: true,
		},
	}
}

func (d *queueItemDbQueries) ListQueueItems(ctx context.Context) ([]*dbsqlc.QueueItem, error) {
	return d.queries.ListQueueItemsForQueue(ctx, d.pool, dbsqlc.ListQueueItemsForQueueParams{
		Tenantid: d.tenantId,
		Queue:    d.queueName,
		GtId:     d.gtId,
		Limit:    d.limit,
	})
}

func (d *queueItemDbQueries) MarkQueueItemsProcessed(ctx context.Context, r *assignResults) (
	succeeded []*AssignedQueueItem, failed []*AssignedQueueItem, err error,
) {
	start := time.Now()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, d.pool, d.l, 5000)

	if err != nil {
		return nil, nil, err
	}

	defer rollback()

	timeAfterPrepare := time.Since(start)

	// d.queries.UpdateStepRunsToAssigned
	idsToUnqueue := make([]int64, len(r.assigned))
	stepRunIds := make([]pgtype.UUID, len(r.assigned))
	workerIds := make([]pgtype.UUID, len(r.assigned))
	stepTimeouts := make([]string, len(r.assigned))

	for i, assignedItem := range r.assigned {
		// d.gtIdMu.Lock()

		// if !d.gtId.Valid {
		// 	d.gtId = pgtype.Int8{
		// 		Int64: assignedItem.QueueItem.ID,
		// 		Valid: true,
		// 	}
		// } else if assignedItem.QueueItem.ID > d.gtId.Int64 {
		// 	d.gtId = pgtype.Int8{
		// 		Int64: assignedItem.QueueItem.ID,
		// 		Valid: true,
		// 	}
		// }

		// d.gtIdMu.Unlock()

		idsToUnqueue[i] = assignedItem.QueueItem.ID
		stepRunIds[i] = assignedItem.QueueItem.StepRunId
		workerIds[i] = assignedItem.WorkerId
		stepTimeouts[i] = assignedItem.QueueItem.StepTimeout.String
	}

	for _, id := range r.schedulingTimedOut {
		idsToUnqueue = append(idsToUnqueue, id.ID)

		// TODO: WRITE AN ERROR MESSAGE AND ENQUEUE A STATUS UPDATE
	}

	// TODO: ADD UNIQUE CONSTRAINT TO SEMAPHORES WITH ON CONFLICT DO NOTHING, THEN DON'T
	// QUEUE ITEMS THAT ALREADY HAVE SEMAPHORES
	err = d.queries.UpdateStepRunsToAssigned(ctx, tx, dbsqlc.UpdateStepRunsToAssignedParams{
		Steprunids:      stepRunIds,
		Workerids:       workerIds,
		Stepruntimeouts: stepTimeouts,
		Tenantid:        d.tenantId,
	})

	if err != nil {
		return nil, nil, err
	}

	timeAfterUpdateStepRuns := time.Since(start)

	err = d.queries.BulkQueueItems(ctx, tx, idsToUnqueue)

	if err != nil {
		return nil, nil, err
	}

	timeAfterBulkQueueItems := time.Since(start)

	dispatcherIdWorkerIds, err := d.queries.ListDispatcherIdsForWorkers(ctx, tx, dbsqlc.ListDispatcherIdsForWorkersParams{
		Tenantid:  d.tenantId,
		Workerids: sqlchelpers.UniqueSet(workerIds),
	})

	if err := commit(ctx); err != nil {
		return nil, nil, err
	}

	workerIdToDispatcherId := make(map[string]pgtype.UUID, len(dispatcherIdWorkerIds))

	for _, dispatcherIdWorkerId := range dispatcherIdWorkerIds {
		workerIdToDispatcherId[sqlchelpers.UUIDToStr(dispatcherIdWorkerId.WorkerId)] = dispatcherIdWorkerId.DispatcherId
	}

	succeeded = make([]*AssignedQueueItem, 0, len(r.assigned))
	failed = make([]*AssignedQueueItem, 0, len(r.assigned))

	for _, assignedItem := range r.assigned {
		dispatcherId, ok := workerIdToDispatcherId[sqlchelpers.UUIDToStr(assignedItem.WorkerId)]

		if !ok {
			failed = append(failed, assignedItem)
			continue
		}

		assignedItem.DispatcherId = &dispatcherId
		succeeded = append(succeeded, assignedItem)
	}

	d.l.Warn().Msgf("marking queue items processed took %s (prepare=%s, update=%s, bulkqueue=%s)", time.Since(start), timeAfterPrepare, timeAfterUpdateStepRuns, timeAfterBulkQueueItems)

	return succeeded, failed, nil
}

type Queuer struct {
	repo      queueItemRepo
	tenantId  pgtype.UUID
	queueName string

	l *zerolog.Logger

	s *Scheduler

	lastReplenished *time.Time
	qis             []*queueItem
	qiMu            deadlock.Mutex

	// unackedQis is a list of queue items that have been unqueued but have not been
	// flushed to the database yet
	unackedQis map[int64]*queueItem
	unackedMu  deadlock.Mutex

	limit int

	resultsCh chan<- *QueueResults

	notifyQueueCh chan struct{}

	queueMu deadlock.Mutex

	cleanup func()
}

func newQueuer(conf *sharedConfig, tenantId pgtype.UUID, queueName string, s *Scheduler, resultsCh chan<- *QueueResults) *Queuer {
	defaultLimit := 100

	if conf.singleQueueLimit > 0 {
		defaultLimit = int(conf.singleQueueLimit)
	}

	repo := newQueueItemDbQueries(conf, tenantId, queueName, int32(defaultLimit))

	notifyQueueCh := make(chan struct{})

	q := &Queuer{
		repo:          repo,
		tenantId:      tenantId,
		queueName:     queueName,
		l:             conf.l,
		s:             s,
		limit:         100,
		resultsCh:     resultsCh,
		unackedQis:    make(map[int64]*queueItem),
		notifyQueueCh: notifyQueueCh,
	}

	ctx, cancel := context.WithCancel(context.Background())
	q.cleanup = cancel

	go q.loopQueue(ctx)

	return q
}

func (q *Queuer) Cleanup() {
	q.cleanup()
}

func (q *Queuer) activeQisLen() int {
	res := 0

	for _, qi := range q.qis {
		if qi.active() {
			res++
		}
	}

	return res
}

func (q *Queuer) ack(id int64) {
	delete(q.unackedQis, id)
}

func (q *Queuer) queue() {
	if ok := q.queueMu.TryLock(); !ok {
		return
	}

	go func() {
		defer q.queueMu.Unlock()

		q.notifyQueueCh <- struct{}{}
	}()
}

func (q *Queuer) loopQueue(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	qis := make([]*dbsqlc.QueueItem, 0)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-q.notifyQueueCh:
		}

		start := time.Now()

		qis, err := q.refillQueue(ctx, qis)

		if err != nil {
			q.l.Error().Err(err).Msg("error refilling queue")
			continue
		}

		timeToRefill := time.Since(start)

		assignCh := q.s.tryAssign(ctx, qis)
		count := 0
		countMu := sync.Mutex{}

		wg := sync.WaitGroup{}

		for r := range assignCh {
			wg.Add(1)

			go func() {
				defer wg.Done()

				startFlush := time.Now()

				numFlushed := q.flushToDatabase(ctx, r)

				countMu.Lock()
				count += numFlushed
				countMu.Unlock()

				q.l.Warn().Msgf("flushed %d items in %s", numFlushed, time.Since(startFlush))
			}()
		}

		wg.Wait()

		elapsed := time.Since(start)

		q.l.Warn().Msgf("queue %s took %s to process %d items (time to refill %s)", q.queueName, elapsed, len(qis), timeToRefill)

		// if we processed all queue items, queue again
		if len(qis) > 0 && count == len(qis) {
			go q.queue()
		}
	}
}

func (q *Queuer) refillQueue(ctx context.Context, curr []*dbsqlc.QueueItem) ([]*dbsqlc.QueueItem, error) {
	// determine whether we need to replenish with the following cases:
	// - we last replenished more than 1 second ago
	// - if we are at less than 50% of the limit, we always attempt to replenish
	replenish := false
	now := time.Now()

	// q.unackedMu.Lock()
	// defer q.unackedMu.Unlock()

	if len(curr) < q.limit/2 {
		// fmt.Println("REPLENISHING BECAUSE LENGTH IS", len(curr))
		replenish = true
	}

	if q.lastReplenished != nil {
		// fmt.Println("REPLENISHING BECAUSE LAST REPLENISHED")

		if time.Since(*q.lastReplenished) > 990*time.Millisecond {
			replenish = true
		}
	}

	if !replenish {
		return curr, nil
	}

	q.lastReplenished = &now

	return q.repo.ListQueueItems(ctx)
}

// func (q *Queuer) replenish(ctx context.Context) error {
// 	now := time.Now()

// 	if ok := q.replenishMu.TryLock(); !ok {
// 		return nil
// 	}

// 	defer func() {
// 		q.replenishMu.Unlock()
// 	}()

// 	// determine whether we need to replenish with the following cases:
// 	// - mustReplenish is true
// 	// - we last replenished more than 1 second ago
// 	// - if we are at less than 50% of the limit, we always attempt to replenish
// 	replenish := false

// 	q.unackedMu.Lock()
// 	defer q.unackedMu.Unlock()

// 	// fmt.Println("ACTIVE QIS LEN", q.activeQisLen(), q.queueName)
// 	// fmt.Println("UNACKED QIS LEN", len(q.unackedQis), q.queueName)

// 	if q.activeQisLen() < q.limit/2 {
// 		replenish = true
// 	}

// 	if q.lastReplenished != nil {
// 		if time.Since(*q.lastReplenished) > 990*time.Millisecond {
// 			replenish = true
// 		}
// 	}

// 	if !replenish {
// 		return nil
// 	}

// 	// before we read from the database, we need to make sure there are no operations that can
// 	// change the state of the queue items currently in progress
// 	q.qiMu.Lock()
// 	defer q.qiMu.Unlock()

// 	qis, err := q.repo.ListQueueItems(ctx)

// 	// fmt.Println("GOT QUEUE ITEMS", len(qis), q.queueName)

// 	if err != nil {
// 		return err
// 	}

// 	if len(qis) == 0 {
// 		return nil
// 	}

// 	// construct new queue items, merging the new queue items with any unacked queue items
// 	newQis := make([]*queueItem, 0, len(qis)+len(q.unackedQis))

// 	for _, qi := range qis {
// 		if unackedQi, ok := q.unackedQis[qi.ID]; ok {
// 			newQis = append(newQis, unackedQi)
// 		} else {
// 			newQis = append(newQis, &queueItem{
// 				QueueItem: qi,
// 			})
// 		}
// 	}

// 	fmt.Println("GOT", len(qis), "BUT LENGTH OF NEW QIS IS", len(newQis))

// 	q.qis = newQis

// 	q.lastReplenished = &now

// 	go q.queue(ctx)

// 	return nil
// }

type QueueResults struct {
	TenantId pgtype.UUID
	Assigned []*AssignedQueueItem
}

// func (q *Queuer) queuev0(ctx context.Context) {
// 	if ok := q.queueMu.TryLock(); !ok {
// 		return
// 	}

// 	if len(q.qis) == 0 {
// 		q.queueMu.Unlock()
// 		return
// 	}

// 	// TODO: change this mechanism to append to a set of queue items which should be removed
// 	// during the next replenish, so we don't need to lock all queue items for this long
// 	q.unackedMu.Lock()

// 	fmt.Println("ATTEMPTING TO ASSIGN QUEUE ITEMS", len(q.qis))

// 	assignCh := q.s.tryAssign(ctx, q.qis)

// 	go func() {
// 		defer q.queueMu.Unlock()
// 		defer q.unackedMu.Unlock()

// 		wg := sync.WaitGroup{}

// 		for r := range assignCh {
// 			wg.Add(1)

// 			fmt.Println("ASSIGNED", len(r.assigned))

// 			// save all unacked queue items
// 			for i := range r.assigned {
// 				qi := r.assigned[i]
// 				q.unackedQis[qi.QueueItem.ID] = qi.QueueItem
// 			}

// 			go func() {
// 				defer wg.Done()
// 				q.flushToDatabase(ctx, r)
// 			}()
// 		}

// 		wg.Wait()
// 	}()
// }

func (q *Queuer) flushToDatabase(ctx context.Context, r *assignResults) int {
	succeeded, failed, err := q.repo.MarkQueueItemsProcessed(ctx, r)

	if err != nil {
		q.l.Error().Err(err).Msg("error marking queue items processed")

		for _, assignedItem := range r.assigned {
			q.s.nack(assignedItem.AckId)
		}

		return 0
	}

	// TODO: MOVE ACKS AND NACKS INTO THE TRANSACTION??
	// TODO: ACK AND NACK QUEUE ITEMS??
	for _, failedItem := range failed {
		q.s.nack(failedItem.AckId)
		q.ack(failedItem.QueueItem.ID)
	}

	for _, assignedItem := range succeeded {
		q.s.ack(assignedItem.AckId)
		q.ack(assignedItem.QueueItem.ID)
	}

	q.resultsCh <- &QueueResults{
		TenantId: q.tenantId,
		Assigned: succeeded,
	}

	return len(succeeded)
}
