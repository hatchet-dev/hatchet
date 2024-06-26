package metadata

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (u *MetadataService) CloudMetadataGet(ctx echo.Context, request gen.CloudMetadataGetRequestObject) (gen.CloudMetadataGetResponseObject, error) {
	return gen.CloudMetadataGet200JSONResponse(
		apierrors.NewAPIErrors("oss"),
	), nil
}
