package authz

import (
	"testing"

	"github.com/stretchr/testify/require"

	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

func TestListedCapabilitiesAndRolesFollowTheExtension(t *testing.T) {
	byName := func(caps []pkgauthz.CapabilityInfo) map[string]pkgauthz.CapabilityInfo {
		out := map[string]pkgauthz.CapabilityInfo{}
		for _, c := range caps {
			out[c.Name] = c
		}

		return out
	}

	// without an extension the infrastructure capabilities grant nothing, so they are never offered
	_, oss, err := LoadPermissions(&server.ServerConfig{})
	require.NoError(t, err)

	ossCaps := byName(oss.Capabilities())
	require.NotContains(t, ossCaps, "infra:read")
	require.NotContains(t, ossCaps[pkgauthz.ReadOnlyCapability].Includes, "infra:read")
	require.Contains(t, ossCaps["worker"].Includes, "runs:trigger", "worker lists what it inherits")
	require.Equal(t, "Read", ossCaps[pkgauthz.ReadOnlyCapability].Label)

	config := &server.ServerConfig{}
	config.Auth.AllowedOperations = []string{"ManagedWorkerList"}
	config.Auth.AllowedWriteOperations = []string{"ManagedWorkerCreate"}

	perms, cloud, err := LoadPermissions(config)
	require.NoError(t, err)

	cloudCaps := byName(cloud.Capabilities())
	require.Contains(t, cloudCaps, "infra:write")
	require.Contains(t, cloudCaps[pkgauthz.ReadOnlyCapability].Includes, "infra:read")
	require.NotContains(t, cloudCaps[pkgauthz.ReadOnlyCapability].Includes, "infra:write")

	all := cloudCaps[pkgauthz.AllCapabilities]
	require.True(t, all.Bundle)
	require.ElementsMatch(t, all.Includes, func() []string {
		var selectable []string
		for _, c := range cloud.Capabilities() {
			if !c.Bundle {
				selectable = append(selectable, c.Name)
			}
		}

		return selectable
	}(), "all stands for every listed capability")

	roles := map[string][]string{}
	for _, r := range perms.AssignableRoles() {
		roles[r.Name] = r.Operations
	}

	require.Len(t, roles, 4, "only assignable roles are listed")
	require.Contains(t, roles["VIEWER"], "ManagedWorkerList")
	require.NotContains(t, roles["VIEWER"], "ManagedWorkerCreate")
	require.NotContains(t, roles["VIEWER"], "V1TaskCancel")
	require.Contains(t, roles["MEMBER"], "ManagedWorkerCreate")
	require.Contains(t, roles["ADMIN"], "ApiTokenCreate")
	require.NotContains(t, roles["MEMBER"], "ApiTokenCreate")
}
