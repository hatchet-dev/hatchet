package rampup

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

var l zerolog.Logger

type taskTracker struct {
	eventPushed    bool
	taskExecuted   bool
	pushedAt       time.Time
	executedAt     time.Time
	checkedTimeout bool
}

func do(duration time.Duration, startEventsPerSecond, amount int, increase, delay, wait, maxAcceptableDuration, maxAcceptableSchedule time.Duration, includeDroppedEvents bool, concurrency int) error {
	l.Debug().Msgf("testing with duration=%s, amount=%d, increase=%d, delay=%s, wait=%s, concurrency=%d", duration, amount, increase, delay, wait, concurrency)

	ctx, cancel := context.WithTimeout(context.Background(), duration+10*time.Second+wait+5*time.Second)
	defer cancel()

	after := 10 * time.Second

	hook := make(chan time.Duration, 1)

	scheduledTimes := make(chan time.Duration, 100000)
	executionTimes := make(chan time.Duration, 100000)
	scheduled := make(chan int64, 100000)
	executed := make(chan int64, 100000)

	tasks := make(map[int64]*taskTracker)
	tasksLock := sync.Mutex{}

	// Helper function to check if task is complete and clean up
	checkComplete := func(id int64, tracker *taskTracker) {
		if tracker.eventPushed && tracker.taskExecuted {
			l.Debug().Msgf("task %d complete (pushed and executed)", id)
			delete(tasks, id)
		}
	}

	// Compute running average for scheduled times
	type avgResult struct {
		count int64
		avg   time.Duration
	}
	scheduledResult := make(chan avgResult)
	go func() {
		var count int64
		var avg time.Duration
		for d := range scheduledTimes {
			count++
			if count == 1 {
				avg = d
			} else {
				avg += (d - avg) / time.Duration(count)
			}
		}
		scheduledResult <- avgResult{count: count, avg: avg}
	}()

	// Compute running average for execution times
	executionResult := make(chan avgResult)
	go func() {
		var count int64
		var avg time.Duration
		for d := range executionTimes {
			count++
			if count == 1 {
				avg = d
			} else {
				avg += (d - avg) / time.Duration(count)
			}
		}
		executionResult <- avgResult{count: count, avg: avg}
	}()

	// Handler for scheduled events
	go func() {
		for s := range scheduled {
			l.Debug().Msgf("scheduled %d", s)
			tasksLock.Lock()
			tracker, exists := tasks[s]
			if !exists {
				tracker = &taskTracker{}
				tasks[s] = tracker
			}
			tracker.eventPushed = true
			tracker.pushedAt = time.Now()
			checkComplete(s, tracker)
			tasksLock.Unlock()
		}
	}()

	// Handler for executed events
	go func() {
		for e := range executed {
			l.Debug().Msgf("executed %d", e)
			tasksLock.Lock()
			tracker, exists := tasks[e]
			if !exists {
				tracker = &taskTracker{}
				tasks[e] = tracker
			}
			tracker.taskExecuted = true
			tracker.executedAt = time.Now()
			checkComplete(e, tracker)
			tasksLock.Unlock()
		}
	}()

	// Single timeout checker goroutine
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				tasksLock.Lock()
				for id, tracker := range tasks {
					if tracker.eventPushed && !tracker.taskExecuted && !tracker.checkedTimeout {
						if time.Since(tracker.pushedAt) > maxAcceptableDuration {
							tracker.checkedTimeout = true
							if includeDroppedEvents {
								tasksLock.Unlock()
								panic(fmt.Errorf("event %d did not execute in time", id))
							}
							l.Warn().Msgf("event %d did not execute in time", id)
						}
					}
				}
				tasksLock.Unlock()
			}
		}
	}()

	go func() {
		run(ctx, delay, concurrency, maxAcceptableDuration, hook, executed, executionTimes)
	}()

	emitted := emit(ctx, startEventsPerSecond, amount, increase, duration, maxAcceptableSchedule, hook, scheduled, scheduledTimes)

	time.Sleep(after)

	close(scheduled)
	close(executed)
	close(scheduledTimes)
	close(executionTimes)
	close(hook)

	finalScheduledResult := <-scheduledResult
	finalExecutionResult := <-executionResult

	log.Printf("ℹ️ emitted %d events", emitted)
	log.Printf("ℹ️ final average scheduling time per event: %s", finalScheduledResult.avg)
	log.Printf("ℹ️ final average execution time per event: %s", finalExecutionResult.avg)

	log.Printf("✅ success")

	return nil
}
