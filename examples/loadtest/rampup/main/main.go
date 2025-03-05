package main

import (
	"context"
	"log"
	"time"

	"github.com/hatchet-dev/hatchet/examples/loadtest/rampup"
)

type RampupArgs struct {
	startEventsPerSecond       int
	duration                   time.Duration
	increase                   time.Duration
	amount                     int
	wait                       time.Duration
	includeDroppedEvents       bool
	maxAcceptableTotalDuration time.Duration
	maxAcceptableScheduleTime  time.Duration
	concurrency                int
	passingEventNumber         int // number of events that should be executed to pass at these settings
}

func main() {
	ctx := context.Background()

	testArgs := RampupArgs{
		startEventsPerSecond:       1,
		duration:                   300 * time.Second,
		increase:                   5 * time.Second,
		amount:                     0,
		wait:                       30 * time.Second,
		includeDroppedEvents:       true,
		maxAcceptableTotalDuration: time.Duration(100 * time.Second),
		maxAcceptableScheduleTime:  5 * time.Millisecond,
		concurrency:                0,
		passingEventNumber:         1,
	}

	if err := rampup.Do(ctx, testArgs.duration, testArgs.startEventsPerSecond, testArgs.amount, testArgs.increase, testArgs.wait, testArgs.maxAcceptableTotalDuration, testArgs.maxAcceptableScheduleTime, testArgs.includeDroppedEvents, testArgs.concurrency, testArgs.passingEventNumber); err != nil {
		log.Println(err)
		panic("load test failed")
	}

}
