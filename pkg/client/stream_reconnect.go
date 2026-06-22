// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/hatchet-dev/hatchet/pkg/client/retry"
)

func classifyStreamRecvError(ctx context.Context, err error, reconnectOnEOF bool) retry.StreamDecision {
	if errors.Is(err, io.EOF) {
		if !reconnectOnEOF || ctx.Err() != nil {
			return retry.StreamDecisionStop
		}

		return retry.StreamDecisionRetry
	}

	return retry.ClassifyStreamError(ctx, err)
}

func streamDecisionStopsReconnect(decision retry.StreamDecision) bool {
	return decision == retry.StreamDecisionStop || decision == retry.StreamDecisionNoProgress
}

func retryStreamConnectSync(
	ctx context.Context,
	isClosed func() bool,
	connect func(context.Context) error,
	logAttempt func(error, int),
	exhaustedFormat string,
) error {
	for attempt := 0; attempt < retry.StreamSyncMaxAttempts; attempt++ {
		if attempt > 0 {
			if err := retry.SleepStreamBackoff(ctx, attempt-1); err != nil {
				return err
			}
		}

		if isClosed() {
			return errListenerClosed
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		err := connect(ctx)
		if err == nil {
			return nil
		}

		if retry.ClassifyStreamError(ctx, err) == retry.StreamDecisionStop {
			return err
		}

		logAttempt(err, attempt+1)
	}

	return fmt.Errorf(exhaustedFormat, retry.StreamSyncMaxAttempts)
}

func retryStreamConnectBackground(
	ctx context.Context,
	isClosed func() bool,
	connect func(context.Context) error,
	logAttempt func(error, int),
	noProgressFormat string,
) error {
	attempt := 0
	consecutiveNoProgress := 0

	for {
		if attempt > 0 {
			if err := retry.SleepStreamBackoff(ctx, attempt-1); err != nil {
				return err
			}
		}

		if isClosed() {
			return errListenerClosed
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		err := connect(ctx)
		if err == nil {
			return nil
		}

		switch retry.ClassifyStreamError(ctx, err) {
		case retry.StreamDecisionStop:
			return err
		case retry.StreamDecisionNoProgress:
			consecutiveNoProgress++
			if consecutiveNoProgress >= maxConsecutiveStreamNoProgress {
				return fmt.Errorf(noProgressFormat, consecutiveNoProgress, err)
			}

			return err
		}

		consecutiveNoProgress = 0
		logAttempt(err, attempt+1)
		attempt++
	}
}
