package info

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (i *InfoService) InfoGetVersion(ctx echo.Context, req gen.InfoGetVersionRequestObject) (gen.InfoGetVersionResponseObject, error) {
	return gen.InfoGetVersion200JSONResponse{
		Version: i.config.Version,
	}, nil
}
