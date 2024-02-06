package main

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client"
)

type Event struct {
	ID        uint64    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

func emit(ctx context.Context, sleep time.Duration, delay time.Duration, amount int, runFor time.Duration) int {
	c, err := client.New()

	if err != nil {
		panic(err)
	}

	var id uint64
	go func() {
		for {
			select {
			case <-time.After(runFor):
				return
			case <-time.After(sleep):
				for i := 0; i < amount; i++ {
					id++

					ev := Event{CreatedAt: time.Now(), ID: id}
					fmt.Println("pushed event", ev.ID)
					err = c.Event().Push(context.Background(), "test:event", ev)
					if err != nil {
						panic(err)
					}

					time.Sleep(delay)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-time.After(sleep):
			return int(id)
		case <-ctx.Done():
			return int(id)
		default:
			time.Sleep(time.Second)
		}
	}
}
