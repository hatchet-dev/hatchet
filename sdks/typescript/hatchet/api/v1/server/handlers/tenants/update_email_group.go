package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (t *TenantService) AlertEmailGroupUpdate(ctx echo.Context, request gen.AlertEmailGroupUpdateRequestObject) (gen.AlertEmailGroupUpdateResponseObject, error) {
	emailGroup := ctx.Get("alert-email-group").(*db.TenantAlertEmailGroupModel)

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

	emailGroup, err := t.config.APIRepository.TenantAlertingSettings().UpdateTenantAlertGroup(emailGroup.ID, updateOpts)

	if err != nil {
		return nil, err
	}

	return gen.AlertEmailGroupUpdate200JSONResponse(
		*transformers.ToTenantAlertEmailGroup(emailGroup),
	), nil
}
