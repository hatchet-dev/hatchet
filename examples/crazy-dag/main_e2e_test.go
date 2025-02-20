//go:build e2e

package main

import (
	"context"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/internal/testutils"
)

func TestCrazyDAG(t *testing.T) {

	testutils.Prepare(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	results := make(chan *stepOutput, 50)

	go func() {
		err := run(ctx, results)

		if err != nil {
			t.Fatalf("/run() error = %v", err)
		}
	}()

	var count int
outer:
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("ctx done waiting for DAG to complete finished %d of %d steps", count, 40)
			break outer

		case <-results:
			count++
			if count == 40 {
				// 40 is the number of steps in the DAG
				break outer
			}

		// timeout is longer because of how long it takes things to start up
		case <-time.After(120 * time.Second):
			t.Fatalf("timeout waiting for DAG to complete finished %d of %d steps", count, 40)
		}
	}

	if count != 40 {
		t.Fatalf("expected 40 steps to complete, got %d", count)

	}

	// give the worker time to handle the last event
	time.Sleep(50 * time.Millisecond)
}
