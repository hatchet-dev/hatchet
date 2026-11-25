package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *TenantService) TenantRoleList(ctx echo.Context, request gen.TenantRoleListRequestObject) (gen.TenantRoleListResponseObject, error) {
	perms, _, err := t.permissions()
	if err != nil {
		return nil, err
	}

	roles := perms.AssignableRoles()
	rows := make([]gen.TenantRole, 0, len(roles))

	for _, r := range roles {
		rows = append(rows, gen.TenantRole{
			Name:        r.Name,
			Label:       r.Label,
			Description: r.Description,
			Operations:  r.Operations,
		})
	}

	return gen.TenantRoleList200JSONResponse(gen.TenantRoleList{Rows: rows}), nil
}
