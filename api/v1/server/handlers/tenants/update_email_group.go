package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *TenantService) AlertEmailGroupUpdate(ctx echo.Context, request gen.AlertEmailGroupUpdateRequestObject) (gen.AlertEmailGroupUpdateResponseObject, error) {
	populator := populator.FromContext(ctx)

	emailGroup, err := populator.GetAlertEmailGroup()
	if err != nil {
		return nil, err
	}

	// validate the request
	if apiErrors, err := t.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.AlertEmailGroupUpdate400JSONResponse(*apiErrors), nil
	}

	// construct the database query
	updateOpts := &repository.UpdateTenantAlertGroupOpts{
		Emails: request.Body.Emails,
	}

	emailGroup, err = t.config.APIRepository.TenantAlertingSettings().UpdateTenantAlertGroup(ctx.Request().Context(), sqlchelpers.UUIDToStr(emailGroup.ID), updateOpts)

	if err != nil {
		return nil, err
	}

	return gen.AlertEmailGroupUpdate200JSONResponse(
		*transformers.ToTenantAlertEmailGroup(emailGroup),
	), nil
}
