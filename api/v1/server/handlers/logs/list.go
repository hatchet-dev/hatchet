package logs

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *LogService) LogLineList(ctx echo.Context, request gen.LogLineListRequestObject) (gen.LogLineListResponseObject, error) {
	return gen.LogLineList400JSONResponse(apierrors.NewAPIErrors(
		"LogLineList is deprecated",
	)), nil
}
