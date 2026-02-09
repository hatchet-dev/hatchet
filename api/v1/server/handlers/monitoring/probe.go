package monitoring

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (m *MonitoringService) MonitoringPostRunProbe(ctx echo.Context, request gen.MonitoringPostRunProbeRequestObject) (gen.MonitoringPostRunProbeResponseObject, error) {
	if !m.enabled {
		m.l.Error().Msg("monitoring is not enabled")
		return gen.MonitoringPostRunProbe403JSONResponse{}, nil
	}

	return gen.MonitoringPostRunProbe403JSONResponse{}, nil
}
