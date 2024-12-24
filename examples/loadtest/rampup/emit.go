package rampup

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client"
)

type Event struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

func emit(ctx context.Context, client client.Client, startEventsPerSecond, amount int, increase, duration, maxAcceptableSchedule time.Duration, errChan chan<- error) int64 {

	var id int64
	mx := sync.Mutex{}
	go func() {
		timer := time.After(duration)
		start := time.Now()

		var eventsPerSecond int

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
				mx.Lock()
				id++

				go func(id int64) {
					var err error
					ev := Event{CreatedAt: time.Now(), ID: id}
					l.Debug().Msgf("pushed event %d", ev.ID)
					err = client.Event().Push(context.Background(), "load-test:event", ev)
					if err != nil {
						errChan <- fmt.Errorf("error pushing event %d: %w", id, err)
						return
					}
					took := time.Since(ev.CreatedAt)
					l.Debug().Msgf("pushed event %d took %s", ev.ID, took)

					if took > maxAcceptableSchedule {
						errChan <- fmt.Errorf("event %d took too long to schedule: %s at %d events/s", id, took, eventsPerSecond)
						return
					}

				}(id)

				mx.Unlock()
			case <-timer:
				l.Debug().Msgf("done emitting events due to timer at %d", id)
				return
			case <-ctx.Done():
				l.Debug().Msgf("done emitting events due to interruption at %d", id)
				return
			}
		}
	}()

	<-ctx.Done()
	mx.Lock()
	defer mx.Unlock()
	return id

}
