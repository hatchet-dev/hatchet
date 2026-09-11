package streaming

import (
	"context"
	"errors"
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/pkg/client/retry"
)

// MaxConsecutiveNoProgress caps consecutive no-progress failures (recv or
// reconnect) before a listen loop surfaces the error to its owner.
const MaxConsecutiveNoProgress = 10

// ErrListenerClosed is returned by operations on a stream or listener after
// it was closed.
var ErrListenerClosed = errors.New("listener is closed")

// Verdict is the single classification of an error observed on a long-lived
// stream, for both recv and reconnect failures.
type Verdict int

const (
	// VerdictRetry reconnects without counting toward the no-progress cap.
	VerdictRetry Verdict = iota
	// VerdictNoProgress reconnects and counts toward MaxConsecutiveNoProgress.
	VerdictNoProgress
	// VerdictStopClean exits the listen loop returning nil.
	VerdictStopClean
	// VerdictStopError exits the listen loop returning the error.
	VerdictStopError
)

// Classifier maps a stream error to a verdict. The listen loop consults it
// once per error, so implementations may carry per-error side effects (e.g.
// the action listener's V2→V1 strategy fallback).
type Classifier func(ctx context.Context, err error) Verdict

// NewClassifier builds the default classifier. reconnectOnEOF is consulted
// every time io.EOF is observed because handler registration can change
// between errors.
func NewClassifier(reconnectOnEOF func(ctx context.Context) bool) Classifier {
	return func(ctx context.Context, err error) Verdict {
		switch {
		case ctx.Err() != nil,
			errors.Is(err, ErrListenerClosed),
			errors.Is(err, context.Canceled),
			status.Code(err) == codes.Canceled:
			return VerdictStopClean
		case errors.Is(err, io.EOF):
			if reconnectOnEOF(ctx) {
				return VerdictRetry
			}
			return VerdictStopClean
		}
		switch retry.ClassifyStreamError(ctx, err) {
		case retry.StreamDecisionRetry:
			return VerdictRetry
		case retry.StreamDecisionStop:
			return VerdictStopError
		default:
			return VerdictNoProgress
		}
	}
}

// shouldLogReconnectMilestone rate-limits reconnect warnings: first attempt
// and every fifth thereafter.
func shouldLogReconnectMilestone(attempt int) bool {
	return attempt == 1 || attempt%5 == 0
}

// errorCode names the gRPC status code of err for log fields.
func errorCode(err error) string {
	if err == nil {
		return ""
	}

	if st, ok := status.FromError(err); ok {
		return st.Code().String()
	}

	return "unknown"
}

// SendListenerError delivers err on errCh unless ctx ends first, so a
// listener whose consumer is gone does not block on the report.
func SendListenerError(ctx context.Context, errCh chan<- error, err error) {
	select {
	case errCh <- err:
	case <-ctx.Done():
	}
}
