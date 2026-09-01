package tenants

import (
	"sort"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TenantService) TenantMemberList(ctx echo.Context, request gen.TenantMemberListRequestObject) (gen.TenantMemberListResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	members, err := t.config.V1.Tenant().ListTenantMembers(ctx.Request().Context(), tenantId)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.TenantMember, len(members))

	for i := range members {
		rows[i] = *transformers.ToTenantMember(members[i])
	}

	databaseMappings := []*sqlcv1.TenantOIDCGroupMapping(nil)
	if t.config.Auth.OIDCProvider != nil {
		issuer, err := authn.OIDCIssuer(t.config)
		if err != nil {
			return nil, err
		}
		databaseMappings, err = t.config.V1.Tenant().ListTenantOIDCGroupMappings(ctx.Request().Context(), tenantId, issuer)
		if err != nil {
			return nil, err
		}
	}
	oidcMappings := make([]gen.OIDCGroupMapping, 0, len(databaseMappings))
	for _, mapping := range databaseMappings {
		oidcMappings = append(oidcMappings, gen.OIDCGroupMapping{
			Id: mapping.ID, Group: mapping.Group, Role: gen.OIDCGroupMappingRole(mapping.Role),
		})
	}
	sort.Slice(oidcMappings, func(i, j int) bool {
		return oidcMappings[i].Group < oidcMappings[j].Group
	})

	return gen.TenantMemberList200JSONResponse{
		Rows: &rows, OidcGroupMappings: &oidcMappings,
	}, nil
}
