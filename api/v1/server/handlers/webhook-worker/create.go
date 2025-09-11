package webhookworker

import (
	"errors"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/random"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (i *WebhookWorkersService) WebhookCreate(ctx echo.Context, request gen.WebhookCreateRequestObject) (gen.WebhookCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	var secret string
	if request.Body.Secret == nil {
		s, err := random.GenerateWebhookSecret()
		if err != nil {
			return nil, err
		}
		secret = s
	} else {
		secret = *request.Body.Secret
	}

	encSecret, err := i.config.Encryption.EncryptString(secret, tenantId)
	if err != nil {
		return nil, err
	}

	ww, err := i.config.EngineRepository.WebhookWorker().CreateWebhookWorker(ctx.Request().Context(), &repository.CreateWebhookWorkerOpts{
		TenantId: tenantId,
		Name:     request.Body.Name,
		URL:      request.Body.Url,
		Secret:   encSecret,
		Deleted:  repository.BoolPtr(false),
	})

	if errors.Is(err, repository.ErrDuplicateKey) {
		return gen.WebhookCreate400JSONResponse(
			apierrors.NewAPIErrors("A webhook with the same url already exists, please delete it and try again.", "url"),
		), nil
	}

	if err != nil {
		return nil, err
	}

	ww.Secret = secret

	return gen.WebhookCreate200JSONResponse(*transformers.ToWebhookWorkerCreated(ww)), nil
}
