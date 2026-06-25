package concurrency

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/hatchet-dev/pgoutbox"
	outboxsqlc "github.com/hatchet-dev/pgoutbox/sqlc"
	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/internal/queueutils"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"

	"github.com/rs/zerolog"
)

const (
	minBackoffDuration = 100 * time.Millisecond
	maxBackoffDuration = 10 * time.Second
)

type ConcurrencyStrategy struct {
	outbox pgoutbox.Outbox
	repo   repository.ConcurrencyRepository
	// TODO(memory): buildIndex reads every queued slot (and its schedule timeout) into memory. Empty
	// sub-queues are pruned after each batch (see pruneEmpty), so idle keys don't accumulate, but a
	// large live backlog still grows this without limit - we eventually need a bounded or evicting
	// strategy (e.g. spill to disk/db) for very high key cardinality. Not critical yet.
	subQueues      map[string]*subQueue
	strategy       *sqlcv1.V1StepConcurrency
	l              *zerolog.Logger
	compare        func(a, b slot) int
	built          chan struct{}
	topic          string
	pending        []*repository.RunConcurrencyResult
	openScopes     []*subQueue
	mu             sync.RWMutex
	pendingMu      sync.Mutex
	buildingMu     sync.Mutex
	initialQueueMu sync.Mutex
	initialQueued  bool
}

// commitScopes discards the undo log on each open sub-queue, making this batch's in-memory
// mutations permanent. Called by Run after ProcessMessages confirms the transaction committed. It
// returns the sub-queues it committed so Run can prune the ones that are now empty.
func (c *ConcurrencyStrategy) commitScopes() []*subQueue {
	committed := c.openScopes
	for _, sq := range c.openScopes {
		sq.commit()
	}
	c.openScopes = nil
	return committed
}

// rollbackScopes reverses every in-memory mutation made by this batch. Called by Run when
// ProcessMessages returns an error, meaning the outbox transaction (and our slot writes) rolled
// back and the messages will be redelivered.
func (c *ConcurrencyStrategy) rollbackScopes() {
	for _, sq := range c.openScopes {
		sq.rollback()
	}
	c.openScopes = nil
}

// appendPending records a single batch's result for the in-flight Run to collect.
func (c *ConcurrencyStrategy) appendPending(res *repository.RunConcurrencyResult) {
	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()

	c.pending = append(c.pending, res)
}

// takePending returns the results accumulated since the last call and clears the buffer.
func (c *ConcurrencyStrategy) takePending() []*repository.RunConcurrencyResult {
	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()

	pending := c.pending
	c.pending = nil

	return pending
}

// NewConcurrencyStrategy constructs a strategy index for a single (tenant, strategy) and
// registers it as the pgoutbox flusher for its topic (<tenant_id>.<strategy_id>). It kicks off
// the initial index hydration asynchronously on the provided (lifecycle) context, which must
// outlive any single Run - building can take much longer than a Run's deadline, and we must not
// abandon a partially-built index.
func NewConcurrencyStrategy(
	ctx context.Context,
	repo repository.ConcurrencyRepository,
	strategy *sqlcv1.V1StepConcurrency,
	outbox pgoutbox.Outbox,
	l *zerolog.Logger,
) *ConcurrencyStrategy {
	c := &ConcurrencyStrategy{
		subQueues: make(map[string]*subQueue),
		strategy:  strategy,
		repo:      repo,
		l:         l,
		compare:   priorityCompare,
		outbox:    outbox,
		topic:     getTopic(strategy),
		built:     make(chan struct{}),
	}

	outbox.AddFlusher(c.topic, c)

	go c.buildIndexLoop(ctx)

	return c
}

func NewNoOpFlusher(
	ctx context.Context,
	outbox pgoutbox.Outbox,
	strategy *sqlcv1.V1StepConcurrency,
	l *zerolog.Logger,
) {
	topic := getTopic(strategy)
	f := pgoutbox.NewNopFlusher()

	outbox.AddFlusher(topic, f)

	go func() {
		err := outbox.AcquireTopic(ctx, topic)

		if err != nil {
			l.Error().Err(err).Msgf("failed to acquire topic %s", topic)
		}

		for {
			if ctx.Err() != nil {
				return
			}

			_, err = outbox.ProcessMessages(ctx, topic)

			if err != nil {
				l.Error().Err(err).Msgf("failed to process messages for topic %s", topic)
			}

			time.Sleep(5 * time.Second)
		}
	}()
}

// important: this needs to be kept in sync with the triggers in v1-core.sql
func getTopic(strategy *sqlcv1.V1StepConcurrency) string {
	return fmt.Sprintf("concurrency.%s.%d", strategy.TenantID, strategy.ID)
}

// buildIndexLoop hydrates the in-memory index, retrying with backoff until it succeeds or the
// lifecycle context is cancelled (strategy teardown). On success it closes c.built to unblock Run.
func (c *ConcurrencyStrategy) buildIndexLoop(ctx context.Context) {
	retryCount := 0

	for {
		if ctx.Err() != nil {
			return
		}

		err := c.buildIndex(ctx)
		if err == nil {
			close(c.built)
			return
		}

		c.l.Error().Err(err).Msgf("failed to build concurrency index for topic %s, retrying", c.topic)

		queueutils.SleepWithExponentialBackoff(minBackoffDuration, maxBackoffDuration, retryCount)
		retryCount++
	}
}

// Run drains the strategy's outbox topic, replaying every WAL message into the in-memory
// index and flushing the resulting slot decisions to the database. It returns the merged
// *repository.RunConcurrencyResult across all batches processed this tick.
func (c *ConcurrencyStrategy) Run(ctx context.Context) (*repository.RunConcurrencyResult, error) {
	ctx, span := telemetry.NewSpan(ctx, "concurrency-strategy-run")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "concurrency.strategy.id", Value: c.strategy.ID},
		telemetry.AttributeKV{Key: "tenant.id", Value: c.strategy.TenantID},
	)

	// wait for the initial (async) index build to complete before processing WAL messages. If the
	// caller's context expires first, surface a clear error rather than running against an
	// incomplete index - the build keeps going on its own lifecycle context and a later Run will
	// proceed once it's ready.
	select {
	case <-c.built:
	case <-ctx.Done():
		return nil, fmt.Errorf("timed out waiting for concurrency index to finish building for topic %s: %w", c.topic, ctx.Err())
	}

	// discard any results left over from a previous aborted Run before we start draining
	c.takePending()

	// Before draining the WAL, run the one-time post-build queueing pass. buildIndex hydrates the
	// current DB state into the sub-queues but never runs the decide step over it, so queued backlog
	// loaded at build time would otherwise sit unqueued until a new WAL message happened to touch its
	// sub-queue. This must happen before we process any WAL messages.
	initialResult, err := c.runInitialQueueing(ctx)

	if err != nil {
		return nil, err
	}

	for {
		msgs, err := c.outbox.ProcessMessages(ctx, c.topic)

		if err != nil {
			// the outbox transaction rolled back (flush, message-delete, or commit failure), so undo
			// this batch's in-memory mutations to keep the index consistent with the database. the
			// messages are not deleted and will be redelivered on a later Run.
			c.rollbackScopes()
			return nil, fmt.Errorf("failed to process outbox messages for topic %s: %w", c.topic, err)
		}

		// ProcessMessages only returns without error once the transaction has committed, so the
		// in-memory mutations are now durable - discard the undo log and prune any sub-queue this
		// batch emptied (its slots were all deleted/cancelled), keeping the index from accumulating
		// idle keys.
		c.pruneEmpty(c.commitScopes())

		// no more messages queued for this topic; we've drained it
		if len(msgs) == 0 {
			break
		}
	}

	return mergeResults(append(c.takePending(), initialResult)), nil
}

// runInitialQueueing runs the post-build queueing pass exactly once. It is idempotent across Runs:
// the pass only marks itself done on success, so a transient failure (e.g. the flush transaction
// rolling back) leaves it to retry on the next Run rather than silently skipping queued backlog.
func (c *ConcurrencyStrategy) runInitialQueueing(ctx context.Context) (*repository.RunConcurrencyResult, error) {
	c.initialQueueMu.Lock()
	defer c.initialQueueMu.Unlock()

	if c.initialQueued {
		return nil, nil
	}

	res, err := c.queueAllSubQueues(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to run initial concurrency queueing for topic %s: %w", c.topic, err)
	}

	c.initialQueued = true

	return res, nil
}

// queueAllSubQueues runs the decide step over every sub-queue hydrated by buildIndex and flushes the
// result in its own transaction. The WAL path rides along with the transaction pgoutbox uses to
// delete messages, but this post-build pass has no outbox message to attach to, so it manages its
// own transaction (and finalizes the undo scopes inline rather than handing them to Run).
func (c *ConcurrencyStrategy) queueAllSubQueues(ctx context.Context) (*repository.RunConcurrencyResult, error) {
	ctx, span := telemetry.NewSpan(ctx, "concurrency-initial-queueing")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "concurrency.strategy.id", Value: c.strategy.ID},
		telemetry.AttributeKV{Key: "tenant.id", Value: c.strategy.TenantID},
	)

	// hold buildingMu so we don't queue against a half-built index (same guard as processWALMessages)
	c.buildingMu.Lock()
	defer c.buildingMu.Unlock()

	// process every sub-queue with an empty WAL: the decide step runs against the hydrated slots
	// alone, promoting queued backlog into free capacity and (for cancel strategies) cancelling slots
	// that don't fit.
	c.mu.RLock()
	grouped := make(map[string][]walMessage, len(c.subQueues))
	for key := range c.subQueues {
		grouped[key] = nil
	}
	c.mu.RUnlock()

	// nothing hydrated: skip the (otherwise empty) flush transaction entirely.
	if len(grouped) == 0 {
		return &repository.RunConcurrencyResult{}, nil
	}

	now := time.Now().UTC()

	touched, slotsToSetFilled, slotsToDelete, slotsToTimeout := c.decideSubQueues(ctx, grouped, now, c.decide())

	tasksToSetFilled, cancelledSlots := buildSlotInputs(slotsToSetFilled, slotsToDelete, slotsToTimeout)

	res, err := c.repo.UpdateConcurrencySlots(ctx, c.strategy.TenantID, c.strategy.ID, tasksToSetFilled, cancelledSlots)

	if err != nil {
		// the transaction rolled back, so undo the in-memory mutations to keep the index consistent
		// with the database; the pass will retry on the next Run.
		for _, sq := range touched {
			sq.rollback()
		}
		return nil, err
	}

	for _, sq := range touched {
		sq.commit()
	}

	// drop any sub-queue this pass emptied (e.g. all slots cancelled), same as the WAL path.
	c.pruneEmpty(touched)

	return res, nil
}

// pruneEmpty removes sub-queues that hold no running or queued slots. Only the sub-queues mutated by
// the just-committed batch are passed in, so this stays O(touched) rather than scanning every
// sub-queue each Run. It must be called only after the batch's undo scope has been committed:
// pruning a sub-queue whose mutations later roll back would drop live state.
func (c *ConcurrencyStrategy) pruneEmpty(candidates []*subQueue) {
	if len(candidates) == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, sq := range candidates {
		if sq.running.len() != 0 || sq.queued.len() != 0 {
			continue
		}

		// guard against pointer identity drift: only delete if the map still holds this exact
		// sub-queue under its key (it can't be replaced without concurrency today, but this keeps the
		// prune safe if that ever changes).
		if existing, ok := c.subQueues[sq.key]; ok && existing == sq {
			delete(c.subQueues, sq.key)
		}
	}
}

// Flush satisfies the pgoutbox.Flusher interface. It runs inside the same transaction
// pgoutbox uses to acquire and delete the messages, so the slot writes performed here commit (or
// roll back) atomically with the message delete. We unmarshal the WAL payloads, replay them into
// the index, and stash the result for Run to collect. If we return an error, pgoutbox rolls the
// transaction back and the messages are redelivered on a later Run.
func (c *ConcurrencyStrategy) Flush(ctx pgoutbox.FlushContext, msgs []*outboxsqlc.Message) error {
	tx := ctx.Tx()

	wal := make([]walMessage, 0, len(msgs))

	for _, msg := range msgs {
		var m walMessage

		if err := json.Unmarshal(msg.Payload, &m); err != nil {
			return fmt.Errorf("failed to unmarshal wal message: %w", err)
		}

		wal = append(wal, m)
	}

	res, err := c.processWALMessages(ctx, tx, wal)

	if err != nil {
		return err
	}

	c.appendPending(res)

	return nil
}

func mergeResults(results []*repository.RunConcurrencyResult) *repository.RunConcurrencyResult {
	merged := &repository.RunConcurrencyResult{
		Queued:                    make([]repository.TaskWithQueue, 0),
		Cancelled:                 make([]repository.TaskWithCancelledReason, 0),
		NextConcurrencyStrategies: make([]int64, 0),
	}

	for _, res := range results {
		if res == nil {
			continue
		}

		merged.Queued = append(merged.Queued, res.Queued...)
		merged.Cancelled = append(merged.Cancelled, res.Cancelled...)
		merged.NextConcurrencyStrategies = append(merged.NextConcurrencyStrategies, res.NextConcurrencyStrategies...)
	}

	return merged
}

type walMessage struct {
	TaskInsertedAt      time.Time `json:"taskInsertedAt"`
	Operation           string    `json:"operation"`
	Key                 string    `json:"key"`
	TaskId              int64     `json:"taskId"`
	ScheduleTimeoutAtMs int64     `json:"scheduleTimeoutAtMs"`
	Priority            int32     `json:"priority"`
	TaskRetryCount      int32     `json:"taskRetryCount"`
}

func (c *ConcurrencyStrategy) buildIndex(ctx context.Context) error {
	ctx, span := telemetry.NewSpan(ctx, "concurrency-build-index")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "concurrency.strategy.id", Value: c.strategy.ID},
		telemetry.AttributeKV{Key: "tenant.id", Value: c.strategy.TenantID},
	)

	c.buildingMu.Lock()
	defer c.buildingMu.Unlock()

	if c.outbox != nil {
		if err := c.outbox.AcquireTopic(ctx, c.topic); err != nil {
			return err
		}
	}

	writeCh := make(chan *sqlcv1.ListConcurrencySlotsForIndexingRow, 10000)
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case row, ok := <-writeCh:
				if !ok {
					return // channel closed and drained
				}

				sq := c.getOrCreateSubQueue(row.Key)

				s := slot{
					priority:            row.Priority,
					taskId:              row.TaskID,
					taskInsertedAtNs:    row.TaskInsertedAt.Time.UnixNano(),
					taskRetryCount:      row.TaskRetryCount,
					scheduleTimeoutAtMs: row.ScheduleTimeoutAt.Time.UnixMilli(),
				}

				if row.IsFilled {
					sq.running.insert(s)
				} else {
					sq.queued.insert(s)
				}
			}
		}
	}()

	err := c.repo.ReadConcurrencySlotsForIndexing(ctx, c.strategy.TenantID, c.strategy.ID, writeCh)
	if err != nil {
		return err
	}

	close(writeCh)

	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

func (c *ConcurrencyStrategy) processWALMessages(ctx context.Context, tx pgx.Tx, messages []walMessage) (*repository.RunConcurrencyResult, error) {
	// wait until we can acquire a lock on the strategy
	// (so we don't process WAL messages while we're building the index)
	c.buildingMu.Lock()
	defer c.buildingMu.Unlock()

	return c.processStrategy(ctx, tx, messages, c.decide())
}

// decideFn runs after a sub-queue's WAL has been applied and timed-out queued slots evicted. It
// mutates the sub-queue's running/queued indexes to reflect this strategy's policy and returns the
// slots to mark filled (RUNNING, queue notified) and the slots to cancel with CONCURRENCY_LIMIT.
// Timed-out queued slots are handled by the shared pipeline and are not passed here.
type decideFn func(sq *subQueue) (toFill, toCancel []slot)

// decide selects the per-sub-queue decision function for this strategy's kind. All three fill free
// capacity from the queued backlog in priorityCompare order; they differ in what happens to the
// slots that don't fit: GROUP_ROUND_ROBIN leaves them queued, CANCEL_NEWEST cancels them (reject the
// newest arrivals, never touch running work), and CANCEL_IN_PROGRESS cancels them too but may also
// preempt a running slot when a higher-priority slot is waiting.
func (c *ConcurrencyStrategy) decide() decideFn {
	switch c.strategy.Strategy {
	case sqlcv1.V1ConcurrencyStrategyGROUPROUNDROBIN:
		return decideGroupRoundRobin
	case sqlcv1.V1ConcurrencyStrategyCANCELINPROGRESS:
		return decideCancelInProgress
	case sqlcv1.V1ConcurrencyStrategyCANCELNEWEST:
		return decideCancelNewest
	default:
		panic("unknown concurrency strategy")
	}
}

// decideGroupRoundRobin fills free capacity (maxRuns - running) from the queued backlog in comparator
// order. It never cancels in-progress work.
func decideGroupRoundRobin(sq *subQueue) (toFill, toCancel []slot) {
	toFill = sq.queued.pop(int(sq.slotsToRun()))
	for _, s := range toFill {
		sq.running.insert(s)
	}
	return toFill, nil
}

// decideCancelNewest fills free capacity from the queued backlog in comparator order, then cancels
// every remaining queued slot. It never preempts running work: once the group is at capacity, new
// arrivals are rejected (cancel-newest) rather than queued (round-robin) or allowed to evict a runner
// (cancel-in-progress). Timed-out queued slots were already evicted by popTimedOut.
func decideCancelNewest(sq *subQueue) (toFill, toCancel []slot) {
	// fill free capacity with the best queued slots; if already at/over capacity this fills nothing.
	toFill = sq.queued.pop(int(sq.slotsToRun()))
	for _, s := range toFill {
		sq.running.insert(s)
	}
	// everything that didn't fit is cancelled.
	toCancel = sq.queued.pop(sq.queued.len())
	return toFill, toCancel
}

// decideCancelInProgress reconciles a sub-queue to the best maxRuns candidates under its comparator,
// cancelling everything else - including running slots that lost their place (in-progress
// cancellation). It leans on the index ordering rather than re-sorting: the queued index pops
// best-first and the running index pops worst-first (it is built with the reversed comparator), so
// the merge is a single linear pass. Timed-out queued slots were already evicted by popTimedOut, so
// they never enter the ranking (matching the SQL candidate filter
// schedule_timeout_at >= NOW() OR is_filled = TRUE).
func decideCancelInProgress(sq *subQueue) (toFill, toCancel []slot) {
	maxRuns := int(sq.maxRuns)

	// Trim running slots beyond capacity (e.g. the index hydrated more filled slots than maxRuns, or
	// maxRuns was lowered, or maxRuns <= 0). running pops worst-first, so this drops the
	// least-preferred runners.
	for sq.running.len() > maxRuns {
		toCancel = append(toCancel, sq.running.pop(1)...)
	}

	// Merge the queued backlog (best-first) against the running set.
	for sq.queued.len() > 0 {
		cand, _ := sq.queued.peek()

		if sq.running.len() < maxRuns {
			// free capacity: promote the best queued slot to running.
			sq.queued.pop(1)
			sq.running.insert(cand)
			toFill = append(toFill, cand)
			continue
		}

		// at capacity: the best queued slot only runs if it outranks the worst runner.
		if worst, ok := sq.running.peek(); ok && sq.compare(cand, worst) < 0 {
			sq.running.pop(1) // evict the worst runner (in-progress cancellation)
			toCancel = append(toCancel, worst)
			sq.queued.pop(1)
			sq.running.insert(cand)
			toFill = append(toFill, cand)
			continue
		}

		// the best remaining queued slot does not outrank any runner (or maxRuns <= 0 left no
		// runners at all). Because queued pops best-first, no remaining queued slot can either -
		// cancel them all.
		toCancel = append(toCancel, sq.queued.pop(sq.queued.len())...)
		break
	}

	return toFill, toCancel
}

// applyWAL replays a sub-queue's WAL messages into its running/queued indexes, bringing the index in
// sync with the database. It returns the superseded/stale slots that must be cancelled with
// CONCURRENCY_LIMIT: a retry's older slot (the incoming message has a higher retry count) or a stale
// incoming slot for a task already present at an equal-or-higher retry count.
func applyWAL(sq *subQueue, msgs []walMessage) []slot {
	superseded := make([]slot, 0)

	for _, msg := range msgs {
		switch msg.Operation {
		case "INSERT":
			if currentRunningSlot, exists := sq.running.get(msg.TaskId); exists {
				// compare the current running slot's retry count with the message's retry count; the greater wins
				if currentRunningSlot.taskRetryCount < msg.TaskRetryCount {
					superseded = append(superseded, currentRunningSlot)
					sq.running.delete(msg.TaskId)
					sq.running.insert(walMessageToSlot(msg))
				} else if currentRunningSlot.taskRetryCount != msg.TaskRetryCount {
					superseded = append(superseded, walMessageToSlot(msg))
				}
			} else if currentQueuedSlot, exists := sq.queued.get(msg.TaskId); exists {
				// compare the current queued slot's retry count with the message's retry count; the greater wins
				if currentQueuedSlot.taskRetryCount < msg.TaskRetryCount {
					superseded = append(superseded, currentQueuedSlot)
					sq.queued.delete(msg.TaskId)
					sq.queued.insert(walMessageToSlot(msg))
				} else if currentQueuedSlot.taskRetryCount != msg.TaskRetryCount {
					superseded = append(superseded, walMessageToSlot(msg))
				}
			} else {
				sq.queued.insert(walMessageToSlot(msg))
			}
		case "DELETE":
			// note: since we're processing a DELETE, it's already been removed from the database, we're just
			// bringing the index in sync with the database
			if _, exists := sq.running.get(msg.TaskId); exists {
				sq.running.delete(msg.TaskId)
			} else if _, exists := sq.queued.get(msg.TaskId); exists {
				sq.queued.delete(msg.TaskId)
			}
		}
	}

	return superseded
}

// processStrategy is the shared WAL-apply -> evict-timeouts -> decide -> flush pipeline used by every
// strategy. Only the decide step differs per strategy; it is selected once per Run by decide().
func (c *ConcurrencyStrategy) processStrategy(ctx context.Context, tx pgx.Tx, msgs []walMessage, decide decideFn) (*repository.RunConcurrencyResult, error) {
	grouped := groupMessagesBySubQueue(msgs)

	// single "now" so every sub-queue evaluates scheduling timeouts against the same instant
	now := time.Now().UTC()

	touched, slotsToSetFilled, slotsToDelete, slotsToTimeout := c.decideSubQueues(ctx, grouped, now, decide)

	// Hand the open undo scopes to Run, which finalizes them once ProcessMessages returns. We must
	// not commit/rollback here: pgoutbox still deletes the messages and commits the transaction
	// after FlushWithTx returns, so the in-memory mutations only become durable once that whole
	// transaction commits. Run commits the scopes on success and rolls them back on any error
	// (flush failure here, or a later message-delete / commit failure).
	c.openScopes = touched

	// Flush within the outbox transaction handed to us by FlushWithTx. We don't retry here: on
	// failure we return the error so pgoutbox rolls back (the messages are not deleted) and they
	// are redelivered on a later Run.
	runResult, err := c.flushToDatabase(ctx, tx, slotsToSetFilled, slotsToDelete, slotsToTimeout)

	if err != nil {
		return nil, fmt.Errorf("failed to flush concurrency slots to database: %w", err)
	}

	return runResult, nil
}

// decideSubQueues runs the WAL-apply -> evict-timeouts -> decide pipeline over each sub-queue in
// grouped, fanning out one goroutine per sub-queue (grouped is keyed by sub-queue, so each is touched
// by exactly one goroutine). It opens an undo scope on every sub-queue it mutates and returns them as
// touched so the caller can finalize the scopes once its flush is durable. The returned slot slices
// are the merged fill/delete/timeout decisions across all sub-queues. The post-build pass passes nil
// message slices so only the decide step runs against the hydrated slots.
func (c *ConcurrencyStrategy) decideSubQueues(ctx context.Context, grouped map[string][]walMessage, now time.Time, decide decideFn) (touched []*subQueue, slotsToSetFilled, slotsToDelete, slotsToTimeout []slot) {
	_, span := telemetry.NewSpan(ctx, "concurrency-decide-sub-queues")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "concurrency.strategy.id", Value: c.strategy.ID},
		telemetry.AttributeKV{Key: "tenant.id", Value: c.strategy.TenantID},
		telemetry.AttributeKV{Key: "concurrency.sub-queue.count", Value: len(grouped)},
	)
	wg := sync.WaitGroup{}
	var batchMu sync.Mutex

	slotsToSetFilled = make([]slot, 0)
	slotsToDelete = make([]slot, 0)
	slotsToTimeout = make([]slot, 0)

	touched = make([]*subQueue, 0, len(grouped))

	for q, msgs := range grouped {
		sq := c.getOrCreateSubQueue(q)
		sq.begin()
		touched = append(touched, sq)

		wg.Add(1)
		go func(sq *subQueue, m []walMessage) {
			defer wg.Done()

			localSlotsToDelete := applyWAL(sq, m)

			// cancel any queued slots that have exceeded their scheduling timeout before deciding, so a
			// timed-out slot is never promoted to running or ranked by a cancel strategy
			localSlotsToTimeout := sq.queued.popTimedOut(now)

			// the per-strategy decision step: promote slots to running and (for cancel strategies)
			// evict slots that lost their place. Both index mutations are recorded in the undo scope.
			localSlotsToSetFilled, localDecideCancel := decide(sq)

			// slots cancelled by the decision step are CONCURRENCY_LIMIT cancellations, same bucket as
			// the superseded slots from applyWAL.
			localSlotsToDelete = append(localSlotsToDelete, localDecideCancel...)

			batchMu.Lock()

			slotsToSetFilled = append(slotsToSetFilled, localSlotsToSetFilled...)
			slotsToDelete = append(slotsToDelete, localSlotsToDelete...)
			slotsToTimeout = append(slotsToTimeout, localSlotsToTimeout...)

			batchMu.Unlock()
		}(sq, msgs)
	}
	wg.Wait()

	return touched, slotsToSetFilled, slotsToDelete, slotsToTimeout
}

func (c *ConcurrencyStrategy) flushToDatabase(ctx context.Context, tx pgx.Tx, slotsToSetFilled, slotsToDelete, slotsToTimeout []slot) (*repository.RunConcurrencyResult, error) {
	ctx, span := telemetry.NewSpan(ctx, "concurrency-flush-to-database")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "concurrency.strategy.id", Value: c.strategy.ID},
		telemetry.AttributeKV{Key: "tenant.id", Value: c.strategy.TenantID},
		telemetry.AttributeKV{Key: "concurrency.slots.filled", Value: len(slotsToSetFilled)},
		telemetry.AttributeKV{Key: "concurrency.slots.cancelled", Value: len(slotsToDelete) + len(slotsToTimeout)},
	)

	tasksToSetFilled, cancelledSlots := buildSlotInputs(slotsToSetFilled, slotsToDelete, slotsToTimeout)

	runResult, err := c.repo.UpdateConcurrencySlotsTx(ctx, tx, c.strategy.TenantID, c.strategy.ID, tasksToSetFilled, cancelledSlots)

	if err != nil {
		return nil, err
	}

	return runResult, nil
}

// buildSlotInputs converts the decided slots into the repository's flush inputs. Shared by the WAL
// path (flushToDatabase) and the post-build queueing pass (queueAllSubQueues) so both translate
// slots to task identifiers and cancellation reasons identically.
func buildSlotInputs(slotsToSetFilled, slotsToDelete, slotsToTimeout []slot) ([]repository.TaskIdInsertedAtRetryCount, []repository.CancelledSlotInput) {
	tasksToSetFilled := make([]repository.TaskIdInsertedAtRetryCount, len(slotsToSetFilled))
	for i, slot := range slotsToSetFilled {
		tasksToSetFilled[i] = repository.TaskIdInsertedAtRetryCount{
			Id:         slot.taskId,
			InsertedAt: sqlchelpers.TimestamptzFromTime(time.Unix(0, slot.taskInsertedAtNs).UTC()),
			RetryCount: slot.taskRetryCount,
		}
	}

	// cancelled slots carry their reason so the repository can surface SCHEDULING_TIMED_OUT vs
	// CONCURRENCY_LIMIT in the RunConcurrencyResult. slotsToDelete are superseded/stale slots;
	// slotsToTimeout are queued slots that blew past their scheduling timeout.
	cancelledSlots := make([]repository.CancelledSlotInput, 0, len(slotsToDelete)+len(slotsToTimeout))

	for _, slot := range slotsToDelete {
		cancelledSlots = append(cancelledSlots, repository.CancelledSlotInput{
			TaskIdInsertedAtRetryCount: repository.TaskIdInsertedAtRetryCount{
				Id:         slot.taskId,
				InsertedAt: sqlchelpers.TimestamptzFromTime(time.Unix(0, slot.taskInsertedAtNs).UTC()),
				RetryCount: slot.taskRetryCount,
			},
			CancelledReason: repository.CancelledReasonConcurrencyLimit,
		})
	}

	for _, slot := range slotsToTimeout {
		cancelledSlots = append(cancelledSlots, repository.CancelledSlotInput{
			TaskIdInsertedAtRetryCount: repository.TaskIdInsertedAtRetryCount{
				Id:         slot.taskId,
				InsertedAt: sqlchelpers.TimestamptzFromTime(time.Unix(0, slot.taskInsertedAtNs).UTC()),
				RetryCount: slot.taskRetryCount,
			},
			CancelledReason: repository.CancelledReasonSchedulingTimedOut,
		})
	}

	return tasksToSetFilled, cancelledSlots
}

func (c *ConcurrencyStrategy) getOrCreateSubQueue(key string) *subQueue {
	c.mu.RLock()
	sq, ok := c.subQueues[key]
	c.mu.RUnlock()
	if ok {
		return sq
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	// re-check: another goroutine may have created it between RUnlock and Lock
	sq, ok = c.subQueues[key]
	if ok {
		return sq
	}
	sq = newSubQueue(key, c.strategy.MaxConcurrency, c.compare)
	c.subQueues[key] = sq
	return sq
}

func groupMessagesBySubQueue(msgs []walMessage) map[string][]walMessage {
	grouped := make(map[string][]walMessage)
	for _, msg := range msgs {
		grouped[msg.Key] = append(grouped[msg.Key], msg)
	}
	return grouped
}

func walMessageToSlot(msg walMessage) slot {
	return slot{
		priority:            msg.Priority,
		taskId:              msg.TaskId,
		taskInsertedAtNs:    msg.TaskInsertedAt.UnixNano(),
		taskRetryCount:      msg.TaskRetryCount,
		scheduleTimeoutAtMs: msg.ScheduleTimeoutAtMs,
	}
}
