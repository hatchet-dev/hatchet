// Package grpcoperator serves v1.OperatorService, the engine API for operators that run outside
// the engine process (kind = GRPC in v1_operator). It is the gRPC adapter over
// internal/services/operatorsvc, which holds the session logic every operator host shares: this
// package parses the protocol, authenticates the caller, and runs the Listen message loop, and
// delegates the durable task stream to the dispatcher.
package grpcoperator

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1/v1connect"
	"github.com/hatchet-dev/hatchet/internal/services/shared/rpcstream"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

// OperatorIdMetadataKey is the incoming gRPC metadata key that carries the operator id on every
// RPC after Register (Listen, SendStepActionEvent, DurableTask).
const OperatorIdMetadataKey = "hatchet-operator-id"

type OperatorService interface {
	v1connect.OperatorServiceHandler
	Cleanup() error
}

// durableTaskDelegate is the dispatcher's own durable task session. The operator service
// authorizes the stream and reads its first message, then the dispatcher owns the session.
type durableTaskDelegate interface {
	DurableTaskWithReceive(ctx context.Context, receive func() (*v1contracts.DurableTaskRequest, error), sender *rpcstream.Sender[v1contracts.DurableTaskResponse]) error
}

type OperatorServiceImpl struct {
	v1connect.UnimplementedOperatorServiceHandler

	svc     *operatorsvc.Service
	durable durableTaskDelegate
	l       *zerolog.Logger
}

type OperatorServiceOpt func(*OperatorServiceOpts)

type OperatorServiceOpts struct {
	repo      repository.Repository
	analytics analytics.Analytics
	v         validator.Validator

	dispatcher *dispatcher.DispatcherImpl
	l          *zerolog.Logger

	maxListenStreamsPerOperator int
	maxActionsPerOperator       int64
}

func defaultOperatorServiceOpts() *OperatorServiceOpts {
	l := logger.NewDefaultLogger("grpc_operator_service")

	return &OperatorServiceOpts{
		v:                           validator.NewDefaultValidator(),
		analytics:                   analytics.NoOpAnalytics{},
		l:                           &l,
		maxListenStreamsPerOperator: operatorsvc.DefaultMaxListenStreamsPerOperator,
		maxActionsPerOperator:       operatorsvc.DefaultMaxActionsPerOperator,
	}
}

// WithMaxListenStreamsPerOperator caps the Listen streams one operator may hold open on this
// replica; zero disables the cap.
func WithMaxListenStreamsPerOperator(n int) OperatorServiceOpt {
	return func(opts *OperatorServiceOpts) {
		opts.maxListenStreamsPerOperator = n
	}
}

// WithMaxActionsPerOperator caps the action links held across all workers of one operator;
// zero disables the cap.
func WithMaxActionsPerOperator(n int64) OperatorServiceOpt {
	return func(opts *OperatorServiceOpts) {
		opts.maxActionsPerOperator = n
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

	svc, err := operatorsvc.New(
		operatorsvc.WithOperatorStore(opts.repo.Operators()),
		operatorsvc.WithWorkerStore(opts.repo.Workers()),
		operatorsvc.WithDispatcherBackend(operatorsvc.NewDispatcherBackend(opts.dispatcher)),
		operatorsvc.WithDispatcherId(opts.dispatcher.DispatcherId()),
		operatorsvc.WithLogger(opts.l),
		operatorsvc.WithValidator(opts.v),
		operatorsvc.WithAnalytics(opts.analytics),
		operatorsvc.WithMaxListenStreamsPerOperator(opts.maxListenStreamsPerOperator),
		operatorsvc.WithMaxActionsPerOperator(opts.maxActionsPerOperator),
	)

	if err != nil {
		return nil, err
	}

	return &OperatorServiceImpl{
		svc:     svc,
		durable: opts.dispatcher.V1(),
		l:       &newLogger,
	}, nil
}

// Cleanup releases what the service holds between requests.
func (s *OperatorServiceImpl) Cleanup() error {
	return s.svc.Cleanup()
}

// tenantFromContext returns the tenant attached by the auth middleware.
func tenantFromContext(ctx context.Context) (*sqlcv1.Tenant, bool) {
	tenant, ok := ctx.Value("tenant").(*sqlcv1.Tenant)

	return tenant, ok && tenant != nil
}

// authorizeOperator resolves the calling operator from the hatchet-operator-id request header
// and the token's tenant. Parsing the header is this package's job; deciding whether the
// operator may be used is the service's.
func (s *OperatorServiceImpl) authorizeOperator(ctx context.Context) (*sqlcv1.Tenant, *sqlcv1.V1Operator, error) {
	tenant, ok := tenantFromContext(ctx)

	if !ok {
		return nil, nil, connect.NewError(connect.CodeUnauthenticated, errors.New("tenant not found in request context"))
	}

	value := ""

	if info, ok := connect.CallInfoForHandlerContext(ctx); ok {
		value = info.RequestHeader().Get(OperatorIdMetadataKey)
	}

	if value == "" {
		return nil, nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("missing %s metadata", OperatorIdMetadataKey))
	}

	operatorId, err := uuid.Parse(value)

	if err != nil {
		return nil, nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid %s metadata: %s is not a uuid", OperatorIdMetadataKey, value))
	}

	op, err := s.svc.AuthorizeOperator(ctx, tenant, operatorId)

	if err != nil {
		return nil, nil, err
	}

	return tenant, op, nil
}

// authorizeWorker checks that workerIdStr names a worker owned by op.
func (s *OperatorServiceImpl) authorizeWorker(ctx context.Context, tenant *sqlcv1.Tenant, op *sqlcv1.V1Operator, workerIdStr string) (*sqlcv1.GetWorkerForEngineRow, error) {
	workerId, err := uuid.Parse(workerIdStr)

	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid worker ID format: %s", workerIdStr))
	}

	return s.svc.AuthorizeWorker(ctx, tenant, op.ID, workerId)
}
