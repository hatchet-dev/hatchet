// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"
	"errors"
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/pkg/client/retry"
)

// maxConsecutiveStreamNoProgress caps consecutive no-progress failures
// (recv or reconnect) before a listen loop surfaces the error to its owner.
const maxConsecutiveStreamNoProgress = 10

var errListenerClosed = errors.New("listener is closed")

// streamVerdict is the single classification of an error observed on a
// long-lived stream, for both recv and reconnect failures.
type streamVerdict int

const (
	// verdictRetry reconnects without counting toward the no-progress cap.
	verdictRetry streamVerdict = iota
	// verdictNoProgress reconnects and counts toward maxConsecutiveStreamNoProgress.
	verdictNoProgress
	// verdictStopClean exits the listen loop returning nil.
	verdictStopClean
	// verdictStopError exits the listen loop returning the error.
	verdictStopError
)

// streamClassifier maps a stream error to a verdict. The listen loop consults
// it once per error, so implementations may carry per-error side effects
// (e.g. the action listener's V2→V1 strategy fallback).
type streamClassifier func(ctx context.Context, err error) streamVerdict

// newStreamClassifier builds the default classifier. reconnectOnEOF is
// consulted every time io.EOF is observed because handler registration can
// change between errors.
func newStreamClassifier(reconnectOnEOF func(ctx context.Context) bool) streamClassifier {
	return func(ctx context.Context, err error) streamVerdict {
		switch {
		case ctx.Err() != nil,
			errors.Is(err, errListenerClosed),
			errors.Is(err, context.Canceled),
			status.Code(err) == codes.Canceled:
			return verdictStopClean
		case errors.Is(err, io.EOF):
			if reconnectOnEOF(ctx) {
				return verdictRetry
			}
			return verdictStopClean
		}
		switch retry.ClassifyStreamError(ctx, err) {
		case retry.StreamDecisionRetry:
			return verdictRetry
		case retry.StreamDecisionStop:
			return verdictStopError
		default:
			return verdictNoProgress
		}
	}
}

// shouldLogReconnectMilestone rate-limits reconnect warnings: first attempt
// and every fifth thereafter.
func shouldLogReconnectMilestone(attempt int) bool {
	return attempt == 1 || attempt%5 == 0
}

func sendListenerError(ctx context.Context, errCh chan<- error, err error) {
	select {
	case errCh <- err:
	case <-ctx.Done():
	}
}
