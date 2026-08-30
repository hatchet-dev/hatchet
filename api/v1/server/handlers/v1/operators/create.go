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

func (t *V1OperatorsService) V1HttpOperatorCreate(ctx echo.Context, request gen.V1HttpOperatorCreateRequestObject) (gen.V1HttpOperatorCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	// Validate endpoints against the SSRF policy at registration time (UX only; the
	// dial-time check remains the real enforcement point).
	if err := safeclient.ValidateEndpoint(request.Body.TriggerEndpoint); err != nil {
		return gen.V1HttpOperatorCreate400JSONResponse(apierrors.NewAPIErrors(fmt.Sprintf("invalid trigger endpoint: %s", err))), nil
	}

	if err := safeclient.ValidateEndpoint(request.Body.HealthcheckEndpoint); err != nil {
		return gen.V1HttpOperatorCreate400JSONResponse(apierrors.NewAPIErrors(fmt.Sprintf("invalid healthcheck endpoint: %s", err))), nil
	}

	if request.Body.RequestTimeoutSeconds <= 0 {
		return gen.V1HttpOperatorCreate400JSONResponse(apierrors.NewAPIErrors("requestTimeoutSeconds must be greater than 0")), nil
	}

	// Store the signing secret encrypted at rest; the operator decrypts it at startup.
	encryptedSecret, err := t.config.Encryption.EncryptString(request.Body.SigningSecret, httpoperator.SigningSecretEncryptionDataID)

	if err != nil {
		return nil, fmt.Errorf("failed to encrypt signing secret: %w", err)
	}

	config := httpoperator.HTTPOperatorConfig{
		TriggerEndpoint:       request.Body.TriggerEndpoint,
		HealthcheckEndpoint:   request.Body.HealthcheckEndpoint,
		SigningSecret:         encryptedSecret,
		RequestTimeoutSeconds: int(request.Body.RequestTimeoutSeconds),
	}

	configBytes, err := json.Marshal(config)

	if err != nil {
		return nil, fmt.Errorf("failed to marshal operator config: %w", err)
	}

	operator, err := t.config.V1.Operators().CreateOperator(
		ctx.Request().Context(),
		tenant.ID,
		v1.CreateOperatorOpts{
			Name:   request.Body.Name,
			Kind:   sqlcv1.V1OperatorKindHTTPAPI,
			Config: configBytes,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create operator: %w", err)
	}

	transformed, err := transformers.ToV1HTTPOperator(operator)

	if err != nil {
		return nil, fmt.Errorf("failed to transform operator: %w", err)
	}

	return gen.V1HttpOperatorCreate200JSONResponse(transformed), nil
}
