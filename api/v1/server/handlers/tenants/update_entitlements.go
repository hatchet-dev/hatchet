package tenants

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// TenantEntitlementsUpdate sets the tenant's Labs preview opt-ins. Omitted fields keep their current value.
func (t *TenantService) TenantEntitlementsUpdate(ctx echo.Context, request gen.TenantEntitlementsUpdateRequestObject) (gen.TenantEntitlementsUpdateResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	if request.Body == nil {
		return gen.TenantEntitlementsUpdate400JSONResponse(gen.APIErrors{
			Errors: []gen.APIError{{Description: "Request body is required."}},
		}), nil
	}

	if apiErrors, err := t.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.TenantEntitlementsUpdate400JSONResponse(*apiErrors), nil
	}

	updated, err := t.config.V1.TenantEntitlement().UpdateLabsEntitlements(ctx.Request().Context(), tenant.ID, v1.LabsEntitlements{
		DurableStreams:     request.Body.DurableStreams,
		ServerlessOperator: request.Body.ServerlessOperator,
		Sandboxes:          request.Body.Sandboxes,
	})

	if err != nil {
		return nil, err
	}

	return gen.TenantEntitlementsUpdate200JSONResponse(toAPITenantEntitlements(updated)), nil
}
