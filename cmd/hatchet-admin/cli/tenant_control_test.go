package cli

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTenantControlSetOpts(t *testing.T) {
	tenantID := uuid.New()

	opts, err := tenantControlSetOpts(tenantID.String(), 250, "noisy tenant")
	require.NoError(t, err)
	require.Equal(t, tenantID, opts.TenantId)
	require.Equal(t, int32(250), opts.IngestPerSecond)
	require.Equal(t, "noisy tenant", opts.Reason)

	_, err = tenantControlSetOpts(tenantID.String(), 0, "noisy tenant")
	require.EqualError(t, err, "--ingest-per-second must be at least 1, got 0")

	_, err = tenantControlSetOpts(tenantID.String(), 250, "")
	require.EqualError(t, err, "--reason must not be empty")

	_, err = tenantControlSetOpts("not-a-uuid", 250, "noisy tenant")
	require.ErrorContains(t, err, "--tenant-id is not a valid uuid")
}
