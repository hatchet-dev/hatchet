package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TenantService) AlertEmailGroupUpdate(ctx echo.Context, request gen.AlertEmailGroupUpdateRequestObject) (gen.AlertEmailGroupUpdateResponseObject, error) {
	emailGroup := ctx.Get("alert-email-group").(*sqlcv1.TenantAlertEmailGroup)

	// validate the request
	if apiErrors, err := t.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.AlertEmailGroupUpdate400JSONResponse(*apiErrors), nil
	}

	// construct the database query
	updateOpts := &v1.UpdateTenantAlertGroupOpts{
		Emails: request.Body.Emails,
	}

	emailGroup, err := t.config.V1.TenantAlertingSettings().UpdateTenantAlertGroup(ctx.Request().Context(), emailGroup.ID.String(), updateOpts)

	if err != nil {
		return nil, err
	}

	return gen.AlertEmailGroupUpdate200JSONResponse(
		*transformers.ToTenantAlertEmailGroup(emailGroup),
	), nil
}
