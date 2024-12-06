//go:build e2e

package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/internal/testutils"
)

func TestCrazyDAG(t *testing.T) {
	os.Setenv("HATCHET_CLIENT_NAMESPACE", randomNamespace())

	testutils.Prepare(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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
			fmt.Println("ctx.Done()")
			break outer

		case <-results:
			count++
			if count == 90 {
				// 90 is the number of steps in the DAG
				break outer
			}

		// timeout is longer because of how long it takes things to start up
		case <-time.After(120 * time.Second):
			t.Fatalf("timeout waiting for DAG to complete finished %d of %d steps", count, 90)
		}
	}

	if count != 90 {
		t.Fatalf("expected 90 steps to complete, got %d", count)
	}

	// give the worker time to handle the last event
	time.Sleep(50 * time.Millisecond)
}
