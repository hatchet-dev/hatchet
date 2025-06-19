package tenants

import (
	"io"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *TenantService) TenantGetPrometheusMetrics(ctx echo.Context, request gen.TenantGetPrometheusMetricsRequestObject) (gen.TenantGetPrometheusMetricsResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// Build URL to engine's tenant-specific metrics endpoint
	prometheusURL := "http://" + t.config.Prometheus.Address + t.config.Prometheus.Path + "/" + tenantId

	// Make HTTP GET request to the engine's metrics endpoint
	resp, err := http.Get(prometheusURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Return the metrics as text response
	return gen.TenantGetPrometheusMetrics200TextResponse(string(body)), nil
}
