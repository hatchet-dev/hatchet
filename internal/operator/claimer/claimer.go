// Package claimer hosts the operators a dispatcher claims. It polls ClaimOperators for the
// rows the dispatcher claims (leasing manager DISPATCHER) assigned to it, builds each one from a
// factory, opens it through the in-process host with the row as its identity, and starts the
// operator on the session. An operator that leaves the claim result is torn down the way every
// host tears an operator down: pause the worker, drain the operator, close the session. Worker
// rows, heartbeats and the dispatcher's routing table are the host's; the claimer owns
// placement and lifecycle only.
//
// A factory is found by the row's kind, except for GRPC rows, contract operators, which are
// found by the row's name: the kind says only that the row is a contract operator, the name
// says which one the engine should build. A claimed row with no factory is logged once and
// left alone.
//
// It runs inside the dispatcher process, wired from cmd/hatchet-engine/engine next to the
// dispatcher, and stops before the dispatcher drains so events reported during a drain still
// have somewhere to go.
package claimer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/operator/dagoperator"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const (
	// defaultPollInterval is how often the claim query runs.
	defaultPollInterval = 5 * time.Second

	// defaultTeardownTimeout bounds one operator's pause, drain and close. DAG runs abort as
	// soon as they are interrupted, so the bound only matters for a stuck engine call.
	defaultTeardownTimeout = 30 * time.Second
)

// Claims is the claim query: the operator rows this dispatcher should run.
type Claims interface {
	ClaimOperators(ctx context.Context, dispatcherId uuid.UUID) ([]*sqlcv1.V1Operator, error)
}

// Factory builds the operator for a claimed row and the options its session is opened with.
// The handler is set by the claimer.
type Factory func(op *sqlcv1.V1Operator) (operator.Operator, operator.OpenOpts, error)

// DAGFactory builds DAG operators: the worker's durable slots come from the row's config or
// the server default.
func DAGFactory(l *zerolog.Logger, repo repository.Repository, writer operator.TaskEventWriter, defaultSlots int) Factory {
	return func(op *sqlcv1.V1Operator) (operator.Operator, operator.OpenOpts, error) {
		slotConfig, err := dagoperator.SlotConfig(op, defaultSlots)

		if err != nil {
			return nil, operator.OpenOpts{}, fmt.Errorf("could not determine slot config for dag operator: %w", err)
		}

		dagOp, err := dagoperator.NewDAGOperator(op, l, repo, writer, dagoperator.WithSlots(defaultSlots))

		if err != nil {
			return nil, operator.OpenOpts{}, fmt.Errorf("could not construct dag operator: %w", err)
		}

		return dagOp, operator.OpenOpts{SlotConfig: slotConfig}, nil
	}
}

type opts struct {
	host         operator.Host
	claims       Claims
	dispatcherId uuid.UUID
	factories    map[sqlcv1.V1OperatorKind]Factory
	named        map[string]Factory
	l            *zerolog.Logger

	pollInterval    time.Duration
	teardownTimeout time.Duration
}

type Opt func(*opts)

func defaultOpts() *opts {
	l := logger.NewDefaultLogger("operator_claimer")

	return &opts{
		factories:       map[sqlcv1.V1OperatorKind]Factory{},
		named:           map[string]Factory{},
		l:               &l,
		pollInterval:    defaultPollInterval,
		teardownTimeout: defaultTeardownTimeout,
	}
}

// WithHost sets the host claimed operators are opened on. Required.
func WithHost(h operator.Host) Opt {
	return func(o *opts) { o.host = h }
}

// WithClaims sets the claim query. Required.
func WithClaims(c Claims) Opt {
	return func(o *opts) { o.claims = c }
}

// WithDispatcherId sets the dispatcher whose claims this claimer hosts. Required.
func WithDispatcherId(id uuid.UUID) Opt {
	return func(o *opts) { o.dispatcherId = id }
}

// WithFactory registers how claimed rows of one kind are built. GRPC rows are not built by
// kind, see WithNamedFactory. A claimed row of a kind with no factory is left alone.
func WithFactory(kind sqlcv1.V1OperatorKind, f Factory) Opt {
	return func(o *opts) { o.factories[kind] = f }
}

// WithNamedFactory registers how claimed GRPC rows named name are built: a contract operator
// the engine hosts in process under a lease the dispatcher holds. A claimed GRPC row with no factory
// of its name is left alone.
func WithNamedFactory(name string, f Factory) Opt {
	return func(o *opts) { o.named[name] = f }
}

func WithLogger(l *zerolog.Logger) Opt {
	return func(o *opts) { o.l = l }
}

// WithPollInterval sets how often the claim query runs.
func WithPollInterval(d time.Duration) Opt {
	return func(o *opts) { o.pollInterval = d }
}

// WithTeardownTimeout bounds one operator's pause, drain and close.
func WithTeardownTimeout(d time.Duration) Opt {
	return func(o *opts) { o.teardownTimeout = d }
}

// hosted is one claimed operator and the session it runs on.
type hosted struct {
	op      operator.Operator
	session operator.Session
}

// Claimer polls the claim query and keeps the claimed operators hosted.
type Claimer struct {
	host         operator.Host
	claims       Claims
	dispatcherId uuid.UUID
	factories    map[sqlcv1.V1OperatorKind]Factory
	named        map[string]Factory
	l            *zerolog.Logger

	pollInterval    time.Duration
	teardownTimeout time.Duration

	mu      sync.Mutex
	running map[uuid.UUID]*hosted

	// unhostable is every claimed row that was logged as having no factory, so the log line
	// is written once per row rather than once per poll.
	unhostable map[uuid.UUID]struct{}

	// teardowns tracks the operators being torn down in the background, so Stop can wait
	// for them.
	teardowns sync.WaitGroup

	cancel   context.CancelFunc
	pollDone chan struct{}
	stopOnce sync.Once
}

func New(fs ...Opt) (*Claimer, error) {
	o := defaultOpts()

	for _, f := range fs {
		f(o)
	}

	if o.host == nil {
		return nil, errors.New("host is required. use WithHost")
	}

	if o.claims == nil {
		return nil, errors.New("claim query is required. use WithClaims")
	}

	if o.dispatcherId == uuid.Nil {
		return nil, errors.New("dispatcher id is required. use WithDispatcherId")
	}

	cl := o.l.With().Str("service", "operator_claimer").Logger()

	return &Claimer{
		host:            o.host,
		claims:          o.claims,
		dispatcherId:    o.dispatcherId,
		factories:       o.factories,
		named:           o.named,
		l:               &cl,
		pollInterval:    o.pollInterval,
		teardownTimeout: o.teardownTimeout,
		running:         map[uuid.UUID]*hosted{},
		unhostable:      map[uuid.UUID]struct{}{},
		pollDone:        make(chan struct{}),
	}, nil
}

// Start begins polling. The first claim runs at once so the dispatcher's operators are live
// without waiting a full interval; Stop ends the polling.
func (c *Claimer) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	go c.poll(ctx)
}

func (c *Claimer) poll(ctx context.Context) {
	defer close(c.pollDone)

	c.reconcile(ctx)

	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.reconcile(ctx)
		}
	}
}

// reconcile runs the claim query and treats its result as the desired state. A failed query
// says nothing about which operators are still claimed, so it changes nothing.
func (c *Claimer) reconcile(ctx context.Context) {
	claimed, err := c.claims.ClaimOperators(ctx, c.dispatcherId)

	if err != nil {
		if ctx.Err() == nil {
			c.l.Error().Ctx(ctx).Err(err).Msg("could not claim operators")
		}

		return
	}

	c.Reconcile(ctx, claimed)
}

// Reconcile hosts the claimed rows that are not running yet and tears down the running
// operators that are no longer claimed: an operator assigned to this dispatcher cannot be
// claimed by another live dispatcher, so an absence means it was deleted or this dispatcher
// lost it.
func (c *Claimer) Reconcile(ctx context.Context, claimed []*sqlcv1.V1Operator) {
	claimedIds := make(map[uuid.UUID]struct{}, len(claimed))

	for _, op := range claimed {
		claimedIds[op.ID] = struct{}{}

		c.mu.Lock()
		_, running := c.running[op.ID]
		c.mu.Unlock()

		if running {
			continue
		}

		h := c.open(ctx, op)

		if h == nil {
			continue
		}

		c.mu.Lock()
		c.running[op.ID] = h
		c.mu.Unlock()
	}

	c.mu.Lock()
	lost := make(map[uuid.UUID]*hosted)

	for id, h := range c.running {
		if _, ok := claimedIds[id]; ok {
			continue
		}

		lost[id] = h
		delete(c.running, id)
	}
	c.mu.Unlock()

	for id, h := range lost {
		c.l.Info().Ctx(ctx).Msgf("operator %s is no longer claimed by this dispatcher, tearing down worker %s", id, h.session.Registration().WorkerId)

		c.teardowns.Go(func() { c.teardown(h) })
	}
}

// factory resolves how a claimed row is built: GRPC rows by name, every other kind by kind.
func (c *Claimer) factory(op *sqlcv1.V1Operator) (Factory, bool) {
	if op.Kind == sqlcv1.V1OperatorKindGRPC {
		f, ok := c.named[op.Name]
		return f, ok
	}

	f, ok := c.factories[op.Kind]

	return f, ok
}

// open builds the operator for a claimed row, opens its session and starts it. It returns nil,
// after logging, when the row cannot be hosted; the next poll tries again, except for a row
// with no factory, which no poll can host and which is logged once.
func (c *Claimer) open(ctx context.Context, op *sqlcv1.V1Operator) *hosted {
	l := c.l.With().Str("operator_id", op.ID.String()).Str("operator_kind", string(op.Kind)).Str("operator_name", op.Name).Logger()

	factory, ok := c.factory(op)

	if !ok {
		c.mu.Lock()
		_, logged := c.unhostable[op.ID]
		c.unhostable[op.ID] = struct{}{}
		c.mu.Unlock()

		if !logged {
			l.Warn().Ctx(ctx).Msg("claimed operator has no factory in this engine and is left alone")
		}

		return nil
	}

	instance, openOpts, err := factory(op)

	if err != nil {
		l.Error().Ctx(ctx).Err(err).Msg("could not build operator")
		return nil
	}

	openOpts.Handler = instance

	session, err := c.host.Open(ctx, operator.Identity{TenantId: op.TenantID, OperatorId: &op.ID}, openOpts)

	if err != nil {
		l.Error().Ctx(ctx).Err(err).Msg("could not open operator session")
		return nil
	}

	if err := instance.Start(ctx, session); err != nil {
		l.Error().Ctx(ctx).Err(err).Msg("could not start operator")

		if closeErr := session.Close(ctx); closeErr != nil {
			l.Error().Ctx(ctx).Err(closeErr).Msg("could not close the session of an operator that did not start")
		}

		return nil
	}

	l.Info().Ctx(ctx).Str("worker_id", session.Registration().WorkerId.String()).Msg("operator hosted")

	return &hosted{op: instance, session: session}
}

// teardown is the host's order for one operator: pause the worker so nothing new is
// assigned, drain the operator, close the session. It runs on its own bounded context because
// the common reason is a shutdown whose context is already gone.
func (c *Claimer) teardown(h *hosted) {
	ctx, cancel := context.WithTimeout(context.Background(), c.teardownTimeout)
	defer cancel()

	workerId := h.session.Registration().WorkerId

	if err := h.session.Pause(ctx); err != nil {
		c.l.Error().Ctx(ctx).Err(err).Msgf("could not pause worker %s before draining", workerId)
	}

	h.op.Drain(ctx)

	if err := h.session.Close(ctx); err != nil {
		c.l.Error().Ctx(ctx).Err(err).Msgf("could not close the session of worker %s", workerId)
	}
}

// Running is the number of operators hosted right now.
func (c *Claimer) Running() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.running)
}

// Stop ends polling and tears every hosted operator down, concurrently, then waits for the
// teardowns already in flight. It runs once.
func (c *Claimer) Stop(ctx context.Context) {
	c.stopOnce.Do(func() {
		if c.cancel != nil {
			c.cancel()
			<-c.pollDone
		}

		c.mu.Lock()
		all := c.running
		c.running = map[uuid.UUID]*hosted{}
		c.mu.Unlock()

		for _, h := range all {
			c.teardowns.Go(func() { c.teardown(h) })
		}

		done := make(chan struct{})

		go func() {
			c.teardowns.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			c.l.Warn().Ctx(ctx).Msg("operator claimer stopped before every operator finished tearing down")
		}
	})
}
