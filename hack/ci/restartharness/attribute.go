package main

import (
	"math"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// attribute fills each restart's per-tenant counts, Steady, Errors and
// Timeline from the probes among ops. A probe counts by its scheduled
// timestamp: in [sigterm, sigterm+WindowS) it belongs to that restart,
// otherwise in [from, end) it is steady. The timeline takes every probe
// within its span of a SIGTERM, warm-up included.
func (r *Result) attribute(tenants int, ops []*op, sigterms []time.Time, from, end time.Time) {
	window := secs(r.Track.WindowS)
	assignDeadline := secs(r.Track.AssignDeadlineS)
	for i := range r.Restarts {
		r.Restarts[i].Tenants = make([]Counts, tenants)
	}
	// timeline[restart][bucket][tenant]
	timeline := make([][][]Counts, len(sigterms))
	for i := range timeline {
		timeline[i] = make([][]Counts, timelineBuckets)
		for b := range timeline[i] {
			timeline[i][b] = make([]Counts, tenants)
		}
	}

	for _, o := range ops {
		if !o.probe {
			continue
		}
		apiBad := o.err != nil
		assignBad := apiBad || o.started.IsZero() || o.started.Sub(o.at) > assignDeadline
		c := Counts{Offered: 2, Bad: b2i(apiBad) + b2i(assignBad)}

		inWindow := false
		for i, s := range sigterms {
			off := o.at.Sub(s)
			if off >= 0 && off < window {
				inWindow = true
				r.Restarts[i].Tenants[o.tenant].add(c)
				switch {
				case apiBad:
					r.Errors.add(o.err, 2)
				case assignBad:
					r.Errors.Late++
				}
			}
			if b := int(math.Floor(off.Seconds() - timelineStartS)); b >= 0 && b < timelineBuckets {
				timeline[i][b][o.tenant].add(c)
			}
		}
		if !inWindow && !o.at.Before(from) && o.at.Before(end) {
			r.Steady.add(c)
		}
	}

	// Per bucket: the mean bad percentage over tenants, then over restarts,
	// leaving out a tenant or restart with nothing offered in the bucket.
	r.Timeline = make([]float64, timelineBuckets)
	for b := range r.Timeline {
		var sum float64
		n := 0
		for i := range timeline {
			if pct, ok := meanBadPct(timeline[i][b]); ok {
				sum += pct
				n++
			}
		}
		if n > 0 {
			r.Timeline[b] = sum / float64(n)
		}
	}
}

func meanBadPct(cs []Counts) (float64, bool) {
	var sum float64
	n := 0
	for _, c := range cs {
		if c.Offered > 0 {
			sum += 100 * float64(c.Bad) / float64(c.Offered)
			n++
		}
	}
	if n == 0 {
		return 0, false
	}
	return sum / float64(n), true
}

func (c *Counts) add(o Counts) {
	c.Offered += o.Offered
	c.Bad += o.Bad
}

// add counts n bad operations against the cause of a failed trigger.
func (e *Errors) add(err error, n int) {
	//exhaustive:ignore
	switch status.Code(err) {
	case codes.ResourceExhausted:
		e.ResourceExhausted += n
	case codes.Unavailable:
		e.Unavailable += n
	case codes.DeadlineExceeded:
		e.Timeout += n
	default:
		e.Other += n
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
