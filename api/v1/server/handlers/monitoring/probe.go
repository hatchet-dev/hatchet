package monitoring

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (m *MonitoringService) MonitoringPostRunProbe(c echo.Context, request gen.MonitoringPostRunProbeRequestObject) (gen.MonitoringPostRunProbeResponseObject, error) {
	ctx := c.Request().Context()

	if !m.enabled {
		m.l.Error().Ctx(ctx).Msg("monitoring is not enabled")
		return gen.MonitoringPostRunProbe403JSONResponse{}, nil
	}

	return gen.MonitoringPostRunProbe403JSONResponse{}, nil
}
