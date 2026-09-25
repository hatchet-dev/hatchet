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
	status     gen.V1CELDebugResponseStatus
	output     *bool
	outputStr  *string
	outputInt  *int
	outputType *rest.V1CELDebugResponseOutputType
	err        *string
}

// Status returns the CEL evaluation status.
func (r *CELEvaluationResult) Status() gen.V1CELDebugResponseStatus { return r.status }

// Output returns the boolean result, if present.
func (r *CELEvaluationResult) Output() *bool { return r.output }

// OutputStr returns the string result, if present.
func (r *CELEvaluationResult) OutputStr() *string { return r.outputStr }

// OutputInt returns the integer result, if present.
func (r *CELEvaluationResult) OutputInt() *int { return r.outputInt }

// OutputType returns the type of the result, if present.
func (r *CELEvaluationResult) OutputType() *rest.V1CELDebugResponseOutputType {
	return r.outputType
}

// Err returns the evaluation error message, if present.
func (r *CELEvaluationResult) Err() *string { return r.err }

// Deprecated: Debug is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
//
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
		status:     gen.V1CELDebugResponseStatusSUCCESS,
		output:     resp.JSON200.Output,
		outputStr:  resp.JSON200.OutputStr,
		outputInt:  resp.JSON200.OutputInt,
		outputType: resp.JSON200.OutputType,
	}, nil
}
