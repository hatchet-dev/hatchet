package tenants

import (
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TenantService) TenantGetPrometheusMetrics(ctx echo.Context, request gen.TenantGetPrometheusMetricsRequestObject) (gen.TenantGetPrometheusMetricsResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID.String()

	var response string

	// connect to the prometheus server
	if t.config.Prometheus.PrometheusServerURL != "" {
		endpoint := fmt.Sprintf("%s/federate?match[]={tenant_id=\"%s\"}", t.config.Prometheus.PrometheusServerURL, tenantId)

		req, err := http.NewRequestWithContext(ctx.Request().Context(), "GET", endpoint, nil)
		if err != nil {
			return nil, err
		}

		if t.config.Prometheus.PrometheusServerUsername != "" && t.config.Prometheus.PrometheusServerPassword != "" {
			req.SetBasicAuth(t.config.Prometheus.PrometheusServerUsername, t.config.Prometheus.PrometheusServerPassword)
		}

		federatedMetrics, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer federatedMetrics.Body.Close()

		body, err := io.ReadAll(federatedMetrics.Body)
		if err != nil {
			return nil, err
		}

		response = string(body)

		// even if there are no metrics, we still want to return a 200 response
		// and an empty body is what Prometheus metrics expect so we follow convention
	} else {
		return gen.TenantGetPrometheusMetrics400JSONResponse(
			apierrors.NewAPIErrors("Prometheus metrics are not enabled for this tenant."),
		), nil
	}

	// Return the metrics as text response
	return gen.TenantGetPrometheusMetrics200TextResponse(response), nil
}
