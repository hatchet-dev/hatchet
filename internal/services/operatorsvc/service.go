// Package operatorsvc holds the engine-side logic of an operator session: registering the
// operator and its worker, opening and closing the session that makes the worker live,
// applying action set deltas, authorizing an operator and its workers, and opening durable
// invocations.
//
// It is shared by every operator host. internal/services/grpcoperator serves it over
// OperatorService to operators that run outside the engine, and the in-process host will call
// the same functions for operators hosted inside the dispatcher, so the two hosts differ only
// in how assigned actions reach the operator and in how the caller is authenticated.
//
// The API takes no OperatorService protocol messages: callers pass plain values and get
// repository rows back. Errors carry gRPC status codes because they are returned to gRPC
// callers unchanged; in-process callers can ignore the codes.
package operatorsvc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

const (
	// DefaultMaxListenStreamsPerOperator caps the Listen streams one operator may hold open on
	// this engine replica at once. Every stream is a live worker with its own session and
	// action set, so the cap bounds what one tenant token can allocate here.
	DefaultMaxListenStreamsPerOperator = 100

	// DefaultMaxActionsPerOperator caps the action links held across all workers of one
	// operator. Deltas that would exceed it are refused with ResourceExhausted.
	DefaultMaxActionsPerOperator = 1_000_000

	// MaxActionsPerDelta caps the ids (adds plus removes) one delta may carry.
	MaxActionsPerDelta = 1000

	// defaultNotifyInterval throttles scheduler notifications while action deltas keep
	// arriving: the scheduler reloads the worker's action set on each notify, so a burst of
	// deltas is folded into one reload per second.
	defaultNotifyInterval = time.Second

	// defaultOperatorCacheTTL is how long an authorized operator row is reused.
	defaultOperatorCacheTTL = 60 * time.Second

	// heartbeatWriteInterval throttles heartbeat writes to the database; hosts heartbeat every
	// 4 seconds, so this only matters for misbehaving ones.
	heartbeatWriteInterval = time.Second

	// deactivateTimeout bounds the detached deactivation write once the session is gone.
	deactivateTimeout = 20 * time.Second
)

// OperatorStore is the subset of repository.OperatorRepository the service uses. It is narrow
// so tests can substitute a double without stubbing the whole repository tree.
type OperatorStore interface {
	GetOperatorById(ctx context.Context, operatorId uuid.UUID) (*sqlcv1.V1Operator, error)
	UpsertGRPCOperator(ctx context.Context, tenantId uuid.UUID, name string) (*sqlcv1.V1Operator, error)
}

// WorkerStore is the subset of repository.WorkerRepository the service uses.
type WorkerStore interface {
	CreateNewWorker(ctx context.Context, tenantId uuid.UUID, opts *repository.CreateWorkerOpts) (*sqlcv1.Worker, error)
	GetWorkerForEngine(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID) (*sqlcv1.GetWorkerForEngineRow, error)
	UpdateWorker(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, opts *repository.UpdateWorkerOpts) (*sqlcv1.Worker, error)
	ActivateWorkerListener(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, sessionId uuid.UUID) (*sqlcv1.Worker, error)
	DeactivateWorkerListener(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, sessionId uuid.UUID) (*sqlcv1.Worker, error)
	UpdateWorkerHeartbeat(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, lastHeartbeatAt time.Time) error
	UpsertWorkerLabels(ctx context.Context, workerId uuid.UUID, opts []repository.UpsertWorkerLabelOpts) ([]*sqlcv1.WorkerLabel, error)
	AddWorkerActionsWithinBudget(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, actionIds []string, maxNewLinks int64) (int, error)
	RemoveWorkerActions(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, actionIds []string) (int, error)
	CountOperatorWorkerActions(ctx context.Context, tenantId uuid.UUID, operatorId uuid.UUID) (int64, error)
}

// ActionHandler receives assigned actions by direct call. It is the in-process delivery: the
// call runs on the dispatcher's delivery goroutine and its error requeues the task.
type ActionHandler interface {
	HandleAction(ctx context.Context, action *contracts.AssignedAction) error
}

// StreamSession is the dispatcher session handle for a stream-backed session; see
// dispatcher.OperatorStreamSession.
type StreamSession interface {
	Fin() <-chan bool
	Send(ctx context.Context, msg proto.Message) error
	Release()
}

// HandlerSession is the dispatcher session handle for a handler-backed session; see
// dispatcher.OperatorHandlerSession.
type HandlerSession interface {
	Release()
}

// DispatcherBackend is what the service needs from the local dispatcher: session registration
// for either delivery, scheduler notification, task events, and the channel-backed durable
// session.
type DispatcherBackend interface {
	AddOperatorStreamSession(workerId uuid.UUID, sessionId uuid.UUID, stream grpc.ServerStream, wrap func(*contracts.AssignedAction) proto.Message) StreamSession
	AddOperatorSession(workerId uuid.UUID, sessionId uuid.UUID, handler ActionHandler) HandlerSession
	NotifyNewWorker(ctx context.Context, tenant *sqlcv1.Tenant, workerId uuid.UUID)
	SendStepActionEvent(ctx context.Context, req *contracts.StepActionEvent) (*contracts.ActionEventResponse, error)
	RegisterDurableTask(ctx context.Context, externalId uuid.UUID) (chan<- *v1contracts.DurableTaskRequest, <-chan *v1contracts.DurableTaskResponse, error)
}

// Deps are the collaborators a Service cannot run without.
type Deps struct {
	Operators  OperatorStore
	Workers    WorkerStore
	Dispatcher DispatcherBackend

	// DispatcherId is the id of the local dispatcher; every operator worker whose session is
	// opened here is pinned to it, since the delivery path lives here.
	DispatcherId uuid.UUID
}

type Service struct {
	operators  OperatorStore
	workers    WorkerStore
	dispatcher DispatcherBackend

	cache     cache.Cacheable
	analytics analytics.Analytics
	v         validator.Validator
	l         *zerolog.Logger

	dispatcherId uuid.UUID

	notifyInterval time.Duration

	// maxListenStreamsPerOperator and maxActionsPerOperator are the admission limits; see the
	// Default constants. Zero disables the limit. The stream cap only counts stream-backed
	// sessions: an in-process session holds no stream.
	maxListenStreamsPerOperator int
	maxActionsPerOperator       int64

	// listenStreams counts the open stream-backed sessions per operator on this replica.
	listenStreams   map[uuid.UUID]int
	listenStreamsMu sync.Mutex
}

type opts struct {
	analytics       analytics.Analytics
	v               validator.Validator
	l               *zerolog.Logger
	notifyInterval  time.Duration
	cacheTTL        time.Duration
	maxListenStream int
	maxActions      int64
}

type Opt func(*opts)

func WithLogger(l *zerolog.Logger) Opt {
	return func(o *opts) { o.l = l }
}

func WithValidator(v validator.Validator) Opt {
	return func(o *opts) { o.v = v }
}

func WithAnalytics(a analytics.Analytics) Opt {
	return func(o *opts) { o.analytics = a }
}

// WithMaxListenStreamsPerOperator caps the stream-backed sessions one operator may hold open on
// this replica; zero disables the cap.
func WithMaxListenStreamsPerOperator(n int) Opt {
	return func(o *opts) { o.maxListenStream = n }
}

// WithMaxActionsPerOperator caps the action links held across all workers of one operator; zero
// disables the cap.
func WithMaxActionsPerOperator(n int64) Opt {
	return func(o *opts) { o.maxActions = n }
}

// WithNotifyInterval sets the window scheduler notifications are folded into.
func WithNotifyInterval(d time.Duration) Opt {
	return func(o *opts) { o.notifyInterval = d }
}

// WithOperatorCacheTTL sets how long an authorized operator row is reused.
func WithOperatorCacheTTL(d time.Duration) Opt {
	return func(o *opts) { o.cacheTTL = d }
}

func New(deps Deps, fs ...Opt) (*Service, error) {
	if deps.Operators == nil || deps.Workers == nil {
		return nil, fmt.Errorf("operator and worker stores are required")
	}

	if deps.Dispatcher == nil {
		return nil, fmt.Errorf("dispatcher backend is required")
	}

	defaultLogger := logger.NewDefaultLogger("operator_service")

	o := &opts{
		analytics:       analytics.NoOpAnalytics{},
		v:               validator.NewDefaultValidator(),
		l:               &defaultLogger,
		notifyInterval:  defaultNotifyInterval,
		cacheTTL:        defaultOperatorCacheTTL,
		maxListenStream: DefaultMaxListenStreamsPerOperator,
		maxActions:      DefaultMaxActionsPerOperator,
	}

	for _, f := range fs {
		f(o)
	}

	l := o.l.With().Str("service", "operator_service").Logger()

	return &Service{
		operators:                   deps.Operators,
		workers:                     deps.Workers,
		dispatcher:                  deps.Dispatcher,
		dispatcherId:                deps.DispatcherId,
		cache:                       cache.New(o.cacheTTL),
		analytics:                   o.analytics,
		v:                           o.v,
		l:                           &l,
		notifyInterval:              o.notifyInterval,
		maxListenStreamsPerOperator: o.maxListenStream,
		maxActionsPerOperator:       o.maxActions,
		listenStreams:               map[uuid.UUID]int{},
	}, nil
}

// DispatcherId is the dispatcher every session opened here pins its worker to.
func (s *Service) DispatcherId() uuid.UUID {
	return s.dispatcherId
}

// Cleanup stops the operator cache's expiry goroutine.
func (s *Service) Cleanup() error {
	if s.cache != nil {
		s.cache.Stop()
	}

	return nil
}

// acquireListenStream admits one more stream-backed session for the operator, or refuses it
// with ResourceExhausted at the cap. The returned release gives the slot back and must be
// called exactly once. The count is per engine replica: a client that spreads its streams over
// replicas is bounded by the cap times the replica count.
func (s *Service) acquireListenStream(operatorId uuid.UUID) (release func(), err error) {
	s.listenStreamsMu.Lock()
	defer s.listenStreamsMu.Unlock()

	if s.listenStreams == nil {
		s.listenStreams = map[uuid.UUID]int{}
	}

	if s.maxListenStreamsPerOperator > 0 && s.listenStreams[operatorId] >= s.maxListenStreamsPerOperator {
		return nil, status.Errorf(codes.ResourceExhausted, "operator %s already holds %d Listen streams, the limit is %d", operatorId, s.listenStreams[operatorId], s.maxListenStreamsPerOperator)
	}

	s.listenStreams[operatorId]++

	return func() {
		s.listenStreamsMu.Lock()
		defer s.listenStreamsMu.Unlock()

		if s.listenStreams[operatorId] <= 1 {
			delete(s.listenStreams, operatorId)
			return
		}

		s.listenStreams[operatorId]--
	}, nil
}

// ListenStreamCount reports the open stream-backed sessions of the operator on this replica.
func (s *Service) ListenStreamCount(operatorId uuid.UUID) int {
	s.listenStreamsMu.Lock()
	defer s.listenStreamsMu.Unlock()

	return s.listenStreams[operatorId]
}

// SendStepActionEvent reports task progress for an action delivered to one of the operator's
// workers. The event's worker must belong to the operator; the dispatcher then handles it
// exactly like an SDK worker's event.
func (s *Service) SendStepActionEvent(ctx context.Context, tenant *sqlcv1.Tenant, op *sqlcv1.V1Operator, ev *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	workerId, err := uuid.Parse(ev.WorkerId)

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid worker ID format: %s", ev.WorkerId)
	}

	if _, err := s.AuthorizeWorker(ctx, tenant, op.ID, workerId); err != nil {
		return nil, err
	}

	return s.dispatcher.SendStepActionEvent(ctx, ev)
}

// PauseWorker stops the scheduler assigning to the worker, or lets it be assigned to again.
// It returns once the change is committed, so a caller that pauses before draining knows no
// further work will arrive. Register clears the pause when it resumes a worker, so an operator
// that crashed while paused comes back assignable.
func (s *Service) PauseWorker(ctx context.Context, tenant *sqlcv1.Tenant, workerId uuid.UUID, paused bool) error {
	_, err := s.workers.UpdateWorker(ctx, tenant.ID, workerId, &repository.UpdateWorkerOpts{IsPaused: &paused})

	if err != nil {
		s.l.Error().Ctx(ctx).Err(err).Msgf("could not set paused=%t on worker %s", paused, workerId)
		return err
	}

	return nil
}
