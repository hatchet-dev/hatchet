package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/internal/operator/claimer"
	"github.com/hatchet-dev/hatchet/internal/operator/hostinproc"
	adminv1 "github.com/hatchet-dev/hatchet/internal/services/admin/v1"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// operatorStopTimeout bounds the claimer's teardown of every hosted operator at shutdown.
const operatorStopTimeout = 60 * time.Second

// inprocOperators is the engine's in-process operator runtime: the one host every operator
// hosted in this dispatcher process opens its sessions on (the claimed DAG operators and the
// serverless operator alike), and the stop that pauses, drains and closes the claimed
// operators, then the host and its service. The serverless operator closes its own sessions
// before stop runs.
type inprocOperators struct {
	host *hostinproc.Host
	stop func() error
}

// startOperatorClaimer hosts the operators this dispatcher claims inside the engine process:
// the in-process host over the engine's operator session logic, and the claimer that opens
// the DAG operator on it for every claimed row. The stop it returns blocks until every claimed
// operator is torn down; the caller runs it before the dispatcher drains its workers so the
// operators' events still have somewhere to go.
func startOperatorClaimer(sc *server.ServerConfig, d *dispatcher.DispatcherImpl, adminv1Svc adminv1.AdminService) (*inprocOperators, error) {
	l := sc.Logger.With().Str("service", "operator-claimer").Logger()

	svc, err := operatorsvc.New(
		operatorsvc.WithOperatorStore(sc.V1.Operators()),
		operatorsvc.WithWorkerStore(sc.V1.Workers()),
		operatorsvc.WithDispatcherBackend(operatorsvc.NewDispatcherBackend(d)),
		operatorsvc.WithDispatcherId(d.DispatcherId()),
		operatorsvc.WithLogger(&l),
		operatorsvc.WithValidator(sc.Validator),
		operatorsvc.WithAnalytics(sc.Analytics),
		operatorsvc.WithMaxActionsPerOperator(sc.Runtime.GRPCOperatorMaxActionsPerOperator),
	)

	if err != nil {
		return nil, fmt.Errorf("could not create operator service: %w", err)
	}

	host, err := hostinproc.New(
		hostinproc.WithService(svc),
		hostinproc.WithTenantStore(sc.V1.Tenant()),
		hostinproc.WithHeartbeatStore(sc.V1.Workers()),
		hostinproc.WithAdminService(adminv1Svc),
		hostinproc.WithWorkflowStore(sc.V1.Workflows()),
		hostinproc.WithLogger(&l),
	)

	if err != nil {
		_ = svc.Cleanup()
		return nil, fmt.Errorf("could not create in-process operator host: %w", err)
	}

	c, err := claimer.New(
		claimer.WithHost(host),
		claimer.WithClaims(sc.V1.Operators()),
		claimer.WithDispatcherId(d.DispatcherId()),
		claimer.WithFactory(sqlcv1.V1OperatorKindDAG, claimer.DAGFactory(&l, sc.V1, d, sc.Runtime.DagOperatorDefaultSlots)),
		claimer.WithLogger(&l),
	)

	if err != nil {
		host.Close()
		_ = svc.Cleanup()

		return nil, fmt.Errorf("could not create operator claimer: %w", err)
	}

	// The claimer runs on its own context rather than the engine's so that shutdown is
	// ordered by the cleanup chain, not by the engine context's cancellation.
	c.Start(context.Background())

	return &inprocOperators{
		host: host,
		stop: func() error {
			ctx, cancel := context.WithTimeout(context.Background(), operatorStopTimeout)
			defer cancel()

			c.Stop(ctx)
			host.Close()

			return svc.Cleanup()
		},
	}, nil
}
