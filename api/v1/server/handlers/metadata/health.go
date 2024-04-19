package metadata

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (u *MetadataService) LivenessGet(ctx echo.Context, request gen.LivenessGetRequestObject) (gen.LivenessGetResponseObject, error) {
	if !u.config.APIRepository.Health().IsHealthy() {
		return nil, fmt.Errorf("api repository is not healthy")
	}

	if !u.config.EngineRepository.Health().IsHealthy() {
		return nil, fmt.Errorf("engine repository is not healthy")
	}

	if !u.config.MessageQueue.IsReady() {
		return nil, fmt.Errorf("task queue is not healthy")
	}

	return gen.LivenessGet200Response{}, nil
}

func (u *MetadataService) ReadinessGet(ctx echo.Context, request gen.ReadinessGetRequestObject) (gen.ReadinessGetResponseObject, error) {
	if !u.config.APIRepository.Health().IsHealthy() {
		return nil, fmt.Errorf("api repository is not healthy")
	}

	if !u.config.EngineRepository.Health().IsHealthy() {
		return nil, fmt.Errorf("engine repository is not healthy")
	}

	if !u.config.MessageQueue.IsReady() {
		return nil, fmt.Errorf("task queue is not healthy")
	}

	return gen.ReadinessGet200Response{}, nil
}
