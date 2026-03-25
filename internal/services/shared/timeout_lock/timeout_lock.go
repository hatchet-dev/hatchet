package timeout_lock

import (
	"time"
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
	locks    map[T]*TimeoutLock
	Timeout  time.Duration
	lockLock *TimeoutLock
}

func NewKeyedTimeoutLock[T comparable](timeout time.Duration) *KeyedTimeoutLock[T] {
	return &KeyedTimeoutLock[T]{
		locks:    make(map[T]*TimeoutLock),
		Timeout:  timeout,
		lockLock: NewTimeoutLock(100 * time.Millisecond), // secondary lock to protect creation of locks
	}
}

func (k *KeyedTimeoutLock[T]) Acquire(key T) bool {
	acquired := k.lockLock.Acquire()
	if !acquired {
		return acquired
	}
	lock, ok := k.locks[key]
	if !ok {
		lock = NewTimeoutLock(k.Timeout)
		k.locks[key] = lock
	}
	k.lockLock.Release()
	return lock.Acquire()
}

func (k *KeyedTimeoutLock[T]) Release(key T) {
	lock, ok := k.locks[key]
	if !ok {
		return
	}
	lock.Release()
}
