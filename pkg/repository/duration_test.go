//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	hatchetvalidator "github.com/hatchet-dev/hatchet/pkg/validator"
)

type durationField struct {
	Duration string `validate:"duration"`
}

func TestConvertDurationToInterval(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	v := hatchetvalidator.NewDefaultValidator()

	// Shared corpus: everything registration validation accepts must be
	// enforced as the same number of seconds. Everything it rejects must
	// either hit the SQL fallback (5 minutes = 300s) for genuinely garbled
	// input, or raise loudly when the value is well outside the grammar
	// (oversized digit count, beyond Go's max duration).
	epochCases := []struct {
		name         string
		input        string
		seconds      float64
		wantFallback bool
		wantRaise    bool
	}{
		{name: "single second", input: "42s", seconds: 42},
		{name: "single minute", input: "11m", seconds: 660},
		{name: "single hour", input: "1h", seconds: 3600},
		{name: "minute and second", input: "42m30s", seconds: 2550},
		{name: "hour and minute", input: "1h30m", seconds: 5400},
		{name: "hour minute second", input: "1h30m5s", seconds: 5405},
		{name: "milliseconds", input: "1500ms", seconds: 1.5},
		{name: "decimal hour", input: "1.5h", seconds: 5400},
		{name: "zero with unit", input: "0s", seconds: 0},
		{name: "invalid text", input: "bad", wantFallback: true},
		{name: "missing unit", input: "42", wantFallback: true},
		{name: "bare zero", input: "0", wantFallback: true},
		{name: "negative", input: "-1.5h", wantFallback: true},
		{name: "leading plus sign", input: "+1h", wantFallback: true},
		{name: "nanoseconds", input: "100ns", wantFallback: true},
		{name: "microseconds", input: "100us", wantFallback: true},
		{name: "mixed multi-unit and legacy", input: "30s1d", wantFallback: true},
		{name: "decimal legacy unit", input: "1.5d", wantFallback: true},
		{name: "overflows go duration", input: "9999999999999h", wantRaise: true},
		{name: "exceeds digit cap", input: "1234567890123456h", wantRaise: true},
	}

	for _, tc := range epochCases {
		t.Run(tc.name, func(t *testing.T) {
			valErr := v.Validate(&durationField{Duration: tc.input})
			if tc.wantFallback || tc.wantRaise {
				assert.Error(t, valErr, "validator must reject %q", tc.input)
			} else {
				assert.NoError(t, valErr, "validator must accept %q", tc.input)
			}

			if tc.wantRaise {
				var got float64
				err := pool.QueryRow(ctx,
					`SELECT EXTRACT(EPOCH FROM convert_duration_to_interval($1))::double precision`,
					tc.input,
				).Scan(&got)
				require.Error(t, err, "SQL function must raise on %q", tc.input)
				return
			}

			var got float64
			err := pool.QueryRow(ctx,
				`SELECT EXTRACT(EPOCH FROM convert_duration_to_interval($1))::double precision`,
				tc.input,
			).Scan(&got)
			require.NoError(t, err)

			want := tc.seconds
			if tc.wantFallback {
				want = 300
			}

			assert.InDelta(t, want, got, 1e-9, "input=%q", tc.input)
		})
	}

	// Legacy single-unit suffixes are kept for values stored before the
	// migration. Registration validation rejects them; the SQL function
	// still parses them with calendar semantics.
	legacyCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "single day stays calendar", input: "1d", expected: "1 day"},
		{name: "single week stays calendar", input: "1w", expected: "7 days"},
		{name: "single year stays calendar", input: "1y", expected: "12 mons"},
		{name: "multiple days stay calendar", input: "10d", expected: "10 days"},
	}

	for _, tc := range legacyCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Error(t, v.Validate(&durationField{Duration: tc.input}),
				"validator must reject legacy unit %q", tc.input)

			var equal bool
			err := pool.QueryRow(ctx,
				`SELECT convert_duration_to_interval($1) = $2::interval`,
				tc.input, tc.expected,
			).Scan(&equal)
			require.NoError(t, err)

			assert.True(t, equal, "input=%q expected interval=%q", tc.input, tc.expected)
		})
	}

	// Legacy values with oversized digit counts fall back instead of raising.
	t.Run("legacy digit cap falls back", func(t *testing.T) {
		var got float64
		err := pool.QueryRow(ctx,
			`SELECT EXTRACT(EPOCH FROM convert_duration_to_interval($1))::double precision`,
			"999999999d",
		).Scan(&got)
		require.NoError(t, err)

		assert.InDelta(t, 300, got, 1e-9)
	})
}
