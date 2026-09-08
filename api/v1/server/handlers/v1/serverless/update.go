package serverlessv1

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/contract"
)

func (t *V1ServerlessService) V1ServerlessEndpointUpdate(ctx echo.Context, request gen.V1ServerlessEndpointUpdateRequestObject) (gen.V1ServerlessEndpointUpdateResponseObject, error) {
	endpoint := ctx.Get("v1-serverless-endpoint").(*sqlcv1.V1ServerlessEndpoint)
	body := request.Body

	if body.HealthcheckUrl != nil {
		if err := validateEndpointUrl("healthcheckUrl", *body.HealthcheckUrl); err != nil {
			return gen.V1ServerlessEndpointUpdate400JSONResponse(apierrors.NewAPIErrors(err.Error())), nil
		}
	}

	if body.TriggerUrl != nil {
		if err := validateEndpointUrl("triggerUrl", *body.TriggerUrl); err != nil {
			return gen.V1ServerlessEndpointUpdate400JSONResponse(apierrors.NewAPIErrors(err.Error())), nil
		}
	}

	labels, err := marshalLabels(body.Labels)

	if err != nil {
		return gen.V1ServerlessEndpointUpdate400JSONResponse(apierrors.NewAPIErrors(err.Error())), nil
	}

	opts := v1.UpdateServerlessEndpointOpts{
		Name:                  body.Name,
		HealthcheckUrl:        body.HealthcheckUrl,
		TriggerUrl:            body.TriggerUrl,
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

	// A new secret rotates the stored one; the operator picks it up on its next refresh of the
	// endpoint. Omitting it keeps the current secret.
	if body.SigningSecret != nil {
		if err := validateSigningSecret(*body.SigningSecret); err != nil {
			return gen.V1ServerlessEndpointUpdate400JSONResponse(apierrors.NewAPIErrors(err.Error())), nil
		}

		encryptedSecret, err := t.config.Encryption.EncryptString(*body.SigningSecret, contract.SigningSecretEncryptionDataID)

		if err != nil {
			return nil, fmt.Errorf("failed to encrypt signing secret: %w", err)
		}

		opts.SigningSecretEnc = &encryptedSecret
	}

	updated, err := t.config.V1.Serverless().Endpoints().Update(ctx.Request().Context(), endpoint.TenantID, endpoint.ID, opts)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return gen.V1ServerlessEndpointUpdate404JSONResponse(apierrors.NewAPIErrors("serverless endpoint not found")), nil
		}

		if msg, ok := badRequestMessage(err); ok {
			return gen.V1ServerlessEndpointUpdate400JSONResponse(apierrors.NewAPIErrors(msg)), nil
		}

		return nil, fmt.Errorf("failed to update serverless endpoint: %w", err)
	}

	return gen.V1ServerlessEndpointUpdate200JSONResponse(transformers.ToV1ServerlessEndpoint(updated)), nil
}
