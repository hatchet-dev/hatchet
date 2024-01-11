//go:build e2e

package main

import (
	"log"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/internal/testutils"
	"github.com/stretchr/testify/assert"
)

func TestMiddleware(t *testing.T) {
	testutils.Prepare(t)

	ch := make(chan interface{}, 1)

	events := make(chan string, 50)

	go func() {
		time.Sleep(20 * time.Second)
		ch <- struct{}{}
		close(events)
		log.Printf("sent interrupt")
	}()

	if err := run(ch, events); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	var items []string
	for item := range events {
		items = append(items, item)
	}

	assert.Equal(t, []string{
		"1st-middleware",
		"2nd-middleware",
		"svc-middleware",
		"step-one",
		"testvalue",
		"svcvalue",
		"1st-middleware",
		"2nd-middleware",
		"svc-middleware",
		"step-two",
	}, items)
}
