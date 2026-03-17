package dispatcher

import (
	"time"
)

type TimeoutLock struct {
	sem     chan struct{}
	timeout time.Duration
}

func (l *TimeoutLock) Acquire() bool {
	select {
	// attempt to send to the semaphore, blocks on contention because it has a buffer of 1
	case l.sem <- struct{}{}:
		return true
	// timing out dequeues the semaphore send
	case <-time.After(l.timeout):
		return false
	}
}

func (l *TimeoutLock) Release() {
	<-l.sem
}

func NewTimeoutLock(timeout time.Duration) *TimeoutLock {
	return &TimeoutLock{
		sem:     make(chan struct{}, 1),
		timeout: timeout,
	}
}
