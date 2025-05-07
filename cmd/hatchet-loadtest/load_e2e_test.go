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
	// We're using LoadTestConfig directly instead of an args struct

	l = logger.NewStdErr(
		&shared.LoggerConfigFile{
			Level:  "warn",
			Format: "console",
		},
		"loadtest",
	)

	tests := []struct {
		name    string
		config  LoadTestConfig
		wantErr bool
	}{
		{
			name: "test with high step delay",
			config: LoadTestConfig{
				Duration:       240 * time.Second,
				Events:         10,
				Delay:          10 * time.Second,
				Wait:           60 * time.Second,
				Concurrency:    0,
				Slots:          100,
				FailureRate:    0.0,
				PayloadSize:    "0kb",
				EventFanout:    1,
				DagSteps:       1,
				RlKeys:         0,
				RlLimit:        0,
				RlDurationUnit: "",
			},
		}, {
			name: "test simple with unlimited concurrency",
			config: LoadTestConfig{
				Duration:       240 * time.Second,
				Events:         10,
				Delay:          0 * time.Second,
				Wait:           60 * time.Second,
				Concurrency:    0,
				Slots:          100,
				FailureRate:    0.0,
				PayloadSize:    "0kb",
				EventFanout:    1,
				DagSteps:       1,
				RlKeys:         0,
				RlLimit:        0,
				RlDurationUnit: "",
			},
		},
		{
			name: "test with DAG",
			config: LoadTestConfig{
				Duration:       240 * time.Second,
				Events:         10,
				Delay:          0 * time.Second,
				Wait:           60 * time.Second,
				Concurrency:    0,
				Slots:          100,
				FailureRate:    0.0,
				PayloadSize:    "0kb",
				EventFanout:    1,
				DagSteps:       2,
				RlKeys:         0,
				RlLimit:        0,
				RlDurationUnit: "",
			},
		},
		{
			name: "test with event fanout",
			config: LoadTestConfig{
				Duration:       240 * time.Second,
				Events:         10,
				Delay:          0 * time.Second,
				Wait:           60 * time.Second,
				Concurrency:    0,
				Slots:          100,
				FailureRate:    0.0,
				PayloadSize:    "0kb",
				EventFanout:    2,
				DagSteps:       1,
				RlKeys:         0,
				RlLimit:        0,
				RlDurationUnit: "",
			},
		},
		{
			name: "test with global concurrency key",
			config: LoadTestConfig{
				Duration:       240 * time.Second,
				Events:         10,
				Delay:          0 * time.Second,
				Wait:           60 * time.Second,
				Concurrency:    10,
				Slots:          100,
				FailureRate:    0.0,
				PayloadSize:    "0kb",
				EventFanout:    1,
				DagSteps:       1,
				RlKeys:         0,
				RlLimit:        0,
				RlDurationUnit: "",
			},
		},
		{
			name: "test for many queued events and little worker throughput",
			config: LoadTestConfig{
				Duration:       240 * time.Second,
				Events:         10,
				Delay:          0 * time.Second,
				WorkerDelay:    120 * time.Second, // will write 1200 events before the worker is ready
				Wait:           120 * time.Second,
				Concurrency:    0,
				Slots:          100,
				FailureRate:    0.0,
				PayloadSize:    "0kb",
				EventFanout:    1,
				DagSteps:       1,
				RlKeys:         0,
				RlLimit:        0,
				RlDurationUnit: "",
			},
		},
		{
			name: "test with rate limits",
			config: LoadTestConfig{
				Duration:       240 * time.Second,
				Events:         10,
				Delay:          0 * time.Second,
				Wait:           60 * time.Second,
				Concurrency:    0,
				Slots:          100,
				FailureRate:    0.0,
				PayloadSize:    "0kb",
				EventFanout:    1,
				DagSteps:       1,
				RlKeys:         10,
				RlLimit:        100,
				RlDurationUnit: "second",
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

			testConfig := tt.config
			testConfig.Namespace = namespace
			if err := do(testConfig); (err != nil) != tt.wantErr {
				t.Errorf("do() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	log.Printf("test complete")
}
