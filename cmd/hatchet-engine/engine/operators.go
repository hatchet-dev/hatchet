package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/operator/claimer"
	"github.com/hatchet-dev/hatchet/internal/operator/hostinproc"
	adminv1 "github.com/hatchet-dev/hatchet/internal/services/admin/v1"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// operatorStopTimeout bounds the claimer's teardown of every hosted operator at shutdown.
const operatorStopTimeout = 60 * time.Second

// startOperatorClaimer hosts the operators this dispatcher claims inside the engine process:
// the in-process host over the engine's operator session logic, and the claimer that opens
// the DAG operator on it for every claimed row. It returns a stop function that pauses,
// drains and closes every hosted operator and blocks until it has; the caller runs it before
// the dispatcher drains its workers so the operators' events still have somewhere to go.
func startOperatorClaimer(sc *server.ServerConfig, d *dispatcher.DispatcherImpl, adminv1Svc adminv1.AdminService) (stop func() error, err error) {
	l := sc.Logger.With().Str("service", "operator-claimer").Logger()

	svc, err := operatorsvc.New(
		operatorsvc.Deps{
			Operators:    sc.V1.Operators(),
			Workers:      sc.V1.Workers(),
			Dispatcher:   operatorsvc.NewDispatcherBackend(d),
			DispatcherId: d.DispatcherId(),
		},
		operatorsvc.WithLogger(&l),
		operatorsvc.WithValidator(sc.Validator),
		operatorsvc.WithAnalytics(sc.Analytics),
		operatorsvc.WithMaxActionsPerOperator(sc.Runtime.GRPCOperatorMaxActionsPerOperator),
	)

	if err != nil {
		return nil, fmt.Errorf("could not create operator service: %w", err)
	}

	host, err := hostinproc.New(hostinproc.Deps{
		Service:    svc,
		Tenants:    sc.V1.Tenant(),
		Heartbeats: sc.V1.Workers(),
		Workflows:  inprocWorkflows{admin: adminv1Svc, workflows: sc.V1.Workflows()},
		Logger:     &l,
	})

	if err != nil {
		_ = svc.Cleanup()
		return nil, fmt.Errorf("could not create in-process operator host: %w", err)
	}

	c, err := claimer.New(claimer.Deps{
		Host:         host,
		Claims:       sc.V1.Operators(),
		DispatcherId: d.DispatcherId(),
		Factories: map[sqlcv1.V1OperatorKind]claimer.Factory{
			sqlcv1.V1OperatorKindDAG: claimer.DAGFactory(&l, sc.V1, d, sc.Runtime.DagOperatorDefaultSlots),
		},
		Logger: &l,
	})

	if err != nil {
		host.Close()
		_ = svc.Cleanup()

		return nil, fmt.Errorf("could not create operator claimer: %w", err)
	}

	// The claimer runs on its own context rather than the engine's so that shutdown is
	// ordered by the cleanup chain, not by the engine context's cancellation.
	c.Start(context.Background())

	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), operatorStopTimeout)
		defer cancel()

		c.Stop(ctx)
		host.Close()

		return svc.Cleanup()
	}, nil
}

// inprocWorkflows is the in-process host's workflow store: the admin service puts the
// workflow and the repository lists the stored steps of the version it created.
type inprocWorkflows struct {
	admin     adminv1.AdminService
	workflows repository.WorkflowRepository
}

func (w inprocWorkflows) PutWorkflow(ctx context.Context, req *v1.CreateWorkflowVersionRequest) (*v1.CreateWorkflowVersionResponse, error) {
	return w.admin.PutWorkflow(ctx, req)
}

func (w inprocWorkflows) ListStepsByWorkflowVersionId(ctx context.Context, tenantId uuid.UUID, workflowVersionId uuid.UUID) ([]*sqlcv1.ListStepsByWorkflowVersionIdsRow, error) {
	return w.workflows.ListStepsByWorkflowVersionId(ctx, tenantId, workflowVersionId)
}
