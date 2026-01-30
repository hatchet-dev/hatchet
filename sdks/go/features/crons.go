package features

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// RunPriority is the priority for a workflow run.
type RunPriority int32

const (
	// RunPriorityLow is the lowest priority for a workflow run.
	RunPriorityLow RunPriority = 1
	// RunPriorityMedium is the medium priority for a workflow run.
	RunPriorityMedium RunPriority = 2
	// RunPriorityHigh is the highest priority for a workflow run.
	RunPriorityHigh RunPriority = 3
)

// CreateCronTrigger contains the configuration for creating a cron trigger.
type CreateCronTrigger struct {
	// Name is the unique identifier for the cron trigger.
	Name string `json:"name"`

	// Expression is the cron expression that defines the schedule.
	Expression string `json:"expression"`

	// (optional) Input is the input data for the workflow.
	Input map[string]interface{} `json:"input,omitempty"`

	// (optional) AdditionalMetadata is metadata to associate with the cron trigger.
	AdditionalMetadata map[string]interface{} `json:"additionalMetadata,omitempty"`

	// (optional) Priority is the priority of the run triggered by the cron.
	Priority *RunPriority `json:"priority,omitempty"`
}

// CronsClient provides methods for interacting with cron workflow triggers
type CronsClient struct {
	api      *rest.ClientWithResponses
	tenantId uuid.UUID
}

// NewCronsClient creates a new CronsClient
func NewCronsClient(
	api *rest.ClientWithResponses,
	tenantId uuid.UUID,
) *CronsClient {
	tenantIdUUID := tenantId

	return &CronsClient{
		api:      api,
		tenantId: tenantIdUUID,
	}
}

// IsValidCronExpression validates that a string is a valid cron expression.
func IsValidCronExpression(expression string) bool {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(expression)

	return err == nil
}

// Create creates a new cron workflow trigger.
func (c *CronsClient) Create(ctx context.Context, workflowName string, cron CreateCronTrigger) (*rest.CronWorkflows, error) {
	// Validate cron expression
	if !IsValidCronExpression(cron.Expression) {
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
		return nil, errors.Wrap(err, "failed to create cron workflow trigger")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Delete removes a cron workflow trigger.
func (c *CronsClient) Delete(ctx context.Context, cronId string) error {
	cronIdUUID, err := uuid.Parse(cronId)
	if err != nil {
		return err
	}

	resp, err := c.api.WorkflowCronDeleteWithResponse(
		ctx,
		c.tenantId,
		cronIdUUID,
	)
	if err != nil {
		return errors.Wrap(err, "failed to delete cron workflow trigger")
	}

	if err := validateStatusCodeResponse(resp.StatusCode(), resp.Body); err != nil {
		return err
	}

	return nil
}

// List retrieves a collection of cron workflow triggers based on the provided parameters.
func (c *CronsClient) List(ctx context.Context, opts rest.CronWorkflowListParams) (*rest.CronWorkflowsList, error) {
	resp, err := c.api.CronWorkflowListWithResponse(
		ctx,
		c.tenantId,
		&opts,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list cron workflow triggers")
	}

	return resp.JSON200, nil
}

// Get retrieves a specific cron workflow trigger by its ID.
func (c *CronsClient) Get(ctx context.Context, cronId string) (*rest.CronWorkflows, error) {
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
		return nil, errors.Wrap(err, "failed to get cron workflow trigger")
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
