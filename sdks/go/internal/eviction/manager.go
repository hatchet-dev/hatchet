package eviction

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// DurableEvictionConfig controls the eviction manager behavior.
type DurableEvictionConfig struct {
	CheckInterval              time.Duration
	ReserveSlots               int
	MinWaitForCapacityEviction time.Duration
}

// DefaultDurableEvictionConfig provides sensible defaults.
var DefaultDurableEvictionConfig = DurableEvictionConfig{
	CheckInterval:              1 * time.Second,
	ReserveSlots:               0,
	MinWaitForCapacityEviction: 10 * time.Second,
}

// CancelLocalFn cancels a local durable run by its action key.
type CancelLocalFn func(key string)

// RequestEvictionWithAckFn sends an eviction request to the server and waits for ack.
type RequestEvictionWithAckFn func(ctx context.Context, key string, rec *DurableRunRecord) error

// DurableEvictionManager periodically checks for and evicts durable runs.
type DurableEvictionManager struct {
	cancelLocal  CancelLocalFn
	requestEvict RequestEvictionWithAckFn
	cache        *DurableEvictionCache
	l            *zerolog.Logger
	cancel       context.CancelFunc
	config       DurableEvictionConfig
	durableSlots int
	mu           sync.Mutex
}

// NewDurableEvictionManager creates a new eviction manager.
func NewDurableEvictionManager(
	durableSlots int,
	cancelLocal CancelLocalFn,
	requestEvict RequestEvictionWithAckFn,
	config DurableEvictionConfig,
	l *zerolog.Logger,
) *DurableEvictionManager {
	return &DurableEvictionManager{
		durableSlots: durableSlots,
		cancelLocal:  cancelLocal,
		requestEvict: requestEvict,
		config:       config,
		cache:        NewDurableEvictionCache(),
		l:            l,
	}
}

// Cache returns the underlying cache for direct access.
func (m *DurableEvictionManager) Cache() *DurableEvictionCache {
	return m.cache
}

// Start begins the eviction check loop. Safe to call multiple times.
func (m *DurableEvictionManager) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	go m.runLoop(ctx)
}

// Stop halts the eviction check loop.
func (m *DurableEvictionManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

// RegisterRun adds a run to the eviction cache.
func (m *DurableEvictionManager) RegisterRun(key, stepRunID string, invocationCount int, policy *EvictionPolicy) {
	m.cache.RegisterRun(key, stepRunID, invocationCount, time.Now().UTC(), policy)
}

// UnregisterRun removes a run from the eviction cache.
func (m *DurableEvictionManager) UnregisterRun(key string) {
	m.cache.UnregisterRun(key)
}

// MarkWaiting marks a run as waiting.
func (m *DurableEvictionManager) MarkWaiting(key, waitKind, resourceID string) {
	m.cache.MarkWaiting(key, time.Now().UTC(), waitKind, resourceID)
}

// MarkActive marks a run as active (no longer waiting).
func (m *DurableEvictionManager) MarkActive(key string) {
	m.cache.MarkActive(key, time.Now().UTC())
}

func (m *DurableEvictionManager) evictRun(key string) {
	m.cancelLocal(key)
	m.cache.UnregisterRun(key)
}

func (m *DurableEvictionManager) runLoop(ctx context.Context) {
	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.tickSafe(ctx)
		}
	}
}

func (m *DurableEvictionManager) tickSafe(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil && m.l != nil {
			m.l.Error().Interface("panic", r).Msg("DurableEvictionManager: panic in eviction loop")
		}
	}()

	if err := m.tick(ctx); err != nil && m.l != nil {
		m.l.Error().Err(err).Msg("DurableEvictionManager: error in eviction loop")
	}
}

func (m *DurableEvictionManager) tick(ctx context.Context) error {
	evicted := make(map[string]bool)

	for {
		now := time.Now().UTC()
		key := m.cache.SelectEvictionCandidate(
			now,
			m.durableSlots,
			m.config.ReserveSlots,
			m.config.MinWaitForCapacityEviction,
		)
		if key == "" {
			return nil
		}
		if evicted[key] {
			return nil
		}
		evicted[key] = true

		rec := m.cache.Get(key)
		if rec == nil || rec.EvictionPolicy == nil {
			continue
		}

		if m.l != nil {
			m.l.Debug().
				Str("step_run_id", rec.StepRunID).
				Str("wait_kind", rec.WaitKind).
				Str("resource_id", rec.WaitResourceID).
				Msg("DurableEvictionManager: evicting durable run")
		}

		if err := m.requestEvict(ctx, key, rec); err != nil {
			if m.l != nil {
				m.l.Error().Err(err).Str("step_run_id", rec.StepRunID).Msg("DurableEvictionManager: failed to request eviction")
			}
			continue
		}

		m.evictRun(key)
	}
}

// HandleServerEviction processes a server-initiated eviction notification.
func (m *DurableEvictionManager) HandleServerEviction(stepRunID string, invocationCount int) {
	keyPtr := m.cache.FindKeyByStepRunID(stepRunID)
	if keyPtr == nil {
		return
	}
	key := *keyPtr

	if rec := m.cache.Get(key); rec != nil && rec.InvocationCount != invocationCount {
		return
	}

	if m.l != nil {
		m.l.Info().
			Str("step_run_id", stepRunID).
			Int("invocation_count", invocationCount).
			Msg("DurableEvictionManager: server-initiated eviction")
	}

	m.evictRun(key)
}

// EvictAllWaiting evicts every currently-waiting durable run. Used during graceful shutdown.
func (m *DurableEvictionManager) EvictAllWaiting(ctx context.Context) int {
	m.Stop()

	waiting := m.cache.GetAllWaiting()
	evicted := 0

	for _, rec := range waiting {
		rec.mu.Lock()
		rec.EvictionReason = buildEvictionReason(EvictionCauseWorkerShutdown, rec, 0)
		rec.mu.Unlock()

		if m.l != nil {
			m.l.Debug().
				Str("step_run_id", rec.StepRunID).
				Str("wait_kind", rec.WaitKind).
				Msg("DurableEvictionManager: shutdown-evicting durable run")
		}

		if err := m.requestEvict(ctx, rec.Key, rec); err != nil {
			if m.l != nil {
				m.l.Error().Err(err).Str("step_run_id", rec.StepRunID).Msg("DurableEvictionManager: failed to send eviction")
			}
		}

		m.evictRun(rec.Key)
		evicted++
	}

	return evicted
}
