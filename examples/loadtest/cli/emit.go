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

func emit(ctx context.Context, amountPerSecond int, runFor time.Duration) int {
	c, err := client.New()

	if err != nil {
		panic(err)
	}

	var id uint64
	go func() {
		ticker := time.NewTicker(time.Second / time.Duration(amountPerSecond))
		defer ticker.Stop()

		timer := time.After(runFor)

		for {
			select {
			case <-ticker.C:
				id++

				ev := Event{CreatedAt: time.Now(), ID: id}
				fmt.Println("pushed event", ev.ID)
				err = c.Event().Push(context.Background(), "test:event", ev)
				if err != nil {
					panic(err)
				}
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
			return int(id)
		default:
			time.Sleep(time.Second)
		}
	}
}
