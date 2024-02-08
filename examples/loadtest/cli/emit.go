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

func emit(ctx context.Context, amountPerSecond int, duration time.Duration) int64 {
	c, err := client.New()

	if err != nil {
		panic(err)
	}

	var id int64
	mx := sync.Mutex{}
	go func() {
		ticker := time.NewTicker(time.Second / time.Duration(amountPerSecond))
		defer ticker.Stop()

		timer := time.After(duration + 10)

		for {
			select {
			case <-ticker.C:
				mx.Lock()
				id += 1
				mx.Unlock()

				go func(id int64) {
					ev := Event{CreatedAt: time.Now(), ID: id}
					fmt.Println("pushed event", ev.ID)
					err = c.Event().Push(context.Background(), "test:event", ev)
					if err != nil {
						panic(fmt.Errorf("error pushing event: %w", err))
					}
				}(id)
			case <-timer:
				return
			case <-ctx.Done():
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
