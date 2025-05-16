package filtersv1

import (
	"encoding/json"
	"fmt"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/labstack/echo/v4"
)

func (t *V1FiltersService) V1FilterCreate(ctx echo.Context, request gen.V1FilterCreateRequestObject) (gen.V1FilterCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)

	var payload []byte
	if request.Body.Payload != nil {
		marshalledPayload, err := json.Marshal(request.Body.Payload)

		if err != nil {
			return gen.V1FilterCreate400JSONResponse(apierrors.NewAPIErrors("failed to marshal payload to json")), nil
		}
		payload = marshalledPayload
	}

	params := v1.CreateFilterOpts{
		Workflowid: sqlchelpers.UUIDFromStr(request.Body.WorkflowId.String()),
		Scope:      request.Body.Scope,
		Expression: request.Body.Expression,
		Payload:    payload,
	}

	filter, err := t.config.V1.Filters().CreateFilter(
		ctx.Request().Context(),
		tenant.ID.String(),
		params,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create filter")
	}

	transformed := transformers.ToV1Filter(filter)

	return gen.V1FilterCreate200JSONResponse(transformed), nil
}
