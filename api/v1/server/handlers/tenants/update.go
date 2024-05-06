package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

func (t *TenantService) TenantUpdate(ctx echo.Context, request gen.TenantUpdateRequestObject) (gen.TenantUpdateResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	// validate the request
	if apiErrors, err := t.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.TenantUpdate400JSONResponse(*apiErrors), nil
	}

	// construct the database query
	createOpts := &repository.UpdateTenantOpts{}

	if request.Body.AnalyticsOptOut != nil {
		createOpts.AnalyticsOptOut = request.Body.AnalyticsOptOut
	}

	// update the tenant
	tenant, err := t.config.APIRepository.Tenant().UpdateTenant(tenant.ID, createOpts)

	if err != nil {
		return nil, err
	}

	return gen.TenantUpdate200JSONResponse(
		*transformers.ToTenant(tenant),
	), nil
}
