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

func do(config LoadTestConfig) error {
	l.Info().Msgf("testing with duration=%s, eventsPerSecond=%d, delay=%s, wait=%s, concurrency=%d, averageDurationThreshold=%s", config.Duration, config.Events, config.Delay, config.Wait, config.Concurrency, config.AverageDurationThreshold)

	after := 10 * time.Second

	// The worker may intentionally be delayed (WorkerDelay) before it starts consuming tasks.
	// The test timeout must include this delay, otherwise we can cancel while work is still expected to complete.
	timeout := config.WorkerDelay + after + config.Duration + config.Wait + 30*time.Second

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ch := make(chan int64, 2)
	durations := make(chan time.Duration, config.Events)

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

	// Start worker and ensure it has time to register
	workerStarted := make(chan struct{})

	go func() {
		if config.WorkerDelay > 0 {
			// run a worker to register the workflow
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			run(ctx, config, durations)
			cancel()
			l.Info().Msgf("wait %s before starting the worker", config.WorkerDelay)
			time.Sleep(config.WorkerDelay)
		}
		l.Info().Msg("starting worker now")

		// Signal that worker is starting
		close(workerStarted)

		count, uniques := run(ctx, config, durations)
		close(durations)
		ch <- count
		ch <- uniques
	}()

	// Wait for worker to start, then give it time to register workflows
	<-workerStarted
	time.Sleep(after)

	scheduled := make(chan time.Duration, config.Events)

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

	emitted := emit(ctx, config.Namespace, config.Events, config.Duration, scheduled, config.PayloadSize)
	close(scheduled)

	executed := <-ch
	uniques := <-ch

	finalDurationResult := <-durationsResult
	finalScheduledResult := <-scheduledResult

	expected := int64(config.EventFanout) * emitted * int64(config.DagSteps)

	// NOTE: `emit()` returns successfully pushed events (not merely generated IDs),
	// so `emitted` here is effectively "pushed".
	log.Printf(
		"ℹ️ pushed %d, executed %d, uniques %d, using %d events/s (fanout=%d dagSteps=%d expected=%d)",
		emitted,
		executed,
		uniques,
		config.Events,
		config.EventFanout,
		config.DagSteps,
		expected,
	)

	if executed == 0 {
		return fmt.Errorf("❌ no events executed")
	}

	log.Printf("ℹ️ final average duration per executed event: %s", finalDurationResult.avg)
	log.Printf("ℹ️ final average scheduling time per event: %s", finalScheduledResult.avg)

	if expected != executed {
		log.Printf("⚠️ warning: pushed and executed counts do not match: expected=%d got=%d", expected, executed)
	}

	if expected != uniques {
		return fmt.Errorf("❌ pushed and unique executed counts do not match: expected=%d got=%d (fanout=%d pushed=%d dagSteps=%d)", expected, uniques, config.EventFanout, emitted, config.DagSteps)
	}

	// Add a small tolerance (1% or 1ms, whichever is smaller)
	tolerance := config.AverageDurationThreshold / 100 // 1% tolerance
	if tolerance > time.Millisecond {
		tolerance = time.Millisecond
	}
	thresholdWithTolerance := config.AverageDurationThreshold + tolerance

	if finalDurationResult.avg > thresholdWithTolerance {
		return fmt.Errorf("❌ average duration per executed event is greater than the threshold (with tolerance): %s > %s (threshold: %s, tolerance: %s)", finalDurationResult.avg, thresholdWithTolerance, config.AverageDurationThreshold, tolerance)
	}

	log.Printf("✅ success")

	return nil
}
