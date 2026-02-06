package features

import (
	"context"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// The CEL client is a client for debugging CEL expressions within Hatchet
type CELClient interface {
	Debug(ctx context.Context, expression string, input map[string]interface{}, additionalMetadata, filterPayload *map[string]interface{}) (*CELEvaluationResult, error)
}

type celClientImpl struct {
	api      *rest.ClientWithResponses
	tenantId uuid.UUID
}

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

type CELEvaluationResult struct {
	output *bool
	err    *string
	status gen.V1CELDebugResponseStatus
}

// Debug a CEL expression with the provided input, filter payload, and optional metadata. Useful for testing and validating CEL expressions and debugging issues in production.
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
