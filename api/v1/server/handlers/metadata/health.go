package metadata

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (u *MetadataService) LivenessGet(ctx echo.Context, request gen.LivenessGetRequestObject) (gen.LivenessGetResponseObject, error) {
	return gen.LivenessGet200Response{}, nil
}

func (u *MetadataService) ReadinessGet(ctx echo.Context, request gen.ReadinessGetRequestObject) (gen.ReadinessGetResponseObject, error) {
	if !u.config.Repository.Health().IsHealthy() {
		return nil, fmt.Errorf("repository is not healthy")
	}

	if !u.config.MessageQueue.IsReady() {
		return nil, fmt.Errorf("task queue is not healthy")
	}

	return gen.ReadinessGet200Response{}, nil
}
