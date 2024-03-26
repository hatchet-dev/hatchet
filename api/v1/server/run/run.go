package run

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
	"github.com/hatchet-dev/hatchet/api/v1/server/authz"
	apitokens "github.com/hatchet-dev/hatchet/api/v1/server/handlers/api-tokens"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/events"
	githubapp "github.com/hatchet-dev/hatchet/api/v1/server/handlers/github-app"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/ingestors"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/logs"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/metadata"
	stepruns "github.com/hatchet-dev/hatchet/api/v1/server/handlers/step-runs"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/tenants"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/users"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/workers"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/workflows"
	hatchetmiddleware "github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/config/server"
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
	*githubapp.GithubAppService
	*ingestors.IngestorsService
}

func newAPIService(config *server.ServerConfig) *apiService {
	return &apiService{
		UserService:      users.NewUserService(config),
		TenantService:    tenants.NewTenantService(config),
		EventService:     events.NewEventService(config),
		LogService:       logs.NewLogService(config),
		WorkflowService:  workflows.NewWorkflowService(config),
		WorkerService:    workers.NewWorkerService(config),
		MetadataService:  metadata.NewMetadataService(config),
		APITokenService:  apitokens.NewAPITokenService(config),
		StepRunService:   stepruns.NewStepRunService(config),
		GithubAppService: githubapp.NewGithubAppService(config),
		IngestorsService: ingestors.NewIngestorsService(config),
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

func (t *APIServer) Run() (func() error, error) {
	oaspec, err := gen.GetSwagger()
	if err != nil {
		return nil, err
	}

	e := echo.New()

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

		return worker, worker.TenantID, nil
	})

	populatorMW.RegisterGetter("gh-installation", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		ghInstallation, err := config.APIRepository.Github().ReadGithubAppInstallationByID(id)

		if err != nil {
			return nil, "", err
		}

		return ghInstallation, "", nil
	})

	authnMW := authn.NewAuthN(t.config)
	authzMW := authz.NewAuthZ(t.config)

	mw, err := hatchetmiddleware.NewMiddlewareHandler(oaspec)

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

	// register echo middleware
	e.Use(
		middleware.Logger(),
		middleware.Recover(),
		allHatchetMiddleware,
	)

	service := newAPIService(t.config)

	myStrictApiHandler := gen.NewStrictHandler(service, []gen.StrictMiddlewareFunc{})

	gen.RegisterHandlers(e, myStrictApiHandler)

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
