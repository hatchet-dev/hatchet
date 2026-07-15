package authz

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/auth/rbac"
)

var deniedWithAPIToken = []string{
	"ApiTokenCreate",
	"ApiTokenList",
	"ApiTokenUpdateRevoke",
	"TenantInviteCreate",
	"TenantInviteUpdate",
	"TenantMemberDelete",
	"TenantMemberUpdate",
}

func TestBearerPolicyValidateSpec(t *testing.T) {
	_, err := newBearerPolicy()
	assert.Nil(t, err)
}

func TestBearerPolicyClassifiesEveryOperation(t *testing.T) {
	policy, err := newBearerPolicy()
	require.NoError(t, err)

	for _, operationId := range operationIdsFromSpec() {
		assert.Equal(t, rbac.OperationIn(operationId, deniedWithAPIToken), policy.IsDenied(operationId), operationId)
	}
}

func TestBearerPolicyRejectsUnclassifiedOperation(t *testing.T) {
	policy, err := rbac.LoadBearerPolicy([]byte("operations:\n  denied: []\n  allowed: []\n"))
	require.NoError(t, err)

	spec, err := gen.GetSwagger()
	require.NoError(t, err)

	err = policy.ValidateSpec(*spec)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "exists in openapi specs but not in bearer.yaml")
}

func TestBearerPolicyRejectsUnknownOperation(t *testing.T) {
	policy, err := rbac.LoadBearerPolicy([]byte("operations:\n  denied:\n    - NotARealOperation\n  allowed: []\n"))
	require.NoError(t, err)

	spec, err := gen.GetSwagger()
	require.NoError(t, err)

	err = policy.ValidateSpec(*spec)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "NotARealOperation exists in bearer.yaml but not in specs")
}
