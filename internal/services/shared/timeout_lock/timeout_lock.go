package timeout_lock

import (
	"context"
	"errors"
	"time"

	"github.com/hatchet-dev/hatchet/internal/syncx"
)

var ErrTimeout = errors.New("timed out acquiring lock")

type TimeoutLock struct {
	sem     chan struct{}
	Timeout time.Duration
}

func (l *TimeoutLock) Acquire() bool {
	return l.AcquireContext(context.Background()) == nil
}

// AcquireContext bounds admission by both the caller's context and the lock timeout.
func (l *TimeoutLock) AcquireContext(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	timer := time.NewTimer(l.Timeout)
	defer timer.Stop()
	select {
	// attempt to send to the semaphore, blocks on contention because it has a buffer of 1
	case l.sem <- struct{}{}:
		if err := ctx.Err(); err != nil {
			l.Release()
			return err
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	// timing out dequeues the semaphore send
	case <-timer.C:
		return ErrTimeout
	}
}

func (l *TimeoutLock) Release() {
	<-l.sem
}

func NewTimeoutLock(timeout time.Duration) *TimeoutLock {
	return &TimeoutLock{
		sem:     make(chan struct{}, 1),
		Timeout: timeout,
	}
}

type KeyedTimeoutLock[T comparable] struct {
	locks   syncx.Map[T, *TimeoutLock]
	Timeout time.Duration
}

func NewKeyedTimeoutLock[T comparable](timeout time.Duration) *KeyedTimeoutLock[T] {
	return &KeyedTimeoutLock[T]{
		locks:   syncx.Map[T, *TimeoutLock]{},
		Timeout: timeout,
	}
}

func (k *KeyedTimeoutLock[T]) Acquire(key T) bool {
	lock, _ := k.locks.LoadOrStore(key, NewTimeoutLock(k.Timeout))
	return lock.Acquire()
}

func (k *KeyedTimeoutLock[T]) Release(key T) {
	lock, ok := k.locks.Load(key)
	if !ok {
		return
	}
	lock.Release()
}
