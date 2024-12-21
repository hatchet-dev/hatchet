package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	clientconfig "github.com/hatchet-dev/hatchet/pkg/config/client"

	"github.com/hatchet-dev/hatchet/pkg/client"
)

func generateNamespace() string {
	return fmt.Sprintf("loadtest-%d", time.Now().Unix())
}

func do(ctx context.Context, duration time.Duration, eventsPerSecond int, delay time.Duration, concurrency int, workerDelay time.Duration, maxPerEventTime time.Duration, maxPerExecution time.Duration) error {
	l.Info().Msgf("testing with duration=%s, eventsPerSecond=%d, delay=%s,  concurrency=%d", duration, eventsPerSecond, delay, concurrency)
	c, err := client.NewFromConfigFile(&clientconfig.ClientConfigFile{
		Namespace: generateNamespace(),
	})

	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	// catch an interrupt signal
	sigChan := make(chan os.Signal, 1)

	// Notify the channel of interrupt and terminate signals
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	ch := make(chan int64, 2)
	durations := make(chan time.Duration, eventsPerSecond*int(duration.Seconds())*3)
	workerCtx, workerCancel := context.WithCancel(context.Background())

	defer workerCancel()
	executedChan := make(chan int64, eventsPerSecond*int(duration.Seconds())*2)
	emittedChan := make(chan int64, 1)
	duplicateChan := make(chan int64, 1)
	executedCount := int64(0)

	go func() {
		if workerDelay.Seconds() > 0 {

			l.Info().Msgf("wait %s before starting the worker", workerDelay)
			time.Sleep(workerDelay)
		}
		l.Info().Msg("starting worker now")

		uniques := runWorker(workerCtx, c, delay, durations, concurrency, executedChan, duplicateChan)

		select {
		case ch <- uniques:
		case <-workerCtx.Done():
			l.Error().Msg("worker cancelled before finishing")
		}

		l.Info().Msg("worker finished")
	}()

	// we need to wait for the worker to start so that the workflow is registered and we don't miss any events
	// otherwise we could process the events before we have a workflow registered for them

	time.Sleep(15 * time.Second) // wait for the worker to start

	scheduled := make(chan time.Duration, eventsPerSecond*int(duration.Seconds())*2)
	var emittedCount int64

	startedAt := time.Now()
	go func() {

		select {

		case <-ctx.Done():
			l.Error().Msg("context done before finishing emit")
			return
		case emittedChan <- emit(ctx, c, eventsPerSecond, duration, scheduled):
		}

	}()

	// going to allow 10% of the duration to wait for all the events to consumed
	after := duration / 10
	var movingTimeout = time.Now().Add(duration + after)
	var totalTimeout = time.Now().Add(duration + after)

	totalTimeoutTimer := time.NewTimer(time.Until(totalTimeout))
	defer totalTimeoutTimer.Stop()

	movingTimeoutTimer := time.NewTimer(time.Until(movingTimeout))
	defer movingTimeoutTimer.Stop()
outer:
	for {
		select {
		case <-sigChan:
			l.Info().Msg("interrupted")
			return nil
		case <-ctx.Done():
			l.Info().Msg("context done")
			return nil

		case dupeId := <-duplicateChan:
			return fmt.Errorf("❌ duplicate event %d", dupeId)

		case <-totalTimeoutTimer.C:
			l.Info().Msg("timed out")
			return fmt.Errorf("❌ timed out after %s", duration+after)

		case <-movingTimeoutTimer.C:
			l.Info().Msg("timeout")
			return fmt.Errorf("❌ timed out waiting for activity")

		case executed := <-executedChan:
			l.Info().Msgf("executed %d", executed)
			executedCount++
			movingTimeout = time.Now().Add(5 * time.Second)
			l.Info().Msgf("Set the timeout to %s", movingTimeout)
			if !movingTimeoutTimer.Stop() {
				<-movingTimeoutTimer.C
			}
			movingTimeoutTimer.Reset(time.Until(movingTimeout))

			if emittedCount > 0 {

				if executedCount == emittedCount {
					// this is the finished condition
					break outer
				}
				if executedCount > emittedCount {
					l.Error().Msgf("❌ executed more events than emitted executed=%d, emitted=%d", executedCount, emittedCount)
					return fmt.Errorf("❌ executed more events than emitted")
				}
			}

		case emittedCount = <-emittedChan:

			l.Info().Msgf("emitted %d", emittedCount)
		}

	}
	timeTaken := time.Since(startedAt)
	workerCancel()
	executed := <-ch

	l.Info().Msgf("emitted %d, executed %d, using %d events/s", emittedCount, executed, eventsPerSecond)

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
	for i := 0; i < int(emittedCount); i++ {
		totalDurationScheduled += <-scheduled
	}
	scheduleTimePerEvent := totalDurationScheduled / time.Duration(emittedCount)

	log.Printf("ℹ️ average scheduling time per event: %s", scheduleTimePerEvent)

	if emittedCount != executed {
		return fmt.Errorf("❌ emitted and executed counts do not match: %d != %d", emittedCount, executed)
	}

	if maxPerEventTime > 0 && scheduleTimePerEvent > maxPerEventTime {
		return fmt.Errorf("❌ scheduling time per event %s exceeds max %s", scheduleTimePerEvent, maxPerEventTime)
	}

	if maxPerExecution > 0 && durationPerEventExecuted > maxPerExecution {
		return fmt.Errorf("❌ duration per event executed %s exceeds max %s", durationPerEventExecuted, maxPerExecution)
	}
	log.Printf("Executed %d events in %s for %.2f events per second",
		executedCount,
		timeTaken,
		float64(executedCount)/timeTaken.Seconds())

	log.Printf("✅ success")

	return nil
}
