package streaming

import (
	"context"
	"fmt"
)

// Listen runs the receive loop for a reconnecting stream. Reconnects use
// full-jitter backoff; after MaxConsecutiveNoProgress consecutive no-progress
// failures the loop returns an error; classify decides clean vs error exits.
//
// Context contract: ctx scopes the loop itself (recv classification and
// backoff sleeps; cancellation is a clean exit returning nil, because
// background loops shut down cleanly). Reconnect attempts always use
// stream.LifecycleContext(), which only Close() cancels, so one caller's ctx
// cannot destroy the shared stream's ability to reconnect.
//
// If no client is installed yet, the first iteration connects (this serves
// StreamByAdditionalMetadata's initial connect and makes Listen() usable on a
// never-connected listener).
func Listen[C any, E any](
	ctx context.Context,
	stream *ReconnectingStream[C],
	recv func(C) (E, error),
	handle func(E) error,
	classify Classifier,
) error {
	noProgress := 0
	reconnects := 0

	client, generation, connected := stream.Snapshot()
	defer func() {
		if stream.closeSend != nil && connected {
			stream.sendMu.Lock()
			closeErr := stream.closeSend(client)
			stream.sendMu.Unlock()
			if closeErr != nil {
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
			if err := stream.ConnectOnce(stream.LifecycleContext()); err != nil {
				switch classify(ctx, err) {
				case VerdictStopClean:
					return nil
				case VerdictStopError:
					return fmt.Errorf("could not reconnect %s: %w", stream.name, err)
				case VerdictNoProgress:
					noProgress++
					if noProgress >= MaxConsecutiveNoProgress {
						return fmt.Errorf("%s made no progress after %d consecutive errors: %w", stream.name, noProgress, err)
					}
				}
				reconnects++
				if shouldLogReconnectMilestone(reconnects) {
					stream.l.Warn().Err(err).Str("stream", stream.name).
						Int("reconnect_attempt", reconnects).
						Int("consecutive_no_progress", noProgress).
						Str("error_code", errorCode(err)).
						Msg("stream reconnect attempt continuing")
				}
				continue
			}
			client, generation, _ = stream.Snapshot()
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
		case VerdictStopClean:
			return nil
		case VerdictStopError:
			return err
		case VerdictNoProgress:
			noProgress++
			if noProgress >= MaxConsecutiveNoProgress {
				return fmt.Errorf("%s made no progress after %d consecutive errors: %w", stream.name, noProgress, err)
			}
		}

		if c, g, ok := stream.Snapshot(); ok && g != generation {
			client, generation = c, g
			noProgress, reconnects = 0, 0
			continue
		}
		reconnects++
		connected = false
	}
}
