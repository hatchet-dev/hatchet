//go:build e2e

package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/internal/testutils"
)

func TestSimple(t *testing.T) {
	testutils.Prepare(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	events := make(chan string, 50)

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

	assert.Equal(t, []string{
		"step-one",
		"step-two",
	}, items)

	if err := cleanup(); err != nil {
		t.Fatalf("cleanup() error = %v", err)
	}
}
