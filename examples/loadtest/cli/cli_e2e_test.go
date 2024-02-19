//go:build load

package main

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/internal/testutils"
)

func TestLoadCLI(t *testing.T) {
	testutils.Prepare(t)

	type args struct {
		duration        time.Duration
		eventsPerSecond int
		delay           time.Duration
		wait            time.Duration
		concurrency     int
	}
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
			wait:            20 * time.Second,
			concurrency:     0,
		},
	}, {
		name: "test with high step delay",
		args: args{
			duration:        10 * time.Second,
			eventsPerSecond: 10,
			delay:           10 * time.Second,
			wait:            30 * time.Second,
			concurrency:     0,
		},
	}}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	testutils.Setup(t, ctx)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				time.Sleep(1 * time.Second)

				// TODO re-enable
				//goleak.VerifyNone(
				//	t,
				//	goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),
				//	goleak.IgnoreTopFunction("google.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run"),
				//	goleak.IgnoreTopFunction("internal/poll.runtime_pollWait"),
				//	goleak.IgnoreTopFunction("google.golang.org/grpc/internal/transport.(*controlBuffer).get"),
				//)
			}()

			if err := do(tt.args.duration, tt.args.eventsPerSecond, tt.args.delay, tt.args.wait, tt.args.concurrency); (err != nil) != tt.wantErr {
				t.Errorf("do() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	cancel()

	log.Printf("test complete")
	time.Sleep(30 * time.Second)
	log.Printf("cleanup complete")
}
