package features

import (
	"context"
	"regexp"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// CronsClient provides methods for interacting with cron workflow triggers
// in the Hatchet platform.
type CronsClient interface {
	// Create creates a new cron workflow trigger.
	Create(ctx context.Context, workflowName string, cron CreateCronTrigger) (*rest.CronWorkflows, error)

	// Delete removes a cron workflow trigger.
	Delete(ctx context.Context, cronId string) error

	// List retrieves a collection of cron workflow triggers based on the provided parameters.
	List(ctx context.Context, opts rest.CronWorkflowListParams) (*rest.CronWorkflowsList, error)

	// Get retrieves a specific cron workflow trigger by its ID.
	Get(ctx context.Context, cronId string) (*rest.CronWorkflows, error)
}

// CreateCronTrigger contains the configuration for creating a cron trigger.
type CreateCronTrigger struct {
	// Name is the unique identifier for the cron trigger.
	Name string `json:"name"`

	// Expression is the cron expression that defines the schedule.
	Expression string `json:"expression"`

	// Input is the optional input data for the workflow.
	Input map[string]interface{} `json:"input,omitempty"`

	// AdditionalMetadata is optional metadata to associate with the cron trigger.
	AdditionalMetadata map[string]interface{} `json:"additionalMetadata,omitempty"`
}

// cronsClientImpl implements the CronsClient interface.
type cronsClientImpl struct {
	api      *rest.ClientWithResponses
	tenantId uuid.UUID
}

// NewCronsClient creates a new client for interacting with cron workflow triggers.
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

// ValidateCronExpression validates that a string is a valid cron expression.
func ValidateCronExpression(expression string) bool {
	// Basic cron validation regex matching the TypeScript implementation
	cronRegex := regexp.MustCompile(`^(\*|([0-9]|1[0-9]|2[0-9]|3[0-9]|4[0-9]|5[0-9])|\*\/([0-9]|1[0-9]|2[0-3])) (\*|([0-9]|1[0-9]|2[0-3])|\*\/([0-9]|1[0-9]|2[0-3])) (\*|([1-9]|1[0-9]|2[0-9]|3[0-1])|\*\/([1-9]|1[0-9]|2[0-9]|3[0-1])) (\*|([1-9]|1[0-2])|\*\/([1-9]|1[0-2])) (\*|([0-6])|\*\/([0-6]))$`)
	return cronRegex.MatchString(expression)
}

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

// InvalidCronExpressionError represents an error when an invalid cron expression is provided.
type InvalidCronExpressionError struct {
	Expression string
}

func (e *InvalidCronExpressionError) Error() string {
	return "invalid cron expression: " + e.Expression
}
