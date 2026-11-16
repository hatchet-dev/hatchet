package logger

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestShutdownAwareDemotesToDebugOnShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	l := zerolog.New(&buf).Level(zerolog.DebugLevel)

	ShutdownAware(ctx, &l, context.Canceled, zerolog.WarnLevel).Err(context.Canceled).Msg("listener failed")

	out := buf.String()

	if !strings.Contains(out, `"level":"debug"`) {
		t.Errorf("expected debug-level record, got %q", out)
	}

	if !strings.Contains(out, "listener failed") {
		t.Errorf("expected message in record, got %q", out)
	}
}

func TestShutdownAwareDemotedEventRespectsLevelFiltering(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	l := zerolog.New(&buf).Level(zerolog.WarnLevel)

	// a demoted event on a warn-level logger must be a no-op that is still
	// safe to chain
	ShutdownAware(ctx, &l, context.Canceled, zerolog.WarnLevel).Err(context.Canceled).Msg("listener failed")

	if buf.Len() != 0 {
		t.Errorf("expected no output from demoted event on warn-level logger, got %q", buf.String())
	}
}

func TestShutdownAwareKeepsLevelForRealError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	l := zerolog.New(&buf).Level(zerolog.DebugLevel)

	err := fmt.Errorf("connection refused")

	ShutdownAware(ctx, &l, err, zerolog.ErrorLevel).Err(err).Msg("listener failed")

	if !strings.Contains(buf.String(), `"level":"error"`) {
		t.Errorf("expected error-level record for a real failure, got %q", buf.String())
	}
}

func TestShutdownAwareDoesNotDemoteOnLapsedDeadline(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Hour))
	defer cancel()

	<-ctx.Done()

	var buf bytes.Buffer
	l := zerolog.New(&buf).Level(zerolog.DebugLevel)

	ShutdownAware(ctx, &l, context.DeadlineExceeded, zerolog.ErrorLevel).Err(context.DeadlineExceeded).Msg("listener failed")

	if !strings.Contains(buf.String(), `"level":"error"`) {
		t.Errorf("expected error-level record when the governing deadline lapsed, got %q", buf.String())
	}
}

func TestShutdownAwareDoesNotDemoteWrappedChildDeadlineDuringShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	l := zerolog.New(&buf).Level(zerolog.DebugLevel)

	err := fmt.Errorf("query failed: %w", context.DeadlineExceeded)

	ShutdownAware(ctx, &l, err, zerolog.ErrorLevel).Err(err).Msg("listener failed")

	if !strings.Contains(buf.String(), `"level":"error"`) {
		t.Errorf("expected error-level record for a deadline error even during shutdown, got %q", buf.String())
	}
}

func TestShutdownAwareDemotesWrappedCancellationDuringShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	l := zerolog.New(&buf).Level(zerolog.DebugLevel)

	err := fmt.Errorf("query failed: %w", context.Canceled)

	ShutdownAware(ctx, &l, err, zerolog.ErrorLevel).Err(err).Msg("listener failed")

	if !strings.Contains(buf.String(), `"level":"debug"`) {
		t.Errorf("expected debug-level record for wrapped cancellation during shutdown, got %q", buf.String())
	}
}

func TestShutdownAwareDoesNotDemoteOnLiveContext(t *testing.T) {
	var buf bytes.Buffer
	l := zerolog.New(&buf).Level(zerolog.DebugLevel)

	ShutdownAware(context.Background(), &l, context.Canceled, zerolog.WarnLevel).Err(context.Canceled).Msg("listener failed")

	if !strings.Contains(buf.String(), `"level":"warn"`) {
		t.Errorf("expected warn-level record when ctx is not canceled, got %q", buf.String())
	}
}
