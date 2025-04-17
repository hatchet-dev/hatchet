package v2

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

type Queuer struct {
	repo      v1.QueueRepository
	tenantId  pgtype.UUID
	queueName string

	l *zerolog.Logger

	s *Scheduler

	lastReplenished *time.Time

	limit int

	resultsCh chan<- *QueueResults

	notifyQueueCh chan map[string]string

	queueMu mutex

	cleanup func()

	isCleanedUp bool

	unackedMu rwMutex
	unacked   map[int64]struct{}

	unassigned   map[int64]*sqlcv1.V1QueueItem
	unassignedMu mutex
}

func newQueuer(conf *sharedConfig, tenantId pgtype.UUID, queueName string, s *Scheduler, resultsCh chan<- *QueueResults) *Queuer {
	defaultLimit := 100

	if conf.singleQueueLimit > 0 {
		defaultLimit = conf.singleQueueLimit
	}

	queueRepo := conf.repo.QueueFactory().NewQueue(tenantId, queueName)

	notifyQueueCh := make(chan map[string]string, 1)

	q := &Queuer{
		repo:          queueRepo,
		tenantId:      tenantId,
		queueName:     queueName,
		l:             conf.l,
		s:             s,
		limit:         defaultLimit,
		resultsCh:     resultsCh,
		notifyQueueCh: notifyQueueCh,
		queueMu:       newMu(conf.l),
		unackedMu:     newRWMu(conf.l),
		unacked:       make(map[int64]struct{}),
		unassigned:    make(map[int64]*sqlcv1.V1QueueItem),
		unassignedMu:  newMu(conf.l),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cleanupMu := sync.Mutex{}
	q.cleanup = func() {
		cleanupMu.Lock()
		defer cleanupMu.Unlock()

		if q.isCleanedUp {
			return
		}

		q.isCleanedUp = true
		cancel()

		queueRepo.Cleanup()
	}

	go q.loopQueue(ctx)

	return q
}

func (q *Queuer) Cleanup() {
	q.cleanup()
}

func (q *Queuer) queue(ctx context.Context) {
	if ok := q.queueMu.TryLock(); !ok {
		return
	}

	go func() {
		defer q.queueMu.Unlock()

		ctx, span := telemetry.NewSpan(ctx, "notify-queue")
		defer span.End()

		q.notifyQueueCh <- telemetry.GetCarrier(ctx)
	}()
}

func (q *Queuer) loopQueue(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)

	for {
		var carrier map[string]string

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case carrier = <-q.notifyQueueCh:
		}

		prometheus.QueueInvocations.Inc()

		ctx, span := telemetry.NewSpanWithCarrier(ctx, "queue", carrier)

		telemetry.WithAttributes(span, telemetry.AttributeKV{
			Key:   "queue",
			Value: q.queueName,
		})

		start := time.Now()
		checkpoint := start
		var err error
		qis, err := q.refillQueue(ctx)

		if err != nil {
			span.End()
			q.l.Error().Err(err).Msg("error refilling queue")
			continue
		}

		// NOTE: we don't terminate early out of this loop because calling `tryAssign` is necessary
		// for calling the scheduling extensions.

		refillTime := time.Since(checkpoint)
		checkpoint = time.Now()

		rls, err := q.repo.GetTaskRateLimits(ctx, qis)

		if err != nil {
			q.l.Error().Err(err).Msg("error getting rate limits")

			q.unackedToUnassigned(qis)
			continue
		}

		rateLimitTime := time.Since(checkpoint)
		checkpoint = time.Now()

		stepIds := make([]pgtype.UUID, 0, len(qis))

		for _, qi := range qis {
			stepIds = append(stepIds, qi.StepID)
		}

		labels, err := q.repo.GetDesiredLabels(ctx, stepIds)

		if err != nil {
			q.l.Error().Err(err).Msg("error getting desired labels")

			q.unackedToUnassigned(qis)
			continue
		}

		desiredLabelsTime := time.Since(checkpoint)
		checkpoint = time.Now()

		assignCh := q.s.tryAssign(ctx, qis, labels, rls)
		count := 0

		countMu := sync.Mutex{}
		wg := sync.WaitGroup{}

		startingQiLength := len(qis)
		processedQiLength := 0

		for r := range assignCh {
			wg.Add(1)

			// asynchronously flush to database
			go func(ar *assignResults) {
				defer wg.Done()

				startFlush := time.Now()

				numFlushed := q.flushToDatabase(ctx, ar)

				countMu.Lock()
				count += numFlushed
				processedQiLength += len(ar.assigned) + len(ar.unassigned) + len(ar.schedulingTimedOut) + len(ar.rateLimited)
				countMu.Unlock()

				if sinceStart := time.Since(startFlush); sinceStart > 100*time.Millisecond {
					q.l.Warn().Msgf("flushing items to database took longer than 100ms (%d items in %s)", numFlushed, time.Since(startFlush))
				}

				if numFlushed > 0 {
					now := time.Now()

					for _, assignedItem := range ar.assigned {
						prometheus.AssignedTasks.Inc()

						qi := assignedItem.QueueItem

						// we only track queue times if this is the first retry, because the TaskInsertedAt will
						// match the time the first queue item was created. we should eventually write an InsertedAt
						// column to the queue item table to track this better.
						if qi.RetryCount != 0 || !qi.TaskInsertedAt.Valid {
							continue
						}

						prometheus.QueuedToAssigned.Inc()

						timeInQueueSeconds := now.Sub(qi.TaskInsertedAt.Time).Seconds()

						if timeInQueueSeconds > 0 {
							prometheus.QueuedToAssignedTimeBuckets.Observe(timeInQueueSeconds)
						}
					}
				}

				for range ar.schedulingTimedOut {
					prometheus.SchedulingTimedOut.Inc()
				}

				for range ar.rateLimited {
					prometheus.RateLimited.Inc()
				}
			}(r)
		}

		assignTime := time.Since(checkpoint)
		elapsed := time.Since(start)

		if elapsed > 100*time.Millisecond {
			q.l.Warn().Dur(
				"refill_time", refillTime,
			).Dur(
				"rate_limit_time", rateLimitTime,
			).Dur(
				"desired_labels_time", desiredLabelsTime,
			).Dur(
				"assign_time", assignTime,
			).Msgf("queue %s took longer than 100ms (%s) to process %d items", q.queueName, elapsed, len(qis))
		}

		// if we processed all queue items, queue again
		prevQis := qis

		go func(originalStart time.Time) {
			wg.Wait()
			span.End()

			countMu.Lock()
			if len(prevQis) > 0 && count == len(prevQis) {
				q.queue(context.Background())
			}

			if startingQiLength != processedQiLength {
				q.l.Error().Int("starting", startingQiLength).Int("processed", processedQiLength).Msg("queue items processed mismatch")
			}

			countMu.Unlock()

			if sinceStart := time.Since(originalStart); sinceStart > 100*time.Millisecond {
				q.l.Warn().Dur(
					"duration", sinceStart,
				).Msgf("queue %s took longer than 100ms to process and flush %d items", q.queueName, len(prevQis))
			}
		}(start)
	}
}

func (q *Queuer) refillQueue(ctx context.Context) ([]*sqlcv1.V1QueueItem, error) {
	q.unackedMu.Lock()
	defer q.unackedMu.Unlock()

	q.unassignedMu.Lock()
	defer q.unassignedMu.Unlock()

	curr := make([]*sqlcv1.V1QueueItem, 0, len(q.unassigned))

	for _, qi := range q.unassigned {
		curr = append(curr, qi)
	}

	// determine whether we need to replenish with the following cases:
	// - we last replenished more than 1 second ago
	// - if we are at less than 50% of the limit, we always attempt to replenish
	replenish := false

	if len(curr) < q.limit {
		replenish = true
	} else if q.lastReplenished != nil {
		if time.Since(*q.lastReplenished) > 990*time.Millisecond {
			replenish = true
		}
	}

	if replenish {
		now := time.Now()
		q.lastReplenished = &now
		limit := 2 * q.limit

		var err error
		curr, err = q.repo.ListQueueItems(ctx, limit)

		if err != nil {
			return nil, err
		}
	}

	newCurr := make([]*sqlcv1.V1QueueItem, 0, len(curr))

	for _, qi := range curr {
		if _, ok := q.unacked[qi.ID]; !ok {
			newCurr = append(newCurr, qi)
		}
	}

	// add all newCurr to unacked so we don't assign them again
	for _, qi := range newCurr {
		q.unacked[qi.ID] = struct{}{}
	}

	sort.Slice(newCurr, func(i, j int) bool {
		if newCurr[i].Priority == newCurr[j].Priority {
			return newCurr[i].ID < newCurr[j].ID
		}
		return newCurr[i].Priority > newCurr[j].Priority
	})

	return newCurr, nil
}

type QueueResults struct {
	TenantId pgtype.UUID
	Assigned []*v1.AssignedItem

	Unassigned         []*sqlcv1.V1QueueItem
	SchedulingTimedOut []*sqlcv1.V1QueueItem
	RateLimited        []*v1.RateLimitResult
}

func (q *Queuer) ack(r *assignResults) {
	q.unackedMu.Lock()
	defer q.unackedMu.Unlock()

	q.unassignedMu.Lock()
	defer q.unassignedMu.Unlock()

	for _, assignedItem := range r.assigned {
		delete(q.unacked, assignedItem.QueueItem.ID)
		delete(q.unassigned, assignedItem.QueueItem.ID)
	}

	for _, unassignedItem := range r.unassigned {
		delete(q.unacked, unassignedItem.ID)
		q.unassigned[unassignedItem.ID] = unassignedItem
	}

	for _, schedulingTimedOutItem := range r.schedulingTimedOut {
		delete(q.unacked, schedulingTimedOutItem.ID)
		delete(q.unassigned, schedulingTimedOutItem.ID)
	}

	for _, rateLimitedItem := range r.rateLimited {
		delete(q.unacked, rateLimitedItem.qi.ID)
		q.unassigned[rateLimitedItem.qi.ID] = rateLimitedItem.qi
	}
}

func (q *Queuer) unackedToUnassigned(items []*sqlcv1.V1QueueItem) {
	q.unackedMu.Lock()
	defer q.unackedMu.Unlock()

	q.unassignedMu.Lock()
	defer q.unassignedMu.Unlock()

	for _, item := range items {
		delete(q.unacked, item.ID)
		q.unassigned[item.ID] = item
	}
}

func (q *Queuer) flushToDatabase(ctx context.Context, r *assignResults) int {
	// no matter what, we always ack the items in the queuer
	defer q.ack(r)

	ctx, span := telemetry.NewSpan(ctx, "flush-to-database")
	defer span.End()

	begin := time.Now()

	q.l.Debug().Int("assigned", len(r.assigned)).Int("unassigned", len(r.unassigned)).Int("scheduling_timed_out", len(r.schedulingTimedOut)).Msg("flushing to database")

	if len(r.assigned) == 0 && len(r.unassigned) == 0 && len(r.schedulingTimedOut) == 0 && len(r.rateLimited) == 0 {
		return 0
	}

	opts := &v1.AssignResults{
		Assigned:           make([]*v1.AssignedItem, 0, len(r.assigned)),
		Unassigned:         r.unassigned,
		SchedulingTimedOut: r.schedulingTimedOut,
		RateLimited:        make([]*v1.RateLimitResult, 0, len(r.rateLimited)),
	}

	stepRunIdsToAcks := make(map[int64]int, len(r.assigned))

	for _, assignedItem := range r.assigned {
		stepRunIdsToAcks[assignedItem.QueueItem.TaskID] = assignedItem.AckId

		opts.Assigned = append(opts.Assigned, &v1.AssignedItem{
			WorkerId:  assignedItem.WorkerId,
			QueueItem: assignedItem.QueueItem,
		})
	}

	for _, rateLimitedItem := range r.rateLimited {
		opts.RateLimited = append(opts.RateLimited, &v1.RateLimitResult{
			ExceededKey:    rateLimitedItem.exceededKey,
			ExceededUnits:  rateLimitedItem.exceededUnits,
			ExceededVal:    rateLimitedItem.exceededVal,
			TaskId:         rateLimitedItem.qi.TaskID,
			TaskInsertedAt: rateLimitedItem.qi.TaskInsertedAt,
			RetryCount:     rateLimitedItem.qi.RetryCount,
		})
	}

	succeeded, failed, err := q.repo.MarkQueueItemsProcessed(ctx, opts)

	if err != nil {
		q.l.Error().Err(err).Msg("error marking queue items processed")

		nackIds := make([]int, 0, len(r.assigned))

		for _, assignedItem := range r.assigned {
			nackIds = append(nackIds, assignedItem.AckId)
		}

		q.s.nack(nackIds)

		return 0
	}

	writeDuration := time.Since(begin)
	checkpoint := time.Now()

	nackIds := make([]int, 0, len(failed))
	ackIds := make([]int, 0, len(succeeded))

	for _, failedItem := range failed {
		nackId := stepRunIdsToAcks[failedItem.QueueItem.TaskID]
		nackIds = append(nackIds, nackId)
	}

	for _, assignedItem := range succeeded {
		ackId := stepRunIdsToAcks[assignedItem.QueueItem.TaskID]
		ackIds = append(ackIds, ackId)
	}

	q.s.nack(nackIds)

	nackDuration := time.Since(checkpoint)
	checkpoint = time.Now()

	q.s.ack(ackIds)

	ackDuration := time.Since(checkpoint)
	checkpoint = time.Now()

	q.resultsCh <- &QueueResults{
		TenantId:           q.tenantId,
		Assigned:           succeeded,
		SchedulingTimedOut: r.schedulingTimedOut,
		RateLimited:        opts.RateLimited,
		Unassigned:         r.unassigned,
	}

	chWriteDuration := time.Since(checkpoint)

	q.l.Debug().Int("succeeded", len(succeeded)).Int("failed", len(failed)).Msg("flushed to database")

	if time.Since(begin) > 100*time.Millisecond {
		q.l.Warn().Dur(
			"write_duration", writeDuration,
		).Dur(
			"nack_duration", nackDuration,
		).Dur(
			"ack_duration", ackDuration,
		).Dur(
			"ch_write_duration", chWriteDuration,
		).Msgf("flushing %d items to database took longer than 100ms", len(r.assigned)+len(r.unassigned)+len(r.schedulingTimedOut))
	}

	return len(succeeded) + len(r.schedulingTimedOut)
}

func getLargerDuration(s1, s2 string) (string, error) {
	i1, err := getDurationIndex(s1)
	if err != nil {
		return "", err
	}

	i2, err := getDurationIndex(s2)
	if err != nil {
		return "", err
	}

	if i1 > i2 {
		return s1, nil
	}

	return s2, nil
}

func getDurationIndex(s string) (int, error) {
	for i, d := range durationStrings {
		if d == s {
			return i, nil
		}
	}

	return -1, fmt.Errorf("invalid duration string: %s", s)
}

var durationStrings = []string{
	"SECOND",
	"MINUTE",
	"HOUR",
	"DAY",
	"WEEK",
	"MONTH",
	"YEAR",
}

func getWindowParamFromDurString(dur string) string {
	// validate duration string
	found := false

	for _, d := range durationStrings {
		if d == dur {
			found = true
			break
		}
	}

	if !found {
		return "MINUTE"
	}

	return fmt.Sprintf("1 %s", dur)
}
