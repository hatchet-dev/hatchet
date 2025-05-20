package queueutils

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type OpMethod func(ctx context.Context, id string) (bool, error)

// SerialOperation represents a method that can only run serially.
// It can be configured with a maxJitter duration to add a random delay
// before executing, which helps prevent the "thundering herd" problem
// when many operations might start at the same time. The jitter is disabled
// by default (maxJitter=0) and can be enabled via OperationPool.WithJitter().
type SerialOperation struct {
	mu             sync.RWMutex
	shouldContinue bool
	isRunning      bool
	id             string
	lastRun        time.Time
	description    string
	timeout        time.Duration
	method         OpMethod
	maxJitter      time.Duration
}

func (o *SerialOperation) RunOrContinue(ql *zerolog.Logger) {
	// Apply jitter if configured
	if o.maxJitter > 0 {
		jitter := time.Duration(rand.Int63n(int64(o.maxJitter)))
		time.Sleep(jitter)
	}

	o.setContinue(true)
	o.Run(ql)
}

func (o *SerialOperation) Run(ql *zerolog.Logger) {
	if !o.setRunning(true, ql) {
		return
	}

	go func() {
		defer func() {
			o.setRunning(false, ql)
		}()

		f := func() {
			o.setContinue(false)

			ctx, cancel := context.WithTimeout(context.Background(), o.timeout)
			defer cancel()

			shouldContinue, err := o.method(ctx, o.id)

			if err != nil {
				ql.Err(err).Msgf("could not %s", o.description)
				return
			}

			// if a continue was set during execution of the scheduler, we'd like to continue no matter what.
			// if a continue was not set, we'd like to set it to the value returned by the scheduler.
			if !o.getContinue() {
				o.setContinue(shouldContinue)
			}
		}

		f()

		for o.getContinue() {
			f()
		}
	}()
}

// setRunning sets the running state of the operation and returns true if the state was changed,
// false if the state was not changed.
func (o *SerialOperation) setRunning(isRunning bool, ql *zerolog.Logger) bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	if isRunning == o.isRunning {

		return false
	}

	if isRunning {
		ql.Info().Str("tenant_id", o.id).TimeDiff("last_run", time.Now(), o.lastRun).Msg(o.description)

		o.lastRun = time.Now()

	}

	o.isRunning = isRunning

	return true
}

func (o *SerialOperation) setContinue(shouldContinue bool) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.shouldContinue = shouldContinue
}

func (o *SerialOperation) getContinue() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.shouldContinue
}
