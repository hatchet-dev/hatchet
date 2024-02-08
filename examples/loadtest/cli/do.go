package main

import (
	"fmt"
	"log"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
)

func do(duration time.Duration, eventsPerSecond int, wait time.Duration) error {
	log.Printf("testing with runFor=%s, eventsPerSecond=%d, wait=%s", duration, eventsPerSecond, wait)

	ctx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())
	defer cancel()

	after := 10 * time.Second

	go func() {
		time.Sleep(duration + after + wait + 5*time.Second)
		cancel()
	}()

	ch := make(chan int64, 1)
	go func() {
		count, uniques := run(ctx)
		ch <- count
		ch <- uniques
	}()

	time.Sleep(after)

	emitted := emit(ctx, eventsPerSecond, duration)
	executed := <-ch
	uniques := <-ch

	log.Printf("ℹ️ emitted %d, executed %d, uniques %d, using %d events/s", emitted, executed, uniques, eventsPerSecond)

	if emitted != executed {
		log.Printf("⚠️ warning: emitted and executed counts do not match")
	}

	if emitted != uniques {
		return fmt.Errorf("❌ emitted and unique executed counts do not match")
	}

	return nil
}
