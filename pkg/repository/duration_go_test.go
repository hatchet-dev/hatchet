//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Same corpus as TestConvertDurationToInterval: the Go port must produce the interval
// the SQL function returns. TestDurationToIntervalMatchesSQL checks the resulting
// deadlines against the database.
func TestDurationToInterval(t *testing.T) {
	cases := []struct {
		input   string
		want    pgtype.Interval
		wantErr bool
	}{
		{input: "42s", want: pgtype.Interval{Microseconds: 42_000_000, Valid: true}},
		{input: "11m", want: pgtype.Interval{Microseconds: 660_000_000, Valid: true}},
		{input: "1h", want: pgtype.Interval{Microseconds: 3_600_000_000, Valid: true}},
		{input: "42m30s", want: pgtype.Interval{Microseconds: 2_550_000_000, Valid: true}},
		{input: "1h30m", want: pgtype.Interval{Microseconds: 5_400_000_000, Valid: true}},
		{input: "1h30m5s", want: pgtype.Interval{Microseconds: 5_405_000_000, Valid: true}},
		{input: "1500ms", want: pgtype.Interval{Microseconds: 1_500_000, Valid: true}},
		{input: "1.5h", want: pgtype.Interval{Microseconds: 5_400_000_000, Valid: true}},
		{input: "0s", want: pgtype.Interval{Valid: true}},
		// make_interval rounds to the nearest microsecond
		{input: "0.0004996s", want: pgtype.Interval{Microseconds: 500, Valid: true}},
		{input: "0.0014996s", want: pgtype.Interval{Microseconds: 1500, Valid: true}},
		// fallback to 5 minutes
		{input: "", want: pgtype.Interval{Microseconds: 300_000_000, Valid: true}},
		{input: "bad", want: pgtype.Interval{Microseconds: 300_000_000, Valid: true}},
		{input: "42", want: pgtype.Interval{Microseconds: 300_000_000, Valid: true}},
		{input: "0", want: pgtype.Interval{Microseconds: 300_000_000, Valid: true}},
		{input: "-1.5h", want: pgtype.Interval{Microseconds: 300_000_000, Valid: true}},
		{input: "+1h", want: pgtype.Interval{Microseconds: 300_000_000, Valid: true}},
		{input: "100ns", want: pgtype.Interval{Microseconds: 300_000_000, Valid: true}},
		{input: "100us", want: pgtype.Interval{Microseconds: 300_000_000, Valid: true}},
		{input: "30s1d", want: pgtype.Interval{Microseconds: 300_000_000, Valid: true}},
		{input: "1.5d", want: pgtype.Interval{Microseconds: 300_000_000, Valid: true}},
		{input: "999999999d", want: pgtype.Interval{Microseconds: 300_000_000, Valid: true}},
		// legacy single-unit suffixes keep calendar semantics
		{input: "1d", want: pgtype.Interval{Days: 1, Valid: true}},
		{input: "1w", want: pgtype.Interval{Days: 7, Valid: true}},
		{input: "10d", want: pgtype.Interval{Days: 10, Valid: true}},
		{input: "1y", want: pgtype.Interval{Months: 12, Valid: true}},
		{input: "99999999y", want: pgtype.Interval{Months: 1_199_999_988, Valid: true}},
		// the SQL function raises on these
		{input: "9999999999999h", wantErr: true},
		{input: "1234567890123456h", wantErr: true},
		{input: "9223372037s", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := durationToInterval(tc.input)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got, "input=%q", tc.input)
		})
	}
}
