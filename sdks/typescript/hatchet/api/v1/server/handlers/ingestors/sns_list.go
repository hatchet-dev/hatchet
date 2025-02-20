package ingestors

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (i *IngestorsService) SnsList(ctx echo.Context, req gen.SnsListRequestObject) (gen.SnsListResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	// create the SNS integration
	snsIntegrations, err := i.config.APIRepository.SNS().ListSNSIntegrations(tenant.ID)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.SNSIntegration, len(snsIntegrations))

	serverUrl := i.config.Runtime.ServerURL

	for i := range snsIntegrations {
		rows[i] = *transformers.ToSNSIntegration(&snsIntegrations[i], serverUrl)
	}

	return gen.SnsList200JSONResponse(
		gen.ListSNSIntegrations{
			Rows: rows,
		},
	), nil
}
