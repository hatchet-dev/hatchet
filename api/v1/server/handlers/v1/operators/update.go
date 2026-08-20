package operatorsv1

import (
	"encoding/json"
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator/httpoperator"
	"github.com/hatchet-dev/hatchet/pkg/operator/httpoperator/safeclient"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *V1OperatorsService) V1HttpOperatorUpdate(ctx echo.Context, request gen.V1HttpOperatorUpdateRequestObject) (gen.V1HttpOperatorUpdateResponseObject, error) {
	operator := ctx.Get("v1-http-operator").(*sqlcv1.V1Operator)

	// Merge the requested changes onto the existing config so omitted fields are preserved.
	var config httpoperator.HTTPOperatorConfig

	if err := json.Unmarshal(operator.Config, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal existing operator config: %w", err)
	}

	if request.Body.TriggerEndpoint != nil {
		if err := safeclient.ValidateEndpoint(*request.Body.TriggerEndpoint); err != nil {
			return gen.V1HttpOperatorUpdate400JSONResponse(apierrors.NewAPIErrors(fmt.Sprintf("invalid trigger endpoint: %s", err))), nil
		}

		config.TriggerEndpoint = *request.Body.TriggerEndpoint
	}

	if request.Body.HealthcheckEndpoint != nil {
		if *request.Body.HealthcheckEndpoint != "" {
			if err := safeclient.ValidateEndpoint(*request.Body.HealthcheckEndpoint); err != nil {
				return gen.V1HttpOperatorUpdate400JSONResponse(apierrors.NewAPIErrors(fmt.Sprintf("invalid healthcheck endpoint: %s", err))), nil
			}
		}

		config.HealthcheckEndpoint = *request.Body.HealthcheckEndpoint
	}

	if request.Body.SigningSecret != nil {
		encryptedSecret, err := t.config.Encryption.EncryptString(*request.Body.SigningSecret, httpoperator.SigningSecretEncryptionDataID)

		if err != nil {
			return nil, fmt.Errorf("failed to encrypt signing secret: %w", err)
		}

		config.SigningSecret = encryptedSecret
	}

	if request.Body.RequestTimeoutSeconds != nil {
		config.RequestTimeoutSeconds = int(*request.Body.RequestTimeoutSeconds)
	}

	configBytes, err := json.Marshal(config)

	if err != nil {
		return nil, fmt.Errorf("failed to marshal operator config: %w", err)
	}

	updated, err := t.config.V1.Operators().UpdateOperator(
		ctx.Request().Context(),
		operator.TenantID,
		operator.ID,
		v1.UpdateOperatorOpts{
			Config: configBytes,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update operator: %w", err)
	}

	transformed, err := transformers.ToV1HTTPOperator(updated)

	if err != nil {
		return nil, fmt.Errorf("failed to transform operator: %w", err)
	}

	return gen.V1HttpOperatorUpdate200JSONResponse(transformed), nil
}
