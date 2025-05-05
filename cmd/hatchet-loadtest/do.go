package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

type avgResult struct {
	count int64
	avg   time.Duration
}

func do(namespace string, duration time.Duration, eventsPerSecond int, delay time.Duration, wait time.Duration, concurrency int, workerDelay time.Duration, slots int, failureRate float32, payloadSize string, eventFanout int) error {
	l.Info().Msgf("testing with duration=%s, eventsPerSecond=%d, delay=%s, wait=%s, concurrency=%d", duration, eventsPerSecond, delay, wait, concurrency)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	after := 10 * time.Second

	go func() {
		time.Sleep(duration + after + wait + 5*time.Second)
		cancel()
	}()

	ch := make(chan int64, 2)
	durations := make(chan time.Duration, eventsPerSecond)

	// Compute running average for executed durations using a rolling average.
	durationsResult := make(chan avgResult)
	go func() {
		var count int64
		var avg time.Duration
		for d := range durations {
			count++
			if count == 1 {
				avg = d
			} else {
				avg += (d - avg) / time.Duration(count)
			}
		}
		durationsResult <- avgResult{count: count, avg: avg}
	}()

	go func() {
		if workerDelay > 0 {
			// run a worker to register the workflow
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			run(ctx, namespace, delay, durations, concurrency, slots, failureRate, eventFanout)
			cancel()
			l.Info().Msgf("wait %s before starting the worker", workerDelay)
			time.Sleep(workerDelay)
		}
		l.Info().Msg("starting worker now")
		count, uniques := run(ctx, namespace, delay, durations, concurrency, slots, failureRate, eventFanout)
		close(durations)
		ch <- count
		ch <- uniques
	}()

	time.Sleep(after)

	scheduled := make(chan time.Duration, eventsPerSecond)

	// Compute running average for scheduled times using a rolling average.
	scheduledResult := make(chan avgResult)
	go func() {
		var count int64
		var avg time.Duration
		for d := range scheduled {
			count++
			if count == 1 {
				avg = d
			} else {
				avg += (d - avg) / time.Duration(count)
			}
		}
		scheduledResult <- avgResult{count: count, avg: avg}
	}()

	emitted := emit(ctx, namespace, eventsPerSecond, duration, scheduled, payloadSize)
	close(scheduled)

	executed := <-ch
	uniques := <-ch

	finalDurationResult := <-durationsResult
	finalScheduledResult := <-scheduledResult

	log.Printf("ℹ️ emitted %d, executed %d, uniques %d, using %d events/s", emitted, executed, uniques, eventsPerSecond)

	if executed == 0 {
		return fmt.Errorf("❌ no events executed")
	}

	log.Printf("ℹ️ final average duration per executed event: %s", finalDurationResult.avg)
	log.Printf("ℹ️ final average scheduling time per event: %s", finalScheduledResult.avg)

	if int64(eventFanout)*emitted != executed {
		log.Printf("⚠️ warning: emitted and executed counts do not match: %d != %d", int64(eventFanout)*emitted, executed)
	}

	if int64(eventFanout)*emitted != uniques {
		return fmt.Errorf("❌ emitted and unique executed counts do not match: %d != %d", int64(eventFanout)*emitted, uniques)
	}

	log.Printf("✅ success")

	return nil
}
