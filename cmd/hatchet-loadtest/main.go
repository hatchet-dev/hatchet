package main

import (
	"log"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	"github.com/hatchet-dev/hatchet/pkg/logger"

	"net/http"
	_ "net/http/pprof" // nolint: gosec
)

var l zerolog.Logger

// LoadTestConfig holds all configuration for the load test
type LoadTestConfig struct {
	Namespace      string
	Events         int
	Concurrency    int
	Duration       time.Duration
	Wait           time.Duration
	Delay          time.Duration
	WorkerDelay    time.Duration
	Slots          int
	FailureRate    float32
	PayloadSize    string
	EventFanout    int
	DagSteps       int
	RlKeys         int
	RlLimit        int
	RlDurationUnit string
}

func main() {
	config := LoadTestConfig{}

	var loadtest = &cobra.Command{
		Use: "loadtest",
		Run: func(cmd *cobra.Command, args []string) {
			l = logger.NewStdErr(
				&shared.LoggerConfigFile{
					Level:  cmd.Flag("level").Value.String(),
					Format: "console",
				},
				"loadtest",
			)

			// enable pprof if requested
			if os.Getenv("PPROF_ENABLED") == "true" {
				go func() {
					log.Println(http.ListenAndServe("localhost:6060", nil)) // nolint: gosec
				}()
			}

			config.Namespace = os.Getenv("HATCHET_CLIENT_NAMESPACE")

			if err := do(config); err != nil {
				log.Println(err)
				panic("load test failed")
			}
		},
	}

	loadtest.Flags().IntVarP(&config.Events, "events", "e", 10, "events per second")
	loadtest.Flags().IntVarP(&config.Concurrency, "concurrency", "c", 0, "concurrency specifies the maximum events to run at the same time")
	loadtest.Flags().DurationVarP(&config.Duration, "duration", "d", 10*time.Second, "duration specifies the total time to run the load test")
	loadtest.Flags().DurationVarP(&config.Delay, "delay", "D", 0, "delay specifies the time to wait in each event to simulate slow tasks")
	loadtest.Flags().DurationVarP(&config.Wait, "wait", "w", 10*time.Second, "wait specifies the total time to wait until events complete")
	loadtest.Flags().DurationVarP(&config.WorkerDelay, "workerDelay", "p", 0*time.Second, "workerDelay specifies the time to wait before starting the worker")
	loadtest.Flags().IntVarP(&config.Slots, "slots", "s", 0, "slots specifies the number of slots to use in the worker")
	loadtest.Flags().Float32VarP(&config.FailureRate, "failureRate", "f", 0, "failureRate specifies the rate of failure for the worker")
	loadtest.Flags().StringVarP(&config.PayloadSize, "payloadSize", "P", "0kb", "payload specifies the size of the payload to send")
	loadtest.Flags().IntVarP(&config.EventFanout, "eventFanout", "F", 1, "eventFanout specifies the number of events to fanout")
	loadtest.Flags().IntVarP(&config.DagSteps, "dagSteps", "g", 1, "dagSteps specifies the number of steps in the DAG")
	loadtest.Flags().IntVar(&config.RlKeys, "rlKeys", 0, "rlKeys specifies the number of keys to use in the rate limit")
	loadtest.Flags().IntVar(&config.RlLimit, "rlLimit", 0, "rlLimit specifies the rate limit")
	loadtest.Flags().StringVar(&config.RlDurationUnit, "rlDurationUnit", "second", "rlDurationUnit specifies the duration unit for the rate limit (second, minute, hour)")
	loadtest.Flags().StringVarP(&logLevel, "level", "l", "info", "logLevel specifies the log level (debug, info, warn, error)")

	cmd := &cobra.Command{Use: "app"}
	cmd.AddCommand(loadtest)
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}

// Variable to store the log level which is used to configure the logger
var logLevel string
