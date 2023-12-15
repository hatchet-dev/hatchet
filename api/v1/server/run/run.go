package run

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
	"github.com/hatchet-dev/hatchet/api/v1/server/authz"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/events"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/tenants"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/users"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/workers"
	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/workflows"
	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/config/server"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	hatchetmiddleware "github.com/hatchet-dev/hatchet/api/v1/server/middleware"
)

type apiService struct {
	*users.UserService
	*tenants.TenantService
	*events.EventService
	*workflows.WorkflowService
	*workers.WorkerService
}

func newAPIService(config *server.ServerConfig) *apiService {
	return &apiService{
		UserService:     users.NewUserService(config),
		TenantService:   tenants.NewTenantService(config),
		EventService:    events.NewEventService(config),
		WorkflowService: workflows.NewWorkflowService(config),
		WorkerService:   workers.NewWorkerService(config),
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

func (t *APIServer) Run() error {
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

	if err := e.Start(fmt.Sprintf(":%d", t.config.Runtime.Port)); err != nil {
		return err
	}

	return nil
}

// func IDGetter[T any](getter func(id string) (T, error), parentGetter func(val T) string) populator.PopulatorFunc {
// 	return func(config *server.ServerConfig, parent *populator.PopulatedResourceNode, id string) (res *populator.PopulatorResult, err error) {
// 		gotVal, err := getter(id)
// 		if err != nil {
// 			return nil, err
// 		}

// 		res = &populator.PopulatorResult{
// 			Resource: gotVal,
// 		}

// 		if parentGetter != nil {
// 			res.ParentID = parentGetter(gotVal)
// 		}

// 		return res, nil
// 	}
// }
