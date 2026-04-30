package tenants

import (
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *TenantService) TenantGetTaskStats(ctx echo.Context, request gen.TenantGetTaskStatsRequestObject) (gen.TenantGetTaskStatsResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	stats, err := t.config.V1.Tasks().GetTaskStats(ctx.Request().Context(), tenant.ID)
	if err != nil {
		return nil, err
	}

	requiredNames := normalizeTaskNames(request.Params.TaskNames)
	transformedStats := transformers.ToTaskStats(stats, requiredNames)

	return gen.TenantGetTaskStats200JSONResponse(transformedStats), nil
}

func normalizeTaskNames(raw *[]string) []string {
	if raw == nil {
		return nil
	}
	seen := make(map[string]struct{}, len(*raw))
	out := make([]string, 0, len(*raw))
	for _, entry := range *raw {
		for _, name := range strings.Split(entry, ",") {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, name)
		}
	}
	return out
}
