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

// gatedClassifier coordinates clean termination with the listen gate. When the
// base classifier stops cleanly, keep is evaluated while holding the gate lock.
// If keep returns true, the gate remains active and no-progress is returned so
// listenStream reconnects using its normal backoff and limit. Otherwise, the
// gate is released and *released is set.
//
// Classification and deferred cleanup run on the same goroutine, so released
// needs no synchronization. It prevents cleanup from clearing the gate after
// another listen loop has acquired it.
func gatedClassifier(base streamClassifier, gate *listenGate, keep func(context.Context) bool, released *bool) streamClassifier {
	return func(ctx context.Context, err error) streamVerdict {
		v := base(ctx, err)
		if v != verdictStopClean {
			return v
		}
		if gate.release(func() bool { return keep(ctx) }) {
			*released = true
			return verdictStopClean
		}
		return verdictNoProgress
	}
}

// finishGatedListen releases the gate after listenStream returns unless the
// classifier already released it. On error, registered handlers are failed
// while the gate is still held, before another listen loop can acquire it.
func finishGatedListen(gate *listenGate, released bool, err error, fail func(error)) {
	if released {
		return
	}
	gate.release(func() bool {
		if err != nil {
			fail(err)
		}
		return false
	})
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
