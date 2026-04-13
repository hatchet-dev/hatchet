package authz

import (
	"testing"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/auth/rbac"
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
	r, err := newHatchetAuthorizer()
	assert.Nil(t, err)
	allOperations := operationIdsFromSpec()
	for _, operationId := range allOperations {
		assert.Equal(t, r.IsAuthorized(string(sqlcv1.TenantMemberRoleADMIN), operationId), true)
		assert.Equal(t, r.IsAuthorized(string(sqlcv1.TenantMemberRoleOWNER), operationId), true)
		if rbac.OperationIn(operationId, adminAndOwnerOnly) {
			assert.Equal(t, r.IsAuthorized(string(sqlcv1.TenantMemberRoleMEMBER), operationId), false)
		} else {
			assert.Equal(t, r.IsAuthorized(string(sqlcv1.TenantMemberRoleMEMBER), operationId), true)
		}
	}
}

func TestValidateSpec(t *testing.T) {
	_, err := newHatchetAuthorizer()
	assert.Nil(t, err)
}
