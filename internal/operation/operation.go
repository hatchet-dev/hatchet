package operation

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

type ID interface {
	string | int64
}

type OpMethod func(ctx context.Context, id string) (shouldContinue bool, err error)

// SerialOperation represents a method that can only run serially. Each operation can
// optionally run on an interval with jitter. This interval can be configured by reading
// from an external source like a database.
//
// When RunOrContinue is called, the operation will run if it is not already running.
// Each method called in RunOrContinue must return a boolean indicating whether the operation should
// continue running, along with an integer indicating the number of rows modified.
//
// Intervals which are configured with withBackoff=true will double their interval each time the method returns
// rowsModified=0 for a given number of times (incBackoffCount). The interval will reset to the original interval
// when rowsModified>0 is returned.
//
// The jitter is disabled by default (maxJitter=0). The jitter is applied to the interval after any backoff is applied.
// This is designed to help prevent the "thundering herd" problem when many operations might start at the same time.
type SerialOperation struct {
	mu             sync.RWMutex
	shouldContinue bool
	isRunning      bool
	id             string
	lastRun        time.Time
	operationId    string
	description    string
	timeout        time.Duration
	method         OpMethod
	l              *zerolog.Logger

	runningCtx context.Context
	cancel     context.CancelFunc

	interval *Interval
}

func WithDescription(description string) func(*SerialOperation) {
	return func(o *SerialOperation) {
		o.description = description
	}
}

func WithTimeout(timeout time.Duration) func(*SerialOperation) {
	return func(o *SerialOperation) {
		o.timeout = timeout
	}
}

func WithInterval(
	l *zerolog.Logger,
	repo v1.IntervalSettingsRepository,
	maxJitter, startInterval, maxInterval time.Duration,
	incBackoffCount int,
	gauge IntervalGauge,
) func(*SerialOperation) {
	return func(o *SerialOperation) {
		o.interval = NewInterval(l, repo, o.operationId, o.id, maxJitter, startInterval, maxInterval, incBackoffCount, gauge)
	}
}

func NewSerialOperation(
	l *zerolog.Logger,
	id string,
	operationId string,
	method OpMethod,
	fs ...func(*SerialOperation),
) *SerialOperation {
	runningCtx, cancel := context.WithCancel(context.Background())

	op := &SerialOperation{
		operationId: operationId,
		id:          id,
		description: "serial operation",
		timeout:     30 * time.Second,
		method:      method,
		runningCtx:  runningCtx,
		cancel:      cancel,
		l:           l,
	}

	for _, f := range fs {
		f(op)
	}

	// if we have an interval, we'd like to start it
	if op.interval != nil {
		// start the interval
		i := op.interval

		go func() {
			triggers := i.RunInterval(runningCtx)

			for {
				select {
				case <-runningCtx.Done():
					return
				case <-triggers:
					op.RunOrContinue(l)
				}
			}
		}()
	}

	return op
}

func (o *SerialOperation) Stop() {
	o.cancel()
}

func (o *SerialOperation) RunOrContinue(l *zerolog.Logger) {
	o.setContinue(true)
	o.Run(l)
}

func (o *SerialOperation) Run(l *zerolog.Logger) {
	if !o.setRunning(true, l) {
		return
	}

	go func() {
		defer func() {
			o.setRunning(false, l)
		}()

		f := func() {
			o.setContinue(false)

			ctx, cancel := context.WithTimeout(o.runningCtx, o.timeout)
			defer cancel()

			shouldContinue, err := o.method(ctx, o.id)

			if err != nil {
				l.Err(err).Msgf("could not %s", o.description)
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
func (o *SerialOperation) setRunning(isRunning bool, l *zerolog.Logger) bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	if isRunning == o.isRunning {
		return false
	}

	if isRunning {
		var idStr string
		switch id := any(o.id).(type) {
		case string:
			idStr = id
		case int64:
			idStr = strconv.FormatInt(id, 10)
		default:
			panic("unsupported ID type")
		}

		l.Info().Str("tenant_id", idStr).TimeDiff("last_run", time.Now(), o.lastRun).Msg(o.description)

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
