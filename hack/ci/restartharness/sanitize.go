package main

// Input policy for compare.
//
// In CI both results files come from a job that runs pull request code, and
// compare's output is posted as a pull request comment by a job holding a
// write token. The output is safe by construction, not by spotting bad input:
//
//  1. readInput accepts only a regular file (no symlink, pipe or device) of
//     at most maxInputBytes.
//  2. parseResults decodes every Result in full and rejects the whole file
//     unless each one is in bounds: at most maxResults results, restarts and
//     tenants per restart capped, a timeline of exactly timelineBuckets
//     percentages in [0, 100], no negative count, no more bad than offered
//     operations, and a track name that is in the tracks registry.
//  3. The output holds only constant text, track names taken from the
//     registry (never from the file), and numbers the program formatted
//     itself through num, which maps NaN and Inf to 0. No string from the
//     input is ever echoed, not even in an error message.
//  4. Every reported number is recomputed from the raw counts.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
)

const (
	maxInputBytes = 4 << 20
	maxResults    = 16
	maxRestarts   = 100
	maxTenants    = 1000
	maxDownS      = 3600
)

// readInput refuses anything but a regular file, since a pipe would block
// and a device or symlink could point anywhere, and never holds more than
// maxInputBytes of it.
func readInput(path string) ([]byte, error) {
	fi, err := os.Lstat(path) //nolint:gosec // the caller names the file to read
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() || fi.Size() > maxInputBytes {
		return nil, errors.New("not a regular file within the size cap")
	}
	f, err := os.Open(path) //nolint:gosec // the caller names the file to read
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxInputBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxInputBytes {
		return nil, errors.New("input exceeds the size cap")
	}
	return data, nil
}

// parseResults decodes a stream of concatenated Result objects. Errors name
// the result's position and byte offset only, never input text.
func parseResults(data []byte) ([]Result, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	var out []Result
	for {
		var r Result
		err := dec.Decode(&r)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("result %d is not a valid result object (near byte %d)", len(out)+1, dec.InputOffset())
		}
		if len(out) == maxResults {
			return nil, fmt.Errorf("more than %d results", maxResults)
		}
		if err := checkResult(r); err != nil {
			return nil, fmt.Errorf("result %d: %w", len(out)+1, err)
		}
		out = append(out, r)
	}
	if len(out) == 0 {
		return nil, errors.New("no results")
	}
	return out, nil
}

func checkResult(r Result) error {
	if _, ok := trackByName(r.Track.Name); !ok {
		return errors.New("unknown track name")
	}
	if len(r.Restarts) > maxRestarts {
		return fmt.Errorf("more than %d restarts", maxRestarts)
	}
	if len(r.Timeline) != timelineBuckets {
		return fmt.Errorf("timeline must have %d buckets", timelineBuckets)
	}
	for _, p := range r.Timeline {
		if p < 0 || p > 100 {
			return errors.New("timeline percentage outside 0 to 100")
		}
	}
	counts := []Counts{r.Steady}
	for _, rs := range r.Restarts {
		if len(rs.Tenants) > maxTenants {
			return fmt.Errorf("more than %d tenants in a restart", maxTenants)
		}
		if rs.DownS < 0 || rs.DownS > maxDownS {
			return fmt.Errorf("down_s outside 0 to %d", maxDownS)
		}
		counts = append(counts, rs.Tenants...)
	}
	for _, c := range counts {
		if c.Offered < 0 || c.Bad < 0 || c.Bad > c.Offered {
			return errors.New("counts must satisfy 0 <= bad <= offered")
		}
	}
	e := r.Errors
	for _, n := range []int{e.ResourceExhausted, e.Unavailable, e.Timeout, e.Late, e.Other,
		r.Lost, r.Duplicates, r.MaxRunsViolations, r.PeakConns} {
		if n < 0 {
			return errors.New("negative count")
		}
	}
	return nil
}

// num formats every float that reaches the output. NaN and Inf can only come
// from arithmetic on extreme recorded settings and become 0.
func num(f float64, decimals int) string {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		f = 0
	}
	return strconv.FormatFloat(f, 'f', decimals, 64)
}
