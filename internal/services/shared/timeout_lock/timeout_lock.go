package timeout_lock

import (
	"time"

	"github.com/hatchet-dev/hatchet/internal/syncx"
)

type TimeoutLock struct {
	sem     chan struct{}
	Timeout time.Duration
}

func (l *TimeoutLock) Acquire() bool {
	select {
	// attempt to send to the semaphore, blocks on contention because it has a buffer of 1
	case l.sem <- struct{}{}:
		return true
	// timing out dequeues the semaphore send
	case <-time.After(l.Timeout):
		return false
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
