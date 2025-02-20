//go:build e2e

package main

import (
	"context"
	"os"
	"os/signal"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/internal/testutils"
)

func TestAdvancedConcurrency(t *testing.T) {
	testutils.Prepare(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ctx.Done()
		ch <- os.Interrupt
	}()

	err := run(ctx)

	if err != nil {
		t.Fatalf("/run() error = %v", err)
	}

}
