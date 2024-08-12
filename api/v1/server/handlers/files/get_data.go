package files

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

func (t *FileService) FileDataGet(ctx echo.Context, request gen.FileDataGetRequestObject) (gen.FileDataGetResponseObject, error) {
	file := ctx.Get("file").(*dbsqlc.File)

	fileContent, err := t.blob_storage.GetObject(ctx.Request().Context(), file.FileName)

	if err != nil {
		return nil, fmt.Errorf("unable to get file: %v", err)
	}

	dataStr := string(fileContent)

	return gen.FileDataGet200JSONResponse(
		gen.FileData{
			Data: dataStr,
		},
	), nil
}
