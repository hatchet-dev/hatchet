package cleanup

import (
	"context"
	"errors"
	"fmt"
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

// A failing cleanup fn stops Run, and the returned error carries both
// ErrTeardown and the fn's own error so callers can match on either.
func TestRunWrapsTeardownFailure(t *testing.T) {
	logger := zerolog.Nop()
	c := New(&logger)

	cause := errors.New("boom")
	ranAfter := false
	c.Add(func() error { return cause }, "first")
	c.Add(func() error {
		ranAfter = true
		return nil
	}, "second")

	err := c.Run()
	if err == nil {
		t.Fatal("expected an error")
	}

	if !errors.Is(err, ErrTeardown) {
		t.Fatalf("error does not wrap ErrTeardown: %v", err)
	}

	if !errors.Is(err, cause) {
		t.Fatalf("error does not wrap the fn's error: %v", err)
	}

	if ranAfter {
		t.Fatal("Run continued past the failing fn")
	}
}

// The engine joins a start error to Run's error when a start is aborted. The
// harness accepts a tree that carries the cancellation and no teardown
// failure; a cancellation joined to a failed cleanup fn, or a start error that
// is not a cancellation, is rejected.
func TestOnlyCancelled(t *testing.T) {
	logger := zerolog.Nop()

	clean := New(&logger)
	clean.Add(func() error { return nil }, "ok")

	failing := New(&logger)
	failing.Add(func() error { return errors.New("boom") }, "bad")

	startCancelled := fmt.Errorf("could not run with config: %w", context.Canceled)

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, true},
		{"cancelled, clean teardown", errors.Join(startCancelled, clean.Run()), true},
		{"cancelled, failed teardown", errors.Join(startCancelled, failing.Run()), false},
		{"cancelled, bare", context.Canceled, true},
		{"other start error", errors.Join(errors.New("could not start health"), clean.Run()), false},
		{"deadline", context.DeadlineExceeded, false},
		{"teardown only", failing.Run(), false},
	}

	for _, tc := range cases {
		if got := OnlyCancelled(tc.err); got != tc.want {
			t.Errorf("%s: OnlyCancelled(%v) = %t, want %t", tc.name, tc.err, got, tc.want)
		}
	}
}
