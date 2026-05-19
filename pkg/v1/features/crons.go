// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package features

import (
	"context"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// Deprecated: CronsClient is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type CronsClient interface {
	Create(ctx context.Context, workflowName string, cron CreateCronTrigger) (*rest.CronWorkflows, error)

	Delete(ctx context.Context, cronId string) error

	List(ctx context.Context, opts rest.CronWorkflowListParams) (*rest.CronWorkflowsList, error)

	Get(ctx context.Context, cronId string) (*rest.CronWorkflows, error)
}

// Deprecated: CreateCronTrigger is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type CreateCronTrigger struct {
	Name               string                 `json:"name"`
	Expression         string                 `json:"expression"`
	Input              map[string]interface{} `json:"input,omitempty"`
	AdditionalMetadata map[string]interface{} `json:"additionalMetadata,omitempty"`
	Priority           *int32                 `json:"priority,omitempty"`
}

// cronsClientImpl implements the CronsClient interface.
type cronsClientImpl struct {
	api      *rest.ClientWithResponses
	tenantId uuid.UUID
}

// Deprecated: NewCronsClient is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func NewCronsClient(
	api *rest.ClientWithResponses,
	tenantId *string,
) CronsClient {
	tenantIdUUID := uuid.MustParse(*tenantId)

	return &cronsClientImpl{
		api:      api,
		tenantId: tenantIdUUID,
	}
}

// Deprecated: ValidateCronExpression is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func ValidateCronExpression(expression string) bool {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(expression)

	return err == nil
}

// Deprecated: Create is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
//
// Create creates a new cron workflow trigger.
func (c *cronsClientImpl) Create(ctx context.Context, workflowName string, cron CreateCronTrigger) (*rest.CronWorkflows, error) {
	// Validate cron expression
	if !ValidateCronExpression(cron.Expression) {
		return nil, &InvalidCronExpressionError{Expression: cron.Expression}
	}

	// Prepare input and metadata maps if nil
	input := cron.Input
	if input == nil {
		input = make(map[string]interface{})
	}

	additionalMetadata := cron.AdditionalMetadata
	if additionalMetadata == nil {
		additionalMetadata = make(map[string]interface{})
	}

	request := rest.CronWorkflowTriggerCreateJSONRequestBody{
		CronName:           cron.Name,
		CronExpression:     cron.Expression,
		Input:              input,
		AdditionalMetadata: additionalMetadata,
	}

	resp, err := c.api.CronWorkflowTriggerCreateWithResponse(
		ctx,
		c.tenantId,
		workflowName,
		request,
	)
	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Deprecated: Delete is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
//
// Delete removes a cron workflow trigger.
func (c *cronsClientImpl) Delete(ctx context.Context, cronId string) error {
	cronIdUUID, err := uuid.Parse(cronId)
	if err != nil {
		return err
	}

	_, err = c.api.WorkflowCronDeleteWithResponse(
		ctx,
		c.tenantId,
		cronIdUUID,
	)
	return err
}

// Deprecated: List is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
//
// List retrieves a collection of cron workflow triggers based on the provided parameters.
func (c *cronsClientImpl) List(ctx context.Context, opts rest.CronWorkflowListParams) (*rest.CronWorkflowsList, error) {
	resp, err := c.api.CronWorkflowListWithResponse(
		ctx,
		c.tenantId,
		&opts,
	)
	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Deprecated: Get is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
//
// Get retrieves a specific cron workflow trigger by its ID.
func (c *cronsClientImpl) Get(ctx context.Context, cronId string) (*rest.CronWorkflows, error) {
	cronIdUUID, err := uuid.Parse(cronId)
	if err != nil {
		return nil, err
	}

	resp, err := c.api.WorkflowCronGetWithResponse(
		ctx,
		c.tenantId,
		cronIdUUID,
	)
	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Deprecated: InvalidCronExpressionError is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type InvalidCronExpressionError struct {
	Expression string
}

// Deprecated: Error is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (e *InvalidCronExpressionError) Error() string {
	return "invalid cron expression: " + e.Expression
}
