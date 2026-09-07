package serverlessoperator

import (
	"context"
	"sync"
)

// slotLimiter is a counting semaphore whose capacity can change while it is held, which is
// what a per-endpoint slots update through the routing cache needs: deliveries in flight
// keep their slot and the new limit applies to the next acquire.
type slotLimiter struct {
	cond *sync.Cond
	mu   sync.Mutex
	max  int
	held int
}

func newSlotLimiter(limit int) *slotLimiter {
	s := &slotLimiter{max: limit}
	s.cond = sync.NewCond(&s.mu)

	return s
}

// acquire blocks until a slot is free or ctx is done.
func (s *slotLimiter) acquire(ctx context.Context) error {
	stop := context.AfterFunc(ctx, func() {
		s.mu.Lock()
		s.cond.Broadcast()
		s.mu.Unlock()
	})
	defer stop()

	s.mu.Lock()
	defer s.mu.Unlock()

	for s.held >= s.max {
		if err := ctx.Err(); err != nil {
			return err
		}

		s.cond.Wait()
	}

	s.held++

	return nil
}

func (s *slotLimiter) release() {
	s.mu.Lock()
	s.held--
	s.cond.Broadcast()
	s.mu.Unlock()
}

// resize changes the capacity; waiters re-check against the new limit.
func (s *slotLimiter) resize(limit int) {
	s.mu.Lock()
	s.max = limit
	s.cond.Broadcast()
	s.mu.Unlock()
}

func (s *slotLimiter) inUse() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.held
}
