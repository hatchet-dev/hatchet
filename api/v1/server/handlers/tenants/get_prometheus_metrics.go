package tenants

import (
	"bytes"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *TenantService) TenantGetPrometheusMetrics(ctx echo.Context, request gen.TenantGetPrometheusMetricsRequestObject) (gen.TenantGetPrometheusMetricsResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// Create Prometheus API client to query external Prometheus endpoint
	client, err := api.NewClient(api.Config{
		Address: "http://localhost:9090", // Replace with your Prometheus server URL
	})
	if err != nil {
		return nil, err
	}

	// Create API client for querying
	promAPI := v1.NewAPI(client)

	// Query metrics filtered by tenant_id
	query := fmt.Sprintf(`{tenant_id="%s"}`, tenantId)
	result, warnings, err := promAPI.Query(ctx.Request().Context(), query, time.Now())
	if err != nil {
		return nil, err
	}

	// Log any warnings
	if len(warnings) > 0 {
		// Handle warnings if needed
	}

	// Convert result to Prometheus text format
	var buf bytes.Buffer

	// Convert model.Value to metric families
	switch result.Type() {
	case model.ValVector:
		vector := result.(model.Vector)
		metricFamilies := convertVectorToMetricFamilies(vector)

		encoder := expfmt.NewEncoder(&buf, expfmt.FmtText)
		for _, mf := range metricFamilies {
			if err := encoder.Encode(mf); err != nil {
				return nil, err
			}
		}
	default:
		// For other types, fall back to string representation
		buf.WriteString(result.String())
	}

	return gen.TenantGetPrometheusMetrics200TextResponse(buf.String()), nil
}

func convertVectorToMetricFamilies(vector model.Vector) []*dto.MetricFamily {
	families := make(map[string]*dto.MetricFamily)

	for _, sample := range vector {
		metricName := string(sample.Metric[model.MetricNameLabel])

		// Create metric family if it doesn't exist
		if _, exists := families[metricName]; !exists {
			// Generate HELP text based on metric name
			helpText := generateHelpText(metricName)
			families[metricName] = &dto.MetricFamily{
				Name: &metricName,
				Help: &helpText,
				Type: dto.MetricType_COUNTER.Enum(), // Use counter for most Hatchet metrics
			}
		}

		// Convert labels, filtering out instance and job
		labels := make([]*dto.LabelPair, 0, len(sample.Metric))
		for name, value := range sample.Metric {
			labelName := string(name)
			if name != model.MetricNameLabel && labelName != "instance" && labelName != "job" {
				labelValue := string(value)
				labels = append(labels, &dto.LabelPair{
					Name:  &labelName,
					Value: &labelValue,
				})
			}
		}

		// Create metric (using counter value instead of gauge)
		counterValue := float64(sample.Value)
		metric := &dto.Metric{
			Label: labels,
			Counter: &dto.Counter{
				Value: &counterValue,
			},
		}

		families[metricName].Metric = append(families[metricName].Metric, metric)
	}

	// Convert map to slice
	result := make([]*dto.MetricFamily, 0, len(families))
	for _, family := range families {
		result = append(result, family)
	}

	return result
}

func generateHelpText(metricName string) string {
	// Map common metric names to their help text
	helpTexts := map[string]string{
		"tenant_finished_workflows_total": "The total number of finished workflow runs",
	}

	if help, exists := helpTexts[metricName]; exists {
		return help
	}

	// Default help text for unknown metrics
	return fmt.Sprintf("Metric %s", metricName)
}
