package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *TenantService) TenantUpdate(ctx echo.Context, request gen.TenantUpdateRequestObject) (gen.TenantUpdateResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// validate the request
	if apiErrors, err := t.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.TenantUpdate400JSONResponse(*apiErrors), nil
	}

	// construct the database query
	updateOpts := &repository.UpdateTenantOpts{}

	if request.Body.AnalyticsOptOut != nil {
		updateOpts.AnalyticsOptOut = request.Body.AnalyticsOptOut
	}

	if request.Body.AlertMemberEmails != nil {
		updateOpts.AlertMemberEmails = request.Body.AlertMemberEmails
	}

	if request.Body.Name != nil {
		updateOpts.Name = request.Body.Name
	}

	// update the tenant
	tenant, err := t.config.APIRepository.Tenant().UpdateTenant(ctx.Request().Context(), tenantId, updateOpts)

	if err != nil {
		return nil, err
	}

	if request.Body.MaxAlertingFrequency != nil ||
		request.Body.EnableExpiringTokenAlerts != nil ||
		request.Body.EnableTenantResourceLimitAlerts != nil ||
		request.Body.EnableWorkflowRunFailureAlerts != nil {

		_, err = t.config.APIRepository.TenantAlertingSettings().UpsertTenantAlertingSettings(
			ctx.Request().Context(),
			tenantId,
			&repository.UpsertTenantAlertingSettingsOpts{
				MaxFrequency:                    request.Body.MaxAlertingFrequency,
				EnableExpiringTokenAlerts:       request.Body.EnableExpiringTokenAlerts,
				EnableWorkflowRunFailureAlerts:  request.Body.EnableWorkflowRunFailureAlerts,
				EnableTenantResourceLimitAlerts: request.Body.EnableTenantResourceLimitAlerts,
			},
		)

		if err != nil {
			return nil, err
		}
	}

	return gen.TenantUpdate200JSONResponse(
		*transformers.ToTenant(tenant),
	), nil
}
