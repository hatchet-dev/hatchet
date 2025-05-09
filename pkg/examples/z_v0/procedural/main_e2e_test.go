//go:build e2e

package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/internal/testutils"
)

func TestProcedural(t *testing.T) {
	testutils.Prepare(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	events := make(chan string, 5*NUM_CHILDREN)

	cleanup, err := run(events)
	if err != nil {
		t.Fatalf("/run() error = %v", err)
	}

	var items []string

outer:
	for {
		select {
		case item := <-events:
			items = append(items, item)
		case <-ctx.Done():
			break outer
		}
	}

	expected := []string{}

	for i := 0; i < NUM_CHILDREN; i++ {
		expected = append(expected, fmt.Sprintf("child-%d-started", i))
		expected = append(expected, fmt.Sprintf("child-%d-completed", i))
	}

	assert.ElementsMatch(t, expected, items)

	if err := cleanup(); err != nil {
		t.Fatalf("cleanup() error = %v", err)
	}
}
