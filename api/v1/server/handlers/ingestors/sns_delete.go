package ingestors

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (i *IngestorsService) SnsDelete(ctx echo.Context, req gen.SnsDeleteRequestObject) (gen.SnsDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID
	sns := ctx.Get("sns").(*sqlcv1.SNSIntegration)

	// create the SNS integration
	err := i.config.V1.SNS().DeleteSNSIntegration(ctx.Request().Context(), tenantId, sns.ID.String())

	if err != nil {
		return nil, err
	}

	return gen.SnsDelete204Response{}, nil
}
