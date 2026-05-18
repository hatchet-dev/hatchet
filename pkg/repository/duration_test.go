//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertDurationToInterval(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()

	epochCases := []struct {
		name    string
		input   string
		seconds float64
	}{
		{"single second", "42s", 42},
		{"single minute", "11m", 660},
		{"single hour", "1h", 3600},
		{"minute and second", "42m30s", 2550},
		{"hour and minute", "1h30m", 5400},
		{"hour minute second", "1h30m5s", 5405},
		{"milliseconds", "1500ms", 1.5},
		{"decimal hour", "1.5h", 5400},
		{"negative decimal hour", "-1.5h", -5400},
		{"zero with unit", "0s", 0},
		{"invalid falls back to five minutes", "bad", 300},
		{"mixed multi-unit and legacy is rejected", "30s1d", 300},
		{"missing unit falls back", "42", 300},
	}

	for _, tc := range epochCases {
		t.Run(tc.name, func(t *testing.T) {
			var got float64
			err := pool.QueryRow(ctx,
				`SELECT EXTRACT(EPOCH FROM convert_duration_to_interval($1))::double precision`,
				tc.input,
			).Scan(&got)
			require.NoError(t, err)

			assert.InDelta(t, tc.seconds, got, 1e-9, "input=%q", tc.input)
		})
	}

	legacyCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"single day stays calendar", "1d", "1 day"},
		{"single week stays calendar", "1w", "7 days"},
		{"single year stays calendar", "1y", "12 mons"},
		{"multiple days stay calendar", "10d", "10 days"},
	}

	for _, tc := range legacyCases {
		t.Run(tc.name, func(t *testing.T) {
			var equal bool
			err := pool.QueryRow(ctx,
				`SELECT convert_duration_to_interval($1) = $2::interval`,
				tc.input, tc.expected,
			).Scan(&equal)
			require.NoError(t, err)

			assert.True(t, equal, "input=%q expected interval=%q", tc.input, tc.expected)
		})
	}
}
