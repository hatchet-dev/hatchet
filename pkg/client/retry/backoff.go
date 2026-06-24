package retry

import (
	"context"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

const (
	// gRPC retry backoff uses full jitter over exponential windows so clients
	// spread retry pressure. After the initial RPC fails, middleware invokes
	// backoff with attempt=1 for the first retry; delay windows are 0-5s,
	// 0-10s, 0-20s, 0-40s, then capped at 0-80s.
	grpcMaxRetries    = 5
	grpcBaseDelay     = 5 * time.Second
	grpcBackoffFactor = 2.0
	grpcMaxDelay      = 80 * time.Second

	// REST retry timing defaults keep SDK calls responsive while still covering
	// short gateway and rate-limit blips. With 5 total attempts, retry delay
	// windows are 0-250ms, 0-500ms, 0-1s, and 0-2s, unless a valid Retry-After
	// header supplies the delay for an HTTP 429 response.
	restMaxAttempts       = 5
	restBaseDelay         = 250 * time.Millisecond
	restBackoffFactor     = 2.0
	restMaxDelay          = 5 * time.Second
	restMaxRetryAfter     = 5 * time.Second
	restPerAttemptTimeout = 30 * time.Second

	// restDiscardDrainLimitBytes is a heuristic cap, not a protocol threshold.
	// Draining small intermediate retry responses lets net/http reuse the TCP
	// connection after Close. The cap is large enough for typical API error
	// bodies, but small enough that an unexpectedly large or slow discarded body
	// does not delay the next retry for long.
	restDiscardDrainLimitBytes = 64 * 1024
)

type clock struct {
	now    func() time.Time
	sleep  func(context.Context, time.Duration) error
	jitter func(int64) int64
}

func defaultClock() clock {
	return clock{
		now:    time.Now,
		sleep:  sleepContext,
		jitter: rand.Int64N,
	}
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		return nil
	}

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func fullJitterDelay(attempt int, base time.Duration, factor float64, maxDelay time.Duration, jitter func(int64) int64) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	delayLimit := float64(base) * math.Pow(factor, float64(attempt))
	if delayLimit > float64(maxDelay) {
		delayLimit = float64(maxDelay)
	}
	if delayLimit <= 0 {
		return 0
	}

	return time.Duration(jitter(int64(delayLimit) + 1))
}

// parseRetryAfter parses an HTTP Retry-After header value.
// It supports delta-seconds and HTTP-date forms. Past HTTP-date values yield zero delay.
// Returns ok=false when the header is missing, invalid, negative, or exceeds max.
func parseRetryAfter(header string, now time.Time, maxDelay time.Duration) (time.Duration, bool) {
	if header == "" {
		return 0, false
	}

	if seconds, err := strconv.Atoi(header); err == nil {
		if seconds < 0 {
			return 0, false
		}

		delay := time.Duration(seconds) * time.Second
		if delay > maxDelay {
			return 0, false
		}

		return delay, true
	}

	retryAt, err := http.ParseTime(header)
	if err != nil {
		return 0, false
	}

	delay := retryAt.Sub(now)
	if delay < 0 {
		delay = 0
	}
	if delay > maxDelay {
		return 0, false
	}

	return delay, true
}
