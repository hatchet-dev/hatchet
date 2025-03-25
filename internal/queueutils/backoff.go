package queueutils

import (
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

	// Calculate exponential backoff
	backoff := base * (1 << retryCount)
	if backoff > max {
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
