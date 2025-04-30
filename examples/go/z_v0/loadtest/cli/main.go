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
	_ "net/http/pprof"
)

var l zerolog.Logger

func main() {
	var events int
	var concurrency int
	var duration time.Duration
	var wait time.Duration
	var delay time.Duration
	var workerDelay time.Duration
	var logLevel string
	var slots int
	var failureRate float32
	var payloadSize string
	var eventFanout int

	var loadtest = &cobra.Command{
		Use: "loadtest",
		Run: func(cmd *cobra.Command, args []string) {
			l = logger.NewStdErr(
				&shared.LoggerConfigFile{
					Level:  logLevel,
					Format: "console",
				},
				"loadtest",
			)

			// enable pprof if requested
			if os.Getenv("PPROF_ENABLED") == "true" {
				go func() {
					log.Println(http.ListenAndServe("localhost:6060", nil))
				}()
			}

			if err := do(duration, events, delay, wait, concurrency, workerDelay, slots, failureRate, payloadSize, eventFanout); err != nil {
				log.Println(err)
				panic("load test failed")
			}
		},
	}

	loadtest.Flags().IntVarP(&events, "events", "e", 10, "events per second")
	loadtest.Flags().IntVarP(&concurrency, "concurrency", "c", 0, "concurrency specifies the maximum events to run at the same time")
	loadtest.Flags().DurationVarP(&duration, "duration", "d", 10*time.Second, "duration specifies the total time to run the load test")
	loadtest.Flags().DurationVarP(&delay, "delay", "D", 0, "delay specifies the time to wait in each event to simulate slow tasks")
	loadtest.Flags().DurationVarP(&wait, "wait", "w", 10*time.Second, "wait specifies the total time to wait until events complete")
	loadtest.Flags().DurationVarP(&workerDelay, "workerDelay", "p", 0*time.Second, "workerDelay specifies the time to wait before starting the worker")
	loadtest.Flags().StringVarP(&logLevel, "level", "l", "info", "logLevel specifies the log level (debug, info, warn, error)")
	loadtest.Flags().IntVarP(&slots, "slots", "s", 0, "slots specifies the number of slots to use in the worker")
	loadtest.Flags().Float32VarP(&failureRate, "failureRate", "f", 0, "failureRate specifies the rate of failure for the worker")
	loadtest.Flags().StringVarP(&payloadSize, "payloadSize", "P", "0kb", "payload specifies the size of the payload to send")
	loadtest.Flags().IntVarP(&eventFanout, "eventFanout", "F", 1, "eventFanout specifies the number of events to fanout")

	cmd := &cobra.Command{Use: "app"}
	cmd.AddCommand(loadtest)
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
