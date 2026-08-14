package operation

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

// gaugeInterval is the period between activity-gauge polls. It is a var so
// tests can shrink it; production always uses the 5s default.
var gaugeInterval = 5 * time.Second

// IntervalGauge is a function that determines whether or not to increase or reset the interval.
// If the returned integer is >0, the interval is reset to the start interval. If 0, the no-rows count is increased,
// and if it exceeds the incBackoffCount, the interval is doubled.
type IntervalGauge func(ctx context.Context, resourceId string) (int, error)

type Interval struct {
	l               *zerolog.Logger
	repo            v1.IntervalSettingsRepository
	gauge           IntervalGauge
	operationId     string
	resourceId      string // tenant ID, queue name, etc.
	maxJitter       time.Duration
	startInterval   time.Duration
	currInterval    time.Duration
	maxInterval     time.Duration
	noActivityCount int
	incBackoffCount int
	intervalMu      sync.RWMutex

	// firstTrigger is true until the first getNextTrigger call, which uses a
	// full-window random phase instead of currInterval+jitter so ops created
	// together at startup do not all fire within a narrow band.
	firstTrigger bool

	// needsIntervalLoad is true when NewInterval deferred the persisted-interval
	// DB read; RunInterval loads it lazily before the first trigger.
	needsIntervalLoad bool
}

func NewInterval(
	l *zerolog.Logger,
	repo v1.IntervalSettingsRepository,
	operationId, resourceId string,
	maxJitter, startInterval, maxInterval time.Duration,
	incBackoffCount int,
	gauge IntervalGauge,
) *Interval {
	if maxInterval < 0 {
		maxInterval = time.Minute
	}

	// Do not block the constructor on a per-tenant DB read. Thousands of
	// intervals are created in a tight loop when a controller discovers tenants;
	// the persisted value is loaded lazily in RunInterval instead.
	return &Interval{
		l:                 l,
		repo:              repo,
		operationId:       operationId,
		resourceId:        resourceId,
		maxJitter:         maxJitter,
		startInterval:     startInterval,
		currInterval:      startInterval,
		maxInterval:       maxInterval,
		noActivityCount:   0,
		incBackoffCount:   incBackoffCount,
		gauge:             gauge,
		firstTrigger:      true,
		needsIntervalLoad: true,
	}
}

// loadPersistedInterval reads the stored interval for this resource (if any)
// and clamps it into [startInterval, maxInterval]. Safe to call concurrently
// with getNextTrigger / SetIntervalGauge.
func (i *Interval) loadPersistedInterval(ctx context.Context) {
	i.intervalMu.Lock()
	needsLoad := i.needsIntervalLoad
	i.needsIntervalLoad = false
	i.intervalMu.Unlock()

	if !needsLoad || i.repo == nil {
		return
	}

	readCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	currInterval, err := i.repo.ReadInterval(readCtx, i.operationId, uuid.MustParse(i.resourceId))
	if err != nil {
		if i.l != nil {
			i.l.Error().Err(err).Msg(fmt.Sprintf("error reading interval for resource %s, defaulting to start interval", i.resourceId))
		}
		return
	}

	i.intervalMu.Lock()
	defer i.intervalMu.Unlock()

	if currInterval < i.startInterval {
		currInterval = i.startInterval
	}

	if currInterval > i.maxInterval {
		currInterval = i.maxInterval
	}

	i.currInterval = currInterval
}

// runInterval sends a struct{} on the returned channel at the configured interval,
// and exits when the context is cancelled.
func (i *Interval) RunInterval(ctx context.Context) <-chan struct{} {
	res := make(chan struct{})

	// run the gauge at a regular interval to adjust the current interval if needed
	if i.gauge != nil {
		go func() {
			// Randomize start phase so gauges created together at startup do not
			// all hit the DB on the same 5s boundary forever.
			phase := safeRandomDuration(gaugeInterval)
			if phase > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(phase):
				}
			}

			ticker := time.NewTicker(gaugeInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					// call the gauge function to get the number of rows modified
					rowsModified, err := i.gauge(ctx, i.resourceId)

					if err != nil {
						if i.l != nil {
							i.l.Error().Ctx(ctx).Err(err).Msg(fmt.Sprintf("error calling interval gauge for resource %s", i.resourceId))
						}
					} else {
						i.SetIntervalGauge(rowsModified)
					}
				}
			}
		}()
	}

	go func() {
		i.loadPersistedInterval(ctx)

		trigger := i.getNextTrigger()

		for {
			select {
			case <-ctx.Done():
				return
			case <-trigger:
				res <- struct{}{}

				trigger = i.getNextTrigger()
			}
		}
	}()

	return res
}

// gets the next trigger time. The first call uses a random phase in
// [0, currInterval+maxJitter); subsequent calls use currInterval + jitter.
func (i *Interval) getNextTrigger() <-chan time.Time {
	i.intervalMu.Lock()
	defer i.intervalMu.Unlock()

	var delay time.Duration
	if i.firstTrigger {
		i.firstTrigger = false
		delay = safeRandomDuration(i.currInterval + i.maxJitter)
	} else {
		delay = i.currInterval + safeRandomDuration(i.maxJitter)
	}

	return time.After(delay)
}

func safeRandomDuration(maxJitter time.Duration) time.Duration {
	if maxJitter <= 0 {
		return 0
	}

	return time.Duration(rand.Int63n(int64(maxJitter))) // nolint: gosec
}

func (i *Interval) SetIntervalGauge(rowsModified int) {
	i.intervalMu.Lock()
	defer i.intervalMu.Unlock()

	previousInterval := i.currInterval

	if rowsModified > 0 {
		i.currInterval = i.startInterval
		i.noActivityCount = 0
	} else {
		i.noActivityCount++

		if i.noActivityCount >= i.incBackoffCount {
			i.currInterval *= 2
			i.noActivityCount = 0
		}
	}

	if i.currInterval > i.maxInterval {
		i.currInterval = i.maxInterval
	}

	// Only update the database if the interval has changed
	if i.currInterval != previousInterval {
		// Use background context since this is for persistence
		ctx := context.Background()
		newInterval, err := i.repo.SetInterval(ctx, i.operationId, uuid.MustParse(i.resourceId), i.currInterval)

		if err != nil {
			if i.l != nil {
				i.l.Error().Ctx(ctx).Err(err).Msg(fmt.Sprintf("error setting interval for resource %s", i.resourceId))
			}
		} else {
			i.currInterval = newInterval
		}
	}
}
