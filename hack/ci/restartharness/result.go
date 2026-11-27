package main

import "fmt"

// A measured operation is one of two things, both offered open-loop at a
// fixed timestamp and attributed to that timestamp:
//   - an API call: one trigger RPC, bad if it errors or misses the track's
//     APIDeadlineS;
//   - a trigger→assigned: the triggered task starting on a worker, bad if it
//     has not started AssignDeadlineS after the trigger's timestamp (which
//     includes every trigger that failed).
//
// Every probe trigger offers one of each.

const (
	// steadyBadMax is the largest bad fraction allowed outside restart
	// windows; above it the engine was not keeping up before any restart and
	// the disruption numbers mean nothing.
	steadyBadMax = 0.01

	// The timeline has one bucket per second, the first starting
	// timelineStartS relative to SIGTERM.
	timelineStartS  = -10
	timelineBuckets = 40

	// minRestarts is the fewest restarts per side for a verdict: with 5 a
	// side, two samples of one distribution have non-overlapping ranges
	// 2/C(10,5) = 0.8% of the time.
	minRestarts = 5
)

// Result is one track's run against one engine build. A results file is a
// stream of Result objects, one per track, so results files concatenate.
type Result struct {
	Track    Track     `json:"track"`
	Restarts []Restart `json:"restarts"`

	// Steady counts the measured operations after warm-up that fall outside
	// every restart window.
	Steady Counts `json:"steady"`

	// Errors breaks the bad operations inside restart windows down by cause.
	Errors Errors `json:"errors"`

	// Timeline is the bad percentage per second around SIGTERM: tenants
	// weighted equally, then averaged over restarts.
	Timeline []float64 `json:"timeline"`

	// Lost counts accepted triggers whose task never ran, Duplicates the
	// triggers whose task ran more than once.
	Lost       int `json:"lost"`
	Duplicates int `json:"duplicates"`

	// MaxRunsViolations counts task starts that found more than max_runs
	// tasks of their concurrency key already running on the worker.
	MaxRunsViolations int `json:"max_runs_violations"`

	// PeakConns is the most connections the engine held at once outside its
	// DDL pool, as seen in pg_stat_activity.
	PeakConns int `json:"peak_conns"`
}

// Restart is what one restart did to each tenant.
type Restart struct {
	// DownS is the time from SIGTERM until the new engine reported ready.
	DownS float64 `json:"down_s"`

	// Tenants holds each tenant's measured operations scheduled in
	// [SIGTERM, SIGTERM+window), in the track's tenant order.
	Tenants []Counts `json:"tenants"`
}

type Counts struct {
	Offered int `json:"offered"`
	Bad     int `json:"bad"`
}

// Errors attributes each bad operation to one cause. A trigger→assigned whose
// trigger failed takes the trigger's cause; one whose trigger succeeded is
// late.
type Errors struct {
	ResourceExhausted int `json:"resource_exhausted"`
	Unavailable       int `json:"unavailable"`
	Timeout           int `json:"timeout"`
	Late              int `json:"late"`
	Other             int `json:"other"`
}

// lostS is the seconds an average tenant was down in this restart's window:
// every tenant counts the same, so a large tenant cannot hide starved small
// ones.
func (r Restart) lostS(windowS float64) float64 {
	if len(r.Tenants) == 0 {
		return 0
	}
	var sum float64
	for _, c := range r.Tenants {
		if c.Offered > 0 {
			sum += float64(c.Bad) / float64(c.Offered)
		}
	}
	return windowS * sum / float64(len(r.Tenants))
}

// lostS returns lostS per restart.
func (r Result) lostS() []float64 {
	out := make([]float64, len(r.Restarts))
	for i, rs := range r.Restarts {
		out[i] = rs.lostS(r.Track.WindowS)
	}
	return out
}

// violations lists the hard constraints this run broke. Any violation makes
// the run INVALID; none of them is folded into the score.
func (r Result) violations() []string {
	var v []string
	if r.Lost > 0 {
		v = append(v, fmt.Sprintf("%d accepted trigger(s) never ran", r.Lost))
	}
	if r.Duplicates > 0 {
		v = append(v, fmt.Sprintf("%d trigger(s) ran more than once", r.Duplicates))
	}
	if r.MaxRunsViolations > 0 {
		v = append(v, fmt.Sprintf("%d task start(s) exceeded max_runs for their concurrency key", r.MaxRunsViolations))
	}
	if r.Steady.Offered == 0 {
		v = append(v, "no operations measured outside restart windows")
	} else if frac := float64(r.Steady.Bad) / float64(r.Steady.Offered); frac > steadyBadMax {
		v = append(v, fmt.Sprintf("%.2f%% of operations outside restart windows were bad (limit %.2f%%)", 100*frac, 100*steadyBadMax))
	}
	if r.PeakConns > r.Track.MaxConns {
		v = append(v, fmt.Sprintf("peak of %d database connections exceeds DATABASE_MAX_CONNS=%d", r.PeakConns, r.Track.MaxConns))
	}
	if len(r.Restarts) == 0 {
		v = append(v, "no restarts recorded")
	}
	for _, rs := range r.Restarts {
		for _, c := range rs.Tenants {
			if c.Offered == 0 {
				v = append(v, "a tenant was offered no operations in a restart window")
				return v
			}
		}
	}
	return v
}
