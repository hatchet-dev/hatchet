package shutdown

import (
	"context"
	"sync"
)

// Signaller is to signal from outside that any goroutines should begin to close.
//
// NOTE(gregfurman): This approach is a simplified adaptation from github.com/Jeffail/shutdown.
// If we want more complicated shutdown handling, i.e the ability to distinguish between
// a graceful shutdown vs a forced, we can use the package directly,
type Signaller struct {
	stopChan chan struct{}
	stopOnce sync.Once
}

// NewSignaller creates a new signaller.
func NewSignaller() *Signaller {
	return &Signaller{
		stopChan: make(chan struct{}),
	}
}

// TriggerShutdown signals to the owner of this Signaller that it should terminate.
func (s *Signaller) TriggerShutdown() {
	s.stopOnce.Do(func() {
		close(s.stopChan)
	})
}

// WithShutdown derives a context.Context that will be terminated when either the
// parent context is cancelled or the signal to stop has been made.
func (s *Signaller) WithShutdown(parent context.Context) (context.Context, context.CancelFunc) {
	var cancel context.CancelFunc
	parent, cancel = context.WithCancel(parent)
	go func() {
		select {
		case <-parent.Done():
		case <-s.stopChan:
		}
		cancel()
	}()
	return parent, cancel
}
