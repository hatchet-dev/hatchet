package repository

import (
	"fmt"
	"math"
	"regexp"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
)

// The two grammars below mirror the convert_duration_to_interval SQL function
// (sql/schema/v1-core.sql). Keep them in sync: the flush parses step_timeout in
// Go so the statement does not call the plpgsql function per row. The statement
// still adds the interval to CURRENT_TIMESTAMP, so deadlines stay on the database
// clock and keep Postgres' calendar arithmetic and timestamp range checks.
var (
	legacyDurationRegex    = regexp.MustCompile(`^([0-9]{1,8})(d|w|y)$`)
	durationComponentRegex = regexp.MustCompile(`^([0-9]+(?:\.[0-9]*)?|\.[0-9]+)(ms|s|m|h)`)
)

const (
	defaultDurationFallbackMicros = 5 * 60 * 1_000_000
	maxDurationSeconds            = 9223372036
	maxDurationDigits             = 15
)

// durationToInterval returns the interval convert_duration_to_interval(duration)
// produces: empty or unparseable input falls back to 5 minutes, legacy d/w/y
// suffixes produce calendar days or months, and the inputs the SQL function raises
// on return an error.
func durationToInterval(duration string) (pgtype.Interval, error) {
	if duration == "" {
		return pgtype.Interval{Microseconds: defaultDurationFallbackMicros, Valid: true}, nil
	}

	if m := legacyDurationRegex.FindStringSubmatch(duration); m != nil {
		// at most eight digits, so the int32 fields cannot overflow
		val, err := strconv.ParseInt(m[1], 10, 32)

		if err != nil {
			return pgtype.Interval{}, err
		}

		n := int32(val)

		switch m[2] {
		case "d":
			return pgtype.Interval{Days: n, Valid: true}, nil
		case "w":
			return pgtype.Interval{Days: n * 7, Valid: true}, nil
		default:
			return pgtype.Interval{Months: n * 12, Valid: true}, nil
		}
	}

	rest := duration
	totalSeconds := 0.0

	for len(rest) > 0 {
		m := durationComponentRegex.FindStringSubmatch(rest)

		if m == nil {
			return pgtype.Interval{Microseconds: defaultDurationFallbackMicros, Valid: true}, nil
		}

		if len(m[1]) > maxDurationDigits {
			return pgtype.Interval{}, fmt.Errorf("duration %s has a numeric component exceeding %d digits", duration, maxDurationDigits)
		}

		val, err := strconv.ParseFloat(m[1], 64)

		if err != nil {
			return pgtype.Interval{}, err
		}

		switch m[2] {
		case "ms":
			totalSeconds += val * 1e-3
		case "s":
			totalSeconds += val
		case "m":
			totalSeconds += val * 60
		case "h":
			totalSeconds += val * 3600
		}

		rest = rest[len(m[0]):]
	}

	if totalSeconds > maxDurationSeconds {
		return pgtype.Interval{}, fmt.Errorf("duration %s exceeds maximum supported value (~292 years)", duration)
	}

	// make_interval(secs => ...) rounds to the nearest microsecond
	return pgtype.Interval{Microseconds: int64(math.RoundToEven(totalSeconds * 1e6)), Valid: true}, nil
}
