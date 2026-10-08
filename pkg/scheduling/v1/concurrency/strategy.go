package concurrency

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/pgoutbox"
	outboxsqlc "github.com/hatchet-dev/pgoutbox/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/listutils"
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

const (
	// DefaultEagerIndexMaxSlots is the slot count at or above which a strategy stops hydrating its
	// whole backlog into memory at build time and switches to on-demand hydration.
	DefaultEagerIndexMaxSlots int32 = 1_000_000

	onDemandInitialScanKeysPerPage = 1000
	onDemandInitialScanPagesPerRun = 10
	// onDemandOutsideWindowCancelLimit caps how many queued slots outside the hydration window a single
	// batch cancels; keys whose cap was hit are revisited on the next Run.
	onDemandOutsideWindowCancelLimit = 10000
	// onDemandExpiredLoadLimit caps how many expired queued slots a single batch loads (and so times
	// out); keys whose cap was hit are revisited on the next Run.
	onDemandExpiredLoadLimit  = 1000
	onDemandRevisitKeysPerRun = 1000
)

// maxRunsObservation is a sub-queue's dynamically observed limit, retained across on-demand evictions
// so a deleted newer task cannot hand the limit back to an older slot's value (see observeMaxRuns).
type maxRunsObservation struct {
	maxRuns int32
	from    int64
}

type ConcurrencyStrategy struct {
	outbox pgoutbox.Outbox
	repo   repository.ConcurrencyRepository
	// subQueues holds the hydrated keys. With eager hydration (the default) buildIndex loads every slot
	// of the strategy and pruneEmpty drops keys once they empty out. With on-demand hydration (chosen
	// at build time when the strategy holds at least eagerIndexMaxSlots slots) only the keys touched by
	// the current batch are loaded from the database, and all of them are evicted once the batch is
	// finalized, so memory is bounded by the batch size rather than the backlog.
	subQueues map[string]*subQueue
	strategy  *sqlcv1.V1StepConcurrency

	eagerIndexMaxSlots int32
	// residentSlots is the number of slots held across all hydrated sub-queues on the eager path,
	// kept current from each finalized batch's sub-queue size deltas. Guarded by mu.
	residentSlots int
	// hydrateOnDemand is set by buildIndex, or later by switchToOnDemandHydration when an eager
	// index outgrows the bound; both run under buildingMu, and Run only reads it between batches.
	hydrateOnDemand bool
	// initialScanLastKey is the on-demand initial scan's cursor over the strategy's distinct keys;
	// the scan is spread across Runs so each one only holds a bounded number of keys in memory.
	// Invalid means the scan has not started, so the first page includes the empty-string key.
	initialScanLastKey pgtype.Text
	// keysToRevisit are keys whose outside-window cancellations were capped in a batch; each Run
	// re-decides a bounded number of them. Guarded by mu.
	keysToRevisit map[string]struct{}
	// observedMaxRuns keeps each key's dynamic limit while its sub-queue is evicted. Only populated
	// for strategies with a max_runs_expression, and dropped once the key holds no slots, mirroring
	// pruneEmpty on the eager path. Guarded by mu.
	observedMaxRuns map[string]maxRunsObservation
	// immutable copies of the strategy identity, safe to read without holding any lock
	// (strategy itself is swapped in place by UpdateStrategy under buildingMu + mu)
	strategyId       int64
	strategyTenantId uuid.UUID
	l                *zerolog.Logger
	compare          func(a, b slot) int
	built            chan struct{}
	topic            string
	pending          []*repository.RunConcurrencyResult
	openScopes       []*subQueue
	mu               sync.RWMutex
	pendingMu        sync.Mutex
	buildingMu       sync.Mutex
	initialQueueMu   sync.Mutex
	initialQueued    bool
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
	c.evictIfHydratedOnDemand(c.openScopes)
	c.recountResidentSlots(c.openScopes)
	c.openScopes = nil
}

// finalizeCommittedSubQueues drops the sub-queues a committed batch no longer needs in memory: all of
// them under on-demand hydration (the next batch that touches a key reloads it from the database),
// otherwise only the ones the batch emptied.
func (c *ConcurrencyStrategy) finalizeCommittedSubQueues(committed []*subQueue) {
	if c.hydrateOnDemand {
		c.evictIfHydratedOnDemand(committed)
		return
	}

	c.recountResidentSlots(committed)
	c.pruneEmpty(committed)
}

// recountResidentSlots folds each sub-queue's size change since it was last counted into the eager
// index's resident slot total. Only the sub-queues a batch touched can have changed, so this is
// O(touched) per batch rather than a scan of every key.
func (c *ConcurrencyStrategy) recountResidentSlots(subQueues []*subQueue) {
	if c.hydrateOnDemand {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, sq := range subQueues {
		size := sq.size()
		c.residentSlots += size - sq.countedSlots
		sq.countedSlots = size
	}
}

func (c *ConcurrencyStrategy) evictIfHydratedOnDemand(subQueues []*subQueue) {
	if !c.hydrateOnDemand || len(subQueues) == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, sq := range subQueues {
		if existing, ok := c.subQueues[sq.key]; ok && existing == sq {
			delete(c.subQueues, sq.key)
		}

		c.retainObservedMaxRunsLocked(sq)
	}
}

// retainObservedMaxRunsLocked carries a dynamic strategy's observed limit across the sub-queue's
// eviction. A key with no slots left forgets its observation, as pruneEmpty does on the eager path.
// Requires mu.
func (c *ConcurrencyStrategy) retainObservedMaxRunsLocked(sq *subQueue) {
	if !c.strategy.MaxRunsExpression.Valid {
		return
	}

	if sq.running.len() == 0 && sq.queued.len() == 0 {
		delete(c.observedMaxRuns, sq.key)
		return
	}

	if sq.maxRunsFrom != 0 {
		c.observedMaxRuns[sq.key] = maxRunsObservation{maxRuns: sq.maxRuns, from: sq.maxRunsFrom}
	}
}

// observeMaxRunsForKey records a dynamic limit observation for a key that may not be hydrated, so the
// value is available when the key's sub-queue is next created. Newest task wins, as in observeMaxRuns.
func (c *ConcurrencyStrategy) observeMaxRunsForKey(key string, maxRuns int32, taskInsertedAtNs int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.strategy.MaxRunsExpression.Valid {
		return
	}

	if current, ok := c.observedMaxRuns[key]; ok && taskInsertedAtNs < current.from {
		return
	}

	c.observedMaxRuns[key] = maxRunsObservation{maxRuns: maxRuns, from: taskInsertedAtNs}
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
	opts ...StrategyOption,
) *ConcurrencyStrategy {
	c := &ConcurrencyStrategy{
		subQueues:          make(map[string]*subQueue),
		strategy:           strategy,
		strategyId:         strategy.ID,
		strategyTenantId:   strategy.TenantID,
		repo:               repo,
		l:                  l,
		compare:            comparatorForStrategy(strategy.Strategy),
		outbox:             outbox,
		topic:              getTopic(strategy),
		built:              make(chan struct{}),
		eagerIndexMaxSlots: DefaultEagerIndexMaxSlots,
		keysToRevisit:      make(map[string]struct{}),
		observedMaxRuns:    make(map[string]maxRunsObservation),
	}

	for _, opt := range opts {
		opt(c)
	}

	outbox.AddFlusher(c.topic, c)

	go c.buildIndexLoop(ctx)

	return c
}

// StrategyOption configures a ConcurrencyStrategy at construction.
type StrategyOption func(*ConcurrencyStrategy)

// WithEagerIndexMaxSlots sets the slot count at or above which the strategy hydrates keys on demand
// instead of loading its whole backlog at build time. Zero or negative keeps the default.
func WithEagerIndexMaxSlots(maxSlots int32) StrategyOption {
	return func(c *ConcurrencyStrategy) {
		if maxSlots > 0 {
			c.eagerIndexMaxSlots = maxSlots
		}
	}
}

func NewNoOpFlusher(
	ctx context.Context,
	outbox pgoutbox.Outbox,
	strategy *sqlcv1.V1StepConcurrency,
	l *zerolog.Logger,
) {
	topic := getTopic(strategy)

	outbox.AddFlusher(topic, pgoutbox.NewNopFlusher())

	go func() {
		for {
			if ctx.Err() != nil {
				return
			}

			// Subscribe owns the exclusive lease for the duration of the call: it blocks
			// acquiring it, re-acquires it if it is ever lost mid-subscribe, and releases it
			// on return so the topic's next consumer can take over immediately. It only
			// returns early if the initial acquisition fails (e.g. a transient database
			// error), so retry rather than leaving the topic undrained.
			err := outbox.Subscribe(ctx, topic, pgoutbox.WithExclusive())

			if err != nil && ctx.Err() == nil {
				l.Error().Err(err).Msgf("failed to subscribe to topic %s, retrying", topic)
			}

			// context-aware sleep so this goroutine exits promptly on shutdown rather than
			// lingering in a fixed sleep (which otherwise trips goleak and delays teardown).
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
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
		telemetry.AttributeKV{Key: "concurrency.strategy.id", Value: c.strategyId},
		telemetry.AttributeKV{Key: "tenant.id", Value: c.strategyTenantId},
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

	// a failed revisit re-marks its keys and must not discard the already-committed scan results above
	revisitResult, err := c.revisitKeys(ctx)

	if err != nil {
		c.l.Error().Err(err).Msgf("failed to revisit concurrency keys for topic %s, will retry on the next run", c.topic)
	}

	for {
		msgs, err := c.outbox.ProcessMessages(ctx, c.topic)

		if err != nil {
			// the outbox transaction rolled back (flush, message-delete, or commit failure), so undo
			// this batch's in-memory mutations to keep the index consistent with the database. the
			// messages are not deleted and will be redelivered on a later Run. Earlier batches in this
			// Run did commit, so their results are returned alongside the error: the caller publishes
			// the cancelled-task messages from them and they must not be lost to the failed batch.
			c.rollbackScopes()
			return mergeResults(append(c.takePending(), initialResult, revisitResult)), fmt.Errorf("failed to process outbox messages for topic %s: %w", c.topic, err)
		}

		// ProcessMessages only returns without error once the transaction has committed, so the
		// in-memory mutations are now durable - discard the undo log and drop the sub-queues this
		// batch no longer needs in memory.
		c.finalizeCommittedSubQueues(c.commitScopes())

		if c.eagerIndexOutgrewBound() {
			c.switchToOnDemandHydration()
		}

		// no more messages queued for this topic; we've drained it
		if len(msgs) == 0 {
			break
		}
	}

	return mergeResults(append(c.takePending(), initialResult, revisitResult)), nil
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

	if c.hydrateOnDemand {
		res, scanComplete, err := c.queueNextKeyPages(ctx)

		if err != nil {
			return nil, fmt.Errorf("failed to run initial concurrency queueing scan for topic %s: %w", c.topic, err)
		}

		c.initialQueued = scanComplete

		return res, nil
	}

	res, err := c.queueAllSubQueues(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to run initial concurrency queueing for topic %s: %w", c.topic, err)
	}

	c.initialQueued = true

	return res, nil
}

// queueNextKeyPages is the on-demand counterpart of queueAllSubQueues. The whole backlog cannot be
// held in memory, so the pass walks the strategy's distinct keys from initialScanLastKey, a page of
// keys at a time: each page is hydrated from the database, decided, flushed in its own transaction
// and evicted again before the next page is loaded. It processes a bounded number of pages per call
// so a Run returns promptly and interleaves with WAL processing (which is safe under on-demand
// hydration, since every batch reloads the keys it touches from the database). It reports whether
// the scan reached the end of the key space.
func (c *ConcurrencyStrategy) queueNextKeyPages(ctx context.Context) (*repository.RunConcurrencyResult, bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "concurrency-initial-queueing-scan")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "concurrency.strategy.id", Value: c.strategy.ID},
		telemetry.AttributeKV{Key: "tenant.id", Value: c.strategy.TenantID},
	)

	c.buildingMu.Lock()
	defer c.buildingMu.Unlock()

	results := make([]*repository.RunConcurrencyResult, 0, onDemandInitialScanPagesPerRun)

	for range onDemandInitialScanPagesPerRun {
		res, scanComplete, err := c.queueNextKeyPage(ctx)

		if err != nil {
			// Earlier pages in this call have already committed and advanced the cursor, so their
			// results must still reach the caller (cancelled-task messages are published from them).
			// Surface the error only when nothing was committed; otherwise the scan simply resumes
			// from the cursor on the next Run.
			if len(results) == 0 {
				return nil, false, err
			}

			c.l.Error().Err(err).Msgf("initial concurrency queueing scan for topic %s stopped early, will resume on the next run", c.topic)

			return mergeResults(results), false, nil
		}

		if scanComplete {
			return mergeResults(results), true, nil
		}

		results = append(results, res)
	}

	return mergeResults(results), false, nil
}

// queueNextKeyPage processes the page of keys after the scan cursor and advances the cursor past it.
// It reports scanComplete when no keys remain.
func (c *ConcurrencyStrategy) queueNextKeyPage(ctx context.Context) (res *repository.RunConcurrencyResult, scanComplete bool, err error) {
	keys, err := c.repo.ListDistinctConcurrencyKeysAfter(ctx, c.strategy.TenantID, c.strategy.ID, c.initialScanLastKey, onDemandInitialScanKeysPerPage)

	if err != nil {
		return nil, false, err
	}

	if len(keys) == 0 {
		return nil, true, nil
	}

	res, err = c.decideAndFlushHydratedKeys(ctx, keys)

	if err != nil {
		return nil, false, err
	}

	c.initialScanLastKey = pgtype.Text{String: keys[len(keys)-1], Valid: true}

	return res, false, nil
}

// decideAndFlushHydratedKeys hydrates the given keys from the database (outside any transaction), runs
// the decide step over them with no WAL messages, flushes in its own transaction and evicts the
// sub-queues afterwards regardless of the outcome.
func (c *ConcurrencyStrategy) decideAndFlushHydratedKeys(ctx context.Context, keys []string) (*repository.RunConcurrencyResult, error) {
	hydrated, err := c.hydrateKeysFromDatabase(ctx, nil, keys)

	if err != nil {
		return nil, err
	}

	grouped := make(map[string][]walMessage, len(keys))
	for _, key := range keys {
		grouped[key] = nil
	}

	touched, slotsToSetFilled, slotsToDelete, slotsToTimeout := c.decideSubQueues(ctx, grouped, time.Now().UTC(), c.decide())

	defer c.evictIfHydratedOnDemand(touched)

	tasksToSetFilled, cancelledSlots := buildSlotInputs(slotsToSetFilled, append(hydrated.toCancel, slotsToDelete...), slotsToTimeout)

	return c.repo.UpdateConcurrencySlots(ctx, c.strategy.TenantID, c.strategy.ID, tasksToSetFilled, cancelledSlots)
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

	// only populated by UPDATE messages
	IsFilled bool `json:"isFilled"`

	// only populated by INSERT messages, and only for strategies with a
	// max_runs_expression: the slot's insert-time evaluation of the per-group limit
	MaxRuns *int32 `json:"maxRuns"`
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

	slotCount, err := c.repo.CountConcurrencySlotsUpToLimit(ctx, c.strategy.TenantID, c.strategy.ID, c.eagerIndexMaxSlots)

	if err != nil {
		return fmt.Errorf("failed to count concurrency slots for topic %s: %w", c.topic, err)
	}

	if slotCount >= int64(c.eagerIndexMaxSlots) {
		c.hydrateOnDemand = true
		c.l.Warn().Msgf("concurrency strategy %d holds at least %d slots; hydrating keys on demand instead of loading the whole index", c.strategy.ID, c.eagerIndexMaxSlots)
		return nil
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

				c.insertIndexRow(row)
			}
		}
	}()

	err = c.repo.ReadConcurrencySlotsForIndexing(ctx, c.strategy.TenantID, c.strategy.ID, writeCh)
	if err != nil {
		return err
	}

	close(writeCh)

	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}

	c.mu.Lock()
	for _, sq := range c.subQueues {
		sq.countedSlots = sq.size()
		c.residentSlots += sq.countedSlots
	}
	c.mu.Unlock()

	return nil
}

// insertIndexRow places a database slot row into its sub-queue's running or queued index. Rows are
// inserted one at a time, in any order: the timestamp guard in observeMaxRuns makes each group
// converge to the value evaluated for its most recently created live slot.
func (c *ConcurrencyStrategy) insertIndexRow(row *sqlcv1.ListConcurrencySlotsForIndexingRow) {
	sq := c.getOrCreateSubQueue(row.Key)

	if row.MaxRuns.Valid {
		sq.observeMaxRuns(row.MaxRuns.Int32, row.TaskInsertedAt.Time.UnixNano())
	}

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

// hydrateSubQueuesFromRows loads the given slot rows into fresh sub-queues. It is the on-demand
// counterpart of buildIndex: callers pass the rows of exactly the keys the current batch touches,
// and evict those sub-queues again once the batch is finalized. Rows are applied as INSERT
// messages (filled rows as UPDATE messages) so a retry's superseded slot is detected and cancelled
// exactly as it would be on the live WAL path; the returned slots are those superseded slots.
func (c *ConcurrencyStrategy) hydrateSubQueuesFromRows(rows []*sqlcv1.ListConcurrencySlotsForIndexingRow) []slot {
	superseded := make([]slot, 0)

	for _, row := range rows {
		sq := c.getOrCreateSubQueue(row.Key)

		if row.MaxRuns.Valid {
			sq.observeMaxRuns(row.MaxRuns.Int32, row.TaskInsertedAt.Time.UnixNano())
		}

		superseded = append(superseded, applyWAL(sq, []walMessage{indexRowToWALMessage(row)})...)
	}

	return superseded
}

func indexRowToWALMessage(row *sqlcv1.ListConcurrencySlotsForIndexingRow) walMessage {
	operation := "INSERT"
	if row.IsFilled {
		operation = "UPDATE"
	}

	return walMessage{
		Operation:           operation,
		Key:                 row.Key,
		TaskId:              row.TaskID,
		TaskInsertedAt:      row.TaskInsertedAt.Time,
		TaskRetryCount:      row.TaskRetryCount,
		Priority:            row.Priority,
		ScheduleTimeoutAtMs: row.ScheduleTimeoutAt.Time.UnixMilli(),
		IsFilled:            row.IsFilled,
	}
}

func (c *ConcurrencyStrategy) processWALMessages(ctx context.Context, tx pgx.Tx, messages []walMessage) (*repository.RunConcurrencyResult, error) {
	// wait until we can acquire a lock on the strategy
	// (so we don't process WAL messages while we're building the index)
	c.buildingMu.Lock()
	defer c.buildingMu.Unlock()

	return c.processStrategy(ctx, tx, messages, c.decide())
}

// hydratedKeys is the outcome of loading a batch's keys from the database: the slots the hydration
// itself decided to cancel (a retry's superseded slot, or a queued slot outside the window on a
// strategy that cancels everything it does not keep), and whether some key's outside-window
// cancellations were capped and so need another pass.
type hydratedKeys struct {
	toCancel  []slot
	truncated bool
}

// hydrateKeysFromDatabase loads a bounded window of each key's slots into fresh sub-queues. With a nil
// tx it reads from the pool; the WAL path passes the outbox transaction so the decide step runs against
// the committed state the batch's messages describe. Keys touched by a batch were evicted when the
// previous batch was finalized, so none are expected to be hydrated already; any that are get reloaded
// over in place (insert replaces an existing entry for the same task).
//
// The window per key is every filled slot plus the best and worst windowSizeForKey queued slots under
// the strategy's comparator. For every decide function the slots it fills come from the best end and
// the slots it keeps queued come from one end or the other, so queued slots outside the window never
// change a decision: GROUP_ROUND_ROBIN leaves them queued and every cancel strategy cancels them, which
// is done here from the rows rather than by loading them.
func (c *ConcurrencyStrategy) hydrateKeysFromDatabase(ctx context.Context, tx pgx.Tx, keys []string) (hydratedKeys, error) {
	query := repository.ConcurrencySlotWindowQuery{
		Keys:         keys,
		Limits:       listutils.Map(keys, c.knownLimitForKey),
		LimitFactor:  c.windowLimitFactor(),
		Ordering:     slotOrderingForStrategy(c.strategy.Strategy),
		ExpiredLimit: onDemandExpiredLoadLimit,
		OutsideLimit: c.outsideWindowCancelLimit(),
	}

	var window *repository.ConcurrencySlotWindow
	var err error

	if tx != nil {
		window, err = c.repo.ListConcurrencySlotWindowForKeysTx(ctx, tx, c.strategy.TenantID, c.strategy.ID, query)
	} else {
		window, err = c.repo.ListConcurrencySlotWindowForKeys(ctx, c.strategy.TenantID, c.strategy.ID, query)
	}

	if err != nil {
		return hydratedKeys{}, fmt.Errorf("failed to hydrate concurrency keys for topic %s: %w", c.topic, err)
	}

	toCancel := c.hydrateSubQueuesFromRows(window.InWindow)

	// expired slots are loaded so popTimedOut cancels them with SCHEDULING_TIMED_OUT, exactly as on
	// the eager path, rather than being lumped in with the outside-window CONCURRENCY_LIMIT cancels
	toCancel = append(toCancel, c.hydrateSubQueuesFromRows(window.Expired)...)

	// The newest slot's evaluation is the key's effective limit even when that slot is outside the
	// window. It is applied after every row so it is the final observation: observeMaxRuns lets an
	// equal timestamp overwrite, and slots created together share one, so a row loaded later with the
	// same timestamp would otherwise win the tie the query already resolved by task id.
	for key, newest := range window.NewestLimitByKey {
		c.getOrCreateSubQueue(key).observeMaxRuns(newest.MaxRuns, newest.TaskInsertedAt.UnixNano())
	}

	for _, row := range window.OutsideWindow {
		toCancel = append(toCancel, walMessageToSlot(indexRowToWALMessage(row)))
	}

	truncated := len(window.Expired) >= int(query.ExpiredLimit) ||
		(query.OutsideLimit > 0 && len(window.OutsideWindow) >= int(query.OutsideLimit))

	if truncated {
		c.markKeysForRevisit(keys)
	}

	return hydratedKeys{toCancel: toCancel, truncated: truncated}, nil
}

// slotOrderingForStrategy maps the strategy's comparator (see comparatorForStrategy) onto the flags the
// window query ranks by. "Best first" under every comparator is highest priority (when the comparator
// uses priority), then oldest or newest.
func slotOrderingForStrategy(kind sqlcv1.V1ConcurrencyStrategy) repository.ConcurrencySlotOrdering {
	switch kind {
	case sqlcv1.V1ConcurrencyStrategyCANCELINPROGRESS:
		return repository.ConcurrencySlotOrdering{OrderByPriority: true, NewestFirst: true}
	case sqlcv1.V1ConcurrencyStrategyCANCELQUEUEDEXCEPTNEWEST, sqlcv1.V1ConcurrencyStrategyCANCELQUEUEDEXCEPTOLDEST:
		return repository.ConcurrencySlotOrdering{OrderByPriority: false, NewestFirst: false}
	default:
		return repository.ConcurrencySlotOrdering{OrderByPriority: true, NewestFirst: false}
	}
}

// knownLimitForKey is the lower bound this process knows for a key's concurrency limit: the static
// limit, or a retained dynamic observation if larger. The window query widens it further from the
// key's own rows, so loading more than needed is the only possible error, and that is always safe.
func (c *ConcurrencyStrategy) knownLimitForKey(key string) int32 {
	limit := max(c.strategy.MaxConcurrency, 0)

	c.mu.RLock()
	defer c.mu.RUnlock()

	if observed, ok := c.observedMaxRuns[key]; ok {
		limit = max(limit, observed.maxRuns)
	}

	return limit
}

// windowLimitFactor is how many multiples of the limit to load from each end of the comparator.
// CANCEL_QUEUED_EXCEPT_OLDEST keeps the maxRuns queued slots directly after the ones it fills, so it
// needs twice the limit from the best end.
func (c *ConcurrencyStrategy) windowLimitFactor() int32 {
	if c.strategy.Strategy == sqlcv1.V1ConcurrencyStrategyCANCELQUEUEDEXCEPTOLDEST {
		return 2
	}

	return 1
}

func (c *ConcurrencyStrategy) outsideWindowCancelLimit() int32 {
	if c.strategy.Strategy == sqlcv1.V1ConcurrencyStrategyGROUPROUNDROBIN {
		return 0
	}

	return onDemandOutsideWindowCancelLimit
}

func (c *ConcurrencyStrategy) markKeysForRevisit(keys []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, key := range keys {
		c.keysToRevisit[key] = struct{}{}
	}
}

func (c *ConcurrencyStrategy) takeKeysToRevisit(limit int) []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	keys := make([]string, 0, min(limit, len(c.keysToRevisit)))

	for key := range c.keysToRevisit {
		if len(keys) == limit {
			break
		}

		keys = append(keys, key)
		delete(c.keysToRevisit, key)
	}

	return keys
}

// revisitKeys re-decides keys whose outside-window cancellations were capped in an earlier batch. Each
// pass handles a bounded number of keys; any that are still capped re-enter the set.
func (c *ConcurrencyStrategy) revisitKeys(ctx context.Context) (*repository.RunConcurrencyResult, error) {
	keys := c.takeKeysToRevisit(onDemandRevisitKeysPerRun)

	if len(keys) == 0 {
		return nil, nil
	}

	c.buildingMu.Lock()
	defer c.buildingMu.Unlock()

	res, err := c.decideAndFlushHydratedKeys(ctx, keys)

	if err != nil {
		c.markKeysForRevisit(keys)
		return nil, fmt.Errorf("failed to revisit concurrency keys for topic %s: %w", c.topic, err)
	}

	return res, nil
}

// decideFn runs after a sub-queue's WAL has been applied and timed-out queued slots evicted. It
// mutates the sub-queue's running/queued indexes to reflect this strategy's policy and returns the
// slots to mark filled (RUNNING, queue notified) and the slots to cancel with CONCURRENCY_LIMIT.
// Timed-out queued slots are handled by the shared pipeline and are not passed here.
type decideFn func(sq *subQueue) (toFill, toCancel []slot)

// comparatorForStrategy selects the slot ordering for a strategy kind. GROUP_ROUND_ROBIN and
// CANCEL_NEWEST keep the oldest among equal-priority slots (priorityCompare); CANCEL_IN_PROGRESS
// keeps the newest (cancelInProgressCompare), so a newer arrival preempts an older run. "Smaller"
// under the chosen comparator always means "should run".
func comparatorForStrategy(kind sqlcv1.V1ConcurrencyStrategy) func(a, b slot) int {
	if kind == sqlcv1.V1ConcurrencyStrategyCANCELINPROGRESS {
		return cancelInProgressCompare
	}
	if kind == sqlcv1.V1ConcurrencyStrategyCANCELQUEUEDEXCEPTOLDEST || kind == sqlcv1.V1ConcurrencyStrategyCANCELQUEUEDEXCEPTNEWEST {
		return cancelQueuedExceptCompare
	}
	return priorityCompare
}

// decide selects the per-sub-queue decision function for this strategy's kind. All three fill free
// capacity from the queued backlog in comparator order; they differ in what happens to the slots
// that don't fit: GROUP_ROUND_ROBIN leaves them queued, CANCEL_NEWEST cancels them (reject the
// newest arrivals, never touch running work), and CANCEL_IN_PROGRESS cancels them too but may also
// preempt a running slot when a higher-priority-or-newer slot is waiting.
func (c *ConcurrencyStrategy) decide() decideFn {
	switch c.strategy.Strategy {
	case sqlcv1.V1ConcurrencyStrategyGROUPROUNDROBIN:
		return decideGroupRoundRobin
	case sqlcv1.V1ConcurrencyStrategyCANCELINPROGRESS:
		return decideCancelInProgress
	case sqlcv1.V1ConcurrencyStrategyCANCELNEWEST:
		return decideCancelNewest
	case sqlcv1.V1ConcurrencyStrategyCANCELQUEUEDEXCEPTNEWEST:
		return decideCancelQueuedExceptNewest
	case sqlcv1.V1ConcurrencyStrategyCANCELQUEUEDEXCEPTOLDEST:
		return decideCancelQueuedExceptOldest
	default:
		panic("unknown concurrency strategy")
	}
}

// decideCancelQueuedExceptNewest fills free capacity from queued buffer, then cancels everything
// except for the maxRuns newest, which will stick around to be queued up the next time
func decideCancelQueuedExceptNewest(sq *subQueue) (toFill, toCancel []slot) {
	toFill = sq.queued.pop(int(sq.slotsToRun()))
	for _, s := range toFill {
		sq.running.insert(s)
	}
	toCancel = sq.queued.pop(sq.queued.len() - int(sq.maxRuns))
	return toFill, toCancel
}

// decideCancelQueuedExceptOldest is the same as CancelQueuedExceptNewest except that it keeps the oldest slot around
func decideCancelQueuedExceptOldest(sq *subQueue) (toFill, toCancel []slot) {
	toFill = sq.queued.pop(int(sq.slotsToRun()))
	for _, s := range toFill {
		sq.running.insert(s)
	}
	// somewhat awkward here, but we want to keep the first elements (because oldest-first comparator),
	// so we need to pop and then reinsert.
	// maxRuns is clamped to 0 here to guard against a negative value (e.g. from data written before
	// MaxRuns positivity validation was enforced) producing an out-of-range slice index below.
	maxRuns := max(0, int(sq.maxRuns))
	if sq.queued.len() > maxRuns {
		popped := sq.queued.pop(sq.queued.len())
		toCancel = popped[maxRuns:]
		for _, s := range popped[:maxRuns] {
			sq.queued.insert(s)
		}
	}
	return toFill, toCancel
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
			// Applied even when the slot itself ends up superseded below: the value is
			// still the evaluation of the group's newest task. UPDATE/DELETE messages
			// never carry a value, so retries and completions cannot move the limit.
			if msg.MaxRuns != nil {
				sq.observeMaxRuns(*msg.MaxRuns, msg.TaskInsertedAt.UnixNano())
			}

			if currentRunningSlot, exists := sq.running.get(msg.TaskId); exists {
				// compare the current running slot's retry count with the message's retry count; the greater wins
				if currentRunningSlot.taskRetryCount < msg.TaskRetryCount {
					superseded = append(superseded, currentRunningSlot)
					sq.running.delete(msg.TaskId)

					// place the new slot in the queued index, so it goes through the regular promotion pipeline
					sq.queued.insert(walMessageToSlot(msg))
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
		case "UPDATE":
			// UPDATE never represents a duplicate row - it's the same physical v1_concurrency_slot row
			// being resynced in place (a retry-reset: task_retry_count/schedule_timeout_at/priority
			// changed, is_filled reset to FALSE). Unlike INSERT, nothing is ever superseded/cancelled
			// here - just move the slot into whichever index matches msg.IsFilled.
			newSlot := walMessageToSlot(msg)
			if msg.IsFilled {
				sq.queued.delete(msg.TaskId)
				sq.running.insert(newSlot)
			} else {
				sq.running.delete(msg.TaskId)
				sq.queued.insert(newSlot)
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

	cancelledByHydration := make([]slot, 0)

	if c.hydrateOnDemand {
		// The touched keys are reloaded from the database inside the outbox transaction, which sees
		// every slot change the messages describe (and any later ones), so the rows are the truth
		// and the messages only tell us which keys to look at. Applying them on top would reintroduce
		// slots whose DELETE is still queued behind this batch. The one thing kept from the messages
		// is a dynamic limit evaluation, which the rows cannot carry once that task's slot is gone.
		keys := make([]string, 0, len(grouped))

		for key, keyMsgs := range grouped {
			keys = append(keys, key)

			for _, msg := range keyMsgs {
				if msg.Operation == "INSERT" && msg.MaxRuns != nil {
					c.observeMaxRunsForKey(key, *msg.MaxRuns, msg.TaskInsertedAt.UnixNano())
				}
			}

			grouped[key] = nil
		}

		hydrated, err := c.hydrateKeysFromDatabase(ctx, tx, keys)

		if err != nil {
			return nil, err
		}

		cancelledByHydration = hydrated.toCancel
	}

	// single "now" so every sub-queue evaluates scheduling timeouts against the same instant
	now := time.Now().UTC()

	touched, slotsToSetFilled, slotsToDelete, slotsToTimeout := c.decideSubQueues(ctx, grouped, now, decide)

	slotsToDelete = append(cancelledByHydration, slotsToDelete...)

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

// UpdateStrategy applies a changed definition to the live index without a rebuild. The
// caller guarantees the strategy kind and parent linkage are unchanged (those alter the
// sub-queue comparators and heap ordering, so they require a rebuild); expression and
// static max-concurrency changes are safe to swap in place. buildingMu serializes this
// against WAL processing, index builds, and the queueing pass, so no batch observes a
// half-applied definition; mu covers getOrCreateSubQueue's reads.
//
// The WAL path only re-decides sub-queues its messages touch, so a changed limit would
// otherwise not reach an idle group's backlog until new traffic arrived for it. Re-arm
// the all-sub-queue queueing pass (the same one that runs post-build) so the next Run
// applies the new limit everywhere: raises promote queued backlog immediately, lowers
// trim or grandfather per the strategy kind. Lock order matches runInitialQueueing
// (initialQueueMu before buildingMu).
func (c *ConcurrencyStrategy) UpdateStrategy(next *sqlcv1.V1StepConcurrency) {
	c.initialQueueMu.Lock()
	defer c.initialQueueMu.Unlock()
	c.buildingMu.Lock()
	defer c.buildingMu.Unlock()
	c.mu.Lock()
	defer c.mu.Unlock()

	c.strategy = next

	for _, sq := range c.subQueues {
		// only groups still on the static default move to the new static limit; a
		// dynamically observed value (maxRunsFrom set) stays until a newer task's
		// evaluation replaces it. The begin-scope snapshot moves too so a rollback of an
		// in-flight batch restores the new static value rather than the old one.
		if sq.maxRunsFrom == 0 {
			sq.maxRuns = next.MaxConcurrency
			sq.maxRunsAtBegin = next.MaxConcurrency
		}
	}

	c.initialQueued = false
	c.initialScanLastKey = pgtype.Text{}
}

// switchToOnDemandHydration converts an eager index that has outgrown the bound into on-demand mode:
// every hydrated sub-queue is dropped and the paged initial scan is re-armed so the backlog is
// re-decided from the database, a page at a time, on the following Runs. Run calls this between
// batches, so no batch observes a half-switched index. Lock order matches UpdateStrategy.
func (c *ConcurrencyStrategy) switchToOnDemandHydration() {
	c.initialQueueMu.Lock()
	defer c.initialQueueMu.Unlock()
	c.buildingMu.Lock()
	defer c.buildingMu.Unlock()
	c.mu.Lock()
	defer c.mu.Unlock()

	c.l.Warn().Msgf("concurrency strategy %d grew to %d hydrated keys; switching to hydrating keys on demand", c.strategy.ID, len(c.subQueues))

	c.hydrateOnDemand = true

	for _, sq := range c.subQueues {
		c.retainObservedMaxRunsLocked(sq)
	}

	c.subQueues = make(map[string]*subQueue)
	c.residentSlots = 0
	c.initialQueued = false
	c.initialScanLastKey = pgtype.Text{}
}

// eagerIndexOutgrewBound reports whether an eagerly hydrated index now holds at least
// eagerIndexMaxSlots slots, or as many keys (each key carries a fixed per-sub-queue cost that
// dominates memory, so a key count on its own is enough to trip the bound).
func (c *ConcurrencyStrategy) eagerIndexOutgrewBound() bool {
	if c.hydrateOnDemand {
		return false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	bound := int(c.eagerIndexMaxSlots)

	return len(c.subQueues) >= bound || c.residentSlots >= bound
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

	if observed, ok := c.observedMaxRuns[key]; ok {
		sq.observeMaxRuns(observed.maxRuns, observed.from)
	}

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
