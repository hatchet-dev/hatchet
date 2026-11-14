package cli

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTenantControlSetOpts(t *testing.T) {
	tenantID := uuid.New()
	limit := int32(250)
	zero := int32(0)

	opts, err := tenantControlSetOpts(tenantID.String(), &limit, false, "noisy tenant")
	require.NoError(t, err)
	require.Equal(t, tenantID, opts.TenantId)
	require.Equal(t, &limit, opts.IngestPerSecond)
	require.False(t, opts.Paused)
	require.Equal(t, "noisy tenant", opts.Reason)

	opts, err = tenantControlSetOpts(tenantID.String(), nil, true, "noisy tenant")
	require.NoError(t, err)
	require.Nil(t, opts.IngestPerSecond)
	require.True(t, opts.Paused)

	_, err = tenantControlSetOpts(tenantID.String(), nil, false, "noisy tenant")
	require.EqualError(t, err, "set at least one of --paused and --ingest-per-second; clear removes every control")

	_, err = tenantControlSetOpts(tenantID.String(), &zero, true, "noisy tenant")
	require.EqualError(t, err, "--ingest-per-second must be at least 1, got 0")

	_, err = tenantControlSetOpts(tenantID.String(), &limit, false, "")
	require.EqualError(t, err, "--reason must not be empty")

	_, err = tenantControlSetOpts("not-a-uuid", &limit, false, "noisy tenant")
	require.ErrorContains(t, err, "--tenant-id is not a valid uuid")
}
