package serverlessv1

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/contract"
)

func (t *V1ServerlessService) V1ServerlessEndpointCreate(ctx echo.Context, request gen.V1ServerlessEndpointCreateRequestObject) (gen.V1ServerlessEndpointCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	body := request.Body

	if err := validateEndpointUrl("healthcheckUrl", body.HealthcheckUrl); err != nil {
		return gen.V1ServerlessEndpointCreate400JSONResponse(apierrors.NewAPIErrors(err.Error())), nil
	}

	if err := validateEndpointUrl("triggerUrl", body.TriggerUrl); err != nil {
		return gen.V1ServerlessEndpointCreate400JSONResponse(apierrors.NewAPIErrors(err.Error())), nil
	}

	if err := validateSigningSecret(body.SigningSecret); err != nil {
		return gen.V1ServerlessEndpointCreate400JSONResponse(apierrors.NewAPIErrors(err.Error())), nil
	}

	labels, err := marshalLabels(body.Labels)

	if err != nil {
		return gen.V1ServerlessEndpointCreate400JSONResponse(apierrors.NewAPIErrors(err.Error())), nil
	}

	// The secret is stored encrypted at rest; the operator decrypts it when it loads the
	// endpoint and holds it in memory only.
	encryptedSecret, err := t.config.Encryption.EncryptString(body.SigningSecret, contract.SigningSecretEncryptionDataID)

	if err != nil {
		return nil, fmt.Errorf("failed to encrypt signing secret: %w", err)
	}

	opts := v1.CreateServerlessEndpointOpts{
		Name:                  body.Name,
		HealthcheckUrl:        body.HealthcheckUrl,
		TriggerUrl:            body.TriggerUrl,
		SigningSecretEnc:      encryptedSecret,
		RequestTimeoutSeconds: body.RequestTimeoutSeconds,
		PollIntervalSeconds:   body.PollIntervalSeconds,
		InlineWaitBudgetMs:    body.InlineWaitBudgetMs,
		Labels:                labels,
		Enabled:               body.Enabled,
	}

	if body.Kind != nil {
		kind := sqlcv1.V1ServerlessEndpointKind(*body.Kind)
		opts.Kind = &kind
	}

	endpoint, err := t.config.V1.Serverless().Endpoints().Create(ctx.Request().Context(), tenant.ID, opts)

	if err != nil {
		if msg, ok := badRequestMessage(err); ok {
			return gen.V1ServerlessEndpointCreate400JSONResponse(apierrors.NewAPIErrors(msg)), nil
		}

		return nil, fmt.Errorf("failed to create serverless endpoint: %w", err)
	}

	return gen.V1ServerlessEndpointCreate200JSONResponse(transformers.ToV1ServerlessEndpoint(endpoint)), nil
}
