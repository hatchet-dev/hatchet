// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package features

import (
	"context"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// Deprecated: CELClient is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type CELClient interface {
	Debug(ctx context.Context, expression string, input map[string]interface{}, additionalMetadata, filterPayload *map[string]interface{}) (*CELEvaluationResult, error)
}

type celClientImpl struct {
	api      *rest.ClientWithResponses
	tenantId uuid.UUID
}

// Deprecated: NewCELClient is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func NewCELClient(
	api *rest.ClientWithResponses,
	tenantId *string,
) CELClient {
	tenantIdUUID := uuid.MustParse(*tenantId)

	return &celClientImpl{
		api:      api,
		tenantId: tenantIdUUID,
	}
}

// Deprecated: CELEvaluationResult is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type CELEvaluationResult struct {
	status gen.V1CELDebugResponseStatus
	output *bool
	err    *string
}

// Deprecated: Debug is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (c *celClientImpl) Debug(ctx context.Context, expression string, input map[string]interface{}, additionalMetadata, filterPayload *map[string]interface{}) (*CELEvaluationResult, error) {
	resp, err := c.api.V1CelDebugWithResponse(
		ctx,
		c.tenantId,
		rest.V1CELDebugRequest{
			Expression:         expression,
			AdditionalMetadata: additionalMetadata,
			FilterPayload:      filterPayload,
			Input:              input,
		},
	)

	if err != nil {
		return nil, err
	}

	if resp.JSON200.Status == rest.V1CELDebugResponseStatus(gen.V1CELDebugResponseStatusERROR) {
		return &CELEvaluationResult{
			status: gen.V1CELDebugResponseStatusERROR,
			err:    resp.JSON200.Error,
		}, nil
	}

	return &CELEvaluationResult{
		status: gen.V1CELDebugResponseStatusSUCCESS,
		output: resp.JSON200.Output,
	}, nil
}
