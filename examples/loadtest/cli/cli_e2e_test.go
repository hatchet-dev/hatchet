//go:build load

package main

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
	// This runs before all tests
	t := &testing.T{}
	postRun := harness.StartEngine(t)

	// Run all tests in this package
	code := m.Run()

	if code != 0 {
		log.Printf("TestMain: code %d", code)
		os.Exit(code)
	}

	postRun()

	// determine if t is failed
	if t.Failed() {
		log.Printf("TestMain: test failed")
		os.Exit(1)
	}
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
	}{{
		name: "test with high step delay",
		args: args{
			duration:        10 * time.Second,
			eventsPerSecond: 10,
			delay:           10 * time.Second,
			wait:            60 * time.Second,
			concurrency:     0,
		},
	}, {
		name: "test simple with unlimited concurrency",
		args: args{
			duration:        10 * time.Second,
			eventsPerSecond: 10,
			delay:           0 * time.Second,
			wait:            60 * time.Second,
			concurrency:     0,
		},
	}, {
		name: "test for many queued events and little worker throughput",
		args: args{
			duration:        60 * time.Second,
			eventsPerSecond: 100,
			delay:           0 * time.Second,
			workerDelay:     60 * time.Second,
			wait:            240 * time.Second,
			concurrency:     0,
		},
	}}

	// TODO instead of waiting, figure out when the engine setup is complete
	time.Sleep(15 * time.Second)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := do(tt.args.duration, tt.args.eventsPerSecond, tt.args.delay, tt.args.wait, tt.args.concurrency, tt.args.workerDelay, 100, 0.0, "0kb", 1); (err != nil) != tt.wantErr {
				t.Errorf("do() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	log.Printf("test complete")
}
