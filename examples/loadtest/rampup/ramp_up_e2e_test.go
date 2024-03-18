//go:build load

package rampup

import (
	"context"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/internal/testutils"
)

func TestRampUp(t *testing.T) {
	testutils.Prepare(t)

	type args struct {
		duration             time.Duration
		increase             time.Duration
		amount               int
		delay                time.Duration
		wait                 time.Duration
		maxAcceptableDelay   time.Duration
		concurrency          int
		startEventsPerSecond int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{{
		name: "test with high step delay",
		args: args{
			startEventsPerSecond: 5,
			duration:             300 * time.Second,
			increase:             10 * time.Second,
			amount:               1,
			delay:                10 * time.Second,
			wait:                 30 * time.Second,
			maxAcceptableDelay:   1 * time.Second,
			concurrency:          0,
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
	time.Sleep(10 * time.Second)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if err := do(tt.args.duration, tt.args.startEventsPerSecond, tt.args.amount, tt.args.increase, tt.args.delay, tt.args.wait, tt.args.maxAcceptableDelay, tt.args.concurrency); (err != nil) != tt.wantErr {
				t.Errorf("do() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	cancel()

	log.Printf("test complete")
	setup.Wait()
	log.Printf("cleanup complete")
}
