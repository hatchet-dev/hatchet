package features

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"

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
	tenantId uuid.UUID,
) *CELClient {
	tenantIdUUID := tenantId

	return &CELClient{
		api:      api,
		tenantId: tenantIdUUID,
	}
}

// Debug evaluates a CEL expression with the provided input, filter payload, and optional metadata.
// Useful for testing and validating CEL expressions and debugging issues in production.
func (c *CELClient) Debug(ctx context.Context, expression string, input map[string]any, additionalMetadata, filterPayload *map[string]any) (*rest.V1CELDebugResponse, error) {
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

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}
