// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"fmt"
	"sync"
)

// registration is one stored handler plus its optional permanent-failure
// callback and the registration id that makes stale removal a no-op.
type registration[E any] struct {
	handle  func(E) error
	onError func(error)
	regID   uint64
}

// handlerRegistry is a keyed registry of event handlers shared by the
// subscription listeners. It is a pure container: one RWMutex over nested
// plain maps, with stale-remove protection via registration ids (replaces the
// tombstone protocol). E is the public handler event type.
type handlerRegistry[K comparable, E any] struct {
	buckets map[K]map[string]registration[E]
	nextReg uint64
	mu      sync.RWMutex
}

func newHandlerRegistry[K comparable, E any]() handlerRegistry[K, E] {
	return handlerRegistry[K, E]{
		buckets: make(map[K]map[string]registration[E]),
	}
}

// store registers h under (k, session) and returns an idempotent remove
// closure. An empty session auto-generates a unique one (durable events'
// auto-increment semantics). The closure removes only the exact registration
// it created: if (k, session) was re-registered since, the stale closure is a
// no-op (the ABA guarantee formerly provided by tombstoned buckets).
func (r *handlerRegistry[K, E]) store(k K, session string, h func(E) error, onError func(error)) (remove func()) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if session == "" {
		r.nextReg++
		session = fmt.Sprintf("auto-%d", r.nextReg)
	}

	r.nextReg++
	regID := r.nextReg

	bucket := r.buckets[k]
	if bucket == nil {
		bucket = make(map[string]registration[E])
		r.buckets[k] = bucket
	}

	bucket[session] = registration[E]{
		handle:  h,
		onError: onError,
		regID:   regID,
	}

	return func() {
		r.mu.Lock()
		defer r.mu.Unlock()

		bucket, ok := r.buckets[k]
		if !ok {
			return
		}

		existing, ok := bucket[session]
		if !ok || existing.regID != regID {
			return
		}

		delete(bucket, session)
		if len(bucket) == 0 {
			delete(r.buckets, k)
		}
	}
}

// removeSession unconditionally removes whatever is registered under
// (k, session). Backs RemoveWorkflowRun.
func (r *handlerRegistry[K, E]) removeSession(k K, session string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	bucket, ok := r.buckets[k]
	if !ok {
		return
	}

	delete(bucket, session)
	if len(bucket) == 0 {
		delete(r.buckets, k)
	}
}

// hasAny reports whether any handler is registered (EOF reconnect policy).
func (r *handlerRegistry[K, E]) hasAny() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, bucket := range r.buckets {
		if len(bucket) > 0 {
			return true
		}
	}
	return false
}

// keys snapshots the registered keys (replay after reconnect).
func (r *handlerRegistry[K, E]) keys() []K {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keys := make([]K, 0, len(r.buckets))
	for k := range r.buckets {
		keys = append(keys, k)
	}
	return keys
}

// snapshot returns the registrations for k without holding the lock during
// dispatch.
func (r *handlerRegistry[K, E]) snapshot(k K) []registration[E] {
	r.mu.RLock()
	defer r.mu.RUnlock()

	bucket, ok := r.buckets[k]
	if !ok || len(bucket) == 0 {
		return nil
	}

	regs := make([]registration[E], 0, len(bucket))
	for _, reg := range bucket {
		regs = append(regs, reg)
	}
	return regs
}

// removeRegistrations removes exactly the given registrations (matched by
// regID) under k. Backs one-shot dispatch: a handler stored concurrently with
// a dispatch is never removed.
func (r *handlerRegistry[K, E]) removeRegistrations(k K, regs []registration[E]) {
	r.mu.Lock()
	defer r.mu.Unlock()

	bucket, ok := r.buckets[k]
	if !ok {
		return
	}

	for _, reg := range regs {
		for session, existing := range bucket {
			if existing.regID == reg.regID {
				delete(bucket, session)
				break
			}
		}
	}

	if len(bucket) == 0 {
		delete(r.buckets, k)
	}
}

// failAll empties the registry, invokes every non-nil onError with err, and
// returns how many registrations were dropped (for the caller's log line).
func (r *handlerRegistry[K, E]) failAll(err error) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	count := 0
	for _, bucket := range r.buckets {
		for _, reg := range bucket {
			count++
			if reg.onError != nil {
				reg.onError(err)
			}
		}
	}

	r.buckets = make(map[K]map[string]registration[E])
	return count
}

// listenGate ensures at most one listen loop runs per listener.
type listenGate struct {
	mu        sync.Mutex
	listening bool
}

func (g *listenGate) tryStart(closed bool) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.listening || closed {
		return false
	}

	g.listening = true
	return true
}

func (g *listenGate) stop() {
	g.mu.Lock()
	g.listening = false
	g.mu.Unlock()
}

func (g *listenGate) active() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.listening
}
