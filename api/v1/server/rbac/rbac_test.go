package rbac

import (
	"testing"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"

	"github.com/stretchr/testify/assert"
)

var adminAndOwnerOnly = []string{
	"TenantInviteList",
	"TenantInviteCreate",
	"TenantInviteUpdate",
	"TenantInviteDelete",
	"TenantMemberList",
	"TenantMemberUpdate",
	// members cannot create API tokens for a tenant, because they have admin permissions
	"ApiTokenList",
	"ApiTokenCreate",
	"ApiTokenUpdateRevoke",
}

func operationIdsFromSpec() []string {
	spec, _ := gen.GetSwagger()
	allOperationIds := make([]string, 0)
	for _, v := range spec.Paths.Map() {
		for _, vv := range v.Operations() {
			allOperationIds = append(allOperationIds, vv.OperationID)
		}
	}
	return allOperationIds
}

func TestAuthorizeTenantOperations(t *testing.T) {
	r, err := NewAuthorizer()
	assert.Nil(t, err)
	allOperations := operationIdsFromSpec()
	for _, operationId := range allOperations {
		assert.Equal(t, r.IsAuthorized(sqlcv1.TenantMemberRoleADMIN, operationId), true)
		assert.Equal(t, r.IsAuthorized(sqlcv1.TenantMemberRoleOWNER, operationId), true)
		if OperationIn(operationId, adminAndOwnerOnly) {
			assert.Equal(t, r.IsAuthorized(sqlcv1.TenantMemberRoleMEMBER, operationId), false)
		} else {
			assert.Equal(t, r.IsAuthorized(sqlcv1.TenantMemberRoleMEMBER, operationId), true)
		}
	}
}
