package features

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// CELClient provides methods for evaluating CEL expressions
type CELClient struct {
	api      *rest.ClientWithResponses
	tenantId uuid.UUID
}

// NewCELClient creates a new CELClient
func NewCELClient(
	api *rest.ClientWithResponses,
	tenantId string,
) *CELClient {
	tenantIdUUID := uuid.MustParse(tenantId)

	return &CELClient{
		api:      api,
		tenantId: tenantIdUUID,
	}
}

// CELEvaluationResult represents the result of a CEL expression evaluation.
type CELEvaluationResult struct {
	// Status is the status of the CEL expression evaluation. Can be one of:
	// - SUCCESS
	// - ERROR
	Status gen.V1CELDebugResponseStatus `json:"status"`
	// Output is the boolean evaluation result of the CEL expression when the Status was SUCCESS.
	Output *bool `json:"output"`
	// Error is the error message if the Status was ERROR.
	Error *string `json:"error"`
}

// Debug evaluates a CEL expression with the provided input, filter payload, and optional metadata.
// Useful for testing and validating CEL expressions and debugging issues in production.
func (c *CELClient) Debug(ctx context.Context, expression string, input map[string]any, additionalMetadata, filterPayload *map[string]any) (*CELEvaluationResult, error) {
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
		return nil, errors.Wrap(err, "failed to evaluate CEL expression")
	}

	if resp.JSON200 == nil {
		return nil, errors.Newf("received non-200 response from server. got status %d with body '%s'", resp.StatusCode(), string(resp.Body))
	}

	if resp.JSON200.Status == rest.V1CELDebugResponseStatus(gen.V1CELDebugResponseStatusERROR) {
		return &CELEvaluationResult{
			Status: gen.V1CELDebugResponseStatusERROR,
			Error:  resp.JSON200.Error,
		}, nil
	}

	return &CELEvaluationResult{
		Status: gen.V1CELDebugResponseStatusSUCCESS,
		Output: resp.JSON200.Output,
	}, nil
}
