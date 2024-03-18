package rampup

import (
	"context"
	"fmt"
	"log"
	"time"
)

func do(duration time.Duration, startEventsPerSecond, amount int, increase, delay, wait, maxAcceptableDelay time.Duration, concurrency int) error {
	log.Printf("testing with duration=%s, amount=%d, increase=%d, delay=%s, wait=%s, concurrency=%d", duration, amount, increase, delay, wait, concurrency)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	after := 10 * time.Second

	go func() {
		time.Sleep(duration + after + wait + 5*time.Second)
		cancel()
	}()

	go func() {
		run(ctx, delay, concurrency)
	}()

	scheduled := make(chan int, 1)

	go func() {
		x := <-scheduled
		panic(fmt.Errorf("event took too long to schedule: %s at %d events/s", maxAcceptableDelay, x))
	}()

	emit(ctx, startEventsPerSecond, amount, increase, duration, maxAcceptableDelay, scheduled)

	time.Sleep(after)

	log.Printf("✅ success")

	return nil
}
