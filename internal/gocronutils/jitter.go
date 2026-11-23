// Package gocronutils holds helpers shared by the services that schedule periodic work with gocron.
package gocronutils

import (
	"math/rand"
	"time"

	"github.com/go-co-op/gocron/v2"
)

// minPhase keeps the first run safely in the future: gocron rejects a start time that has already passed.
const minPhase = 100 * time.Millisecond

// NewJitteredDurationJob registers task on s to run every interval, like gocron.DurationJob, except that the
// first run lands at a random point inside the first interval instead of exactly one interval after
// registration. The period is unchanged; only the phase moves.
//
// Periodic jobs should be registered through this function. Jobs registered with a bare gocron.DurationJob
// at startup all share one phase, so every 5s, 10s, 15s and 60s job across the controllers fires in the
// same instant once a minute and the database takes that burst at once.
//
// options are applied after the random start time, so a caller that needs a specific first run can still
// pass its own gocron.WithStartAt.
func NewJitteredDurationJob(s gocron.Scheduler, interval time.Duration, task gocron.Task, options ...gocron.JobOption) (gocron.Job, error) {
	startAt := gocron.WithStartAt(gocron.WithStartDateTime(time.Now().Add(randomPhase(interval))))

	return s.NewJob(
		gocron.DurationJob(interval),
		task,
		append([]gocron.JobOption{startAt}, options...)...,
	)
}

// randomPhase returns a duration in [minPhase, interval), or minPhase when the interval is no longer than that.
func randomPhase(interval time.Duration) time.Duration {
	if interval <= minPhase {
		return minPhase
	}

	return minPhase + time.Duration(rand.Int63n(int64(interval-minPhase))) // nolint:gosec
}
