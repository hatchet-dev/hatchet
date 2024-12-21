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

// this function is going to emit on a schedule and then return

func emit(ctx context.Context, c client.Client, amountPerSecond int, duration time.Duration, scheduled chan<- time.Duration) int64 {

	var done = make(chan struct{})
	var id int64
	mx := sync.Mutex{}
	go func() {
		defer func() { done <- struct{}{} }()
		ticker := time.NewTicker(time.Second / time.Duration(amountPerSecond))
		defer ticker.Stop()

		timer := time.After(duration)
		wg := sync.WaitGroup{}

		for {
			select {
			case <-ticker.C:
				mx.Lock()
				id++

				wg.Add(1)
				go func(id int64) {

					defer wg.Done()
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

				wg.Wait()
				return
			case <-ctx.Done():
				wg.Wait()

				l.Info().Msgf("done emitting events due to interruption at %d", id)

				return
			case <-time.After(duration + 20*time.Second):
				l.Fatal().Msg("timed out emitting events")

			}
		}
	}()

	for {
		select {
		case <-done:
			l.Info().Msgf("done emitting events at %d", id)
			mx.Lock()
			defer mx.Unlock()
			return id
		case <-ctx.Done():
			l.Info().Msgf("context done s done emitting events at %d", id)
			mx.Lock()
			defer mx.Unlock()
			return id

		}
	}
}
