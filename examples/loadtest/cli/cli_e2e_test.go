//go:build load

package main

import (
	"context"
	"log"
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
		name: "test simple with unlimited concurrency",
		args: args{
			duration:        10 * time.Second,
			eventsPerSecond: 10,
			delay:           0 * time.Second,
			wait:            60 * time.Second,
			concurrency:     0,
		},
	}, {
		name: "test with high step delay",
		args: args{
			duration:        10 * time.Second,
			eventsPerSecond: 10,
			delay:           10 * time.Second,
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)

	setup := sync.WaitGroup{}

	go func() {
		setup.Add(1)
		log.Printf("setup start")
		testutils.SetupEngine(ctx, t)
		setup.Done()
		log.Printf("setup end")
	}()

	// TODO instead of waiting, figure out when the engine setup is complete
	time.Sleep(15 * time.Second)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := do(tt.args.duration, tt.args.eventsPerSecond, tt.args.delay, tt.args.wait, tt.args.concurrency, tt.args.workerDelay); (err != nil) != tt.wantErr {
				t.Errorf("do() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	cancel()

	log.Printf("test complete")
	setup.Wait()
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
