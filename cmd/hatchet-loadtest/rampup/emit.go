package rampup

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client"
)

type Event struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

func emit(ctx context.Context, startEventsPerSecond, amount int, increase, duration, maxAcceptableSchedule time.Duration, hook <-chan time.Duration, scheduled chan<- int64, scheduledTimes chan<- time.Duration) int64 {
	c, err := client.New()

	if err != nil {
		panic(err)
	}

	var id int64

	// Create a buffered channel for events
	jobCh := make(chan Event, startEventsPerSecond*2)

	// Worker pool to handle event pushes
	numWorkers := 10
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ev := range jobCh {
				l.Debug().Msgf("pushing event %d", ev.ID)

				err := c.Event().Push(context.Background(), "load-test:event", ev)
				if err != nil {
					panic(fmt.Errorf("error pushing event: %w", err))
				}
				took := time.Since(ev.CreatedAt)
				l.Debug().Msgf("pushed event %d took %s", ev.ID, took)

				if took > maxAcceptableSchedule {
					panic(fmt.Errorf("event took over %s to schedule: %s", maxAcceptableSchedule, took))
				}

				scheduledTimes <- took
				scheduled <- ev.ID
			}
		}()
	}

	// Hook handler for execution timeout
	go func() {
		took := <-hook

		if took == 0 {
			return
		}

		panic(fmt.Errorf("event took too long to run: %s", took))
	}()

	timer := time.After(duration)
	start := time.Now()

	var eventsPerSecond int

loop:
	for {
		// emit amount * increase events per second
		eventsPerSecond = startEventsPerSecond + (amount * int(time.Since(start).Seconds()) / int(increase.Seconds()))
		increase++
		if eventsPerSecond < 1 {
			eventsPerSecond = 1
		}
		l.Debug().Msgf("emitting %d events per second", eventsPerSecond)
		select {
		case <-time.After(time.Second / time.Duration(eventsPerSecond)):
			newID := atomic.AddInt64(&id, 1)
			ev := Event{
				ID:        newID,
				CreatedAt: time.Now(),
			}
			select {
			case jobCh <- ev:
			case <-ctx.Done():
				l.Debug().Msgf("done emitting events due to interruption at %d", id)
				break loop
			}
		case <-timer:
			l.Debug().Msgf("done emitting events due to timer at %d", id)
			break loop
		case <-ctx.Done():
			l.Debug().Msgf("done emitting events due to interruption at %d", id)
			break loop
		}
	}

	close(jobCh)
	wg.Wait()
	return atomic.LoadInt64(&id)
}
