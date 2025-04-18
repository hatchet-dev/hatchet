package populator

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

// Populator provides methods to access and populate resources.
type Populator struct {
	config *server.ServerConfig
}

// NewPopulator creates a new populator instance.
func NewPopulator(config *server.ServerConfig) *Populator {
	return &Populator{
		config: config,
	}
}

// Middleware intercepts HTTP requests to populate resources.
func (p *Populator) Middleware(r *middleware.RouteInfo) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := p.populate(c, r)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return c.NoContent(404)
			}
			return err
		}
		return nil
	}
}

// populate processes resource population for a request.
func (p *Populator) populate(c echo.Context, r *middleware.RouteInfo) error {
	if len(r.Resources) == 0 {
		return nil
	}

	// Create a map of keys to identifiers from path parameters
	keysToIds := make(map[string]string)
	for _, paramName := range c.ParamNames() {
		keysToIds[paramName] = c.Param(paramName)
	}

	rootResource := &resource{}
	currResource := rootResource
	var prevResource *resource

	// Build resource chain
	for _, resourceKey := range r.Resources {
		currResource.ResourceKey = resourceKey

		if resourceId, exists := keysToIds[resourceKey]; exists && resourceId != "" {
			currResource.ResourceID = resourceId
		}

		if prevResource != nil {
			currResource.ParentID = prevResource.ResourceID
			prevResource.Children = append(prevResource.Children, currResource)
		}

		prevResource = currResource
		currResource = &resource{}
	}

	// Traverse and populate resource tree
	err := p.traverseNode(c, rootResource)
	if err != nil {
		return err
	}

	// Add populated resources to context
	currResource = rootResource
	for {
		if currResource.Resource != nil {
			c.Set(currResource.ResourceKey, currResource.Resource)
		}

		if len(currResource.Children) == 0 {
			break
		}

		currResource = currResource.Children[0]
	}

	return nil
}

type resource struct {
	ResourceKey string
	ResourceID  string
	Resource    interface{}
	ParentID    string
	Children    []*resource
}

// traverseNode recursively populates a resource node and its children.
func (p *Populator) traverseNode(c echo.Context, node *resource) error {
	populated := false

	// Populate current node if ID is available
	if node.ResourceID != "" {
		err := p.populateResource(c.Request().Context(), node)
		if err != nil {
			return fmt.Errorf("could not populate resource %s: %w", node.ResourceKey, err)
		}
		populated = true
	}

	// Process child nodes
	if node.Children != nil {
		for _, child := range node.Children {
			if populated {
				child.ParentID = node.ResourceID
			}

			err := p.traverseNode(c, child)
			if err != nil {
				return err
			}

			if !populated && child.ParentID != "" {
				// Use parent info to populate current resource
				err = p.populateResource(c.Request().Context(), node)
				if err != nil {
					return err
				}
				populated = true
			} else if populated && child.ParentID != node.ResourceID {
				return fmt.Errorf("resource %s could not be populated - parent ID mismatch", node.ResourceKey)
			}
		}
	}

	// Ensure resource was populated
	if !populated {
		return fmt.Errorf("resource %s could not be populated", node.ResourceKey)
	}

	return nil
}

// populateResource calls the appropriate Setter based on resource type.
func (p *Populator) populateResource(ctx context.Context, node *resource) error {
	switch node.ResourceKey {
	case "tenant":
		tenant, err := p.SetTenant(ctx, node.ResourceID)
		if err != nil {
			return err
		}
		node.Resource = tenant
		return nil
	case "api-token":
		token, err := p.SetAPIToken(ctx, node.ParentID, node.ResourceID)
		if err != nil {
			return err
		}
		node.Resource = token
		return nil
	case "tenant-invite":
		invite, err := p.SetTenantInvite(ctx, node.ResourceID)
		if err != nil {
			return err
		}
		node.Resource = invite
		return nil
	case "slack":
		webhook, err := p.SetSlackWebhook(ctx, node.ResourceID)
		if err != nil {
			return err
		}
		node.Resource = webhook
		return nil
	case "alert-email-group":
		group, err := p.SetAlertEmailGroup(ctx, node.ResourceID)
		if err != nil {
			return err
		}
		node.Resource = group
		return nil
	case "sns":
		integration, err := p.SetSNSIntegration(ctx, node.ResourceID)
		if err != nil {
			return err
		}
		node.Resource = integration
		return nil
	case "workflow":
		workflow, err := p.SetWorkflow(ctx, node.ResourceID)
		if err != nil {
			return err
		}
		node.Resource = workflow
		return nil
	case "workflow-run":
		run, err := p.SetWorkflowRun(ctx, node.ParentID, node.ResourceID)
		if err != nil {
			return err
		}
		node.Resource = run
		return nil
	case "scheduled-workflow-run":
		scheduled, err := p.SetScheduledWorkflow(ctx, node.ParentID, node.ResourceID)
		if err != nil {
			return err
		}
		node.Resource = scheduled
		return nil
	case "cron-workflow":
		cron, err := p.SetCronWorkflow(ctx, node.ParentID, node.ResourceID)
		if err != nil {
			return err
		}
		node.Resource = cron
		return nil
	case "step-run":
		stepRun, err := p.SetStepRun(ctx, node.ParentID, node.ResourceID)
		if err != nil {
			return err
		}
		node.Resource = stepRun
		return nil
	case "event":
		event, err := p.SetEvent(ctx, node.ResourceID)
		if err != nil {
			return err
		}
		node.Resource = event
		return nil
	case "worker":
		worker, err := p.SetWorker(ctx, node.ResourceID)
		if err != nil {
			return err
		}
		node.Resource = worker
		return nil
	case "webhook":
		webhook, err := p.SetWebhookWorker(ctx, node.ResourceID)
		if err != nil {
			return err
		}
		node.Resource = webhook
		return nil
	case "task":
		task, err := p.SetTask(ctx, node.ResourceID)
		if err != nil {
			return err
		}
		node.Resource = task
		return nil
	case "v1-workflow-run":
		workflowRun, err := p.SetV1WorkflowRun(ctx, node.ResourceID)
		if err != nil {
			return err
		}
		node.Resource = workflowRun
		return nil
	default:
		return fmt.Errorf("unknown resource type: %s", node.ResourceKey)
	}
}

// Resource Setters with strict typing
// Each method below should be implemented with proper return types
// For now, using interface{} as placeholders

func (p *Populator) SetTenant(ctx context.Context, id string) (*dbsqlc.Tenant, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return p.config.APIRepository.Tenant().GetTenantByID(timeoutCtx, id)
}

func (p *Populator) SetAPIToken(ctx context.Context, tenantID, id string) (*dbsqlc.APIToken, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	apiToken, err := p.config.APIRepository.APIToken().GetAPITokenById(timeoutCtx, id)
	if err != nil {
		return nil, err
	}

	// Validate tenant association if provided
	if !apiToken.TenantId.Valid {
		return nil, fmt.Errorf("api token has no tenant id")
	}

	return apiToken, nil
}

func (p *Populator) SetTenantInvite(ctx context.Context, id string) (interface{}, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return p.config.APIRepository.TenantInvite().GetTenantInvite(timeoutCtx, id)
}

func (p *Populator) SetSlackWebhook(ctx context.Context, id string) (interface{}, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return p.config.APIRepository.Slack().GetSlackWebhookById(timeoutCtx, id)
}

func (p *Populator) SetAlertEmailGroup(ctx context.Context, id string) (interface{}, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return p.config.APIRepository.TenantAlertingSettings().GetTenantAlertGroupById(timeoutCtx, id)
}

func (p *Populator) SetSNSIntegration(ctx context.Context, id string) (interface{}, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return p.config.APIRepository.SNS().GetSNSIntegrationById(timeoutCtx, id)
}

func (p *Populator) SetWorkflow(ctx context.Context, id string) (interface{}, error) {
	return p.config.APIRepository.Workflow().GetWorkflowById(ctx, id)
}

func (p *Populator) SetWorkflowRun(ctx context.Context, workflowID, id string) (interface{}, error) {
	return p.config.APIRepository.WorkflowRun().GetWorkflowRunById(ctx, workflowID, id)
}

func (p *Populator) SetScheduledWorkflow(ctx context.Context, workflowID, id string) (interface{}, error) {
	return p.config.APIRepository.WorkflowRun().GetScheduledWorkflow(ctx, workflowID, id)
}

func (p *Populator) SetCronWorkflow(ctx context.Context, workflowID, id string) (interface{}, error) {
	return p.config.APIRepository.Workflow().GetCronWorkflow(ctx, workflowID, id)
}

func (p *Populator) SetStepRun(ctx context.Context, tenantID, id string) (interface{}, error) {
	stepRun, err := p.config.APIRepository.StepRun().GetStepRunById(id)
	if err != nil {
		return nil, err
	}

	// Validate tenant ID if provided
	if tenantID != "" {
		tenantIDFromStep := sqlchelpers.UUIDToStr(stepRun.TenantId)
		if tenantIDFromStep != tenantID {
			return nil, fmt.Errorf("tenant id mismatch when populating step run")
		}
	}

	return stepRun, nil
}

func (p *Populator) SetEvent(ctx context.Context, id string) (interface{}, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return p.config.APIRepository.Event().GetEventById(timeoutCtx, id)
}

func (p *Populator) SetWorker(ctx context.Context, id string) (interface{}, error) {
	return p.config.APIRepository.Worker().GetWorkerById(id)
}

func (p *Populator) SetWebhookWorker(ctx context.Context, id string) (interface{}, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return p.config.APIRepository.WebhookWorker().GetWebhookWorkerByID(timeoutCtx, id)
}

func (p *Populator) SetTask(ctx context.Context, id string) (interface{}, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return p.config.V1.OLAP().ReadTaskRun(timeoutCtx, id)
}

func (p *Populator) SetV1WorkflowRun(ctx context.Context, id string) (interface{}, error) {
	return p.config.V1.OLAP().ReadWorkflowRun(ctx, sqlchelpers.UUIDFromStr(id))
}

type PopulateGetter struct {
	c echo.Context
}

var ErrNotFound = fmt.Errorf("populator not found")

func FromContext(c echo.Context) *PopulateGetter {
	return &PopulateGetter{
		c: c,
	}
}

func (p *PopulateGetter) GetCronWorkflow() (*dbsqlc.GetCronWorkflowByIdRow, error) {
	cronValue := p.c.Get("cron-workflow")
	if cronValue == nil {
		return nil, ErrNotFound
	}

	cron, ok := cronValue.(*dbsqlc.GetCronWorkflowByIdRow)
	if !ok || cron == nil {
		return nil, ErrNotFound
	}

	return cron, nil
}
