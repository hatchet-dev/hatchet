package eviction

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// ErrEvicted is the cause set on a run's context when it is evicted by the manager.
var ErrEvicted = errors.New("durable run evicted")

// ManagerConfig holds per-worker eviction settings.
type ManagerConfig struct {
	// CheckInterval is how often the manager checks for eviction candidates.
	CheckInterval time.Duration

	// ReserveSlots is the number of durable slots to reserve from capacity-based eviction.
	ReserveSlots int

	// MinWaitForCapacityEviction avoids immediately evicting runs that just entered a wait.
	MinWaitForCapacityEviction time.Duration
}

// DefaultManagerConfig returns sensible defaults.
func DefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		CheckInterval:              1 * time.Second,
		ReserveSlots:               0,
		MinWaitForCapacityEviction: 10 * time.Second,
	}
}

// EvictionHook is an optional callback invoked during eviction.
type EvictionHook func(key string, rec *DurableRunRecord)

// Manager orchestrates the background eviction loop.
type Manager struct {
	cache               DurableEvictionCache
	l                   *zerolog.Logger
	onEvictionSelected  EvictionHook
	onEvictionCancelled EvictionHook
	cancel              context.CancelFunc
	config              ManagerConfig
	wg                  sync.WaitGroup
	durableSlots        int
	mu                  sync.Mutex
}

// NewManager creates a new eviction manager.
func NewManager(durableSlots int, config ManagerConfig, l *zerolog.Logger, opts ...ManagerOption) *Manager {
	m := &Manager{
		durableSlots: durableSlots,
		config:       config,
		cache:        NewInMemoryDurableEvictionCache(),
		l:            l,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// ManagerOption configures optional manager behavior.
type ManagerOption func(*Manager)

// WithEvictionCache overrides the default in-memory cache.
func WithEvictionCache(cache DurableEvictionCache) ManagerOption {
	return func(m *Manager) {
		m.cache = cache
	}
}

// WithOnEvictionSelected sets a hook called when a candidate is selected.
func WithOnEvictionSelected(hook EvictionHook) ManagerOption {
	return func(m *Manager) {
		m.onEvictionSelected = hook
	}
}

// WithOnEvictionCancelled sets a hook called after local cancellation.
func WithOnEvictionCancelled(hook EvictionHook) ManagerOption {
	return func(m *Manager) {
		m.onEvictionCancelled = hook
	}
}

// Cache returns the underlying eviction cache.
func (m *Manager) Cache() DurableEvictionCache {
	return m.cache
}

// Start begins the background eviction loop.
func (m *Manager) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.runLoop(ctx)
	}()
}

// Stop halts the background loop and waits for it to finish.
func (m *Manager) Stop() {
	m.mu.Lock()
	cancel := m.cancel
	m.cancel = nil
	m.mu.Unlock()

	if cancel != nil {
		cancel()
		m.wg.Wait()
	}
}

// RegisterRun registers a durable run for eviction tracking.
func (m *Manager) RegisterRun(key, stepRunId string, ctx context.Context, cancel context.CancelCauseFunc, eviction *Policy) {
	m.cache.RegisterRun(key, stepRunId, ctx, cancel, time.Now().UTC(), eviction)
}

// UnregisterRun removes a run from eviction tracking.
func (m *Manager) UnregisterRun(key string) {
	m.cache.UnregisterRun(key)
}

// MarkWaiting marks a run as entering a wait state.
func (m *Manager) MarkWaiting(key, waitKind, resourceID string) {
	m.cache.MarkWaiting(key, time.Now().UTC(), waitKind, resourceID)
}

// MarkActive marks a run as leaving a wait state.
func (m *Manager) MarkActive(key string) {
	m.cache.MarkActive(key)
}

func (m *Manager) runLoop(ctx context.Context) {
	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.tickSafe()
		}
	}
}

func (m *Manager) tickSafe() {
	defer func() {
		if r := recover(); r != nil {
			m.l.Error().Interface("panic", r).Msg("eviction manager: panic in tick")
		}
	}()
	m.tick()
}

func (m *Manager) tick() {
	evictedThisTick := map[string]bool{}

	for {
		now := time.Now().UTC()
		key := m.cache.SelectEvictionCandidate(
			now,
			m.durableSlots,
			m.config.ReserveSlots,
			m.config.MinWaitForCapacityEviction,
		)

		if key == "" {
			return
		}

		if evictedThisTick[key] {
			return
		}
		evictedThisTick[key] = true

		rec := m.cache.Get(key)
		if rec == nil {
			continue
		}

		if rec.Eviction == nil {
			continue
		}

		m.l.Warn().
			Str("step_run_id", rec.StepRunId).
			Str("wait_kind", rec.WaitKind).
			Str("resource_id", rec.WaitResourceID).
			Msg("eviction manager: evicting durable run")

		// Observability hook: emitted when selected from eviction cache.
		if m.onEvictionSelected != nil {
			func() {
				defer func() {
					if r := recover(); r != nil {
						m.l.Error().Interface("panic", r).Msg("eviction manager: panic in onEvictionSelected hook")
					}
				}()
				m.onEvictionSelected(key, rec)
			}()
		}

		// Cancel the run's context with ErrEvicted cause.
		rec.Cancel(ErrEvicted)

		// Observability hook: emitted after local cancellation.
		if m.onEvictionCancelled != nil {
			func() {
				defer func() {
					if r := recover(); r != nil {
						m.l.Error().Interface("panic", r).Msg("eviction manager: panic in onEvictionCancelled hook")
					}
				}()
				m.onEvictionCancelled(key, rec)
			}()
		}
	}
}
