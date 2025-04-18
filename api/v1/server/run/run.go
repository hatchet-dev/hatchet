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
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/v1/tasks"
	workflowrunsv1 "github.com/hatchet-dev/hatchet/api/v1/server/handlers/v1/workflow-runs"
	webhookworker "github.com/hatchet-dev/hatchet/api/v1/server/handlers/webhook-worker"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/workers"
	workflowruns "github.com/hatchet-dev/hatchet/api/v1/server/handlers/workflow-runs"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/workflows"
	"github.com/hatchet-dev/hatchet/api/v1/server/headers"
	hatchetmiddleware "github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
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
	e.HideBanner = true
	e.HidePort = true

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
