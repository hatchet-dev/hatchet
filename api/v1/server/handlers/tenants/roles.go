package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/enumerables"
	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
)

func (t *TenantService) TenantRoleList(ctx echo.Context, request gen.TenantRoleListRequestObject) (gen.TenantRoleListResponseObject, error) {
	perms, _, err := t.permissions()
	if err != nil {
		return nil, err
	}

	rows := enumerables.Map(perms.AssignableRoles(), func(r pkgauthz.AssignableRole) gen.TenantRole {
		return gen.TenantRole{
			Name:        r.Name,
			Label:       r.Label,
			Description: r.Description,
			Operations:  r.Operations,
		}
	})

	return gen.TenantRoleList200JSONResponse(gen.TenantRoleList{Rows: rows}), nil
}
