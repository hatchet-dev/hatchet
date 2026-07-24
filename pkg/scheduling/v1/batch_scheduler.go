package v1

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/randomticker"

	v1repo "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

const defaultBatchPollInterval = 200 * time.Millisecond
const batchFetchLimit int32 = 256
const defaultBatchIdleTTL = 30 * time.Second
const maxBufferedPayloadBytes int32 = 4_000_000

// small struct for storing in the buffer to avoid worst-case memory growth,
// has only the fields we need for evaluating when to flush, hydrated when needed
// total size of struct including padding is 40 bytes
type bufferedItem struct {
	ScheduleTimeoutAt pgtype.Timestamp
	Queue             string
	ID                int64
	PayloadSize       int32
}

func toBufferedItem(item *sqlcv1.V1BatchedQueueItem) bufferedItem {
	return bufferedItem{
		ID:                item.ID,
		Queue:             item.Queue,
		ScheduleTimeoutAt: item.ScheduleTimeoutAt,
		PayloadSize:       item.PayloadSize,
	}
}

func toV1QueueItem(item *sqlcv1.V1BatchedQueueItem) *sqlcv1.V1QueueItem {
	return &sqlcv1.V1QueueItem{
		ID:                item.ID,
		TenantID:          item.TenantID,
		Queue:             item.Queue,
		TaskID:            item.TaskID,
		TaskInsertedAt:    item.TaskInsertedAt,
		ExternalID:        item.ExternalID,
		ActionID:          item.ActionID,
		StepID:            item.StepID,
		WorkflowID:        item.WorkflowID,
		WorkflowRunID:     item.WorkflowRunID,
		ScheduleTimeoutAt: item.ScheduleTimeoutAt,
		StepTimeout:       item.StepTimeout,
		Priority:          item.Priority,
		Sticky:            item.Sticky,
		DesiredWorkerID:   item.DesiredWorkerID,
		RetryCount:        item.RetryCount,
		BatchKey: pgtype.Text{
			String: item.BatchKey,
			Valid:  strings.TrimSpace(item.BatchKey) != "",
		},
	}
}

// batchGroup buffers queue items in-memory for a single batch_key within a step, flushing
// them once batch requirements are satisfied. Batch config (size/interval/max runs) lives on
// the owning BatchScheduler, since it's defined per-step rather than per-batch_key.
type batchGroup struct {
	l             *zerolog.Logger
	lastActiveAt  *time.Time
	flushDeadline *time.Time
	flushInterval *time.Duration
	batchKey      string
	buffer        []bufferedItem
}

func (g *batchGroup) getBufferedIds() []int64 {
	ids := make([]int64, 0, len(g.buffer))
	for _, item := range g.buffer {
		ids = append(ids, item.ID)
	}
	return ids
}

func (g *batchGroup) popTimedOut() []bufferedItem {
	timedOutItems := make([]bufferedItem, 0)
	remainingItems := make([]bufferedItem, 0, len(g.buffer))
	now := time.Now()
	for _, item := range g.buffer {
		if item.ScheduleTimeoutAt.Valid && item.ScheduleTimeoutAt.Time.Before(now) {
			timedOutItems = append(timedOutItems, item)
		} else {
			remainingItems = append(remainingItems, item)
		}
	}
	g.buffer = remainingItems
	return timedOutItems
}

func (g *batchGroup) popN(n int) []bufferedItem {
	popped := g.buffer[:n]
	g.buffer = g.buffer[n:]
	return popped
}

func (g *batchGroup) resetFlushDeadline() {
	if g.flushInterval != nil {
		g.flushDeadline = new(time.Now().Add(*g.flushInterval))
	}
}

type BatchScheduler struct {
	cf             *sharedConfig
	tenantId       uuid.UUID
	stepId         uuid.UUID
	repo           v1repo.BatchQueueRepository
	queueFactory   v1repo.QueueFactoryRepository
	scheduler      *Scheduler
	emitResults    func(*QueueResults)
	assignOverride assignmentFn
	maxRuns        int

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	groups map[string]*batchGroup // keyed by batch_key

	batchSize     int
	flushInterval *time.Duration
	idleTTL       time.Duration
	lastActiveAt  time.Time

	l zerolog.Logger
}

// reconcileBuffers drops cancelled/deleted items out of every group's buffer, using a single
// ListExistingBatchedQueueItemIds call across all groups rather than one call per group -- this
// is the tenant-wide (well, step-wide) analog of what fetchNewItems already does for fetching.
func (b *BatchScheduler) reconcileBuffers(ctx context.Context) {
	ctx, span := telemetry.NewSpan(ctx, "reconcile-buffers")
	defer span.End()

	if ctx.Err() != nil {
		return
	}

	ids := make([]int64, 0)
	for _, group := range b.groups {
		ids = append(ids, group.getBufferedIds()...)
	}

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "ids.checked", Value: len(ids)})

	if len(ids) == 0 {
		return
	}

	existing, err := b.repo.ListExistingBatchedQueueItemIds(ctx, ids)
	if err != nil {
		span.RecordError(err)
		b.l.Debug().Err(err).Msg("failed to reconcile batch buffers")
		return
	}

	if len(existing) == len(ids) {
		return
	}

	dropped := 0
	for _, group := range b.groups {
		if len(group.buffer) == 0 {
			continue
		}

		newBuf := make([]bufferedItem, 0, len(group.buffer))
		groupDropped := 0

		for _, item := range group.buffer {
			if _, ok := existing[item.ID]; ok {
				newBuf = append(newBuf, item)
			} else {
				groupDropped++
			}
		}

		if groupDropped > 0 {
			group.l.Debug().Int("dropped", groupDropped).Msg("dropped stale/cancelled batched queue items from buffer")
			group.buffer = newBuf
			dropped += groupDropped
		}
	}

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "ids.dropped", Value: dropped})
}

type assignmentFn func(ctx context.Context, queueItems []*sqlcv1.V1QueueItem, labels map[string][]*sqlcv1.GetDesiredLabelsRow) ([]*assignedQueueItem, []*sqlcv1.V1QueueItem, error)

// newBatchScheduler creates a scheduler for the step identified by resource.StepID. resource is
// only used to seed the shared step-level batch config (max size/interval/max runs); its
// BatchKey field is otherwise unused, since batch keys are discovered dynamically as items for
// this step are fetched.
func newBatchScheduler(
	cf *sharedConfig,
	tenantId uuid.UUID,
	resource *sqlcv1.ListDistinctBatchResourcesRow,
	queueFactory v1repo.QueueFactoryRepository,
	scheduler *Scheduler,
	emitResults func(*QueueResults),
) *BatchScheduler {
	if resource == nil {
		return nil
	}

	logger := cf.l.With().
		Str("tenant_id", tenantId.String()).
		Str("step_id", resource.StepID.String()).
		Logger()

	batchSize := resource.BatchMaxSize
	if batchSize <= 0 {
		batchSize = 1
	}

	var flushInterval *time.Duration
	if resource.BatchMaxInterval.Valid {
		flushInterval = new(time.Duration(resource.BatchMaxInterval.Int32) * time.Millisecond)
	}

	var maxRuns int32
	if resource.BatchGroupMaxRuns.Valid {
		maxRuns = resource.BatchGroupMaxRuns.Int32
	}

	return &BatchScheduler{
		cf:            cf,
		tenantId:      tenantId,
		stepId:        resource.StepID,
		repo:          cf.repo.BatchQueue().NewBatchQueue(tenantId),
		queueFactory:  queueFactory,
		scheduler:     scheduler,
		emitResults:   emitResults,
		batchSize:     int(batchSize),
		flushInterval: flushInterval,
		maxRuns:       int(maxRuns),
		idleTTL:       defaultBatchIdleTTL,
		lastActiveAt:  time.Now().UTC(),
		groups:        make(map[string]*batchGroup),
		l:             logger,
	}
}

func (b *BatchScheduler) touch() {
	b.lastActiveAt = time.Now().UTC()
}

func (b *BatchScheduler) Start(ctx context.Context) {
	if b.cancel != nil {
		// already started
		return
	}

	b.ctx, b.cancel = context.WithCancel(ctx)
	b.done = make(chan struct{})
	b.touch()

	go b.run()
}

func (b *BatchScheduler) Cleanup(ctx context.Context) error {
	if b.cancel != nil {
		b.cancel()
	}

	if b.done != nil {
		select {
		case <-b.done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

func (b *BatchScheduler) run() {
	defer close(b.done)

	if b.batchSize <= 0 {
		// fall back to singleton batching if unspecified
		b.batchSize = 1
	}

	ticker := randomticker.NewRandomTicker(defaultBatchPollInterval, defaultBatchPollInterval*2)
	defer ticker.Stop()

	for {
		if err := b.tick(); err != nil {
			if b.ctx.Err() != nil {
				return
			}

			b.l.Error().Err(err).Msg("batch scheduler tick failed")
		}

		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (b *BatchScheduler) shouldMemoryLimitFlush(group *batchGroup) (bool, int) {
	totalPayloadSize := int32(0)
	for i, item := range group.buffer {
		totalPayloadSize += item.PayloadSize
		if totalPayloadSize > maxBufferedPayloadBytes {
			// Flush everything strictly before the item that pushed us over the limit. If 1 item exceeds limit,
			// then flush that one to make some progress.
			if i == 0 {
				return true, 1
			}
			return true, i
		}
	}
	return false, 0
}

func (b *BatchScheduler) tick() error {
	ctx, span := telemetry.NewSpan(b.ctx, "batch-scheduler-tick")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant.id", Value: b.tenantId},
		telemetry.AttributeKV{Key: "step.id", Value: b.stepId},
		telemetry.AttributeKV{Key: "groups.count", Value: len(b.groups)},
	)

	// new items are added to buffer--items that are timed-out before they arrive are
	// handled by `handleScheduleTimeouts`
	timedOutItems, err := b.fetchNewItems(ctx)
	if err != nil {
		span.RecordError(err)
		return err
	}

	// The buffers now have all fresh new items, but may potentially have stale/cancelled items
	// drop those
	b.reconcileBuffers(ctx)

	// Remove timed out items that are already in buffer. These only have the slim bufferedItem
	// record, so hydrate full rows before handleScheduleTimeouts can report/delete them.
	var poppedTimedOut []bufferedItem
	for _, group := range b.groups {
		poppedTimedOut = append(poppedTimedOut, group.popTimedOut()...)
	}

	if len(poppedTimedOut) > 0 {
		hydrated, err := b.hydrateItems(poppedTimedOut)
		if err != nil {
			span.RecordError(err)
			return err
		}
		timedOutItems = append(timedOutItems, hydrated...)
	}
	// Send timeout messages for all timed-out tasks
	if err := b.handleScheduleTimeouts(ctx, timedOutItems); err != nil {
		span.RecordError(err)
		return err
	}
	for key, group := range b.groups {
		// automatically flush when payloads go over 4mb limit
		for {
			flush, count := b.shouldMemoryLimitFlush(group)
			if !flush {
				break
			}
			if err := b.flush(ctx, group, count, v1repo.FlushReasonBufferMemorySizeReached); err != nil {
				span.RecordError(err)
				return err
			}
		}

		// flush for batch size
		if b.batchSize > 0 && len(group.buffer) >= b.batchSize {
			if err := b.flush(ctx, group, b.batchSize, v1repo.FlushReasonBatchSizeReached); err != nil {
				span.RecordError(err)
				return err
			}
		}

		// flush if deadline is exceeded
		if group.flushDeadline != nil && time.Now().After(*group.flushDeadline) && len(group.buffer) > 0 {
			if err := b.flush(ctx, group, len(group.buffer), v1repo.FlushReasonIntervalElapsed); err != nil {
				span.RecordError(err)
				return err
			}
		}

		if b.groupIsIdle(group) {
			delete(b.groups, key)
		}
	}

	b.maybeStopIfIdle()

	return nil
}

func (b *BatchScheduler) fetchNewItems(ctx context.Context) ([]*sqlcv1.V1BatchedQueueItem, error) {
	ctx, span := telemetry.NewSpan(ctx, "fetch-new-items")
	defer span.End()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	limit := pgtype.Int4{
		Int32: batchFetchLimit,
		Valid: true,
	}

	items, err := b.repo.ListBatchedQueueItems(ctx, b.stepId, limit.Int32)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "items.fetched", Value: len(items)})

	if len(items) == 0 {
		return nil, nil
	}

	b.touch()

	now := time.Now().UTC()

	itemsByGroup := make(map[string][]*sqlcv1.V1BatchedQueueItem)
	for _, item := range items {
		if item == nil {
			continue
		}
		itemsByGroup[item.BatchKey] = append(itemsByGroup[item.BatchKey], item)
	}
	timedOutItems := make([]*sqlcv1.V1BatchedQueueItem, 0)
	newItemsTotal := 0
	for batchKey, groupItems := range itemsByGroup {
		group, ok := b.groups[batchKey]
		if !ok {
			group = &batchGroup{
				batchKey:      batchKey,
				lastActiveAt:  new(now),
				flushInterval: b.flushInterval,
				l: new(b.l.With().
					Str("batch_key", batchKey).
					Logger()),
			}
			group.resetFlushDeadline()
			b.groups[batchKey] = group
		}

		// ListBatchedQueueItems has no cursor: it keeps returning the same still-pending rows on
		// every call until they're actually flushed. Without this check, a group that doesn't flush
		// on the very next tick (e.g. flushInterval > defaultBatchPollInterval) would have duplicates
		alreadyBuffered := make(map[int64]struct{}, len(group.buffer))
		for _, item := range group.buffer {
			alreadyBuffered[item.ID] = struct{}{}
		}

		// newItems holds the full rows only long enough to emit "waiting" events and derive the
		// slim bufferedItem records actually kept in group.buffer.
		newItems := make([]*sqlcv1.V1BatchedQueueItem, 0, len(groupItems))
		for _, item := range groupItems {
			if item == nil {
				continue
			}

			if _, ok := alreadyBuffered[item.ID]; ok {
				continue
			}

			// Check if item has exceeded schedule timeout
			if item.ScheduleTimeoutAt.Valid && item.ScheduleTimeoutAt.Time.Before(now) {
				timedOutItems = append(timedOutItems, item)
				continue
			}
			newItems = append(newItems, item)
		}
		for _, item := range newItems {
			group.buffer = append(group.buffer, toBufferedItem(item))
		}
		b.emitWaitingEvents(group, newItems)
		if len(newItems) > 0 {
			group.lastActiveAt = &now
		}
		newItemsTotal += len(newItems)
	}

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "items.new", Value: newItemsTotal},
		telemetry.AttributeKV{Key: "items.timed_out", Value: len(timedOutItems)},
		telemetry.AttributeKV{Key: "groups.touched", Value: len(itemsByGroup)},
	)

	return timedOutItems, nil
}

// groupIsIdle reports whether a batch_key group looks safe to evict from memory. It relies on
// fetchNewItems having already run earlier in the same tick: since ListBatchedQueueItems has no
// cursor and returns every still-pending row for the step on each call, a group with pending DB
// rows would have been repopulated (and its lastActiveAt touched) this same tick. No separate
// confirming DB read is needed here, unlike the whole-scheduler check in maybeStopIfIdle.
func (b *BatchScheduler) groupIsIdle(group *batchGroup) bool {
	if len(group.buffer) > 0 || group.flushDeadline != nil {
		return false
	}

	if b.idleTTL <= 0 {
		return false
	}
	if group.lastActiveAt == nil {
		return false
	}
	if time.Since(*group.lastActiveAt) < b.idleTTL {
		return false
	}

	if b.maxRuns > 0 && b.cf != nil && b.cf.taskRepo != nil && strings.TrimSpace(group.batchKey) != "" {
		cnt, err := b.cf.taskRepo.CountActiveTaskBatchRuns(
			b.ctx,
			b.tenantId.String(),
			b.stepId.String(),
			strings.TrimSpace(group.batchKey),
		)
		if err != nil {
			group.l.Debug().Err(err).Msg("idle check failed to count active batch runs")
			return false
		}

		if cnt > 0 {
			group.lastActiveAt = new(time.Now().UTC())
			return false
		}
	}

	return true
}

func (b *BatchScheduler) maybeStopIfIdle() {
	if b.ctx == nil || b.cancel == nil {
		return
	}

	if b.ctx.Err() != nil {
		return
	}

	if len(b.groups) > 0 {
		return
	}

	if b.idleTTL <= 0 {
		return
	}

	if time.Since(b.lastActiveAt) < b.idleTTL {
		return
	}

	// Confirm there are no DB items left for this step at all.
	rows, err := b.repo.ListBatchedQueueItems(b.ctx, b.stepId, 1)
	if err != nil {
		b.l.Debug().Err(err).Msg("idle check failed to list batched queue items")
		return
	}

	if len(rows) > 0 {
		b.touch()
		return
	}

	b.l.Info().Msg("batch scheduler idle; stopping")
	b.cancel()
}

func (b *BatchScheduler) flush(ctx context.Context, group *batchGroup, count int, reason v1repo.BatchFlushReason) error {
	if len(group.buffer) == 0 || count <= 0 {
		return nil
	}

	ctx, span := telemetry.NewSpan(ctx, "batch-flush")
	defer span.End()

	if count > len(group.buffer) {
		count = len(group.buffer)
	}

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "batch_key", Value: group.batchKey},
		telemetry.AttributeKV{Key: "flush.reason", Value: string(reason)},
		telemetry.AttributeKV{Key: "count", Value: count},
	)

	toFlush, err := b.hydrateItems(group.popN(count))
	if err != nil {
		span.RecordError(err)
		return err
	}

	remaining, err := b.assignAndDispatch(ctx, group, toFlush, reason)
	if err != nil {
		span.RecordError(err)
		return err
	}

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "remaining", Value: len(remaining)})

	if len(remaining) > 0 {
		// Requeue remaining items at the front to preserve ordering.
		remainingRefs := make([]bufferedItem, 0, len(remaining))
		for _, item := range remaining {
			if item == nil {
				continue
			}
			remainingRefs = append(remainingRefs, toBufferedItem(item))
		}
		group.buffer = append(remainingRefs, group.buffer...)
	}

	group.resetFlushDeadline()

	group.lastActiveAt = new(time.Now().UTC())
	b.touch()

	return nil
}

// hydrateItems fetches full rows for a set of slim bufferedItem records. Refs whose row no
// longer exists in the DB (cancelled/deleted since being buffered) are simply absent from the
// result -- callers should treat that as "already gone" rather than an error.
func (b *BatchScheduler) hydrateItems(refs []bufferedItem) ([]*sqlcv1.V1BatchedQueueItem, error) {
	if len(refs) == 0 {
		return nil, nil
	}

	ids := make([]int64, len(refs))
	for i, ref := range refs {
		ids[i] = ref.ID
	}

	items, err := b.repo.GetBatchedQueueItemsByIds(b.ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("hydrate batched queue items: %w", err)
	}

	return items, nil
}

func (b *BatchScheduler) batchMaxIntervalMs() int32 {
	if b.flushInterval == nil || *b.flushInterval <= 0 {
		return 0
	}
	// #nosec G115
	return int32(*b.flushInterval / time.Millisecond)
}

func (b *BatchScheduler) emitWaitingEvents(group *batchGroup, newItems []*sqlcv1.V1BatchedQueueItem) {
	if b.emitResults == nil || len(newItems) == 0 {
		return
	}

	batchMaxIntervalMs := b.batchMaxIntervalMs()

	pending := (int32)(len(group.buffer))

	buffered := make([]*v1repo.AssignedItem, 0, len(newItems))

	for _, item := range newItems {
		if item == nil {
			continue
		}

		queueItem := toV1QueueItem(item)

		triggeredAt := time.Now().UTC()
		if item.InsertedAt.Valid {
			triggeredAt = item.InsertedAt.Time.UTC()
		}

		metaBatchKey := ""
		if queueItem.BatchKey.Valid {
			metaBatchKey = strings.TrimSpace(queueItem.BatchKey.String)
		}

		buffered = append(buffered, &v1repo.AssignedItem{
			QueueItem: queueItem,
			Batch: &v1repo.BatchAssignmentMetadata{
				State:                        "waiting",
				TriggeredAt:                  triggeredAt,
				ConfiguredBatchMaxSize:       int32(b.batchSize),
				ConfiguredBatchMaxIntervalMs: batchMaxIntervalMs,
				ConfiguredBatchGroupMaxRuns:  int32(b.maxRuns),
				Pending:                      pending,
				NextFlushAt:                  group.flushDeadline,
				BatchID:                      "",
				StepID:                       queueItem.StepID.String(),
				ActionID:                     queueItem.ActionID,
				BatchGroupKey:                metaBatchKey,
			},
		})
	}

	if len(buffered) == 0 {
		return
	}

	b.emitResults(&QueueResults{
		TenantId: b.tenantId,
		Buffered: buffered,
	})
}

func (b *BatchScheduler) assignQueueItems(
	ctx context.Context,
	queueItems []*sqlcv1.V1QueueItem,
	labels map[string][]*sqlcv1.GetDesiredLabelsRow,
) ([]*assignedQueueItem, []*sqlcv1.V1QueueItem, error) {
	if b.assignOverride != nil {
		return b.assignOverride(ctx, queueItems, labels)
	}

	// Batch flush scheduling is intentionally a separate path: we only need ONE slot for the whole batch.
	if len(queueItems) == 0 || queueItems[0] == nil {
		return nil, nil, nil
	}

	schedulingItem := queueItems[0]
	stepKey := schedulingItem.StepID.String()
	stepLabels := labels[stepKey]

	res, err := b.scheduler.tryAssignBatchQueueItem(ctx, schedulingItem, stepLabels)
	if err != nil {
		return nil, nil, err
	}

	if !res.succeeded {
		return nil, []*sqlcv1.V1QueueItem{schedulingItem}, nil
	}

	return []*assignedQueueItem{{
		AckId:     res.ackId,
		WorkerId:  res.workerId,
		QueueItem: schedulingItem,
	}}, nil, nil
}

func (b *BatchScheduler) assignAndDispatch(ctx context.Context, group *batchGroup, items []*sqlcv1.V1BatchedQueueItem, reason v1repo.BatchFlushReason) ([]*sqlcv1.V1BatchedQueueItem, error) {
	ctx, span := telemetry.NewSpan(ctx, "assign-and-dispatch")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant.id", Value: b.tenantId},
		telemetry.AttributeKV{Key: "step.id", Value: b.stepId},
		telemetry.AttributeKV{Key: "batch_key", Value: group.batchKey},
		telemetry.AttributeKV{Key: "flush.reason", Value: string(reason)},
		telemetry.AttributeKV{Key: "items.count", Value: len(items)},
	)

	if b.scheduler == nil {
		err := fmt.Errorf("batch scheduler missing core scheduler")
		span.RecordError(err)
		return items, err
	}

	if b.queueFactory == nil {
		err := fmt.Errorf("batch scheduler missing queue factory")
		span.RecordError(err)
		return items, err
	}

	if b.emitResults == nil {
		err := fmt.Errorf("batch scheduler missing results emitter")
		span.RecordError(err)
		return items, err
	}

	if len(items) == 0 {
		return nil, nil
	}

	queueToItems := make(map[string][]*sqlcv1.V1BatchedQueueItem)

	for _, item := range items {
		if item == nil {
			continue
		}
		queueToItems[item.Queue] = append(queueToItems[item.Queue], item)
	}

	remaining := make([]*sqlcv1.V1BatchedQueueItem, 0)
	allAssigned := make([]*v1repo.AssignedItem, 0)
	allAckIds := make([]int, 0)

	for queueName, itemGroup := range queueToItems {
		group.l.Debug().
			Str("queue", queueName).
			Int("group_size", len(itemGroup)).
			Msg("processing batched queue group")
		queueRepo := b.queueFactory.NewQueue(b.tenantId, queueName)
		if queueRepo == nil {
			remaining = append(remaining, itemGroup...)
			continue
		}

		queueItems := make([]*sqlcv1.V1QueueItem, 0, len(itemGroup))
		queueItemsByID := make(map[int64]*sqlcv1.V1QueueItem, len(itemGroup))

		stepID := b.stepId
		actionID := ""
		batchKey := strings.TrimSpace(group.batchKey)

		if len(itemGroup) > 0 && itemGroup[0] != nil {
			if itemGroup[0].StepID != uuid.Nil {
				stepID = itemGroup[0].StepID
			}

			actionID = itemGroup[0].ActionID

			if trimmed := strings.TrimSpace(itemGroup[0].BatchKey); trimmed != "" {
				batchKey = trimmed
			}
		}

		for _, batched := range itemGroup {
			if batched == nil {
				continue
			}

			queueItem := toV1QueueItem(batched)

			queueItems = append(queueItems, queueItem)
			queueItemsByID[queueItem.ID] = queueItem
		}

		if len(queueItems) == 0 {
			queueRepo.Cleanup()
			continue
		}

		group.l.Debug().
			Str("queue", queueName).
			Int("queue_items", len(queueItems)).
			Msg("built queue items for batched group")

		schedulingItem := queueItems[0]

		stepLabelsMap, err := queueRepo.GetDesiredLabels(ctx, nil, []uuid.UUID{b.stepId})
		if err != nil {
			queueRepo.Cleanup()
			err = fmt.Errorf("get desired labels: %w", err)
			span.RecordError(err)
			return items, err
		}

		stepKey := b.stepId.String()
		assigned, failedQueueItems, err := b.assignQueueItems(ctx, []*sqlcv1.V1QueueItem{schedulingItem}, map[string][]*sqlcv1.GetDesiredLabelsRow{
			stepKey: stepLabelsMap[b.stepId],
		})
		if err != nil {
			queueRepo.Cleanup()
			return items, err
		}

		queueRepo.Cleanup()

		ackIds := make([]int, 0, len(assigned))

		requeueGroup := func() {
			if len(ackIds) > 0 {
				b.scheduler.nack(ackIds)
			}

			remaining = append(remaining, itemGroup...)
			b.emitWaitingEvents(group, itemGroup)
			group.resetFlushDeadline()
		}

		if len(assigned) == 0 || len(failedQueueItems) > 0 {
			requeueGroup()
			continue
		}

		assignedItem := assigned[0]
		if assignedItem == nil || assignedItem.QueueItem == nil || assignedItem.WorkerId == uuid.Nil {
			requeueGroup()
			continue
		}

		workerID := assignedItem.WorkerId
		if assignedItem.AckId > 0 {
			ackIds = append(ackIds, assignedItem.AckId)
		}

		batchKeyNormalized := strings.TrimSpace(batchKey)
		if b.maxRuns > 0 && batchKeyNormalized != "" && b.cf.taskRepo != nil {
			stepIDStr := stepID.String()
			tenantIDStr := b.tenantId.String()

			activeCount, err := b.cf.taskRepo.CountActiveTaskBatchRuns(ctx, tenantIDStr, stepIDStr, batchKeyNormalized)
			if err != nil {
				group.l.Error().Err(err).Msg("failed counting active batch runs; deferring batch")
				requeueGroup()
				continue
			}

			if activeCount >= b.maxRuns {
				requeueGroup()
				continue
			}
		}

		batchID := uuid.NewString()

		batchAssignments := make([]*v1repo.BatchAssignment, 0, len(itemGroup))
		queueResultsByTaskID := make(map[int64]*v1repo.AssignedItem, len(itemGroup))
		triggeredAt := time.Now().UTC()
		batchMaxIntervalMs := b.batchMaxIntervalMs()

		for _, batched := range itemGroup {
			if batched == nil {
				continue
			}

			queueItem := queueItemsByID[batched.ID]
			if queueItem == nil {
				continue
			}

			batchAssignments = append(batchAssignments, &v1repo.BatchAssignment{
				BatchQueueItemID: batched.ID,
				TaskID:           batched.TaskID,
				TaskInsertedAt:   batched.TaskInsertedAt,
				RetryCount:       batched.RetryCount,
				WorkerID:         workerID,
				BatchID:          batchID,
				StepID:           stepID,
				ActionID:         actionID,
				BatchKey:         batchKeyNormalized,
			})

			queueResultsByTaskID[queueItem.TaskID] = &v1repo.AssignedItem{
				WorkerId:  workerID,
				QueueItem: queueItem,
				Batch: &v1repo.BatchAssignmentMetadata{
					State:                        "flushed",
					Reason:                       reason,
					TriggeredAt:                  triggeredAt,
					ConfiguredBatchMaxSize:       int32(b.batchSize),
					ConfiguredBatchMaxIntervalMs: batchMaxIntervalMs,
					ConfiguredBatchGroupMaxRuns:  int32(b.maxRuns),
					Pending:                      0,
					NextFlushAt:                  nil,
					BatchID:                      batchID,
					StepID:                       stepID.String(),
					ActionID:                     actionID,
					BatchGroupKey:                batchKeyNormalized,
				},
			}
		}

		if len(batchAssignments) == 0 {
			if len(ackIds) > 0 {
				b.scheduler.nack(ackIds)
			}
			continue
		}

		// Reserving the batch run slot and committing these assignments happen atomically in one transaction
		// to make sure there isn't a race with maxRuns that could lead it to not being respected with multiple
		// concurrent schedulers
		reserved, succeededAssignments, err := b.repo.ReserveAndCommitBatchRun(
			ctx, b.tenantId, stepID, actionID, batchKeyNormalized, batchID, b.maxRuns, batchAssignments,
		)
		if err != nil {
			if len(ackIds) > 0 {
				b.scheduler.nack(ackIds)
			}
			err = fmt.Errorf("reserve and commit batch assignments: %w", err)
			span.RecordError(err)
			return items, err
		}

		if !reserved {
			requeueGroup()
			continue
		}

		// Only emit/ack tasks that were actually assigned (e.g. drop cancellations).
		queueResults := make([]*v1repo.AssignedItem, 0, len(succeededAssignments))
		for _, a := range succeededAssignments {
			if a == nil {
				continue
			}
			if item, ok := queueResultsByTaskID[a.TaskID]; ok && item != nil {
				queueResults = append(queueResults, item)
			}
		}

		if len(queueResults) == 0 {
			// Nothing actually assigned; release the slot and continue.
			if len(ackIds) > 0 {
				b.scheduler.nack(ackIds)
			}
			continue
		}

		allAssigned = append(allAssigned, queueResults...)
		allAckIds = append(allAckIds, ackIds...)
	}

	if len(allAckIds) > 0 {
		b.scheduler.ack(allAckIds)
	}

	if len(allAssigned) > 0 {
		result := &QueueResults{
			TenantId: b.tenantId,
			Assigned: allAssigned,
		}
		b.emitResults(result)
	}

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "batch.assigned", Value: len(allAssigned)},
		telemetry.AttributeKV{Key: "batch.remaining", Value: len(remaining)},
	)

	return remaining, nil
}

func (b *BatchScheduler) handleScheduleTimeouts(ctx context.Context, timedOutItems []*sqlcv1.V1BatchedQueueItem) error {
	if len(timedOutItems) == 0 {
		return nil
	}

	ctx, span := telemetry.NewSpan(ctx, "handle-schedule-timeouts")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "items.timed_out", Value: len(timedOutItems)})

	idsToDelete := make([]int64, 0, len(timedOutItems))
	schedulingTimedOut := make([]*sqlcv1.V1QueueItem, 0, len(timedOutItems))

	for _, item := range timedOutItems {
		if item == nil {
			continue
		}

		idsToDelete = append(idsToDelete, item.ID)

		schedulingTimedOut = append(schedulingTimedOut, toV1QueueItem(item))
	}

	// Delete the timed out items from the batched queue
	if len(idsToDelete) > 0 {
		if err := b.repo.DeleteBatchedQueueItems(ctx, idsToDelete); err != nil {
			err = fmt.Errorf("failed to delete timed out batched queue items: %w", err)
			span.RecordError(err)
			return err
		}
	}

	// Emit the timed out items so they can be processed downstream
	if b.emitResults != nil && len(schedulingTimedOut) > 0 {
		b.emitResults(&QueueResults{
			TenantId:           b.tenantId,
			SchedulingTimedOut: schedulingTimedOut,
		})
	}

	b.l.Info().Int("count", len(timedOutItems)).Msg("batched tasks timed out during scheduling")

	return nil
}
