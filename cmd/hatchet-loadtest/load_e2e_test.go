//go:build load

package main

import (
	"log"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/random"
	"github.com/hatchet-dev/hatchet/pkg/testing/harness"
)

func TestMain(m *testing.M) {
	harness.RunTestWithEngine(m)
}

func TestLoadCLI(t *testing.T) {
	type args struct {
		duration        time.Duration
		eventsPerSecond int
		delay           time.Duration
		wait            time.Duration
		workerDelay     time.Duration
		concurrency     int
	}

	l = logger.NewStdErr(
		&shared.LoggerConfigFile{
			Level:  "warn",
			Format: "console",
		},
		"loadtest",
	)

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test with high step delay",
			args: args{
				duration:        240 * time.Second,
				eventsPerSecond: 10,
				delay:           10 * time.Second,
				wait:            60 * time.Second,
				concurrency:     0,
			},
		}, {
			name: "test simple with unlimited concurrency",
			args: args{
				duration:        240 * time.Second,
				eventsPerSecond: 10,
				delay:           0 * time.Second,
				wait:            60 * time.Second,
				concurrency:     0,
			},
		},
		{
			name: "test with global concurrency key",
			args: args{
				duration:        240 * time.Second,
				eventsPerSecond: 10,
				delay:           0 * time.Second,
				wait:            60 * time.Second,
				concurrency:     10,
			},
		},
		{
			name: "test for many queued events and little worker throughput",
			args: args{
				duration:        240 * time.Second,
				eventsPerSecond: 10,
				delay:           0 * time.Second,
				workerDelay:     120 * time.Second, // will write 1200 events before the worker is ready
				wait:            120 * time.Second,
				concurrency:     0,
			},
		},
	}

	// TODO instead of waiting, figure out when the engine setup is complete
	time.Sleep(15 * time.Second)

	for _, tt := range tests {
		tt := tt // pin the loop variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			namespace, err := random.Generate(8)

			if err != nil {
				t.Fatalf("could not generate random namespace: %s", err)
			}

			if err := do(namespace, tt.args.duration, tt.args.eventsPerSecond, tt.args.delay, tt.args.wait, tt.args.concurrency, tt.args.workerDelay, 100, 0.0, "0kb", 1); (err != nil) != tt.wantErr {
				t.Errorf("do() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	log.Printf("test complete")
}
