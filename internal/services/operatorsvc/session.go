package operatorsvc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// ErrNoStream is returned by Send on a session whose delivery is a handler: there is no stream
// to write protocol messages onto.
var ErrNoStream = errors.New("operatorsvc: session has no stream to send on")

// OpenOpts describes how a session delivers assigned actions to its operator. Exactly one of
// Stream and Handler is set: Stream encodes actions onto a gRPC stream, Handler receives them
// by direct call in the engine process.
type OpenOpts struct {
	// Stream is the gRPC stream the dispatcher fans actions out on.
	Stream grpc.ServerStream

	// Wrap converts an assigned action into Stream's server message type; nil means the
	// stream's server message type is AssignedAction itself.
	Wrap func(*contracts.AssignedAction) proto.Message

	// Handler receives assigned actions by direct call.
	Handler ActionHandler

	// Worker is the worker row, when the caller already read it (the gRPC path reads it to
	// authorize the session). It is loaded here when nil.
	Worker *sqlcv1.GetWorkerForEngineRow
}

// Session is one live worker: its dispatcher routing entry, its listener fence on the worker
// row, its action budget and its scheduler notifications. It lives from OpenSession until
// Close.
//
// The methods that drive the session (Heartbeat, ApplyDelta, Send) are meant to be called from
// the one goroutine that owns the session's protocol loop, which is how the throttles read as
// one sequence; Pause and Close are safe to call from anywhere, and Close runs once.
type Session struct {
	svc *Service

	tenant   *sqlcv1.Tenant
	op       *sqlcv1.V1Operator
	workerId uuid.UUID

	// sessionId is both the dispatcher's session key and the listener fence on the worker row:
	// activation records it, and the deactivation in Close only succeeds while it is still the
	// id on the row, so a session superseded by a newer one on the same worker never marks the
	// live session's worker inactive.
	sessionId uuid.UUID

	l zerolog.Logger

	stream   StreamSession
	handler  HandlerSession
	notifier *throttledNotifier
	budget   *actionBudget

	releaseStream func()

	mu                 sync.Mutex
	lastHeartbeatWrite time.Time
	paused             bool

	closeOnce sync.Once
	closeErr  error
}

// OpenSession makes the worker live: it re-pins the worker to this dispatcher, activates it
// under a fresh session id that doubles as the listener fence, registers the delivery with the
// dispatcher, notifies the scheduler and reads the operator's action budget. Stream-backed
// sessions are also counted against the per-operator stream cap; in-process sessions hold no
// stream, so only the action budget applies to them.
func (s *Service) OpenSession(ctx context.Context, tenant *sqlcv1.Tenant, op *sqlcv1.V1Operator, workerId uuid.UUID, opts OpenOpts) (*Session, error) {
	if tenant == nil {
		return nil, status.Error(codes.Unauthenticated, "tenant not found in request context")
	}

	if (opts.Stream == nil) == (opts.Handler == nil) {
		return nil, fmt.Errorf("a session takes exactly one of a stream or a handler")
	}

	l := s.l.With().
		Str("tenant_id", tenant.ID.String()).
		Str("operator_name", op.Name).
		Str("operator_id", op.ID.String()).
		Str("worker_id", workerId.String()).
		Logger()

	ss := &Session{
		svc:       s,
		tenant:    tenant,
		op:        op,
		workerId:  workerId,
		sessionId: uuid.New(),
		l:         l,
	}

	if opts.Stream != nil {
		release, err := s.acquireListenStream(op.ID)

		if err != nil {
			return nil, err
		}

		ss.releaseStream = release
	}

	if err := s.pinWorker(ctx, &l, tenant, workerId, opts.Worker); err != nil {
		ss.unwind(ctx, false)
		return nil, err
	}

	if _, err := s.workers.ActivateWorkerListener(ctx, tenant.ID, workerId, ss.sessionId); err != nil {
		l.Error().Ctx(ctx).Err(err).Msgf("could not activate worker for listener session %s", ss.sessionId)
		ss.unwind(ctx, false)

		return nil, err
	}

	if opts.Stream != nil {
		ss.stream = s.dispatcher.AddOperatorStreamSession(workerId, ss.sessionId, opts.Stream, opts.Wrap)
	} else {
		ss.handler = s.dispatcher.AddOperatorSession(workerId, ss.sessionId, opts.Handler)
	}

	// the opening notify goes through the notifier so a burst of deltas right after it folds
	// into the same throttle window
	ss.notifier = newThrottledNotifier(ctx, s.dispatcher, tenant, workerId, s.notifyInterval)
	ss.notifier.fire()

	// The action budget is read once per session and then tracked from the deltas this session
	// applies. Sessions of the same operator that run concurrently on this or another replica
	// do not see each other's changes until they reopen, so the cap is exact per session and
	// approximate across sessions, by at most one chunk per session.
	budget, err := s.newActionBudget(ctx, tenant.ID, op.ID)

	if err != nil {
		l.Error().Ctx(ctx).Err(err).Msg("could not count operator worker actions")
		ss.unwind(ctx, true)

		return nil, err
	}

	ss.budget = budget

	s.analytics.Count(ctx, analytics.Worker, analytics.Listen, analytics.Props("operator_kind", string(op.Kind)))

	l.Info().Ctx(ctx).Int64("linked_actions", budget.linked).Msg("operator worker listening")

	return ss, nil
}

// pinWorker points the worker at this dispatcher when it is not there already: the session that
// delivers actions lives here, so a worker resumed after a reconnect to another engine replica
// is re-pinned.
func (s *Service) pinWorker(ctx context.Context, l *zerolog.Logger, tenant *sqlcv1.Tenant, workerId uuid.UUID, known *sqlcv1.GetWorkerForEngineRow) error {
	worker := known

	if worker == nil {
		loaded, err := s.workers.GetWorkerForEngine(ctx, tenant.ID, workerId)

		if err != nil {
			l.Error().Ctx(ctx).Err(err).Msg("could not read worker before opening the session")
			return err
		}

		worker = loaded
	}

	if worker.DispatcherId != nil && *worker.DispatcherId == s.dispatcherId {
		return nil
	}

	dispatcherId := s.dispatcherId

	if _, err := s.workers.UpdateWorker(ctx, tenant.ID, workerId, &repository.UpdateWorkerOpts{DispatcherId: &dispatcherId}); err != nil {
		l.Error().Ctx(ctx).Err(err).Msg("could not update worker dispatcher")
		return err
	}

	return nil
}

// unwind undoes a partially opened session, in the reverse order of OpenSession. activated says
// whether the worker's listener was already activated under this session id.
func (ss *Session) unwind(ctx context.Context, activated bool) {
	if ss.notifier != nil {
		ss.notifier.stop()
	}

	if ss.stream != nil {
		ss.stream.Release()
	}

	if ss.handler != nil {
		ss.handler.Release()
	}

	if activated {
		// the session never became usable, so a deactivation that fails has nothing to report
		// to: the error that is unwinding it is the one the caller sees
		_ = ss.deactivate(ctx)
	}

	if ss.releaseStream != nil {
		ss.releaseStream()
	}
}

// SessionId is the id the worker's listener fence and the dispatcher session are keyed on.
func (ss *Session) SessionId() uuid.UUID { return ss.sessionId }

// WorkerId is the worker this session makes live.
func (ss *Session) WorkerId() uuid.UUID { return ss.workerId }

// Operator is the operator row the session belongs to.
func (ss *Session) Operator() *sqlcv1.V1Operator { return ss.op }

// Tenant is the tenant the session belongs to.
func (ss *Session) Tenant() *sqlcv1.Tenant { return ss.tenant }

// Fin fires when the dispatcher wants a stream-backed session hung up. A handler-backed session
// returns nil, which never selects: the dispatcher has no stream to reclaim.
func (ss *Session) Fin() <-chan bool {
	if ss.stream == nil {
		return nil
	}

	return ss.stream.Fin()
}

// Send writes a protocol message on a stream-backed session's stream, serialised with the
// dispatcher's own action sends.
func (ss *Session) Send(ctx context.Context, msg proto.Message) error {
	if ss.stream == nil {
		return ErrNoStream
	}

	return ss.stream.Send(ctx, msg)
}

// Heartbeat records that the operator is alive. Writes are throttled to one per second, so a
// host that heartbeats faster costs nothing; a throttled call reports no error.
func (ss *Session) Heartbeat(ctx context.Context, at time.Time) error {
	ss.mu.Lock()

	if at.Sub(ss.lastHeartbeatWrite) < heartbeatWriteInterval {
		ss.mu.Unlock()
		return nil
	}

	ss.lastHeartbeatWrite = at
	ss.mu.Unlock()

	return ss.svc.workers.UpdateWorkerHeartbeat(ctx, ss.tenant.ID, ss.workerId, at)
}

// ApplyDelta validates and applies one change to the worker's action set and asks the scheduler
// to reload it. It reports whether the set changed; a delta that only repeats what the worker
// already has needs no notification, and the caller can still acknowledge it.
func (ss *Session) ApplyDelta(ctx context.Context, add, remove []string) (bool, error) {
	changed, err := ss.svc.applyDelta(ctx, &ss.l, ss.tenant.ID, ss.workerId, add, remove, ss.budget)

	if err != nil {
		return false, err
	}

	if changed {
		ss.notifier.request()
	}

	return changed, nil
}

// Pause stops the scheduler assigning to the session's worker, or lets it be assigned to again.
// It returns once the change is committed, so a host that pauses before draining knows no
// further work will arrive.
func (ss *Session) Pause(ctx context.Context, paused bool) error {
	if err := ss.svc.PauseWorker(ctx, ss.tenant, ss.workerId, paused); err != nil {
		return err
	}

	ss.mu.Lock()
	ss.paused = paused
	ss.mu.Unlock()

	return nil
}

type closeOpts struct {
	pause bool
}

type CloseOpt func(*closeOpts)

// WithoutPause closes the session without pausing its worker. It is what a session whose
// operator pauses for itself uses: a gRPC operator pauses through PauseWorker before it hangs
// up, and a stream that ends unexpectedly must leave the worker assignable so the operator's
// next connection resumes a worker that can be given work.
func WithoutPause() CloseOpt {
	return func(o *closeOpts) { o.pause = false }
}

// Close ends the session: pause, then drain, then deactivate. Pausing stops the scheduler
// assigning new work, releasing the dispatcher session stops anything further being delivered,
// and the deactivation, fenced on this session's id, marks the worker inactive. A host that
// waits for its operator's in-flight work does so between Pause and Close.
//
// The deactivation runs detached from ctx because the common exit is the operator being gone,
// at which point ctx is already cancelled. Close runs once; later calls return the first
// result.
func (ss *Session) Close(ctx context.Context, fs ...CloseOpt) error {
	o := &closeOpts{pause: true}

	for _, f := range fs {
		f(o)
	}

	ss.closeOnce.Do(func() {
		if o.pause {
			ss.mu.Lock()
			alreadyPaused := ss.paused
			ss.mu.Unlock()

			if !alreadyPaused {
				pauseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), deactivateTimeout)
				defer cancel()

				if err := ss.svc.PauseWorker(pauseCtx, ss.tenant, ss.workerId, true); err != nil {
					ss.closeErr = err
				}
			}
		}

		ss.notifier.stop()

		if ss.stream != nil {
			ss.stream.Release()
		}

		if ss.handler != nil {
			ss.handler.Release()
		}

		if err := ss.deactivate(ctx); err != nil && ss.closeErr == nil {
			ss.closeErr = err
		}

		if ss.releaseStream != nil {
			ss.releaseStream()
		}
	})

	return ss.closeErr
}

// deactivate marks the worker inactive on behalf of this session. A superseded session (one
// whose id is no longer recorded on the worker) has nothing to do, because the newer session
// owns the worker's active flag.
func (ss *Session) deactivate(ctx context.Context) error {
	deactivateCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), deactivateTimeout)
	defer cancel()

	_, err := ss.svc.workers.DeactivateWorkerListener(deactivateCtx, ss.tenant.ID, ss.workerId, ss.sessionId)

	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		ss.l.Debug().Ctx(deactivateCtx).Msgf("listener session %s was superseded by a newer session, leaving worker active", ss.sessionId)
		return nil
	}

	ss.l.Error().Ctx(deactivateCtx).Err(err).Msgf("could not deactivate worker for listener session %s", ss.sessionId)

	return err
}

// actionDeltaOpts carries the validation rules for an action set delta.
type actionDeltaOpts struct {
	Add    []string `validate:"dive,actionId"`
	Remove []string `validate:"dive,actionId"`
}

// actionBudget tracks the action links of one operator against the per-operator cap for the
// life of a session.
type actionBudget struct {
	linked int64
	limit  int64
}

func (s *Service) newActionBudget(ctx context.Context, tenantId, operatorId uuid.UUID) (*actionBudget, error) {
	budget := &actionBudget{limit: s.maxActionsPerOperator}

	if budget.limit <= 0 {
		return budget, nil
	}

	linked, err := s.workers.CountOperatorWorkerActions(ctx, tenantId, operatorId)

	if err != nil {
		return nil, err
	}

	budget.linked = linked

	return budget, nil
}

// remaining is how many more links the operator may take, or -1 when unlimited. Adds that
// repeat actions the worker already has never consume budget: the repository only counts the
// links it creates, and rolls the delta back when they exceed this.
func (b *actionBudget) remaining() int64 {
	if b.limit <= 0 {
		return -1
	}

	return max(b.limit-b.linked, 0)
}

func (b *actionBudget) apply(added, removed int) {
	b.linked += int64(added) - int64(removed)
}

// applyDelta validates and applies one delta to the worker's action set. It reports whether the
// set changed.
func (s *Service) applyDelta(ctx context.Context, l *zerolog.Logger, tenantId, workerId uuid.UUID, add, remove []string, budget *actionBudget) (bool, error) {
	if n := len(add) + len(remove); n > MaxActionsPerDelta {
		return false, status.Errorf(codes.InvalidArgument, "actions delta carries %d ids, the limit is %d per message", n, MaxActionsPerDelta)
	}

	if err := s.v.Validate(actionDeltaOpts{Add: add, Remove: remove}); err != nil {
		return false, status.Errorf(codes.InvalidArgument, "invalid actions delta: %s", err.Error())
	}

	changed := false

	if len(add) > 0 {
		added, err := s.workers.AddWorkerActionsWithinBudget(ctx, tenantId, workerId, add, budget.remaining())

		if err != nil {
			if errors.Is(err, repository.ErrWorkerActionBudgetExceeded) {
				return false, status.Errorf(codes.ResourceExhausted, "operator holds %d actions and the delta adds more than the limit of %d allows", budget.linked, budget.limit)
			}

			l.Error().Ctx(ctx).Err(err).Msg("could not add worker actions")

			return false, err
		}

		budget.apply(added, 0)
		changed = changed || added > 0
	}

	if len(remove) > 0 {
		removed, err := s.workers.RemoveWorkerActions(ctx, tenantId, workerId, remove)

		if err != nil {
			l.Error().Ctx(ctx).Err(err).Msg("could not remove worker actions")
			return false, err
		}

		budget.apply(0, removed)
		changed = changed || removed > 0
	}

	l.Debug().Ctx(ctx).
		Int("add", len(add)).
		Int("remove", len(remove)).
		Bool("changed", changed).
		Msg("applied operator actions delta")

	return changed, nil
}

// throttledNotifier coalesces scheduler notifications for one worker: the first request fires
// immediately, and requests that arrive within interval of the last fire are folded into one
// notification sent when the interval elapses. It drives its own timer, so a session that is
// not selecting on anything still notifies.
type throttledNotifier struct {
	ctx      context.Context
	d        DispatcherBackend
	tenant   *sqlcv1.Tenant
	workerId uuid.UUID
	interval time.Duration

	mu       sync.Mutex
	lastFire time.Time
	timer    *time.Timer
	stopped  bool
}

func newThrottledNotifier(ctx context.Context, d DispatcherBackend, tenant *sqlcv1.Tenant, workerId uuid.UUID, interval time.Duration) *throttledNotifier {
	return &throttledNotifier{ctx: ctx, d: d, tenant: tenant, workerId: workerId, interval: interval}
}

// request asks for a notification, immediately when the last one is older than the interval and
// otherwise when the window closes.
func (n *throttledNotifier) request() {
	n.mu.Lock()

	if n.stopped || n.timer != nil {
		n.mu.Unlock()
		return
	}

	if wait := n.interval - time.Since(n.lastFire); wait > 0 {
		n.timer = time.AfterFunc(wait, n.fireDeferred)
		n.mu.Unlock()

		return
	}

	n.lastFire = time.Now()
	n.mu.Unlock()

	n.notify()
}

// fire notifies now and restarts the window; it is the opening notification of a session.
func (n *throttledNotifier) fire() {
	n.mu.Lock()

	if n.stopped {
		n.mu.Unlock()
		return
	}

	if n.timer != nil {
		n.timer.Stop()
		n.timer = nil
	}

	n.lastFire = time.Now()
	n.mu.Unlock()

	n.notify()
}

func (n *throttledNotifier) fireDeferred() {
	n.mu.Lock()
	n.timer = nil

	if n.stopped {
		n.mu.Unlock()
		return
	}

	n.lastFire = time.Now()
	n.mu.Unlock()

	n.notify()
}

// notify runs outside the lock: the dispatcher's publish must not be serialised behind the
// notifier's own bookkeeping.
func (n *throttledNotifier) notify() {
	n.d.NotifyNewWorker(n.ctx, n.tenant, n.workerId)
}

// stop ends the notifier. A callback that already passed the stopped check may still publish
// one notification after this returns; the scheduler simply reloads a worker that is on its way
// out, so stop does not wait for it.
func (n *throttledNotifier) stop() {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.stopped = true

	if n.timer != nil {
		n.timer.Stop()
		n.timer = nil
	}
}
