package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type CronOpts struct {
	// Name is the user-friendly name for the cron trigger
	Name string

	// Expression is the cron expression for the trigger
	Expression string

	// Input is the input to the workflow
	Input map[string]interface{}

	// AdditionalMetadata is additional metadata to be stored with the cron trigger
	AdditionalMetadata map[string]string

	// Priority is the priority of the run triggered by the cron
	Priority *int32
}

type CronClient interface {
	// Create creates a new cron trigger
	Create(ctx context.Context, workflow string, opts *CronOpts) (*gen.CronWorkflows, error)

	// Delete deletes a cron trigger
	Delete(ctx context.Context, id string) error

	// List lists all cron triggers
	List(ctx context.Context) (*gen.CronWorkflowsList, error)
}

type cronClientImpl struct {
	restClient *rest.ClientWithResponses

	l *zerolog.Logger

	v validator.Validator

	tenantId uuid.UUID

	namespace string
}

func NewCronClient(restClient *rest.ClientWithResponses, l *zerolog.Logger, v validator.Validator, tenantId, namespace string) (CronClient, error) {
	tenantIdUUID, err := uuid.Parse(tenantId)

	if err != nil {
		return nil, err
	}

	return &cronClientImpl{
		restClient: restClient,
		l:          l,
		v:          v,
		namespace:  namespace,
		tenantId:   tenantIdUUID,
	}, nil
}

func (c *cronClientImpl) Create(ctx context.Context, workflow string, opts *CronOpts) (*gen.CronWorkflows, error) {
	additionalMeta := make(map[string]any)

	for k, v := range opts.AdditionalMetadata {
		additionalMeta[k] = v
	}

	resp, err := c.restClient.CronWorkflowTriggerCreate(
		ctx,
		c.tenantId,
		workflow,
		rest.CreateCronWorkflowTriggerRequest{
			CronName:           opts.Name,
			CronExpression:     opts.Expression,
			Input:              opts.Input,
			AdditionalMetadata: additionalMeta,
			Priority:           opts.Priority,
		},
	)

	if err != nil {
		return nil, err
	}

	// if response code is not 200-level, return an error
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// parse the response body into a cron trigger
	cron := &gen.CronWorkflows{}

	err = json.NewDecoder(resp.Body).Decode(cron)

	if err != nil {
		return nil, fmt.Errorf("could not unmarshal response body: %w", err)
	}

	return cron, nil
}

func (c *cronClientImpl) Delete(ctx context.Context, id string) error {
	idUUID, err := uuid.Parse(id)

	if err != nil {
		return fmt.Errorf("could not parse id: %w", err)
	}

	resp, err := c.restClient.WorkflowCronDelete(
		ctx,
		c.tenantId,
		idUUID,
	)

	if err != nil {
		return err
	}

	// if response code is not 200-level, return an error
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *cronClientImpl) List(ctx context.Context) (*gen.CronWorkflowsList, error) {
	resp, err := c.restClient.CronWorkflowList(
		ctx,
		c.tenantId,
		&rest.CronWorkflowListParams{},
	)

	if err != nil {
		return nil, err
	}

	// parse the response body into a list of cron triggers
	cronList := &gen.CronWorkflowsList{}

	err = json.NewDecoder(resp.Body).Decode(&cronList)

	if err != nil {
		return nil, fmt.Errorf("could not unmarshal response body: %w", err)
	}

	return cronList, nil
}
