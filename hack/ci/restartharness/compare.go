package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"
)

type verdict int

// Verdicts in increasing precedence: the overall verdict is the highest of
// the tracks' verdicts.
const (
	noChange verdict = iota
	improved
	regressed
	invalid
)

func (v verdict) String() string {
	return [...]string{"NO CHANGE", "IMPROVED", "REGRESSED", "INVALID"}[v]
}

// trackCmp is one registry track compared across the two files. base and
// head hold every result that names the track, so a track that is missing or
// repeated in a file is visible as len != 1.
type trackCmp struct {
	track      Track
	base, head []Result
	// reasons lists why the track is INVALID, for base and for head.
	reasons     [2][]string
	verdict     verdict
	fewRestarts bool
}

// compareCmd scores head against base and writes one markdown report whose
// last line is the SCORE. INVALID and REGRESSED are results, not errors: an
// error means an input could not be read or parsed.
func compareCmd(args []string, w io.Writer) error {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	expect := fs.String("tracks", "", "comma-separated tracks that must have results (default: the tracks either file has)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return errors.New("usage: restartharness compare [--tracks <name>,...] <base.json> <head.json>")
	}
	expected := map[string]bool{}
	for name := range strings.SplitSeq(*expect, ",") {
		if _, ok := trackByName(name); !ok && name != "" {
			return errors.New("--tracks names an unknown track")
		}
		expected[name] = true
	}
	var sides [2][]Result
	for i, path := range fs.Args() {
		data, err := readInput(path)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if sides[i], err = parseResults(data); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}
	_, err := io.WriteString(w, render(compareResults(sides[0], sides[1], expected)))
	return err
}

// compareResults returns the tracks that are expected or present in either
// file, in registry order. An expected track with no result is INVALID
// rather than missing from the report.
func compareResults(base, head []Result, expected map[string]bool) []trackCmp {
	var out []trackCmp
	for _, t := range tracks {
		c := trackCmp{track: t, base: named(base, t.Name), head: named(head, t.Name)}
		if len(c.base)+len(c.head) == 0 && !expected[t.Name] {
			continue
		}
		c.reasons = [2][]string{invalidReasons(t, c.base), invalidReasons(t, c.head)}
		if len(c.reasons[0])+len(c.reasons[1]) > 0 {
			c.verdict = invalid
		} else {
			c.verdict, c.fewRestarts = judge(t, c.base[0].lostS(), c.head[0].lostS())
		}
		out = append(out, c)
	}
	return out
}

func named(rs []Result, name string) []Result {
	var out []Result
	for _, r := range rs {
		if r.Track.Name == name {
			out = append(out, r)
		}
	}
	return out
}

func invalidReasons(t Track, rs []Result) []string {
	switch len(rs) {
	case 0:
		return []string{"no result for this track"}
	case 1:
	default:
		return []string{"more than one result for this track"}
	}
	var v []string
	// The noise floor is compare's own setting, not something a run recorded.
	recorded := rs[0].Track
	recorded.NoiseFloorS = t.NoiseFloorS
	if recorded != t {
		v = append(v, "recorded with different track settings than this build of the harness")
	}
	return append(v, rs[0].violations()...)
}

// judge calls a change only when the two sides' ranges over restarts do not
// overlap and the means differ by more than the track's noise floor.
func judge(t Track, b, h []float64) (v verdict, fewRestarts bool) {
	if len(b) < minRestarts || len(h) < minRestarts {
		return noChange, true
	}
	delta := mean(h) - mean(b)
	switch {
	case slices.Max(h) < slices.Min(b) && -delta > t.NoiseFloorS:
		return improved, false
	case slices.Min(h) > slices.Max(b) && delta > t.NoiseFloorS:
		return regressed, false
	}
	return noChange, false
}

func overall(cmps []trackCmp) verdict {
	v := noChange
	for _, c := range cmps {
		v = max(v, c.verdict)
	}
	return v
}

func mean(xs []float64) float64 {
	var sum float64
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs))
}

func render(cmps []trackCmp) string {
	var b strings.Builder
	final := overall(cmps)
	fmt.Fprintf(&b, "## Restart harness: %s\n\n", final)

	b.WriteString("| track | base mean (min–max) | head mean (min–max) | delta | noise floor | verdict |\n")
	b.WriteString("|---|---:|---:|---:|---:|---|\n")
	var baseTotal, headTotal float64
	for _, c := range cmps {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s |\n", c.track.Name,
			lostCell(c.base), lostCell(c.head), deltaCell(c.base, c.head), num(c.track.NoiseFloorS, 2), c.verdict)
		if final != invalid {
			baseTotal += mean(lost(c.base))
			headTotal += mean(lost(c.head))
		}
	}
	if final == invalid {
		fmt.Fprintf(&b, "| total | - | - | - | | %s |\n", final)
	} else {
		fmt.Fprintf(&b, "| total | %s | %s | %s | | %s |\n",
			num(baseTotal, 2), num(headTotal, 2), signed(headTotal-baseTotal), final)
	}
	b.WriteString("\nEach number is the seconds an average tenant was down in the window after SIGTERM " +
		"(tenants weighted equally), as the mean over restarts with the range in parentheses; lower is better.\n")
	for _, c := range cmps {
		if c.fewRestarts {
			fmt.Fprintf(&b, "\nNote: %s has fewer than %d restarts on a side, too few for a verdict, so it is NO CHANGE.\n",
				c.track.Name, minRestarts)
		}
	}

	if final == invalid {
		b.WriteString("\n### Why INVALID\n\n")
		for _, c := range cmps {
			for i, reasons := range c.reasons {
				for _, r := range reasons {
					fmt.Fprintf(&b, "- %s, %s: %s\n", c.track.Name, sideName(i), r)
				}
			}
		}
	}

	b.WriteString("\n### Diagnostics\n\n")
	b.WriteString("Bad operations in restart windows by cause, summed over restarts.\n\n")
	b.WriteString("| track | side | 429/ResourceExhausted | Unavailable | timeout | late | other | SIGTERM to ready, mean s | peak conns / max | steady bad % |\n")
	b.WriteString("|---|---|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, c := range cmps {
		for i, rs := range [2][]Result{c.base, c.head} {
			if len(rs) != 1 {
				continue
			}
			r := rs[0]
			e := r.Errors
			var down []float64
			for _, rst := range r.Restarts {
				down = append(down, rst.DownS)
			}
			steady := "-"
			if r.Steady.Offered > 0 {
				steady = num(100*float64(r.Steady.Bad)/float64(r.Steady.Offered), 2)
			}
			fmt.Fprintf(&b, "| %s | %s | %d | %d | %d | %d | %d | %s | %d / %d | %s |\n",
				c.track.Name, sideName(i), e.ResourceExhausted, e.Unavailable, e.Timeout, e.Late, e.Other,
				num(mean(down), 1), r.PeakConns, c.track.MaxConns, steady)
		}
	}

	b.WriteString("\n### Bad operations around SIGTERM\n")
	for _, c := range cmps {
		if len(c.base) == 1 && len(c.head) == 1 {
			b.WriteString("\n")
			b.WriteString(chart(c.track.Name, c.base[0].Timeline, c.head[0].Timeline))
		}
	}

	if final == invalid {
		b.WriteString("\nSCORE INVALID\n")
	} else {
		fmt.Fprintf(&b, "\nSCORE %s\n", num(headTotal, 3))
	}
	return b.String()
}

func sideName(i int) string {
	return [...]string{"base", "head"}[i]
}

// lost is a side's lost seconds per restart, or nil when the side has no
// single result with restarts to show.
func lost(rs []Result) []float64 {
	if len(rs) != 1 {
		return nil
	}
	return rs[0].lostS()
}

func lostCell(rs []Result) string {
	l := lost(rs)
	if len(l) == 0 {
		return "-"
	}
	return fmt.Sprintf("%s (%s–%s)", num(mean(l), 2), num(slices.Min(l), 2), num(slices.Max(l), 2))
}

func deltaCell(base, head []Result) string {
	b, h := lost(base), lost(head)
	if len(b) == 0 || len(h) == 0 {
		return "-"
	}
	return signed(mean(h) - mean(b))
}

func signed(f float64) string {
	if f > 0 {
		return "+" + num(f, 2)
	}
	return num(f, 2)
}

// chart draws the two timelines on one Mermaid xychart. xychart has no
// legend, so the title says which line is which.
func chart(track string, base, head []float64) string {
	var b strings.Builder
	b.WriteString("```mermaid\nxychart-beta\n")
	fmt.Fprintf(&b, "    title \"%s: bad %% around SIGTERM (1st line base, 2nd line head)\"\n", track)
	fmt.Fprintf(&b, "    x-axis \"seconds since SIGTERM\" %d --> %d\n", timelineStartS, timelineStartS+timelineBuckets-1)
	b.WriteString("    y-axis \"bad %\" 0 --> 100\n")
	for _, line := range [][]float64{base, head} {
		points := make([]string, len(line))
		for i, p := range line {
			points[i] = num(p, 1)
		}
		fmt.Fprintf(&b, "    line [%s]\n", strings.Join(points, ", "))
	}
	b.WriteString("```\n")
	return b.String()
}
