//go:build load

package rampup

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/testing/harness"
)

func TestMain(m *testing.M) {
	harness.RunTestWithEngine(m)
}

func TestRampUp(t *testing.T) {
	type args struct {
		duration time.Duration
		increase time.Duration
		amount   int
		delay    time.Duration
		wait     time.Duration
		// includeDroppedEvents is whether to fail on events that were dropped due to being scheduled too late
		includeDroppedEvents bool
		// maxAcceptableDuration is the maximum acceptable duration for a single event to be scheduled (from start to finish)
		maxAcceptableDuration time.Duration
		// maxAcceptableSchedule is the maximum acceptable time for an event to be purely scheduled, regardless of whether it will run or not
		maxAcceptableSchedule time.Duration
		concurrency           int
		startEventsPerSecond  int
	}

	l = logger.NewStdErr(
		&shared.LoggerConfigFile{
			Level:  "warn",
			Format: "console",
		},
		"loadtest",
	)

	// get ramp up duration from env
	maxAcceptableDurationSeconds := 2 * time.Second

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
		args    args
		wantErr bool
	}{{
		name: "normal test",
		args: args{
			startEventsPerSecond:  1,
			duration:              300 * time.Second,
			increase:              10 * time.Second,
			amount:                1,
			delay:                 0 * time.Second,
			wait:                  30 * time.Second,
			includeDroppedEvents:  true,
			maxAcceptableDuration: maxAcceptableDurationSeconds,
			maxAcceptableSchedule: 2 * time.Second,
			concurrency:           0,
		},
	}}

	// TODO instead of waiting, figure out when the engine setup is complete
	time.Sleep(15 * time.Second)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if err := do(tt.args.duration, tt.args.startEventsPerSecond, tt.args.amount, tt.args.increase, tt.args.delay, tt.args.wait, tt.args.maxAcceptableDuration, tt.args.maxAcceptableSchedule, tt.args.includeDroppedEvents, tt.args.concurrency); (err != nil) != tt.wantErr {
				t.Errorf("do() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
