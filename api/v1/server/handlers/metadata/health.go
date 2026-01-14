package metadata

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (u *MetadataService) collectHealthErrors(ctx context.Context) []error {
	errs := []error{}

	if !u.config.V1.Health().IsHealthy(ctx) {
		errs = append(errs, errors.New("api repository is not healthy"))
	}

	if !u.config.V1.Health().IsHealthy(ctx) {
		errs = append(errs, errors.New("engine repository is not healthy"))
	}

	if !u.config.MessageQueueV1.IsReady() {
		errs = append(errs, errors.New("task queue is not healthy"))
	}

	return errs
}

func (u *MetadataService) logHealthErrors(errs []error) {
	for _, err := range errs {
		u.config.Logger.Err(err).Msg("health check failed")
	}
}

func (u *MetadataService) LivenessGet(ctx echo.Context, request gen.LivenessGetRequestObject) (gen.LivenessGetResponseObject, error) {
	gCtx, cancel := context.WithTimeout(ctx.Request().Context(), 5*time.Second)
	defer cancel()

	allErrors := u.collectHealthErrors(gCtx)

	if len(allErrors) > 0 {
		u.logHealthErrors(allErrors)

		allErrors = append(allErrors, fmt.Errorf(
			"pg connections - acquired: %d, idle: %d, total: %d",
			u.config.V1.Health().PgStat().AcquiredConns(),
			u.config.V1.Health().PgStat().IdleConns(),
			u.config.V1.Health().PgStat().TotalConns(),
		))

		return gen.LivenessGet500JSONResponse(gen.APIErrors{Errors: errorsToAPIErrors(allErrors)}), nil
	}

	return gen.LivenessGet200Response{}, nil
}

func (u *MetadataService) ReadinessGet(ctx echo.Context, request gen.ReadinessGetRequestObject) (gen.ReadinessGetResponseObject, error) {
	gCtx, cancel := context.WithTimeout(ctx.Request().Context(), 5*time.Second)
	defer cancel()

	allErrors := u.collectHealthErrors(gCtx)

	if len(allErrors) > 0 {
		u.logHealthErrors(allErrors)

		allErrors = append(allErrors, fmt.Errorf(
			"pg connections - acquired: %d, idle: %d, total: %d",
			u.config.V1.Health().PgStat().AcquiredConns(),
			u.config.V1.Health().PgStat().IdleConns(),
			u.config.V1.Health().PgStat().TotalConns(),
		))

		return gen.ReadinessGet500JSONResponse(gen.APIErrors{Errors: errorsToAPIErrors(allErrors)}), nil
	}

	return gen.ReadinessGet200Response{}, nil
}

func errorsToAPIErrors(errors []error) []gen.APIError {
	apiErrors := make([]gen.APIError, len(errors))

	for i, err := range errors {
		apiErrors[i] = gen.APIError{
			Description: err.Error(),
		}
	}

	return apiErrors
}
