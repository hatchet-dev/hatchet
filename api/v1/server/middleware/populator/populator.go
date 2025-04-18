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
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
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

func (p *PopulateGetter) GetTenant() (*dbsqlc.Tenant, error) {
	tenantValue := p.c.Get("tenant")
	if tenantValue == nil {
		return nil, ErrNotFound
	}

	tenant, ok := tenantValue.(*dbsqlc.Tenant)
	if !ok || tenant == nil {
		return nil, ErrNotFound
	}

	return tenant, nil
}

func (p *PopulateGetter) GetUser() (*dbsqlc.User, error) {
	userValue := p.c.Get("user")
	if userValue == nil {
		return nil, ErrNotFound
	}

	user, ok := userValue.(*dbsqlc.User)
	if !ok || user == nil {
		return nil, ErrNotFound
	}

	return user, nil
}

func (p *PopulateGetter) GetTenantMember() (*dbsqlc.PopulateTenantMembersRow, error) {
	tenantMemberValue := p.c.Get("tenant-member")
	if tenantMemberValue == nil {
		return nil, ErrNotFound
	}

	tenantMember, ok := tenantMemberValue.(*dbsqlc.PopulateTenantMembersRow)
	if !ok || tenantMember == nil {
		return nil, ErrNotFound
	}

	return tenantMember, nil
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

func (p *PopulateGetter) GetWorkflow() (*dbsqlc.GetWorkflowByIdRow, error) {
	workflowValue := p.c.Get("workflow")
	if workflowValue == nil {
		return nil, ErrNotFound
	}

	workflow, ok := workflowValue.(*dbsqlc.GetWorkflowByIdRow)
	if !ok || workflow == nil {
		return nil, ErrNotFound
	}

	return workflow, nil
}

func (p *PopulateGetter) GetAPIToken() (*dbsqlc.APIToken, error) {
	apiTokenValue := p.c.Get("api-token")
	if apiTokenValue == nil {
		return nil, ErrNotFound
	}

	apiToken, ok := apiTokenValue.(*dbsqlc.APIToken)
	if !ok || apiToken == nil {
		return nil, ErrNotFound
	}

	return apiToken, nil
}

func (p *PopulateGetter) GetEvent() (*dbsqlc.Event, error) {
	eventValue := p.c.Get("event")
	if eventValue == nil {
		return nil, ErrNotFound
	}

	event, ok := eventValue.(*dbsqlc.Event)
	if !ok || event == nil {
		return nil, ErrNotFound
	}

	return event, nil
}

func (p *PopulateGetter) GetSNSIntegration() (*dbsqlc.SNSIntegration, error) {
	snsIntegrationValue := p.c.Get("sns-integration")
	if snsIntegrationValue == nil {
		return nil, ErrNotFound
	}

	snsIntegration, ok := snsIntegrationValue.(*dbsqlc.SNSIntegration)
	if !ok || snsIntegration == nil {
		return nil, ErrNotFound
	}

	return snsIntegration, nil
}

func (p *PopulateGetter) GetStepRun() (*repository.GetStepRunFull, error) {
	stepRunValue := p.c.Get("step-run")
	if stepRunValue == nil {
		return nil, ErrNotFound
	}

	stepRun, ok := stepRunValue.(*repository.GetStepRunFull)
	if !ok || stepRun == nil {
		return nil, ErrNotFound
	}

	return stepRun, nil
}

func (p *PopulateGetter) GetSlackWebhook() (*dbsqlc.SlackAppWebhook, error) {
	slackWebhookValue := p.c.Get("slack-webhook")
	if slackWebhookValue == nil {
		return nil, ErrNotFound
	}

	slackWebhook, ok := slackWebhookValue.(*dbsqlc.SlackAppWebhook)
	if !ok || slackWebhook == nil {
		return nil, ErrNotFound
	}

	return slackWebhook, nil
}

func (p *PopulateGetter) GetTenantInvite() (*dbsqlc.TenantInviteLink, error) {
	tenantInviteValue := p.c.Get("tenant-invite")
	if tenantInviteValue == nil {
		return nil, ErrNotFound
	}

	tenantInvite, ok := tenantInviteValue.(*dbsqlc.TenantInviteLink)
	if !ok || tenantInvite == nil {
		return nil, ErrNotFound
	}

	return tenantInvite, nil
}

func (p *PopulateGetter) GetAlertEmailGroup() (*dbsqlc.TenantAlertEmailGroup, error) {
	alertGroupValue := p.c.Get("alert-email-group")
	if alertGroupValue == nil {
		return nil, ErrNotFound
	}

	alertGroup, ok := alertGroupValue.(*dbsqlc.TenantAlertEmailGroup)
	if !ok || alertGroup == nil {
		return nil, ErrNotFound
	}

	return alertGroup, nil
}

func (p *PopulateGetter) GetWorkflowRun() (*dbsqlc.GetWorkflowRunByIdRow, error) {
	workflowRunValue := p.c.Get("workflow-run")
	if workflowRunValue == nil {
		return nil, ErrNotFound
	}

	workflowRun, ok := workflowRunValue.(*dbsqlc.GetWorkflowRunByIdRow)
	if !ok || workflowRun == nil {
		return nil, ErrNotFound
	}

	return workflowRun, nil
}

func (p *PopulateGetter) GetScheduledWorkflow() (*dbsqlc.ListScheduledWorkflowsRow, error) {
	scheduledValue := p.c.Get("scheduled-workflow-run")
	if scheduledValue == nil {
		return nil, ErrNotFound
	}

	scheduled, ok := scheduledValue.(*dbsqlc.ListScheduledWorkflowsRow)
	if !ok || scheduled == nil {
		return nil, ErrNotFound
	}

	return scheduled, nil
}

func (p *PopulateGetter) GetWorker() (*dbsqlc.GetWorkerByIdRow, error) {
	workerValue := p.c.Get("worker")
	if workerValue == nil {
		return nil, ErrNotFound
	}

	worker, ok := workerValue.(*dbsqlc.GetWorkerByIdRow)
	if !ok || worker == nil {
		return nil, ErrNotFound
	}

	return worker, nil
}

func (p *PopulateGetter) GetWebhookWorker() (*dbsqlc.WebhookWorker, error) {
	webhookValue := p.c.Get("webhook")
	if webhookValue == nil {
		return nil, ErrNotFound
	}

	webhook, ok := webhookValue.(*dbsqlc.WebhookWorker)
	if !ok || webhook == nil {
		return nil, ErrNotFound
	}

	return webhook, nil
}

func (p *PopulateGetter) GetTask() (*sqlcv1.V1TasksOlap, error) {
	taskValue := p.c.Get("task")
	if taskValue == nil {
		return nil, ErrNotFound
	}

	task, ok := taskValue.(*sqlcv1.V1TasksOlap)
	if !ok || task == nil {
		return nil, ErrNotFound
	}

	return task, nil
}

func (p *PopulateGetter) GetV1WorkflowRun() (*v1.V1WorkflowRunPopulator, error) {
	workflowRunValue := p.c.Get("v1-workflow-run")
	if workflowRunValue == nil {
		return nil, ErrNotFound
	}

	workflowRun, ok := workflowRunValue.(*v1.V1WorkflowRunPopulator)
	if !ok || workflowRun == nil {
		return nil, ErrNotFound
	}

	return workflowRun, nil
}
