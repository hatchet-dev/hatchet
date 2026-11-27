package main

import (
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAttribute(t *testing.T) {
	t0 := time.Unix(1000, 0)
	at := func(s float64) time.Time { return t0.Add(secs(s)) }
	sigterms := []time.Time{at(100), at(200)}
	from, end := at(30), at(300)
	unavailable := status.Error(codes.Unavailable, "down")

	ops := []*op{
		// warm-up: not measured
		{tenant: 0, probe: true, at: at(20), err: unavailable},
		// steady, and inside restart 0's timeline span
		{tenant: 0, probe: true, at: at(95), started: at(95.1)},
		// steady and fast
		{tenant: 0, probe: true, at: at(50), started: at(50.1)},
		// steady but late
		{tenant: 1, probe: true, at: at(60), started: at(63)},
		// first second of restart 0: the trigger failed, so both operations are bad
		{tenant: 0, probe: true, at: at(100), err: unavailable},
		// last instant of restart 0's window: accepted, never started
		{tenant: 1, probe: true, at: at(130).Add(-time.Nanosecond)},
		// first instant after restart 0's window: steady
		{tenant: 1, probe: true, at: at(130), started: at(130.5)},
		// restart 1, fine
		{tenant: 1, probe: true, at: at(205), started: at(205.5)},
		// background load is never measured
		{tenant: 0, at: at(100), err: unavailable},
		// past the end
		{tenant: 0, probe: true, at: at(300), err: unavailable},
	}

	r := Result{
		Track:    Track{WindowS: 30, AssignDeadlineS: 2},
		Restarts: make([]Restart, 2),
	}
	r.attribute(2, ops, sigterms, from, end)

	if want := (Counts{Offered: 8, Bad: 1}); r.Steady != want {
		t.Errorf("steady = %+v, want %+v", r.Steady, want)
	}
	want := [][]Counts{
		{{Offered: 2, Bad: 2}, {Offered: 2, Bad: 1}},
		{{Offered: 0, Bad: 0}, {Offered: 2, Bad: 0}},
	}
	for i, rs := range r.Restarts {
		for tn, c := range rs.Tenants {
			if c != want[i][tn] {
				t.Errorf("restart %d tenant %d = %+v, want %+v", i, tn, c, want[i][tn])
			}
		}
	}
	if want := (Errors{Unavailable: 2, Late: 1}); r.Errors != want {
		t.Errorf("errors = %+v, want %+v", r.Errors, want)
	}

	if len(r.Timeline) != timelineBuckets {
		t.Fatalf("timeline has %d buckets", len(r.Timeline))
	}
	// Bucket 5 (SIGTERM-5s) holds only the probe at 95s; bucket 10 (the
	// SIGTERM second) holds tenant 0's failed probe in restart 0 only.
	// Bucket 39 holds tenant 1's never-started probe; bucket 15 holds
	// restart 1's fine probe.
	for b, want := range map[int]float64{5: 0, 10: 100, 39: 50, 15: 0} {
		if r.Timeline[b] != want {
			t.Errorf("timeline[%d] = %v, want %v", b, r.Timeline[b], want)
		}
	}
}
