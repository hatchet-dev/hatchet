package run

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
	"github.com/hatchet-dev/hatchet/api/v1/server/authz"
	apitokens "github.com/hatchet-dev/hatchet/api/v1/server/handlers/api-tokens"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/events"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/info"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/ingestors"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/logs"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/metadata"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/monitoring"
	rate_limits "github.com/hatchet-dev/hatchet/api/v1/server/handlers/rate-limits"
	slackapp "github.com/hatchet-dev/hatchet/api/v1/server/handlers/slack-app"
	stepruns "github.com/hatchet-dev/hatchet/api/v1/server/handlers/step-runs"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/tenants"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/users"
	celv1 "github.com/hatchet-dev/hatchet/api/v1/server/handlers/v1/cel"
	eventsv1 "github.com/hatchet-dev/hatchet/api/v1/server/handlers/v1/events"
	filtersv1 "github.com/hatchet-dev/hatchet/api/v1/server/handlers/v1/filters"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/v1/tasks"
	webhooksv1 "github.com/hatchet-dev/hatchet/api/v1/server/handlers/v1/webhooks"
	workflowrunsv1 "github.com/hatchet-dev/hatchet/api/v1/server/handlers/v1/workflow-runs"
	webhookworker "github.com/hatchet-dev/hatchet/api/v1/server/handlers/webhook-worker"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/workers"
	workflowruns "github.com/hatchet-dev/hatchet/api/v1/server/handlers/workflow-runs"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/workflows"
	"github.com/hatchet-dev/hatchet/api/v1/server/headers"
	hatchetmiddleware "github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/ratelimit"
	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/telemetry"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type apiService struct {
	*users.UserService
	*tenants.TenantService
	*events.EventService
	*rate_limits.RateLimitService
	*logs.LogService
	*workflows.WorkflowService
	*workers.WorkerService
	*metadata.MetadataService
	*apitokens.APITokenService
	*stepruns.StepRunService
	*ingestors.IngestorsService
	*slackapp.SlackAppService
	*webhookworker.WebhookWorkersService
	*workflowruns.WorkflowRunsService
	*monitoring.MonitoringService
	*info.InfoService
	*tasks.TasksService
	*workflowrunsv1.V1WorkflowRunsService
	*eventsv1.V1EventsService
	*filtersv1.V1FiltersService
	*webhooksv1.V1WebhooksService
	*celv1.V1CELService
}

func newAPIService(config *server.ServerConfig) *apiService {
	return &apiService{
		UserService:           users.NewUserService(config),
		TenantService:         tenants.NewTenantService(config),
		EventService:          events.NewEventService(config),
		RateLimitService:      rate_limits.NewRateLimitService(config),
		LogService:            logs.NewLogService(config),
		WorkflowService:       workflows.NewWorkflowService(config),
		WorkflowRunsService:   workflowruns.NewWorkflowRunsService(config),
		WorkerService:         workers.NewWorkerService(config),
		MetadataService:       metadata.NewMetadataService(config),
		APITokenService:       apitokens.NewAPITokenService(config),
		StepRunService:        stepruns.NewStepRunService(config),
		IngestorsService:      ingestors.NewIngestorsService(config),
		SlackAppService:       slackapp.NewSlackAppService(config),
		WebhookWorkersService: webhookworker.NewWebhookWorkersService(config),
		MonitoringService:     monitoring.NewMonitoringService(config),
		InfoService:           info.NewInfoService(config),
		TasksService:          tasks.NewTasksService(config),
		V1WorkflowRunsService: workflowrunsv1.NewV1WorkflowRunsService(config),
		V1EventsService:       eventsv1.NewV1EventsService(config),
		V1FiltersService:      filtersv1.NewV1FiltersService(config),
		V1WebhooksService:     webhooksv1.NewV1WebhooksService(config),
		V1CELService:          celv1.NewV1CELService(config),
	}
}

type APIServer struct {
	config                *server.ServerConfig
	additionalMiddlewares []hatchetmiddleware.MiddlewareFunc
}

func NewAPIServer(config *server.ServerConfig) *APIServer {
	return &APIServer{
		config: config,
	}
}

// APIServerExtensionOpt returns a spec and a way to register handlers with an echo group
type APIServerExtensionOpt func(config *server.ServerConfig) (*openapi3.T, func(*echo.Group, *populator.Populator) error, error)

func (t *APIServer) Run(opts ...APIServerExtensionOpt) (func() error, error) {
	e, err := t.getCoreEchoService()

	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		// extensions are implemented as their own echo group which validate against the
		// extension's spec
		g := e.Group("")

		spec, f, err := opt(t.config)

		if err != nil {
			return nil, err
		}

		populator, err := t.registerSpec(g, spec)

		if err != nil {
			return nil, err
		}

		if err := f(g, populator); err != nil {
			return nil, err
		}
	}

	return t.RunWithServer(e)
}

func (t *APIServer) RunWithMiddlewares(middlewares []hatchetmiddleware.MiddlewareFunc, opts ...APIServerExtensionOpt) (func() error, error) {
	t.additionalMiddlewares = middlewares

	return t.Run(opts...)
}

func (t *APIServer) RunWithServer(e *echo.Echo) (func() error, error) {
	routes := e.Routes()

	for _, route := range routes {
		t.config.Logger.Debug().Msgf("registered route: %s %s", route.Method, route.Path)
	}

	go func() {
		if err := e.Start(fmt.Sprintf(":%d", t.config.Runtime.Port)); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	cleanup := func() error {
		return e.Shutdown(context.Background())
	}

	return cleanup, nil
}

func (t *APIServer) getCoreEchoService() (*echo.Echo, error) {
	oaspec, err := gen.GetSwagger()

	if err != nil {
		return nil, err
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.IPExtractor = func(r *http.Request) string {
		// Cloudflare sets CF-Connecting-IP header with the original client IP
		if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
			return ip
		}

		// Fallback to X-Forwarded-For
		if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
			// X-Forwarded-For can contain multiple IPs, we only want the first one
			ips := strings.Split(ip, ",")
			if len(ips) > 0 {
				return ips[0]
			}
		}

		// Additional fallback to X-Real-IP used by certain proxies
		if ip := r.Header.Get("X-Real-IP"); ip != "" {
			return ip
		}

		// Final fallback to remote address
		return r.RemoteAddr
	}

	g := e.Group("")

	if _, err := t.registerSpec(g, oaspec); err != nil {
		return nil, err
	}

	service := newAPIService(t.config)

	myStrictApiHandler := gen.NewStrictHandler(service)

	gen.RegisterHandlers(g, myStrictApiHandler)

	return e, nil
}

func (t *APIServer) registerSpec(g *echo.Group, spec *openapi3.T) (*populator.Populator, error) {
	// application middleware
	populatorMW := populator.NewPopulator(t.config)

	populatorMW.RegisterGetter("tenant", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		idUuid, err := uuid.Parse(id)

		if err != nil {
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid tenant id")
		}

		tenant, err := config.V1.Tenant().GetTenantByID(ctxTimeout, idUuid)

		if err != nil {
			return nil, "", err
		}

		return tenant, "", nil
	})

	populatorMW.RegisterGetter("member", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		idUuid, err := uuid.Parse(id)

		if err != nil {
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid tenant member id")
		}

		member, err := config.V1.Tenant().GetTenantMemberByID(ctxTimeout, idUuid)

		if err != nil {
			return nil, "", err
		}

		return member, member.TenantId.String(), nil
	})

	populatorMW.RegisterGetter("api-token", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		idUuid, err := uuid.Parse(id)

		if err != nil {
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid api token id")
		}

		apiToken, err := config.V1.APIToken().GetAPITokenById(ctxTimeout, idUuid)

		if err != nil {
			return nil, "", err
		}

		// at the moment, API tokens should have a tenant id, because there are no other types of
		// API tokens. If we add other types of API tokens, we'll need to pass in a parent id to query
		// for.
		if apiToken.TenantId == nil {
			return nil, "", fmt.Errorf("api token has no tenant id")
		}

		return apiToken, apiToken.TenantId.String(), nil
	})

	populatorMW.RegisterGetter("tenant-invite", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		idUuid, err := uuid.Parse(id)

		if err != nil {
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid tenant invite id")
		}

		tenantInvite, err := config.V1.TenantInvite().GetTenantInvite(timeoutCtx, idUuid)

		if err != nil {
			return nil, "", err
		}

		return tenantInvite, tenantInvite.TenantId.String(), nil
	})

	populatorMW.RegisterGetter("slack", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		idUuid, err := uuid.Parse(id)

		if err != nil {
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid slack integration id")
		}

		slackWebhook, err := config.V1.Slack().GetSlackWebhookById(timeoutCtx, idUuid)

		if err != nil {
			return nil, "", err
		}

		return slackWebhook, slackWebhook.TenantId.String(), nil
	})

	populatorMW.RegisterGetter("alert-email-group", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		idUuid, err := uuid.Parse(id)

		if err != nil {
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid alert email group id")
		}

		emailGroup, err := config.V1.TenantAlertingSettings().GetTenantAlertGroupById(timeoutCtx, idUuid)

		if err != nil {
			return nil, "", err
		}

		return emailGroup, emailGroup.TenantId.String(), nil
	})

	populatorMW.RegisterGetter("sns", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		idUuid, err := uuid.Parse(id)

		if err != nil {
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid sns integration id")
		}

		snsIntegration, err := config.V1.SNS().GetSNSIntegrationById(timeoutCtx, idUuid)

		if err != nil {
			return nil, "", err
		}

		return snsIntegration, snsIntegration.TenantId.String(), nil
	})

	populatorMW.RegisterGetter("workflow", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		idUuid, err := uuid.Parse(id)

		if err != nil {
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid workflow id")
		}

		workflow, err := config.V1.Workflows().GetWorkflowById(context.Background(), idUuid)

		if err != nil {
			return nil, "", err
		}

		return workflow, workflow.Workflow.TenantId.String(), nil
	})

	populatorMW.RegisterGetter("workflow-run", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		config.Logger.Error().Msgf("deprecated call to workflow-run with parent id %s and id %s: use 'v1-workflow-run' getter with parent tenant id", parentId, id)
		return nil, "", echo.NewHTTPError(http.StatusBadRequest, "This endpoint is deprecated.")
	})

	populatorMW.RegisterGetter("scheduled-workflow-run", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		idUuid, err := uuid.Parse(id)
		if err != nil {
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid scheduled workflow run id")
		}

		parentIdUuid, err := uuid.Parse(parentId)
		if err != nil {
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid tenant id")
		}

		scheduled, err := config.V1.WorkflowSchedules().GetScheduledWorkflow(context.Background(), parentIdUuid, idUuid)

		if err != nil {
			return nil, "", err
		}

		if scheduled == nil {
			return nil, "", echo.NewHTTPError(http.StatusNotFound, "scheduled workflow run not found")
		}

		return scheduled, scheduled.TenantId.String(), nil
	})

	populatorMW.RegisterGetter("cron-workflow", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		idUuid, err := uuid.Parse(id)
		if err != nil {
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid cron workflow id")
		}

		parentIdUuid, err := uuid.Parse(parentId)
		if err != nil {
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid tenant id")
		}

		scheduled, err := config.V1.WorkflowSchedules().GetCronWorkflow(context.Background(), parentIdUuid, idUuid)

		if err != nil {
			return nil, "", err
		}

		if scheduled == nil {
			return nil, "", echo.NewHTTPError(http.StatusNotFound, "cron workflow not found")
		}

		return scheduled, scheduled.TenantId.String(), nil
	})

	populatorMW.RegisterGetter("step-run", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		config.Logger.Error().Msgf("deprecated call to step-run with parent id %s and id %s: use 'v1-task' getter with parent tenant id", parentId, id)
		return nil, "", echo.NewHTTPError(http.StatusBadRequest, "This endpoint is deprecated.")
	})

	populatorMW.RegisterGetter("event", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		idUuid, err := uuid.Parse(id)

		if err != nil {
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
		}

		v1Event, err := t.config.V1.OLAP().GetEvent(timeoutCtx, idUuid)

		if err != nil {
			return nil, "", err
		}

		payload, err := t.config.V1.OLAP().ReadPayload(timeoutCtx, v1Event.TenantID, v1Event.ExternalID)

		if err != nil {
			return nil, "", err
		}

		event := &sqlcv1.Event{
			ID:                 v1Event.ExternalID,
			TenantId:           v1Event.TenantID,
			Data:               payload,
			CreatedAt:          pgtype.Timestamp(v1Event.SeenAt),
			AdditionalMetadata: v1Event.AdditionalMetadata,
			Key:                v1Event.Key,
		}

		return event, event.TenantId.String(), nil
	})

	// note: this is a hack to allow for the v0 event getter to use the pk on the v1 event lookup table
	populatorMW.RegisterGetter("event-with-tenant", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		idUuid, err := uuid.Parse(id)

		if err != nil {
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
		}

		parentIdUuid, err := uuid.Parse(parentId)

		if err != nil {
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid tenant id")
		}

		v1Event, err := t.config.V1.OLAP().GetEventWithPayload(timeoutCtx, idUuid, parentIdUuid)

		if err != nil {
			return nil, "", err
		}

		event := &sqlcv1.Event{
			ID:                 v1Event.EventExternalID,
			TenantId:           v1Event.TenantID,
			Data:               v1Event.Payload,
			CreatedAt:          pgtype.Timestamp(v1Event.EventSeenAt),
			AdditionalMetadata: v1Event.EventAdditionalMetadata,
			Key:                v1Event.EventKey,
		}

		return event, event.TenantId.String(), nil
	})

	populatorMW.RegisterGetter("worker", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		idUuid, err := uuid.Parse(id)

		if err != nil {
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid worker id")
		}

		parentIdUuid, err := uuid.Parse(parentId)

		if err != nil {
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid tenant id")
		}

		worker, err := config.V1.Workers().GetWorkerById(ctx, parentIdUuid, idUuid)

		if err != nil {
			return nil, "", err
		}

		return worker, worker.Worker.TenantId.String(), nil
	})

	populatorMW.RegisterGetter("webhook", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		config.Logger.Error().Msgf("deprecated call to webhook with parent id %s and id %s: do not use", parentId, id)
		return nil, "", echo.NewHTTPError(http.StatusBadRequest, "This endpoint is deprecated.")
	})

	populatorMW.RegisterGetter("task", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Validate UUID early to avoid panics deeper in the stack.
		var taskID uuid.UUID
		if err := taskID.Scan(id); err != nil {
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid task id")
		}

		task, err := config.V1.OLAP().ReadTaskRun(ctx, taskID)

		if err != nil {
			return nil, "", err
		}

		if task == nil {
			return nil, "", echo.NewHTTPError(http.StatusNotFound, "task not found")
		}

		return task, task.TenantID.String(), nil
	})

	populatorMW.RegisterGetter("v1-workflow-run", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		// Validate UUID early to avoid panics deeper in the stack.
		var workflowRunID uuid.UUID
		if err := workflowRunID.Scan(id); err != nil {
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid workflow run id")
		}

		workflowRun, err := t.config.V1.OLAP().ReadWorkflowRun(context.Background(), workflowRunID)

		if err != nil {
			return nil, "", err
		}

		return workflowRun, workflowRun.WorkflowRun.TenantID.String(), nil
	})

	populatorMW.RegisterGetter("v1-filter", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		idUuid, err := uuid.Parse(id)

		if err != nil {
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid filter id")
		}

		parentIdUuid, err := uuid.Parse(parentId)

		if err != nil {
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid tenant id")
		}

		filter, err := t.config.V1.Filters().GetFilter(
			context.Background(),
			parentIdUuid,
			idUuid,
		)

		if err != nil {
			return nil, "", err
		}

		return filter, filter.TenantID.String(), nil
	})

	populatorMW.RegisterGetter("v1-event", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		idUuid, err := uuid.Parse(id)

		if err != nil {
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
		}

		parentIdUuid, err := uuid.Parse(parentId)

		if err != nil {
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid tenant id")
		}

		event, err := t.config.V1.OLAP().GetEventWithPayload(
			context.Background(),
			idUuid,
			parentIdUuid,
		)

		if err != nil {
			return nil, "", err
		}

		return event, event.TenantID.String(), nil
	})

	populatorMW.RegisterGetter("v1-webhook", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		parentIdUuid, err := uuid.Parse(parentId)

		if err != nil {
			return nil, "", echo.NewHTTPError(http.StatusBadRequest, "invalid tenant id")
		}

		webhook, err := t.config.V1.Webhooks().GetWebhook(
			context.Background(),
			parentIdUuid,
			id,
		)

		if err != nil {
			return nil, "", err
		}

		return webhook, webhook.TenantID.String(), nil
	})

	authnMW := authn.NewAuthN(t.config)
	authzMW := authz.NewAuthZ(t.config)

	mw, err := hatchetmiddleware.NewMiddlewareHandler(spec)

	if err != nil {
		return nil, err
	}
	mw.Use(headers.Middleware())
	mw.Use(populatorMW.Middleware)
	mw.Use(authnMW.Middleware)
	mw.Use(authzMW.Middleware)
	for _, m := range t.additionalMiddlewares {
		mw.Use(m)
	}

	allHatchetMiddleware, err := mw.Middleware()

	if err != nil {
		return nil, err
	}

	loggerMiddleware := middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:       true,
		LogStatus:    true,
		LogError:     true,
		LogLatency:   true,
		LogRemoteIP:  true,
		LogHost:      true,
		LogMethod:    true,
		LogURIPath:   true,
		LogUserAgent: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			statusCode := v.Status

			// note that the status code is not set yet as it gets picked up by the global err handler
			// see here: https://github.com/labstack/echo/issues/2310#issuecomment-1288196898
			if v.Error != nil && statusCode == 200 {
				statusCode = 500
			}

			var e *zerolog.Event

			switch {
			case statusCode >= 500:
				e = t.config.Logger.Error().Err(v.Error)
			case statusCode >= 400:
				e = t.config.Logger.Warn()
			default:
				e = t.config.Logger.Info()
			}

			e.
				Dur("latency", v.Latency).
				Int("status", statusCode).
				Str("method", v.Method).
				Str("uri", v.URI).
				Str("user_agent", v.UserAgent).
				Str("remote_ip", v.RemoteIP).
				Str("host", v.Host).
				Msg("API")

			return nil
		},
	})

	rateLimitMW := ratelimit.NewRateLimitMiddleware(t.config, spec)
	otelMW := telemetry.NewOTelMiddleware(t.config)

	// register echo middleware
	g.Use(
		loggerMiddleware,
		middleware.Recover(),
		rateLimitMW.Middleware(),
		otelMW.Middleware(),
		allHatchetMiddleware,
	)

	return populatorMW, nil
}
