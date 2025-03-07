package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *TenantService) AlertEmailGroupCreate(ctx echo.Context, request gen.AlertEmailGroupCreateRequestObject) (gen.AlertEmailGroupCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// validate the request
	if apiErrors, err := t.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.AlertEmailGroupCreate400JSONResponse(*apiErrors), nil
	}

	// construct the database query
	createOpts := &repository.CreateTenantAlertGroupOpts{
		Emails: request.Body.Emails,
	}

	emailGroup, err := t.config.APIRepository.TenantAlertingSettings().CreateTenantAlertGroup(ctx.Request().Context(), tenantId, createOpts)

	if err != nil {
		return nil, err
	}

	return gen.AlertEmailGroupCreate201JSONResponse(
		*transformers.ToTenantAlertEmailGroup(emailGroup),
	), nil
}
