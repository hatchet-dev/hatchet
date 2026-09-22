package tenants

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const metricNameLabel = "__tenant_metric_name"

var labelValueEscaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)

type promQueryResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
	Data   struct {
		Result []struct {
			Metric map[string]string  `json:"metric"`
			Value  [2]json.RawMessage `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

func (t *TenantService) TenantGetPrometheusMetrics(ctx echo.Context, request gen.TenantGetPrometheusMetricsRequestObject) (gen.TenantGetPrometheusMetricsResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	if t.config.Prometheus.TenantScoped {
		enabled, err := t.config.V1.TenantEntitlement().IsPrometheusMetricsEnabled(ctx.Request().Context(), tenantId)
		if err != nil {
			return nil, err
		}

		if !enabled {
			return gen.TenantGetPrometheusMetrics403JSONResponse(
				apierrors.NewAPIErrors("Prometheus metrics are not enabled for this tenant."),
			), nil
		}
	}

	if t.config.Prometheus.PrometheusServerURL == "" {
		return gen.TenantGetPrometheusMetrics400JSONResponse(
			apierrors.NewAPIErrors("Prometheus metrics are not enabled for this tenant."),
		), nil
	}

	query := fmt.Sprintf(`sum without (pod) (label_replace({tenant_id=%q}, %q, "$1", "__name__", "(.*)"))`, tenantId.String(), metricNameLabel)
	endpoint := fmt.Sprintf("%s/api/v1/query?%s", t.config.Prometheus.PrometheusServerURL, url.Values{"query": {query}}.Encode())

	req, err := http.NewRequestWithContext(ctx.Request().Context(), "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	if t.config.Prometheus.PrometheusServerUsername != "" && t.config.Prometheus.PrometheusServerPassword != "" {
		req.SetBasicAuth(t.config.Prometheus.PrometheusServerUsername, t.config.Prometheus.PrometheusServerPassword)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var parsed promQueryResponse

	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("could not decode prometheus response: %w", err)
	}

	if parsed.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed: %s", parsed.Error)
	}

	lines := make([]string, 0, len(parsed.Data.Result))

	for _, r := range parsed.Data.Result {
		name := r.Metric[metricNameLabel]

		if name == "" {
			continue
		}

		keys := make([]string, 0, len(r.Metric))

		for k := range r.Metric {
			if k != metricNameLabel {
				keys = append(keys, k)
			}
		}

		sort.Strings(keys)

		var b strings.Builder
		b.WriteString(name)
		b.WriteByte('{')

		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}

			fmt.Fprintf(&b, `%s="%s"`, k, labelValueEscaper.Replace(r.Metric[k]))
		}

		var ts float64
		var value string

		if err := json.Unmarshal(r.Value[0], &ts); err != nil {
			return nil, fmt.Errorf("could not decode sample timestamp: %w", err)
		}

		if err := json.Unmarshal(r.Value[1], &value); err != nil {
			return nil, fmt.Errorf("could not decode sample value: %w", err)
		}

		fmt.Fprintf(&b, "} %s %d", value, int64(ts*1000))
		lines = append(lines, b.String())
	}

	sort.Strings(lines)

	var out strings.Builder
	lastName := ""

	for _, line := range lines {
		name := line[:strings.IndexByte(line, '{')]

		if name != lastName {
			fmt.Fprintf(&out, "# TYPE %s untyped\n", name)
			lastName = name
		}

		out.WriteString(line)
		out.WriteByte('\n')
	}

	return gen.TenantGetPrometheusMetrics200TextResponse(out.String()), nil
}
