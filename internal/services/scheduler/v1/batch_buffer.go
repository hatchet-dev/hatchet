package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog"

	repov1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
)

type batchFlushFunc func(context.Context, *batchFlushRequest) error
type batchFlushDecisionFunc func(context.Context, *batchFlushRequest) (bool, error)

type batchConfig struct {
	batchSize     int
	flushInterval time.Duration
	maxRuns       int
}

type batchFlushReason string

const (
	flushReasonBatchSizeReached  batchFlushReason = "batch_size_reached"
	flushReasonWorkerChanged     batchFlushReason = "worker_changed"
	flushReasonDispatcherChanged batchFlushReason = "dispatcher_changed"
	flushReasonIntervalElapsed   batchFlushReason = "interval_elapsed"
	flushReasonBufferDrained     batchFlushReason = "buffer_drained"
)

type batchBufferManager struct {
	l        *zerolog.Logger
	flushFn  batchFlushFunc
	canFlush batchFlushDecisionFunc

	mu      sync.Mutex
	buffers map[batchBufferKey]*batchBuffer
}

type batchBufferKey struct {
	tenantID     string
	stepID       string
	actionID     string
	dispatcherID string
	workerID     string
	batchKey     string
}

type batchFlushRequest struct {
	TenantID     string
	StepID       string
	ActionID     string
	DispatcherID string
	WorkerID     string
	BatchID      string
	BatchSize    int
	Items        []*repov1.AssignedItem
	FlushReason  batchFlushReason
	TriggeredAt  time.Time

	ConfiguredBatchSize     int
	ConfiguredFlushInterval time.Duration
	BatchKey                string
	MaxRuns                 int
}

type batchAddResult struct {
	Pending        int
	Flushed        bool
	FlushReason    *batchFlushReason
	NextFlushAt    *time.Time
	PendingBatchID string
	FlushedBatchID string
}

func newBatchBufferManager(l *zerolog.Logger, flushFn batchFlushFunc, canFlush batchFlushDecisionFunc) *batchBufferManager {
	return &batchBufferManager{
		l:        l,
		flushFn:  flushFn,
		canFlush: canFlush,
		buffers:  make(map[batchBufferKey]*batchBuffer),
	}
}

func (m *batchBufferManager) Add(ctx context.Context, tenantID, stepID, actionID, dispatcherID, workerID, batchKey string, cfg batchConfig, item *repov1.AssignedItem) (batchAddResult, error) {
	var result batchAddResult

	if cfg.batchSize <= 0 {
		cfg.batchSize = 1
	}

	key := batchBufferKey{
		tenantID:     tenantID,
		stepID:       stepID,
		actionID:     actionID,
		dispatcherID: dispatcherID,
		workerID:     workerID,
		batchKey:     batchKey,
	}

	buf := m.getOrCreateBuffer(key, cfg)

	requests, addResult, err := buf.add(ctx, dispatcherID, workerID, item)

	if err != nil {
		return result, err
	}

	for _, req := range requests {
		if req == nil || len(req.Items) == 0 {
			continue
		}

		if err := m.flushFn(ctx, req); err != nil {
			return result, err
		}
	}

	return addResult, nil
}

func (m *batchBufferManager) FlushAll(ctx context.Context) error {
	m.mu.Lock()
	buffers := make([]*batchBuffer, 0, len(m.buffers))

	for _, buf := range m.buffers {
		buffers = append(buffers, buf)
	}

	m.mu.Unlock()

	var result error

	for _, buf := range buffers {
		req := buf.drainAll(flushReasonBufferDrained)

		if req == nil || len(req.Items) == 0 {
			continue
		}

		if err := m.flushFn(ctx, req); err != nil {
			result = multierror.Append(result, err)
		}
	}

	return result
}

func (m *batchBufferManager) flushFromTimer(key batchBufferKey) {
	buf := m.getBuffer(key)

	if buf == nil {
		return
	}

	req, err := buf.drainForTimer(context.Background())

	if err != nil {
		m.l.Error().
			Err(err).
			Str("tenant_id", key.tenantID).
			Str("action_id", key.actionID).
			Msg("failed to evaluate batch flush on timer")
		return
	}

	if req == nil || len(req.Items) == 0 {
		return
	}

	if err := m.flushFn(context.Background(), req); err != nil {
		m.l.Error().
			Err(err).
			Str("tenant_id", key.tenantID).
			Str("action_id", key.actionID).
			Str("batch_key", key.batchKey).
			Msg("failed to flush batch buffer on timer")
	}
}

func (m *batchBufferManager) getOrCreateBuffer(key batchBufferKey, cfg batchConfig) *batchBuffer {
	m.mu.Lock()
	defer m.mu.Unlock()

	if buf, ok := m.buffers[key]; ok {
		return buf
	}

	buf := newBatchBuffer(m, key, cfg)
	m.buffers[key] = buf
	return buf
}

func (m *batchBufferManager) getBuffer(key batchBufferKey) *batchBuffer {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.buffers[key]
}

type batchBuffer struct {
	manager *batchBufferManager
	key     batchBufferKey
	config  batchConfig

	mu             sync.Mutex
	dispatcherID   string
	workerID       string
	items          []*repov1.AssignedItem
	timer          *time.Timer
	flushDeadline  *time.Time
	currentBatchID string
	nextBatchID    string
}

func newBatchBuffer(manager *batchBufferManager, key batchBufferKey, cfg batchConfig) *batchBuffer {
	return &batchBuffer{
		manager:      manager,
		key:          key,
		config:       cfg,
		dispatcherID: key.dispatcherID,
		workerID:     key.workerID,
		items:        make([]*repov1.AssignedItem, 0),
		nextBatchID:  uuid.NewString(),
	}
}

func (b *batchBuffer) add(ctx context.Context, dispatcherID, workerID string, item *repov1.AssignedItem) ([]*batchFlushRequest, batchAddResult, error) {
	result := batchAddResult{}

	b.mu.Lock()
	defer b.mu.Unlock()

	if dispatcherID == "" {
		return nil, result, fmt.Errorf("dispatcher id required for batch buffering")
	}

	if workerID == "" {
		return nil, result, fmt.Errorf("worker id required for batch buffering")
	}

	if len(b.items) == 0 {
		b.ensureBatchIDLocked()
	}

	var requests []*batchFlushRequest
	flushedWithNewItem := false
	var flushedBatchID string

	if b.dispatcherID == "" {
		b.dispatcherID = dispatcherID
	}

	if b.workerID == "" {
		b.workerID = workerID
	}

	if b.dispatcherID != dispatcherID || b.workerID != workerID {
		if len(b.items) > 0 {
			reason := flushReasonWorkerChanged
			if b.dispatcherID != dispatcherID {
				reason = flushReasonDispatcherChanged
			}

			if req, flushed, err := b.tryFlushLocked(ctx, reason); err != nil {
				return nil, result, err
			} else if flushed {
				requests = append(requests, req)
			}
		}

		b.dispatcherID = dispatcherID
		b.workerID = workerID
	}

	b.items = append(b.items, item)

	if len(b.items) >= b.config.batchSize {
		if req, flushed, err := b.tryFlushLocked(ctx, flushReasonBatchSizeReached); err != nil {
			return nil, result, err
		} else if flushed {
			requests = append(requests, req)
			flushedWithNewItem = true
			flushedBatchID = req.BatchID
		}
	} else if b.config.flushInterval > 0 && b.timer == nil {
		b.startTimerLocked()
	}

	result.Pending = len(b.items)
	result.Flushed = flushedWithNewItem
	if len(b.items) > 0 {
		result.PendingBatchID = b.currentBatchID
	} else {
		result.PendingBatchID = b.nextBatchID
	}

	if flushedWithNewItem {
		reason := flushReasonBatchSizeReached
		result.FlushReason = &reason
		result.FlushedBatchID = flushedBatchID
	}

	if b.flushDeadline != nil {
		deadlineCopy := *b.flushDeadline
		result.NextFlushAt = &deadlineCopy
	}

	return requests, result, nil
}

func (b *batchBuffer) drainForTimer(ctx context.Context) (*batchFlushRequest, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.items) == 0 {
		b.stopTimerLocked()
		return nil, nil
	}

	req, flushed, err := b.tryFlushLocked(ctx, flushReasonIntervalElapsed)
	if err != nil {
		return nil, err
	}

	if !flushed {
		if b.config.flushInterval > 0 && b.timer == nil {
			b.startTimerLocked()
		}
		return nil, nil
	}

	return req, nil
}

func (b *batchBuffer) drainAll(reason batchFlushReason) *batchFlushRequest {
	b.mu.Lock()
	defer b.mu.Unlock()

	req := b.prepareFlushLocked(reason)
	if req != nil {
		b.commitFlushLocked()
	}
	return req
}

func (b *batchBuffer) prepareFlushLocked(reason batchFlushReason) *batchFlushRequest {
	if len(b.items) == 0 {
		return nil
	}

	// Every flush must carry a freshly generated identifier so runtime metadata and
	// monitoring consumers never observe reused batch IDs, even when a buffer flushes
	// repeatedly in rapid succession.
	if b.currentBatchID == "" {
		b.ensureBatchIDLocked()
	}

	copied := make([]*repov1.AssignedItem, len(b.items))
	copy(copied, b.items)

	return &batchFlushRequest{
		TenantID:                b.key.tenantID,
		StepID:                  b.key.stepID,
		ActionID:                b.key.actionID,
		DispatcherID:            b.dispatcherID,
		WorkerID:                b.workerID,
		BatchID:                 b.currentBatchID,
		BatchSize:               len(copied),
		Items:                   copied,
		FlushReason:             reason,
		TriggeredAt:             time.Now().UTC(),
		ConfiguredBatchSize:     b.config.batchSize,
		ConfiguredFlushInterval: b.config.flushInterval,
		BatchKey:                b.key.batchKey,
		MaxRuns:                 b.config.maxRuns,
	}
}

func (b *batchBuffer) commitFlushLocked() {
	b.items = nil
	b.stopTimerLocked()
	b.currentBatchID = ""
}

func (b *batchBuffer) tryFlushLocked(ctx context.Context, reason batchFlushReason) (*batchFlushRequest, bool, error) {
	req := b.prepareFlushLocked(reason)
	if req == nil {
		return nil, false, nil
	}

	allowed := true
	var err error

	if b.manager.canFlush != nil {
		allowed, err = b.manager.canFlush(ctx, req)
		if err != nil {
			return nil, false, err
		}
	}

	if !allowed {
		b.startRetryTimerLocked()
		return nil, false, nil
	}

	b.commitFlushLocked()

	return req, true, nil
}

func (b *batchBuffer) ensureBatchIDLocked() {
	if b.currentBatchID == "" {
		if b.nextBatchID == "" {
			b.nextBatchID = uuid.NewString()
		}
		b.currentBatchID = b.nextBatchID
		b.nextBatchID = uuid.NewString()
	}
}

func (b *batchBuffer) startTimerLocked() {
	if b.config.flushInterval <= 0 {
		return
	}

	deadline := time.Now().Add(b.config.flushInterval)
	b.flushDeadline = &deadline

	b.timer = time.AfterFunc(b.config.flushInterval, func() {
		b.manager.flushFromTimer(b.key)
	})
}

func (b *batchBuffer) startRetryTimerLocked() {
	if b.timer != nil {
		return
	}

	if b.config.flushInterval > 0 {
		b.startTimerLocked()
		return
	}

	interval := time.Second
	deadline := time.Now().Add(interval)
	b.flushDeadline = &deadline

	b.timer = time.AfterFunc(interval, func() {
		b.manager.flushFromTimer(b.key)
	})
}

func (b *batchBuffer) stopTimerLocked() {
	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}

	b.flushDeadline = nil
}
