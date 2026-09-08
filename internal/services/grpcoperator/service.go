// Package grpcoperator serves v1.OperatorService, the engine API for operators that run outside
// the engine process (kind = GRPC in v1_operator). It owns operator and worker registration and
// the Listen stream, and delegates task-event and durable-task RPCs to the dispatcher.
package grpcoperator

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

// OperatorIdMetadataKey is the incoming gRPC metadata key that carries the operator id on every
// RPC after Register (Listen, SendStepActionEvent, DurableTask).
const OperatorIdMetadataKey = "hatchet-operator-id"

type OperatorService interface {
	v1contracts.OperatorServiceServer
	Cleanup() error
}

// operatorStore is the subset of repository.OperatorRepository the service uses. It is narrow
// so tests can substitute a double without stubbing the whole repository tree.
type operatorStore interface {
	GetOperatorById(ctx context.Context, operatorId uuid.UUID) (*sqlcv1.V1Operator, error)
	UpsertGRPCOperator(ctx context.Context, tenantId uuid.UUID, name string) (*sqlcv1.V1Operator, error)
}

// workerStore is the subset of repository.WorkerRepository the service uses.
type workerStore interface {
	CreateNewWorker(ctx context.Context, tenantId uuid.UUID, opts *repository.CreateWorkerOpts) (*sqlcv1.Worker, error)
	GetWorkerForEngine(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID) (*sqlcv1.GetWorkerForEngineRow, error)
	UpdateWorker(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, opts *repository.UpdateWorkerOpts) (*sqlcv1.Worker, error)
	ActivateWorkerListener(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, sessionId uuid.UUID) (*sqlcv1.Worker, error)
	DeactivateWorkerListener(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, sessionId uuid.UUID) (*sqlcv1.Worker, error)
	UpdateWorkerHeartbeat(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, lastHeartbeatAt time.Time) error
	UpsertWorkerLabels(ctx context.Context, workerId uuid.UUID, opts []repository.UpsertWorkerLabelOpts) ([]*sqlcv1.WorkerLabel, error)
	AddWorkerActions(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, actionIds []string) (int, error)
	RemoveWorkerActions(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, actionIds []string) (int, error)
}

// dispatcherBackend is what the service needs from the local dispatcher: session registration
// for the Listen stream, scheduler notification, and the two delegated RPCs.
type dispatcherBackend interface {
	AddOperatorStreamSession(workerId uuid.UUID, sessionId uuid.UUID, stream grpc.ServerStream, wrap func(*contracts.AssignedAction) proto.Message) operatorStreamSession
	NotifyNewWorker(ctx context.Context, tenant *sqlcv1.Tenant, workerId uuid.UUID)
	SendStepActionEvent(ctx context.Context, req *contracts.StepActionEvent) (*contracts.ActionEventResponse, error)
	DurableTask(stream v1contracts.V1Dispatcher_DurableTaskServer) error
}

// operatorStreamSession is the dispatcher session handle for one Listen stream; see
// dispatcher.OperatorStreamSession.
type operatorStreamSession interface {
	Fin() <-chan bool
	Send(ctx context.Context, msg proto.Message) error
	Release()
}

// dispatcherAdapter presents *dispatcher.DispatcherImpl as a dispatcherBackend; the durable
// task stream lives on the dispatcher's V1 service.
type dispatcherAdapter struct {
	*dispatcher.DispatcherImpl
}

func (a dispatcherAdapter) AddOperatorStreamSession(workerId uuid.UUID, sessionId uuid.UUID, stream grpc.ServerStream, wrap func(*contracts.AssignedAction) proto.Message) operatorStreamSession {
	return a.DispatcherImpl.AddOperatorStreamSession(workerId, sessionId, stream, wrap)
}

func (a dispatcherAdapter) DurableTask(stream v1contracts.V1Dispatcher_DurableTaskServer) error {
	return a.V1().DurableTask(stream)
}

type OperatorServiceImpl struct {
	v1contracts.UnimplementedOperatorServiceServer

	operators operatorStore
	workers   workerStore
	cache     cache.Cacheable
	analytics analytics.Analytics
	v         validator.Validator

	dispatcher dispatcherBackend
	l          *zerolog.Logger

	// dispatcherId is the id of the local dispatcher; every operator worker registered through
	// this service is pinned to it, since the Listen stream that delivers actions lives here.
	dispatcherId uuid.UUID

	// notifyInterval throttles scheduler notifications while action deltas keep arriving on a
	// Listen stream; see defaultNotifyInterval.
	notifyInterval time.Duration
}

type OperatorServiceOpt func(*OperatorServiceOpts)

type OperatorServiceOpts struct {
	repo      repository.Repository
	analytics analytics.Analytics
	v         validator.Validator

	dispatcher *dispatcher.DispatcherImpl
	l          *zerolog.Logger
}

func defaultOperatorServiceOpts() *OperatorServiceOpts {
	l := logger.NewDefaultLogger("grpc_operator_service")

	return &OperatorServiceOpts{
		v:         validator.NewDefaultValidator(),
		analytics: analytics.NoOpAnalytics{},
		l:         &l,
	}
}

func WithDispatcher(d *dispatcher.DispatcherImpl) OperatorServiceOpt {
	return func(opts *OperatorServiceOpts) {
		opts.dispatcher = d
	}
}

func WithRepository(r repository.Repository) OperatorServiceOpt {
	return func(opts *OperatorServiceOpts) {
		opts.repo = r
	}
}

func WithLogger(l *zerolog.Logger) OperatorServiceOpt {
	return func(opts *OperatorServiceOpts) {
		opts.l = l
	}
}

func WithAnalytics(a analytics.Analytics) OperatorServiceOpt {
	return func(opts *OperatorServiceOpts) {
		opts.analytics = a
	}
}

func WithValidator(v validator.Validator) OperatorServiceOpt {
	return func(opts *OperatorServiceOpts) {
		opts.v = v
	}
}

func New(fs ...OperatorServiceOpt) (*OperatorServiceImpl, error) {
	opts := defaultOperatorServiceOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.dispatcher == nil {
		return nil, fmt.Errorf("dispatcher is required. use WithDispatcher")
	}

	if opts.repo == nil {
		return nil, fmt.Errorf("repository is required. use WithRepository")
	}

	newLogger := opts.l.With().Str("service", "grpc_operator_service").Logger()

	return &OperatorServiceImpl{
		dispatcher:   dispatcherAdapter{opts.dispatcher},
		dispatcherId: opts.dispatcher.DispatcherId(),
		operators:    opts.repo.Operators(),
		workers:      opts.repo.Workers(),
		cache:        cache.New(60 * time.Second),
		l:            &newLogger,
		analytics:    opts.analytics,
		v:            opts.v,

		notifyInterval: defaultNotifyInterval,
	}, nil
}

// Cleanup stops the operator cache's expiry goroutine.
func (s *OperatorServiceImpl) Cleanup() error {
	if s.cache != nil {
		s.cache.Stop()
	}

	return nil
}

// tenantFromContext returns the tenant attached by the auth middleware.
func tenantFromContext(ctx context.Context) (*sqlcv1.Tenant, bool) {
	tenant, ok := ctx.Value("tenant").(*sqlcv1.Tenant)

	return tenant, ok && tenant != nil
}
