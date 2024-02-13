//go:build e2e

package main

import (
	"syscall"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/hatchet-dev/hatchet/internal/testutils"
)

func TestCLI(t *testing.T) {
	testutils.Prepare(t)
	defer func() {
		if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
			t.Fatalf("syscall.Kill() error = %v", err)
		}

		time.Sleep(2 * time.Second)

		goleak.VerifyNone(
			t,
			goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),
			goleak.IgnoreTopFunction("github.com/hatchet-dev/hatchet/pkg/cmdutils.InterruptChan.func1()"),
			goleak.IgnoreTopFunction("google.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run"),
			goleak.IgnoreTopFunction("internal/poll.runtime_pollWait"),
			goleak.IgnoreTopFunction("google.golang.org/grpc/internal/transport.(*controlBuffer).get"),
		)
	}()

	//goleak.IgnoreCurrent()

	// events per second. do not exceed too many as it might fail on lower end machines
	eventsPerSecond := 10
	concurrency := 0
	duration := 10 * time.Second
	wait := 20 * time.Second
	delay := 0 * time.Second
	if err := do(duration, eventsPerSecond, delay, wait, concurrency); err != nil {
		t.Fatalf("do() error = %v", err)
	}
}
