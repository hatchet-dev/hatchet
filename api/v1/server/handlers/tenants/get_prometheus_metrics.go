package tenants

import (
	"bytes"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/common/expfmt"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	hatchetprom "github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *TenantService) TenantGetPrometheusMetrics(ctx echo.Context, request gen.TenantGetPrometheusMetricsRequestObject) (gen.TenantGetPrometheusMetricsResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	tenantMetrics := hatchetprom.WithTenant(tenantId)

	// Gather metrics
	metricFamilies, err := tenantMetrics.Registry.Gather()
	if err != nil {
		return nil, err
	}

	// Convert to text format
	var buf bytes.Buffer
	encoder := expfmt.NewEncoder(&buf, expfmt.NewFormat(expfmt.TypeTextPlain))
	for _, mf := range metricFamilies {
		if err := encoder.Encode(mf); err != nil {
			return nil, err
		}
	}

	return gen.TenantGetPrometheusMetrics200TextResponse(buf.String()), nil
}
