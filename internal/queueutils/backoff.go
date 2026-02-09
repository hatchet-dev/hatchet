package queueutils

import (
	"math"
	"math/rand"
	"time"
)

// SleepWithExponentialBackoff sleeps for a duration calculated using exponential backoff and jitter,
// based on the retry count. The base sleep time and maximum sleep time are provided as inputs.
// retryCount determines the exponential backoff multiplier.
func SleepWithExponentialBackoff(base, max time.Duration, retryCount int) { // nolint: revive
	if retryCount < 0 {
		retryCount = 0
	}

	// prevent overflow
	pow := time.Duration(math.MaxInt64)
	if retryCount < 63 {
		pow = 1 << retryCount
	}

	// Calculate exponential backoff
	backoff := base * pow

	// if backoff / pow does not recover base, we've overflowed
	if backoff > max || backoff/pow != base {
		backoff = max
	}

	backoffInterval := backoff / 2

	if backoffInterval < 1*time.Millisecond {
		backoffInterval = 1 * time.Millisecond
	}

	// Apply jitter
	jitter := time.Duration(rand.Int63n(int64(backoffInterval))) // nolint: gosec
	sleepDuration := backoffInterval + jitter

	time.Sleep(sleepDuration)
}
