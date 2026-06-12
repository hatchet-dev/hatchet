package cleanup

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// Run must not stall waiting for the log-promotion deadline: the logger
// goroutine only drains the lines channel once the context resolves, so an
// unbuffered channel would block Run for the full TimeLimit before any
// cleanup fn executes.
func TestRunDoesNotStallUntilTimeLimit(t *testing.T) {
	logger := zerolog.Nop()
	c := New(&logger)
	c.TimeLimit = 5 * time.Second

	ran := false
	c.Add(func() error {
		ran = true
		return nil
	}, "noop")

	start := time.Now()
	if err := c.Run(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !ran {
		t.Fatal("cleanup fn did not run")
	}

	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("Run took %s; should complete well before the %s TimeLimit", elapsed, c.TimeLimit)
	}
}
