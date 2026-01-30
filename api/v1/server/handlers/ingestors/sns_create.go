package ingestors

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (i *IngestorsService) SnsCreate(ctx echo.Context, req gen.SnsCreateRequestObject) (gen.SnsCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID.String()

	// validate the request
	if apiErrors, err := i.config.Validator.ValidateAPI(req.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.SnsCreate400JSONResponse(*apiErrors), nil
	}

	opts := &v1.CreateSNSIntegrationOpts{
		TopicArn: req.Body.TopicArn,
	}

	// create the SNS integration
	snsIntegration, err := i.config.V1.SNS().CreateSNSIntegration(ctx.Request().Context(), tenantId, opts)

	if err != nil {
		return nil, err
	}

	resp := transformers.ToSNSIntegration(snsIntegration, i.config.Runtime.ServerURL)

	return gen.SnsCreate201JSONResponse(
		*resp,
	), nil
}
