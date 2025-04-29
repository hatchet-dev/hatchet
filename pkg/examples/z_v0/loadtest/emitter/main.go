package main

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/joho/godotenv"
)

type Event struct {
	ID        uint64    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type stepOneOutput struct {
	Message string `json:"message"`
}

func StepOne(ctx context.Context, input *Event) (result *stepOneOutput, err error) {
	fmt.Println(input.ID, "delay", time.Since(input.CreatedAt))

	return &stepOneOutput{
		Message: "This ran at: " + time.Now().Format(time.RubyDate),
	}, nil
}

func main() {
	err := godotenv.Load()

	if err != nil {
		panic(err)
	}

	client, err := client.New()

	if err != nil {
		panic(err)
	}

	interruptCtx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())
	defer cancel()

	var id uint64
	go func() {
		for {
			select {
			case <-time.After(5 * time.Second):
				for i := 0; i < 100; i++ {
					id++

					ev := Event{CreatedAt: time.Now(), ID: id}
					fmt.Println("pushed event", ev.ID)
					err = client.Event().Push(interruptCtx, "test:event", ev)
					if err != nil {
						panic(err)
					}
				}
			case <-interruptCtx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-interruptCtx.Done():
			return
		default:
			time.Sleep(time.Second)
		}
	}
}
