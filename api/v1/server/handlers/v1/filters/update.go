package filtersv1

import (
	"encoding/json"
	"fmt"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/labstack/echo/v4"
)

func (t *V1FiltersService) V1FilterUpdate(ctx echo.Context, request gen.V1FilterUpdateRequestObject) (gen.V1FilterUpdateResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	filter := ctx.Get("v1-filter").(*sqlcv1.V1Filter)

	var payload []byte
	if request.Body.Payload != nil {
		marshalledPayload, err := json.Marshal(request.Body.Payload)

		if err != nil {
			return gen.V1FilterUpdate400JSONResponse(apierrors.NewAPIErrors("failed to marshal payload to json")), nil
		}
		payload = marshalledPayload
	}

	filter, err := t.config.V1.Filters().UpdateFilter(
		ctx.Request().Context(),
		tenant.ID.String(),
		filter.ID.String(),
		v1.UpdateFilterOpts{
			Scope:      request.Body.Scope,
			Expression: request.Body.Expression,
			Payload:    payload,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update filter: %w", err)
	}

	transformed := transformers.ToV1Filter(filter)

	return gen.V1FilterUpdate200JSONResponse(transformed), nil
}
