package tenants

import (
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TenantService) TenantOidcGroupMappingCreate(ctx echo.Context, request gen.TenantOidcGroupMappingCreateRequestObject) (gen.TenantOidcGroupMappingCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	if t.config.Auth.OIDCProvider == nil {
		return gen.TenantOidcGroupMappingCreate400JSONResponse(apierrors.NewAPIErrors("OIDC authentication is not enabled")), nil
	}
	if apiErrors, err := t.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.TenantOidcGroupMappingCreate400JSONResponse(*apiErrors), nil
	}

	group := strings.TrimSpace(request.Body.Group)
	role := string(request.Body.Role)
	if group == "" || !isOIDCGroupMappingRole(role) {
		return gen.TenantOidcGroupMappingCreate400JSONResponse(apierrors.NewAPIErrors("OIDC group and role must be valid")), nil
	}
	if hasGlobalConfigMapping(group, t.config.Auth.OIDCGroupMappings) {
		return gen.TenantOidcGroupMappingCreate400JSONResponse(apierrors.NewAPIErrors("OIDC group is managed by server configuration")), nil
	}

	mapping, err := t.config.V1.Tenant().UpsertTenantOIDCGroupMapping(ctx.Request().Context(), tenant.ID, &v1.UpsertTenantOIDCGroupMappingOpts{
		Group: group,
		Role:  role,
	})
	if err != nil {
		return nil, err
	}

	t.config.Analytics.Enqueue(ctx.Request().Context(), analytics.OIDCGroupMapping, analytics.Create, mapping.ID.String(), map[string]interface{}{
		"role": role,
	})
	mappingID := mapping.ID
	return gen.TenantOidcGroupMappingCreate200JSONResponse{
		Id: &mappingID, Group: mapping.Group, Role: gen.TenantMemberRole(mapping.Role),
	}, nil
}

func (t *TenantService) TenantOidcGroupMappingDelete(ctx echo.Context, request gen.TenantOidcGroupMappingDeleteRequestObject) (gen.TenantOidcGroupMappingDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	deleted, err := t.config.V1.Tenant().DeleteTenantOIDCGroupMapping(ctx.Request().Context(), tenant.ID, request.Mapping)
	if err != nil {
		return nil, err
	}
	if !deleted {
		return gen.TenantOidcGroupMappingDelete404JSONResponse(apierrors.NewAPIErrors("OIDC group mapping not found")), nil
	}
	t.config.Analytics.Enqueue(ctx.Request().Context(), analytics.OIDCGroupMapping, analytics.Delete, request.Mapping.String(), nil)
	return gen.TenantOidcGroupMappingDelete204Response{}, nil
}

func isOIDCGroupMappingRole(role string) bool {
	return role == "ADMIN" || role == "MEMBER" || role == "VIEWER"
}

func hasGlobalConfigMapping(group string, mappings map[string]string) bool {
	_, ok := mappings[group]
	return ok
}
