//go:build e2e

package main

import (
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/internal/testutils"
)

func TestCLI(t *testing.T) {
	testutils.Prepare(t)

	// events per second. do not exceed too many as it might fail on lower end machines
	eventsPerSecond := 10
	duration := 10 * time.Second
	wait := 10 * time.Second
	delay := 0 * time.Second
	concurrency := 1
	if err := do(duration, eventsPerSecond, delay, wait, concurrency); err != nil {
		t.Fatalf("do() error = %v", err)
	}
}
