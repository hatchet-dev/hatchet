package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	"github.com/hatchet-dev/hatchet/pkg/loadtest/eventkeys"
	"github.com/hatchet-dev/hatchet/pkg/logger"

	"net/http"
	_ "net/http/pprof" // nolint: gosec
)

var l zerolog.Logger

// LoadTestConfig holds all configuration for the load test
type LoadTestConfig struct {
	Namespace                string
	Events                   int
	EmitWorkers              int
	Concurrency              int
	Duration                 time.Duration
	Wait                     time.Duration
	Delay                    time.Duration
	WorkerDelay              time.Duration
	RegistrationTimeout      time.Duration
	Slots                    int
	FailureRate              float32
	PayloadSize              string
	EventFanout              int
	DagSteps                 int
	RlKeys                   int
	RlLimit                  int
	RlDurationUnit           string
	AverageDurationThreshold time.Duration
	PlotDir                  string

	SelectEventKeys  []string
	ExcludeEventKeys []string
	EventKeys        []eventkeys.EventKey

	// ExternalWorker, if set, skips registering a workflow and starting an
	// in-process worker altogether - it assumes a separately-running SDK
	// worker (e.g. cmd/hatchet-loadtest/go) has already registered a
	// compatible workflow named "load-test-0" (or "load-test-{i}" for
	// i<EventFanout).
	ExternalWorker bool

	// TimingSampleRate, in --externalWorker mode, is the proportion (0, 1]
	// of completed runs to fetch full task timings for, so that there are
	// fewer REST calls necessary. E.g. 0.3 samples 30% of runs.
	TimingSampleRate float64
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

			eventKeys, err := resolveEventKeys(config.SelectEventKeys, config.ExcludeEventKeys, availableEventKeys(config.ExternalWorker))
			if err != nil {
				log.Println(err)
				panic("load test failed")
			}
			config.EventKeys = eventKeys
			l.Info().Msgf("publishing event keys: %v", config.EventKeys)

			if err := do(config); err != nil {
				log.Println(err)
				panic("load test failed")
			}
		},
	}

	eventKeyNames := eventkeys.Names(eventkeys.All)

	loadtest.Flags().IntVarP(&config.Events, "events", "e", 10, "events per second")
	loadtest.Flags().IntVar(&config.EmitWorkers, "emitWorkers", 0, "emitWorkers specifies how many concurrent goroutines push events to the engine; each push is a blocking call, so this bounds sustained throughput regardless of --events. 0 auto-scales from --events (see emit.go's defaultEmitWorkers)")
	loadtest.Flags().IntVarP(&config.Concurrency, "concurrency", "c", 0, "concurrency specifies the maximum events to run at the same time")
	loadtest.Flags().DurationVarP(&config.Duration, "duration", "d", 10*time.Second, "duration specifies the total time to run the load test")
	loadtest.Flags().DurationVarP(&config.Delay, "delay", "D", 0, "delay specifies the time to wait in each event to simulate slow tasks")
	loadtest.Flags().DurationVarP(&config.Wait, "wait", "w", 10*time.Second, "wait specifies the total time to wait until events complete")
	loadtest.Flags().DurationVarP(&config.WorkerDelay, "workerDelay", "p", 0*time.Second, "workerDelay specifies the time to wait before starting the worker")
	// CI's load-online-migrate passes 20s: it must stay below that workflow's 30s pre-migration sleep so registration is forced to succeed against the N-1 schema.
	loadtest.Flags().DurationVar(&config.RegistrationTimeout, "registrationTimeout", 60*time.Second, "registrationTimeout specifies the total time to wait for workflow registration before emitting events")
	loadtest.Flags().IntVarP(&config.Slots, "slots", "s", 0, "slots specifies the number of slots to use in the worker")
	loadtest.Flags().Float32VarP(&config.FailureRate, "failureRate", "f", 0, "failureRate specifies the rate of failure for the worker")
	loadtest.Flags().StringVarP(&config.PayloadSize, "payloadSize", "P", "0kb", "payload specifies the size of the payload to send")
	loadtest.Flags().IntVarP(&config.EventFanout, "eventFanout", "F", 1, "eventFanout specifies the number of events to fanout")
	loadtest.Flags().IntVarP(&config.DagSteps, "dagSteps", "g", 1, "dagSteps specifies the number of steps in the DAG")
	loadtest.Flags().IntVar(&config.RlKeys, "rlKeys", 0, "rlKeys specifies the number of keys to use in the rate limit")
	loadtest.Flags().IntVar(&config.RlLimit, "rlLimit", 0, "rlLimit specifies the rate limit")
	loadtest.Flags().StringVar(&config.RlDurationUnit, "rlDurationUnit", "second", "rlDurationUnit specifies the duration unit for the rate limit (second, minute, hour)")
	loadtest.Flags().StringVarP(&logLevel, "level", "l", "info", "logLevel specifies the log level (debug, info, warn, error)")
	loadtest.Flags().DurationVar(&config.AverageDurationThreshold, "averageDurationThreshold", 100*time.Millisecond, "averageDurationThreshold specifies the threshold for the average duration per executed event to be considered a success")
	loadtest.Flags().StringVar(&config.PlotDir, "plotDirectory", "", "plotDirectory specifies where to put the generated plots for latency and task duration")
	loadtest.Flags().StringSliceVar(&config.SelectEventKeys, "select", nil, fmt.Sprintf("select specifies which event keys to publish (comma-separated or repeated), e.g. --select %s. Known keys: %s. Mutually exclusive with --exclude; passing neither publishes only the standard key %q (batch/durable canaries are opt-in)", strings.Join(eventKeyNames, ","), strings.Join(eventKeyNames, ", "), eventkeys.EventKeyDefault.Name()))
	loadtest.Flags().StringSliceVar(&config.ExcludeEventKeys, "exclude", nil, "exclude specifies which event keys to skip (comma-separated or repeated); all other available keys are published. Mutually exclusive with --select")
	loadtest.Flags().BoolVar(&config.ExternalWorker, "externalWorker", false, "externalWorker skips registering a workflow and starting an in-process worker, assuming a separately-running SDK worker (e.g. cmd/hatchet-loadtest/go) has already registered a compatible workflow; worker/workflow flags (slots, dagSteps, eventFanout, rlKeys, workerDelay, failureRate, delay) are ignored in this mode")
	loadtest.Flags().Float64Var(&config.TimingSampleRate, "timingSampleRate", 1, "fetch full task timings for this proportion (0, 1] of completed runs instead of every one - e.g. 0.3 samples 30% of runs. The average still converges, at a fraction of the REST load on the engine; 1 (the default) fetches every run")
	cmd := &cobra.Command{Use: "app"}
	cmd.AddCommand(loadtest)
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}

// Variable to store the log level which is used to configure the logger
var logLevel string
