package run

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
	"github.com/hatchet-dev/hatchet/api/v1/server/authz"
	apitokens "github.com/hatchet-dev/hatchet/api/v1/server/handlers/api-tokens"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/events"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/ingestors"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/logs"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/metadata"
	slackapp "github.com/hatchet-dev/hatchet/api/v1/server/handlers/slack-app"
	stepruns "github.com/hatchet-dev/hatchet/api/v1/server/handlers/step-runs"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/tenants"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/users"
	webhookworker "github.com/hatchet-dev/hatchet/api/v1/server/handlers/webhook-worker"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/workers"
	workflowruns "github.com/hatchet-dev/hatchet/api/v1/server/handlers/workflow-runs"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/workflows"
	hatchetmiddleware "github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

type apiService struct {
	*users.UserService
	*tenants.TenantService
	*events.EventService
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
}

func newAPIService(config *server.ServerConfig) *apiService {
	return &apiService{
		UserService:           users.NewUserService(config),
		TenantService:         tenants.NewTenantService(config),
		EventService:          events.NewEventService(config),
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
	}
}

type APIServer struct {
	config *server.ServerConfig
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

	g := e.Group("")

	if _, err := t.registerSpec(g, oaspec); err != nil {
		return nil, err
	}

	service := newAPIService(t.config)

	myStrictApiHandler := gen.NewStrictHandler(service, []gen.StrictMiddlewareFunc{})

	gen.RegisterHandlers(g, myStrictApiHandler)

	return e, nil
}

func (t *APIServer) registerSpec(g *echo.Group, spec *openapi3.T) (*populator.Populator, error) {
	// application middleware
	populatorMW := populator.NewPopulator(t.config)

	populatorMW.RegisterGetter("tenant", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		tenant, err := config.APIRepository.Tenant().GetTenantByID(id)

		if err != nil {
			return nil, "", err
		}

		return tenant, "", nil
	})

	populatorMW.RegisterGetter("api-token", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		apiToken, err := config.APIRepository.APIToken().GetAPITokenById(id)

		if err != nil {
			return nil, "", err
		}

		// at the moment, API tokens should have a tenant id, because there are no other types of
		// API tokens. If we add other types of API tokens, we'll need to pass in a parent id to query
		// for.
		tenantId, ok := apiToken.TenantID()

		if !ok {
			return nil, "", fmt.Errorf("api token has no tenant id")
		}

		return apiToken, tenantId, nil
	})

	populatorMW.RegisterGetter("tenant-invite", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		tenantInvite, err := config.APIRepository.TenantInvite().GetTenantInvite(id)

		if err != nil {
			return nil, "", err
		}

		return tenantInvite, tenantInvite.TenantID, nil
	})

	populatorMW.RegisterGetter("slack", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		slackWebhook, err := config.APIRepository.Slack().GetSlackWebhookById(id)

		if err != nil {
			return nil, "", err
		}

		return slackWebhook, slackWebhook.TenantID, nil
	})

	populatorMW.RegisterGetter("alert-email-group", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		emailGroup, err := config.APIRepository.TenantAlertingSettings().GetTenantAlertGroupById(id)

		if err != nil {
			return nil, "", err
		}

		return emailGroup, emailGroup.TenantID, nil
	})

	populatorMW.RegisterGetter("sns", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		snsIntegration, err := config.APIRepository.SNS().GetSNSIntegrationById(id)

		if err != nil {
			return nil, "", err
		}

		return snsIntegration, snsIntegration.TenantID, nil
	})

	populatorMW.RegisterGetter("workflow", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		workflow, err := config.APIRepository.Workflow().GetWorkflowById(id)

		if err != nil {
			return nil, "", err
		}

		return workflow, workflow.TenantID, nil
	})

	populatorMW.RegisterGetter("workflow-run", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		workflowRun, err := config.APIRepository.WorkflowRun().GetWorkflowRunById(parentId, id)

		if err != nil {
			return nil, "", err
		}

		return workflowRun, workflowRun.TenantID, nil
	})

	populatorMW.RegisterGetter("step-run", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		stepRun, err := config.APIRepository.StepRun().GetStepRunById(parentId, id)

		if err != nil {
			return nil, "", err
		}

		return stepRun, stepRun.TenantID, nil
	})

	populatorMW.RegisterGetter("event", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		event, err := config.APIRepository.Event().GetEventById(id)

		if err != nil {
			return nil, "", err
		}

		return event, event.TenantID, nil
	})

	populatorMW.RegisterGetter("worker", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		worker, err := config.APIRepository.Worker().GetWorkerById(id)

		if err != nil {
			return nil, "", err
		}

		return worker, sqlchelpers.UUIDToStr(worker.Worker.TenantId), nil
	})

	populatorMW.RegisterGetter("webhook", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		webhookWorker, err := config.APIRepository.WebhookWorker().GetWebhookWorkerByID(id)
		if err != nil {
			return nil, "", err
		}

		return webhookWorker, webhookWorker.TenantID, nil
	})

	authnMW := authn.NewAuthN(t.config)
	authzMW := authz.NewAuthZ(t.config)

	mw, err := hatchetmiddleware.NewMiddlewareHandler(spec)

	if err != nil {
		return nil, err
	}

	mw.Use(populatorMW.Middleware)
	mw.Use(authnMW.Middleware)
	mw.Use(authzMW.Middleware)

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
			if v.Error != nil {
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

	// register echo middleware
	g.Use(
		loggerMiddleware,
		middleware.Recover(),
		allHatchetMiddleware,
	)

	return populatorMW, nil
}
