package run

import (
	"context"
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
	"github.com/hatchet-dev/hatchet/api/v1/server/authz"
	apitokens "github.com/hatchet-dev/hatchet/api/v1/server/handlers/api-tokens"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/events"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/metadata"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/tenants"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/users"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/workers"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/workflows"
	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/config/server"

	hatchetmiddleware "github.com/hatchet-dev/hatchet/api/v1/server/middleware"
)

type apiService struct {
	*users.UserService
	*tenants.TenantService
	*events.EventService
	*workflows.WorkflowService
	*workers.WorkerService
	*metadata.MetadataService
	*apitokens.APITokenService
}

func newAPIService(config *server.ServerConfig) *apiService {
	return &apiService{
		UserService:     users.NewUserService(config),
		TenantService:   tenants.NewTenantService(config),
		EventService:    events.NewEventService(config),
		WorkflowService: workflows.NewWorkflowService(config),
		WorkerService:   workers.NewWorkerService(config),
		MetadataService: metadata.NewMetadataService(config),
		APITokenService: apitokens.NewAPITokenService(config),
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

func (t *APIServer) Run(ctx context.Context) error {
	errCh := make(chan error)

	oaspec, err := gen.GetSwagger()
	if err != nil {
		return err
	}

	e := echo.New()

	// application middleware
	populatorMW := populator.NewPopulator(t.config)

	populatorMW.RegisterGetter("tenant", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		tenant, err := config.Repository.Tenant().GetTenantByID(id)

		if err != nil {
			return nil, "", err
		}

		return tenant, "", nil
	})

	populatorMW.RegisterGetter("api-token", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		apiToken, err := config.Repository.APIToken().GetAPITokenById(id)

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
		tenantInvite, err := config.Repository.TenantInvite().GetTenantInvite(id)

		if err != nil {
			return nil, "", err
		}

		return tenantInvite, tenantInvite.TenantID, nil
	})

	populatorMW.RegisterGetter("workflow", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		workflow, err := config.Repository.Workflow().GetWorkflowById(id)

		if err != nil {
			return nil, "", err
		}

		return workflow, workflow.TenantID, nil
	})

	populatorMW.RegisterGetter("workflow-run", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		workflowRun, err := config.Repository.WorkflowRun().GetWorkflowRunById(parentId, id)

		if err != nil {
			return nil, "", err
		}

		return workflowRun, workflowRun.TenantID, nil
	})

	populatorMW.RegisterGetter("event", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		event, err := config.Repository.Event().GetEventById(id)

		if err != nil {
			return nil, "", err
		}

		return event, event.TenantID, nil
	})

	populatorMW.RegisterGetter("worker", func(config *server.ServerConfig, parentId, id string) (result interface{}, uniqueParentId string, err error) {
		worker, err := config.Repository.Worker().GetWorkerById(id)

		if err != nil {
			return nil, "", err
		}

		return worker, worker.TenantID, nil
	})

	authnMW := authn.NewAuthN(t.config)
	authzMW := authz.NewAuthZ(t.config)

	mw, err := hatchetmiddleware.NewMiddlewareHandler(oaspec)

	if err != nil {
		return err
	}

	mw.Use(populatorMW.Middleware)
	mw.Use(authnMW.Middleware)
	mw.Use(authzMW.Middleware)

	allHatchetMiddleware, err := mw.Middleware()

	if err != nil {
		return err
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
		if err := e.Start(fmt.Sprintf(":%d", t.config.Runtime.Port)); err != nil {
			errCh <- err
		}
	}()

Loop:
	for {
		select {
		case err := <-errCh:
			return err
		case <-ctx.Done():
			break Loop
		}
	}

	err = e.Shutdown(ctx)

	if err != nil {
		return err
	}

	return nil
}
