//go:build load

package main

import (
	"context"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/hatchet-dev/hatchet/internal/testutils"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	"github.com/hatchet-dev/hatchet/pkg/logger"
)

func TestLoadCLI(t *testing.T) {
	testutils.Prepare(t)

	durationMultiplier := 1
	if os.Getenv("SERVER_TASKQUEUE_KIND") == "postgres" {
		t.Logger().Info("using postgres, increasing timings for load test")
		durationMultiplier = 10
	}

	type args struct {
		duration        time.Duration
		eventsPerSecond int
		delay           time.Duration
		wait            time.Duration
		workerDelay     time.Duration
		concurrency     int
		maxPerEventTime time.Duration
		maxPerExecution time.Duration
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
			name: "test simple with unlimited concurrency",
			args: args{
				duration:        10 * time.Second * time.Duration(durationMultiplier),
				eventsPerSecond: 10,
				delay:           0 * time.Second,
				concurrency:     0,
				maxPerEventTime: 0,
				maxPerExecution: 0,
			},
		}, {
			name: "test with high step delay",
			args: args{
				duration:        10 * time.Second * time.Duration(durationMultiplier),
				eventsPerSecond: 10,
				delay:           4 * time.Second, // can't go higher than 5 seconds here because we timeout without activity
				concurrency:     0,
				maxPerEventTime: 0,
				maxPerExecution: 0,
			},
		},
		{
			name: "test for many queued events and little worker throughput",
			args: args{
				duration:        60 * time.Second * time.Duration(durationMultiplier),
				eventsPerSecond: 100,
				delay:           0 * time.Second,
				concurrency:     0,
				maxPerEventTime: 0,
				maxPerExecution: 0,
			},
		},

		{
			name: "test with scheduling and execution time limits",
			args: args{
				duration:        30 * time.Second * time.Duration(durationMultiplier),
				eventsPerSecond: 50,
				delay:           0 * time.Second,
				concurrency:     0,
				maxPerEventTime: 100 * time.Millisecond,
				maxPerExecution: 1 * time.Second,
			},
		}}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	engineCleanupWg := sync.WaitGroup{}

	go func() {
		log.Printf("setup start")
		engineCleanupWg.Add(1)
		testutils.SetupEngine(ctx, t)
		engineCleanupWg.Done()
		log.Printf("setup end")
	}()

	log.Printf("waiting for engine to start")

	time.Sleep(15 * time.Second)

	for _, tt := range tests {

		l.Info().Msgf("running test %s", tt.name)
		t.Run(tt.name, func(t *testing.T) {
			if err := do(ctx, tt.args.duration, tt.args.eventsPerSecond, tt.args.delay, tt.args.concurrency, tt.args.workerDelay, tt.args.maxPerEventTime, tt.args.maxPerExecution); (err != nil) != tt.wantErr {
				t.Errorf("do() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
		l.Info().Msgf("test %s complete", tt.name)

	}
	log.Printf("test complete")
	cancel()
	// wait for engine to cleanup
	engineCleanupWg.Wait()

	log.Printf("cleanup complete")

	goleak.VerifyNone(
		t,
		// worker
		goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),
		goleak.IgnoreTopFunction("google.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run"),
		goleak.IgnoreTopFunction("internal/poll.runtime_pollWait"),
		goleak.IgnoreTopFunction("google.golang.org/grpc/internal/transport.(*controlBuffer).get"),
		// all engine related packages
		goleak.IgnoreTopFunction("github.com/jackc/pgx/v5/pgxpool.(*Pool).backgroundHealthCheck"),
		goleak.IgnoreTopFunction("github.com/rabbitmq/amqp091-go.(*Connection).heartbeater"),
		goleak.IgnoreTopFunction("github.com/rabbitmq/amqp091-go.(*consumers).buffer"),
		goleak.IgnoreTopFunction("google.golang.org/grpc/internal/transport.(*http2Server).keepalive"),
	)
}
