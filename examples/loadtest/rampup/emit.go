package rampup

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client"
)

type Event struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

func emit(ctx context.Context, startEventsPerSecond, amount int, increase, duration, maxAcceptableSchedule time.Duration, hook <-chan time.Duration, scheduled chan<- int64) int64 {
	c, err := client.New()

	if err != nil {
		panic(err)
	}

	var id int64
	mx := sync.Mutex{}
	go func() {
		timer := time.After(duration)
		start := time.Now()

		var eventsPerSecond int
		go func() {
			took := <-hook
			panic(fmt.Errorf("event took too long to schedule: %s at %d events/s", took, eventsPerSecond))
		}()
		for {
			// emit amount * increase events per second
			eventsPerSecond = startEventsPerSecond + (amount * int(time.Since(start).Seconds()) / int(increase.Seconds()))
			increase += 1
			if eventsPerSecond < 1 {
				eventsPerSecond = 1
			}
			log.Printf("emitting %d events per second", eventsPerSecond)
			select {
			case <-time.After(time.Second / time.Duration(eventsPerSecond)):
				mx.Lock()
				id += 1
				mx.Unlock()

				go func(id int64) {
					ev := Event{CreatedAt: time.Now(), ID: id}
					fmt.Println("pushed event", ev.ID)
					err = c.Event().Push(context.Background(), "load-test:event", ev)
					if err != nil {
						panic(fmt.Errorf("error pushing event: %w", err))
					}
					took := time.Since(ev.CreatedAt)
					fmt.Println("pushed event", ev.ID, "took", took)

					if took > maxAcceptableSchedule {
						panic(fmt.Errorf("event took too long to schedule: %s at %d events/s", took, eventsPerSecond))
					}

					scheduled <- id
				}(id)
			case <-timer:
				log.Println("done emitting events due to timer at", id)
				return
			case <-ctx.Done():
				log.Println("done emitting events due to interruption at", id)
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			mx.Lock()
			defer mx.Unlock()
			return id
		default:
			time.Sleep(time.Second)
		}
	}
}
