package gocronutils

import (
	"context"
	"testing"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

func TestRandomPhaseStaysInsideTheInterval(t *testing.T) {
	for _, interval := range []time.Duration{0, time.Millisecond, minPhase, time.Second, time.Minute, time.Hour} {
		for range 1000 {
			phase := randomPhase(interval)

			require.GreaterOrEqual(t, phase, minPhase)

			if interval > minPhase {
				require.Less(t, phase, interval)
			}
		}
	}
}

func TestNewJitteredDurationJobFirstRunsInsideTheIntervalAndKeepsThePeriod(t *testing.T) {
	const interval = time.Minute

	// The scheduler runs on a fake clock so the test never waits on wall-clock time. The helper reads
	// time.Now for the phase, so the fake clock is seeded from it; gocron validates the start time
	// against its own clock, which is at or just before that.
	clock := clockwork.NewFakeClockAt(time.Now())

	s, err := gocron.NewScheduler(gocron.WithClock(clock))
	require.NoError(t, err)

	t.Cleanup(func() { require.NoError(t, s.Shutdown()) })

	runs := make(chan struct{}, 8)
	registered := clock.Now()

	job, err := NewJitteredDurationJob(s, interval, gocron.NewTask(func() { runs <- struct{}{} }))
	require.NoError(t, err)

	s.Start()

	first, err := job.NextRun()
	require.NoError(t, err)
	require.GreaterOrEqual(t, first.Sub(registered), minPhase)
	require.Less(t, first.Sub(registered), interval)

	// Wait for the scheduler to arm the first run's timer, then move the clock one whole interval:
	// that is past the first run and still short of the second, which is one interval after the first.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, clock.BlockUntilContext(ctx, 1))
	clock.Advance(interval)
	awaitRun(t, runs)

	require.Eventually(t, func() bool {
		next, err := job.NextRun()

		return err == nil && next.Sub(first) == interval
	}, 5*time.Second, 10*time.Millisecond, "second run should be exactly one interval after the first")
}

func TestNewJitteredDurationJobLetsTheCallerChooseTheFirstRun(t *testing.T) {
	s, err := gocron.NewScheduler()
	require.NoError(t, err)

	t.Cleanup(func() { require.NoError(t, s.Shutdown()) })

	startAt := time.Now().Add(time.Hour)

	job, err := NewJitteredDurationJob(
		s,
		time.Minute,
		gocron.NewTask(func() {}),
		gocron.WithStartAt(gocron.WithStartDateTime(startAt)),
	)
	require.NoError(t, err)

	s.Start()

	next, err := job.NextRun()
	require.NoError(t, err)
	require.True(t, next.Equal(startAt), "caller's start time should win over the random phase")
}

func awaitRun(t *testing.T, runs <-chan struct{}) {
	t.Helper()

	select {
	case <-runs:
	case <-time.After(5 * time.Second):
		t.Fatal("job did not run")
	}
}
