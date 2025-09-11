package ingestors

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (i *IngestorsService) SnsDelete(ctx echo.Context, req gen.SnsDeleteRequestObject) (gen.SnsDeleteResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	sns := ctx.Get("sns").(*dbsqlc.SNSIntegration)

	// create the SNS integration
	err := i.config.APIRepository.SNS().DeleteSNSIntegration(ctx.Request().Context(), tenantId, sqlchelpers.UUIDToStr(sns.ID))

	if err != nil {
		return nil, err
	}

	return gen.SnsDelete204Response{}, nil
}
