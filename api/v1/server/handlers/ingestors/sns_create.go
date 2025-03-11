package ingestors

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (i *IngestorsService) SnsCreate(ctx echo.Context, req gen.SnsCreateRequestObject) (gen.SnsCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// validate the request
	if apiErrors, err := i.config.Validator.ValidateAPI(req.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.SnsCreate400JSONResponse(*apiErrors), nil
	}

	opts := &repository.CreateSNSIntegrationOpts{
		TopicArn: req.Body.TopicArn,
	}

	// create the SNS integration
	snsIntegration, err := i.config.APIRepository.SNS().CreateSNSIntegration(ctx.Request().Context(), tenantId, opts)

	if err != nil {
		return nil, err
	}

	resp := transformers.ToSNSIntegration(snsIntegration, i.config.Runtime.ServerURL)

	return gen.SnsCreate201JSONResponse(
		*resp,
	), nil
}
