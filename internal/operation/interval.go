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

const (
	gaugeInterval = time.Second * 5
)

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

	// read the current interval from the database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	currInterval, err := repo.ReadInterval(ctx, operationId, uuid.MustParse(resourceId))

	if err != nil {
		l.Error().Err(err).Msg(fmt.Sprintf("error reading interval for resource %s, defaulting to start interval", resourceId))
		currInterval = 0
	}

	if currInterval < startInterval {
		currInterval = startInterval
	}

	if currInterval > maxInterval {
		currInterval = maxInterval
	}

	return &Interval{
		l:               l,
		repo:            repo,
		operationId:     operationId,
		resourceId:      resourceId,
		maxJitter:       maxJitter,
		startInterval:   startInterval,
		currInterval:    currInterval,
		maxInterval:     maxInterval,
		noActivityCount: 0,
		incBackoffCount: incBackoffCount,
		gauge:           gauge,
	}
}

// runInterval sends a struct{} on the returned channel at the configured interval,
// and exits when the context is cancelled.
func (i *Interval) RunInterval(ctx context.Context) <-chan struct{} {
	res := make(chan struct{})

	// run the gauge at a regular interval to adjust the current interval if needed
	if i.gauge != nil {
		go func() {
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
						i.l.Error().Err(err).Msg(fmt.Sprintf("error calling interval gauge for resource %s", i.resourceId))
					} else {
						i.SetIntervalGauge(rowsModified)
					}
				}
			}
		}()
	}

	go func() {
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

// gets the next trigger time, applying jitter if configured.
func (i *Interval) getNextTrigger() <-chan time.Time {
	i.intervalMu.RLock()
	defer i.intervalMu.RUnlock()

	return time.After(i.currInterval + safeRandomDuration(i.maxJitter))
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
			i.l.Error().Err(err).Msg(fmt.Sprintf("error setting interval for resource %s", i.resourceId))
		} else {
			i.currInterval = newInterval
		}
	}
}
