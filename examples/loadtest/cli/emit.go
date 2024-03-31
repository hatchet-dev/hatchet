package main

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

func emit(ctx context.Context, amountPerSecond int, duration time.Duration, scheduled chan<- time.Duration) int64 {
	c, err := client.New()

	if err != nil {
		panic(err)
	}

	var id int64
	mx := sync.Mutex{}
	go func() {
		ticker := time.NewTicker(time.Second / time.Duration(amountPerSecond))
		defer ticker.Stop()

		timer := time.After(duration)

		for {
			select {
			case <-ticker.C:
				mx.Lock()
				id++

				go func(id int64) {
					var err error
					ev := Event{CreatedAt: time.Now(), ID: id}
					l.Info().Msgf("pushed event %d", ev.ID)
					err = c.Event().Push(context.Background(), "load-test:event", ev)
					if err != nil {
						panic(fmt.Errorf("error pushing event: %w", err))
					}
					took := time.Since(ev.CreatedAt)
					l.Info().Msgf("pushed event %d took %s", ev.ID, took)
					scheduled <- took
				}(id)

				mx.Unlock()
			case <-timer:
				l.Info().Msg("done emitting events due to timer")
				return
			case <-ctx.Done():
				l.Info().Msgf("done emitting events due to interruption at %d", id)
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
