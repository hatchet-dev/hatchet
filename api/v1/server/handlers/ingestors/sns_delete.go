package ingestors

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (i *IngestorsService) SnsDelete(ctx echo.Context, req gen.SnsDeleteRequestObject) (gen.SnsDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)
	sns := ctx.Get("sns").(*db.SNSIntegrationModel)

	// create the SNS integration
	err := i.config.APIRepository.SNS().DeleteSNSIntegration(tenant.ID, sns.ID)

	if err != nil {
		return nil, err
	}

	return gen.SnsDelete204Response{}, nil
}
