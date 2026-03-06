package authz

import (
	"testing"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
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

func createRouteInfo(operationId string) *middleware.RouteInfo {
	return &middleware.RouteInfo{
		OperationID: operationId,
		Security:    nil,
		Resources:   nil,
		Route:       nil,
	}
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
	az := AuthZ{
		config: nil,
		rbac:   r,
		l:      nil,
	}
	allOperations := operationIdsFromSpec()
	for _, operationId := range allOperations {
		assert.Equal(t, az.authorizeTenantOperations(sqlcv1.TenantMemberRoleADMIN, createRouteInfo(operationId)), nil)
		assert.Equal(t, az.authorizeTenantOperations(sqlcv1.TenantMemberRoleOWNER, createRouteInfo(operationId)), nil)
		if operationIn(operationId, adminAndOwnerOnly) {
			assert.NotEqual(t, az.authorizeTenantOperations(sqlcv1.TenantMemberRoleMEMBER, createRouteInfo(operationId)), nil)
		} else {
			assert.Equal(t, az.authorizeTenantOperations(sqlcv1.TenantMemberRoleMEMBER, createRouteInfo(operationId)), nil)
		}
	}
}
