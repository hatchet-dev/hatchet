package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// setFlags resets set's flags to their defaults, then sets the given ones as the command line would.
func setFlags(t *testing.T, flags map[string]string) {
	t.Helper()

	for _, name := range []string{"paused", "ingest-per-second", "db-time-percent", "db-conn-percent", "reason"} {
		f := tenantControlSetCmd.Flags().Lookup(name)
		require.NoError(t, f.Value.Set(f.DefValue))
		f.Changed = false
	}

	for name, value := range flags {
		require.NoError(t, tenantControlSetCmd.Flags().Set(name, value))
	}
}

func TestSetIngestPerSecondFlag(t *testing.T) {
	setFlags(t, map[string]string{"ingest-per-second": "250", "reason": "noisy"})

	opts, err := setOptsFromFlags(tenantControlSetCmd)
	require.NoError(t, err)
	require.NotNil(t, opts.IngestPerSecond)
	require.Equal(t, int32(250), *opts.IngestPerSecond)
	require.False(t, opts.Paused)
	require.Nil(t, opts.DBTimePercent)
	require.Nil(t, opts.DBConnPercent)

	setFlags(t, map[string]string{"ingest-per-second": "5000", "db-time-percent": "20", "reason": "noisy"})

	opts, err = setOptsFromFlags(tenantControlSetCmd)
	require.NoError(t, err)
	require.Equal(t, int32(5000), *opts.IngestPerSecond, "the limit has no upper bound of 100")
	require.Equal(t, int32(20), *opts.DBTimePercent)

	setFlags(t, map[string]string{"paused": "true", "reason": "noisy"})

	opts, err = setOptsFromFlags(tenantControlSetCmd)
	require.NoError(t, err)
	require.Nil(t, opts.IngestPerSecond, "a flag left out is unset")

	setFlags(t, map[string]string{"ingest-per-second": "0", "reason": "noisy"})

	_, err = setOptsFromFlags(tenantControlSetCmd)
	require.EqualError(t, err, "--ingest-per-second must be from 1 to 2147483647, got 0")

	setFlags(t, map[string]string{"reason": "noisy"})

	_, err = setOptsFromFlags(tenantControlSetCmd)
	require.ErrorContains(t, err, "--ingest-per-second", "set without any control names the flags")

	setFlags(t, map[string]string{"db-conn-percent": "101", "reason": "noisy"})

	_, err = setOptsFromFlags(tenantControlSetCmd)
	require.EqualError(t, err, "--db-conn-percent must be from 1 to 100, got 101")
}
