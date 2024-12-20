package main

import (
	"context"
	"fmt"
	"log"
	"time"

	clientconfig "github.com/hatchet-dev/hatchet/pkg/config/client"

	"github.com/hatchet-dev/hatchet/pkg/client"
)

func do(duration time.Duration, eventsPerSecond int, delay time.Duration, wait time.Duration, concurrency int, workerDelay time.Duration) error {
	l.Info().Msgf("testing with duration=%s, eventsPerSecond=%d, delay=%s, wait=%s, concurrency=%d", duration, eventsPerSecond, delay, wait, concurrency)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	after := 10 * time.Second

	go func() {
		time.Sleep(duration + after + wait + 5*time.Second)
		cancel()
	}()

	c, err := client.NewFromConfigFile(&clientconfig.ClientConfigFile{}, client.WithLogLevel("warn"))

	if err != nil {
		panic(err)
	}

	ch := make(chan int64, 2)
	durations := make(chan time.Duration, eventsPerSecond*int(duration.Seconds())*3)
	go func() {
		if workerDelay.Seconds() > 0 {
			l.Info().Msgf("wait %s before starting the worker", workerDelay)
			time.Sleep(workerDelay)
		}
		l.Info().Msg("starting worker now")
		c, err := client.New()
		if err != nil {
			panic(err)
		}

		count, uniques := run(ctx, c, delay, durations, concurrency)
		ch <- count
		ch <- uniques
	}()

	time.Sleep(after)

	scheduled := make(chan time.Duration, eventsPerSecond*int(duration.Seconds())*2)
	emitted := emit(ctx, c, eventsPerSecond, duration, scheduled)
	executed := <-ch
	uniques := <-ch

	l.Info().Msgf("emitted %d, executed %d, uniques %d, using %d events/s", emitted, executed, uniques, eventsPerSecond)

	if executed == 0 {
		return fmt.Errorf("❌ no events executed")
	}

	var totalDurationExecuted time.Duration
	for i := 0; i < int(executed); i++ {
		totalDurationExecuted += <-durations
	}
	durationPerEventExecuted := totalDurationExecuted / time.Duration(executed)
	log.Printf("ℹ️ average duration per executed event: %s", durationPerEventExecuted)

	var totalDurationScheduled time.Duration
	for i := 0; i < int(emitted); i++ {
		totalDurationScheduled += <-scheduled
	}
	scheduleTimePerEvent := totalDurationScheduled / time.Duration(emitted)

	log.Printf("ℹ️ average scheduling time per event: %s", scheduleTimePerEvent)

	if emitted != executed {
		log.Printf("⚠️ warning: emitted and executed counts do not match: %d != %d", emitted, executed)
	}

	if emitted != uniques {
		return fmt.Errorf("❌ emitted and unique executed counts do not match: %d != %d", emitted, uniques)
	}

	log.Printf("✅ success")

	return nil
}
