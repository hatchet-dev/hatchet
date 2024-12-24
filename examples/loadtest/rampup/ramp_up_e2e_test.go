//go:build load

package rampup

import (
	"context"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/internal/testutils"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	"github.com/hatchet-dev/hatchet/pkg/logger"
)

func TestRampUp(t *testing.T) {
	testutils.Prepare(t)

	type RampupArgs struct {
		duration time.Duration
		increase time.Duration
		amount   int
		delay    time.Duration
		wait     time.Duration
		// includeDroppedEvents is whether to fail on events that were dropped due to being scheduled too late
		includeDroppedEvents bool
		// maxAcceptableTotalDuration is the maximum acceptable duration for a single event to be scheduled (from start to finish)
		maxAcceptableTotalDuration time.Duration
		// maxAcceptableScheduleTime is the maximum acceptable time for an event to be purely scheduled, regardless of whether it will run or not
		maxAcceptableScheduleTime time.Duration
		concurrency               int
		startEventsPerSecond      int
		passingEventNumber        int
	}

	l = logger.NewStdErr(
		&shared.LoggerConfigFile{
			Level:  "warn",
			Format: "console",
		},
		"loadtest",
	)

	// get ramp up duration from env
	maxAcceptableDurationSeconds := 10 * time.Second

	if os.Getenv("RAMP_UP_DURATION_TIMEOUT") != "" {
		var parseErr error
		maxAcceptableDurationSeconds, parseErr = time.ParseDuration(os.Getenv("RAMP_UP_DURATION_TIMEOUT"))

		if parseErr != nil {
			t.Fatalf("could not parse RAMP_UP_DURATION_TIMEOUT %s: %s", os.Getenv("RAMP_UP_DURATION_TIMEOUT"), parseErr)
		}
	}

	log.Printf("TestRampUp with maxAcceptableDurationSeconds: %s", maxAcceptableDurationSeconds.String())

	tests := []struct {
		name    string
		args    RampupArgs
		wantErr bool
	}{
		{
			name: "normal test",
			args: RampupArgs{
				startEventsPerSecond:       1,
				duration:                   300 * time.Second,
				increase:                   10 * time.Second,
				amount:                     5,
				delay:                      0 * time.Second,
				wait:                       10 * time.Second,
				includeDroppedEvents:       true,
				maxAcceptableTotalDuration: maxAcceptableDurationSeconds,
				maxAcceptableScheduleTime:  2 * time.Second,
				concurrency:                0,
				passingEventNumber:         10000,
			},
		},
		{
			name: "time to first event test",
			args: RampupArgs{
				startEventsPerSecond:       1,
				duration:                   10 * time.Second,
				increase:                   1 * time.Second,
				amount:                     1,
				delay:                      0 * time.Second,
				wait:                       10 * time.Second,
				includeDroppedEvents:       true,
				maxAcceptableTotalDuration: 2 * time.Second,
				maxAcceptableScheduleTime:  50 * time.Millisecond,
				concurrency:                0,
				passingEventNumber:         1,
			},
		},
		{
			name: "time to first execute test",
			args: RampupArgs{
				startEventsPerSecond:       1,
				duration:                   10 * time.Second,
				increase:                   10 * time.Second,
				amount:                     1,
				delay:                      0 * time.Second,
				wait:                       10 * time.Second,
				includeDroppedEvents:       true,
				maxAcceptableTotalDuration: 2 * time.Second,
				maxAcceptableScheduleTime:  150 * time.Millisecond,
				concurrency:                0,
				passingEventNumber:         1,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	engineCleanup := sync.WaitGroup{}

	go func() {
		engineCleanup.Add(1)
		log.Printf("setup start")
		testutils.SetupEngine(ctx, t)
		log.Printf("Returning from SetupEngine ctx must have been cancelled")
		engineCleanup.Done()

	}()

	// TODO instead of waiting, figure out when the engine setup is complete
	time.Sleep(15 * time.Second)

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			doCtx, doCancel := context.WithCancel(ctx)
			if err := Do(doCtx, tt.args.duration, tt.args.startEventsPerSecond, tt.args.amount, tt.args.increase, tt.args.wait, tt.args.maxAcceptableTotalDuration, tt.args.maxAcceptableScheduleTime, tt.args.includeDroppedEvents, tt.args.concurrency, tt.args.passingEventNumber); (err != nil) != tt.wantErr {
				t.Errorf("do() error = %v, wantErr %v", err, tt.wantErr)
			}
			doCancel()
		})
	}
	// give the workers some time to cancel
	time.Sleep(2 * time.Second)

	cancel()

	log.Printf("test complete")

	engineCleanup.Wait()
	log.Printf("cleanup complete")
}
