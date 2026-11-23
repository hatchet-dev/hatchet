package permissions

import (
	"testing"

	"github.com/stretchr/testify/require"

	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
)

func TestTokensLoad(t *testing.T) {
	_, err := Tokens()
	require.NoError(t, err)

	for _, c := range []string{pkgauthz.AllCapabilities, pkgauthz.ReadOnlyCapability, "worker", "events:push"} {
		ok, err := TokenCapabilityExists(c)
		require.NoError(t, err)
		require.True(t, ok, c)
	}

	ok, err := TokenCapabilityExists("workers:fly")
	require.NoError(t, err)
	require.False(t, ok)
}
