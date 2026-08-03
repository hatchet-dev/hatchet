package metadata

import (
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (u *MetadataService) FeedbackCreate(ctx echo.Context, request gen.FeedbackCreateRequestObject) (gen.FeedbackCreateResponseObject, error) {
	if request.Body == nil || strings.TrimSpace(request.Body.Message) == "" {
		return gen.FeedbackCreate400JSONResponse(apierrors.NewAPIErrors("feedback message is required")), nil
	}

	if !u.config.UsageTelemetry.Active() {
		return gen.FeedbackCreate400JSONResponse(apierrors.NewAPIErrors("feedback is not available on this instance")), nil
	}

	email := ""
	if user, ok := ctx.Get("user").(*sqlcv1.User); ok && user != nil {
		email = user.Email
	}

	if err := u.config.UsageTelemetry.SendFeedback(ctx.Request().Context(), request.Body.Message, email); err != nil {
		return nil, err
	}

	return gen.FeedbackCreate200Response{}, nil
}
