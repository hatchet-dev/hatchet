package authz

import (
	"testing"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/stretchr/testify/assert"
)

var permittedWithUnverifiedEmail = []string{
	"UserGetCurrent",
	"UserUpdateLogout",
}

var restrictedWithBearerToken = []string{
	// bearer tokens cannot read, list, or write other bearer tokens
	"ApiTokenList",
	"ApiTokenCreate",
	"ApiTokenUpdateRevoke",
}

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
	az := AuthZ{
		config: nil,
		rbac:   NewAuthorizer(),
		l:      nil,
	}
	for i := 0; i < len(adminAndOwnerOnly); i++ {
		operationId := adminAndOwnerOnly[i]
		assert.Equal(t, az.authorizeTenantOperations(sqlcv1.TenantMemberRoleADMIN, createRouteInfo(operationId)), nil)
		assert.Equal(t, az.authorizeTenantOperations(sqlcv1.TenantMemberRoleOWNER, createRouteInfo(operationId)), nil)
		assert.NotEqual(t, az.authorizeTenantOperations(sqlcv1.TenantMemberRoleMEMBER, createRouteInfo(operationId)), nil)
	}
}

func TestAuthorizeUnverifiedEmailOperations(t *testing.T) {
	allOperations := operationIdsFromSpec()
	az := AuthZ{
		config: nil,
		rbac:   NewAuthorizer(),
		l:      nil,
	}
	for i := 0; i < len(allOperations); i++ {
		operationId := allOperations[i]
		isPermed := az.authorizeTenantOperations(sqlcv1.TenantMemberRoleUNVERIFIEDEMAIL, createRouteInfo(operationId))
		if operationIn(operationId, permittedWithUnverifiedEmail) {
			assert.Equal(t, isPermed, nil)
		} else {
			assert.NotEqual(t, isPermed, nil)
		}
	}
}

func TestAuthorizeBearerTokenOperations(t *testing.T) {
	allOperations := operationIdsFromSpec()
	az := AuthZ{
		config: nil,
		rbac:   NewAuthorizer(),
		l:      nil,
	}
	for i := 0; i < len(allOperations); i++ {
		operationId := allOperations[i]
		isPermed := az.authorizeTenantOperations(sqlcv1.TenantMemberRoleBEARERTOKEN, createRouteInfo(operationId))
		if operationIn(operationId, restrictedWithBearerToken) {
			assert.NotEqual(t, isPermed, nil)
		} else {
			assert.Equal(t, isPermed, nil)
		}
	}
}
