package metadata

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (u *MetadataService) LivenessGet(ctx echo.Context, request gen.LivenessGetRequestObject) (gen.LivenessGetResponseObject, error) {
	return gen.LivenessGet200Response{}, nil
}

func (u *MetadataService) ReadinessGet(ctx echo.Context, request gen.ReadinessGetRequestObject) (gen.ReadinessGetResponseObject, error) {
	// TODO check if db and queue are ready
	return gen.ReadinessGet200Response{}, nil
}
