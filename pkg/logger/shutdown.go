package logger

import (
	"context"
	"errors"

	"github.com/rs/zerolog"
)

// ShutdownAware returns an event on l at level, demoted to debug when err is
// the cancellation produced by ctx being canceled during a graceful shutdown,
// as opposed to a real failure, which keeps logging at level. Both the
// governing context and the error must be canceled specifically: a deadline
// error always logs at its original level, so a genuine timeout is never
// hidden.
func ShutdownAware(ctx context.Context, l *zerolog.Logger, err error, level zerolog.Level) *zerolog.Event {
	if isShutdownErr(ctx, err) {
		return l.Debug()
	}

	return l.WithLevel(level)
}

// isShutdownErr reports whether err is the cancellation produced by ctx being
// canceled during a graceful shutdown.
func isShutdownErr(ctx context.Context, err error) bool {
	return errors.Is(ctx.Err(), context.Canceled) && errors.Is(err, context.Canceled)
}
