// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"
	"fmt"
)

// listenStream runs the receive loop for a reconnecting stream. Reconnects
// use full-jitter backoff; after maxConsecutiveStreamNoProgress consecutive
// no-progress failures the loop returns an error; classify decides clean vs
// error exits.
//
// Context contract: ctx scopes the loop itself (recv classification and
// backoff sleeps; cancellation is a clean exit returning nil, because
// background loops shut down cleanly). Reconnect attempts always use
// stream.lifecycleContext(), which only Close() cancels, so one caller's ctx
// cannot destroy the shared stream's ability to reconnect.
//
// If no client is installed yet, the first iteration connects (this serves
// StreamByAdditionalMetadata's initial connect and makes Listen() usable on a
// never-connected listener).
func listenStream[C any, E any](
	ctx context.Context,
	stream *reconnectingStream[C],
	recv func(C) (E, error),
	handle func(E) error,
	classify streamClassifier,
) error {
	noProgress := 0
	reconnects := 0

	client, generation, connected := stream.snapshot()
	defer func() {
		if stream.closeSend != nil && connected {
			if closeErr := stream.closeSend(client); closeErr != nil {
				stream.l.Warn().Err(closeErr).Str("stream", stream.name).Msg("failed to close stream after listen exit")
			}
		}
	}()

	for {
		if !connected {
			if reconnects > 0 {
				if err := stream.sleep(ctx, reconnects-1); err != nil {
					return nil
				}
			}
			if err := stream.connectOnce(stream.lifecycleContext()); err != nil {
				switch classify(ctx, err) {
				case verdictStopClean:
					return nil
				case verdictStopError:
					return fmt.Errorf("could not reconnect %s: %w", stream.name, err)
				case verdictNoProgress:
					noProgress++
					if noProgress >= maxConsecutiveStreamNoProgress {
						return fmt.Errorf("%s made no progress after %d consecutive errors: %w", stream.name, noProgress, err)
					}
				}
				reconnects++
				if shouldLogReconnectMilestone(reconnects) {
					stream.l.Warn().Err(err).Str("stream", stream.name).
						Int("reconnect_attempt", reconnects).
						Int("consecutive_no_progress", noProgress).
						Str("error_code", streamErrorCode(err)).
						Msg("stream reconnect attempt continuing")
				}
				continue
			}
			client, generation, _ = stream.snapshot()
			if reconnects > 0 {
				stream.l.Info().Str("stream", stream.name).Int("attempts", reconnects).
					Msg("stream reconnected")
			}
			connected, noProgress, reconnects = true, 0, 0
		}

		event, err := recv(client)
		if err == nil {
			noProgress, reconnects = 0, 0
			if herr := handle(event); herr != nil {
				return herr
			}
			continue
		}

		switch classify(ctx, err) {
		case verdictStopClean:
			return nil
		case verdictStopError:
			return err
		case verdictNoProgress:
			noProgress++
			if noProgress >= maxConsecutiveStreamNoProgress {
				return fmt.Errorf("%s made no progress after %d consecutive errors: %w", stream.name, noProgress, err)
			}
		}

		if c, g, ok := stream.snapshot(); ok && g != generation {
			client, generation = c, g
			noProgress, reconnects = 0, 0
			continue
		}
		reconnects++
		connected = false
	}
}
