package main

import (
	"fmt"
	"log"
	"runtime"
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
	durations := make(chan time.Duration, eventsPerSecond*int(duration.Seconds())*3)
	go func() {
		count, uniques := run(ctx, durations)
		ch <- count
		ch <- uniques
	}()

	time.Sleep(after)

	emitted := emit(ctx, eventsPerSecond, duration)
	executed := <-ch
	uniques := <-ch

	var total time.Duration
	for i := 0; i < int(executed); i++ {
		total += <-durations
	}
	durationPerEvent := total / time.Duration(executed)
	log.Printf("ℹ️ average duration per event: %s", durationPerEvent)

	log.Printf("ℹ️ emitted %d, executed %d, uniques %d, using %d events/s", emitted, executed, uniques, eventsPerSecond)

	// num goroutines
	log.Printf("ℹ️ num goroutines: %d", runtime.NumGoroutine())

	if emitted != executed {
		log.Printf("⚠️ warning: emitted and executed counts do not match")
	}

	if emitted != uniques {
		return fmt.Errorf("❌ emitted and unique executed counts do not match")
	}

	log.Printf("✅ success")

	return nil
}
