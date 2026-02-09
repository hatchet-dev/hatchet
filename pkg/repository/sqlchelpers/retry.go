package sqlchelpers

import (
	"math/rand"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

var DeadlockRetry = func(l *zerolog.Logger, f func() error) error {
	return genericRetry(l.Warn(), 3, f, "deadlock", func(err error) (bool, error) {
		return strings.Contains(err.Error(), "deadlock detected"), err
	}, 50*time.Millisecond, 200*time.Millisecond)
}

var genericRetry = func(l *zerolog.Event, maxRetries int, f func() error, msg string, condition func(err error) (bool, error), minSleep, maxSleep time.Duration) error {
	retries := 0

	for {
		err := f()

		if err != nil {
			// condition detected, retry
			if ok, overrideErr := condition(err); ok {
				retries++

				if retries > maxRetries {
					return err
				}

				l.Err(err).Msgf("retry (%s) condition met, retry %d", msg, retries)

				// sleep with jitter
				sleepWithJitter(minSleep, maxSleep)
			} else {
				if overrideErr != nil {
					return overrideErr
				}

				return err
			}
		}

		if err == nil {
			if retries > 0 {
				l.Msgf("retry (%s) condition resolved after %d retries", msg, retries)
			}

			break
		}
	}

	return nil
}

// sleepWithJitter sleeps for a random duration between min and max duration.
// min and max are time.Duration values, specifying the minimum and maximum sleep times.
func sleepWithJitter(min, max time.Duration) { // nolint: revive
	if min > max {
		// Swap if min is greater than max
		min, max = max, min // nolint: revive
	}

	jitter := max - min
	if jitter > 0 {
		sleepDuration := min + time.Duration(rand.Int63n(int64(jitter))) // nolint: gosec
		time.Sleep(sleepDuration)
	} else {
		time.Sleep(min) // Sleep for min duration if jitter is not positive
	}
}
