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

	releaseStream func()

	mu                 sync.Mutex
	lastHeartbeatWrite time.Time
	paused             bool

	closeOnce sync.Once
	closeErr  error
}

// OpenSession makes the worker live: it re-pins the worker to this dispatcher, refreshes its
// action hash when a previous session left it pending, activates it under a fresh session id
// that doubles as the listener fence, registers the delivery with the dispatcher and notifies
// the scheduler. Stream-backed sessions are also counted against the per-operator stream cap;
// in-process sessions hold no stream, so only the action budget applies to them.
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

	worker, err := s.pinWorker(ctx, &l, tenant, workerId, opts.Worker)

	if err != nil {
		ss.unwind(ctx, false)
		return nil, err
	}

	// a worker resumed after its previous session ended between a delta and the refresh that
	// follows it has no hash; the digest of its links is written before it is assignable
	if worker.ActionHash == nil {
		if err := s.workers.RefreshWorkerActionHash(ctx, tenant.ID, workerId); err != nil {
			l.Error().Ctx(ctx).Err(err).Msg("could not refresh the worker's action hash before opening the session")
			ss.unwind(ctx, false)

			return nil, err
		}
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
	// into the same throttle window; the notifier also refreshes the worker's action hash
	// before each notification that follows a delta, so the reload sees the committed set's
	// digest
	ss.notifier = newThrottledNotifier(ctx, s.dispatcher, tenant, workerId, s.notifyInterval, ss.refreshActionHash, &ss.l)
	ss.notifier.fire()

	s.analytics.Count(ctx, analytics.Worker, analytics.Listen, analytics.Props("operator_kind", string(op.Kind)))

	// the count is for the log line only: the budget is enforced by every delta's transaction
	linked, err := s.workers.CountOperatorWorkerActions(ctx, tenant.ID, op.ID)

	if err != nil {
		l.Warn().Ctx(ctx).Err(err).Msg("could not count operator worker actions")
	}

	l.Info().Ctx(ctx).Int64("linked_actions", linked).Msg("operator worker listening")

	return ss, nil
}

// refreshActionHash writes the digest of the worker's committed links. It runs detached from
// the caller's context, bounded by its own timeout: the hash is what the scheduler groups the
// worker by, and it is owed to the row whichever request happened to trigger it.
func (ss *Session) refreshActionHash(ctx context.Context) error {
	refreshCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), actionHashRefreshTimeout)
	defer cancel()

	return ss.svc.workers.RefreshWorkerActionHash(refreshCtx, ss.tenant.ID, ss.workerId)
}

// pinWorker points the worker at this dispatcher when it is not there already: the session that
// delivers actions lives here, so a worker resumed after a reconnect to another engine replica
// is re-pinned. It returns the worker row it read, or was given.
func (s *Service) pinWorker(ctx context.Context, l *zerolog.Logger, tenant *sqlcv1.Tenant, workerId uuid.UUID, known *sqlcv1.GetWorkerForEngineRow) (*sqlcv1.GetWorkerForEngineRow, error) {
	worker := known

	if worker == nil {
		loaded, err := s.workers.GetWorkerForEngine(ctx, tenant.ID, workerId)

		if err != nil {
			l.Error().Ctx(ctx).Err(err).Msg("could not read worker before opening the session")
			return nil, err
		}

		worker = loaded
	}

	if worker.DispatcherId != nil && *worker.DispatcherId == s.dispatcherId {
		return worker, nil
	}

	dispatcherId := s.dispatcherId

	if _, err := s.workers.UpdateWorker(ctx, tenant.ID, workerId, &repository.UpdateWorkerOpts{DispatcherId: &dispatcherId}); err != nil {
		l.Error().Ctx(ctx).Err(err).Msg("could not update worker dispatcher")
		return nil, err
	}

	return worker, nil
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
	changed, err := ss.svc.applyDelta(ctx, &ss.l, ss.tenant.ID, ss.workerId, ss.op.Kind, add, remove)

	if err != nil {
		return false, err
	}

	if changed {
		ss.notifier.request()
	}

	return changed, nil
}

// SendStepActionEvent reports task progress for an action delivered to the session's worker,
// on the tenant-scoped path in-engine operators use: the tenant is put on the context and the
// dispatcher handles the event exactly like an SDK worker's. An empty WorkerId is filled from
// the session; another worker's id is refused, since the session only ever speaks for its own
// worker.
func (ss *Session) SendStepActionEvent(ctx context.Context, ev *contracts.StepActionEvent) error {
	if ev.WorkerId == "" {
		ev.WorkerId = ss.workerId.String()
	} else if ev.WorkerId != ss.workerId.String() {
		return status.Errorf(codes.PermissionDenied, "worker %s is not this session's worker %s", ev.WorkerId, ss.workerId)
	}

	_, err := ss.svc.dispatcher.SendStepActionEvent(WithTenant(ctx, ss.tenant), ev)

	return err
}

// Pause stops the scheduler assigning to the session's worker, or lets it be assigned to again.
// It returns once the change is committed, so a host that pauses before draining knows no
// further work will arrive: the dispatcher session stops delivering before the pause is written,
// and an action the scheduler assigned in the meantime goes back to the queue rather than to
// the operator. Resuming lifts the pause in the opposite order, so nothing is refused once the
// scheduler may assign again. The write is fenced on the session id like the deactivation: a
// session superseded by a newer one on the same worker leaves the worker's scheduling state
// to it.
func (ss *Session) Pause(ctx context.Context, paused bool) error {
	if paused {
		ss.setDelivering(false)
	}

	if err := ss.pauseWorker(ctx, paused); err != nil {
		if paused {
			ss.setDelivering(true)
		}

		return err
	}

	if !paused {
		ss.setDelivering(true)
	}

	ss.mu.Lock()
	ss.paused = paused
	ss.mu.Unlock()

	return nil
}

// pauseWorker writes the pause on behalf of this session. A superseded session's write is
// skipped by the fence, which is not an error: the newer session owns the worker's pause.
func (ss *Session) pauseWorker(ctx context.Context, paused bool) error {
	err := ss.svc.workers.PauseWorkerForListener(ctx, ss.tenant.ID, ss.workerId, ss.sessionId, paused)

	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		ss.l.Debug().Ctx(ctx).Msgf("listener session %s was superseded by a newer session, leaving the worker's pause to it", ss.sessionId)
		return nil
	}

	ss.l.Error().Ctx(ctx).Err(err).Msgf("could not set paused=%t on worker for listener session %s", paused, ss.sessionId)

	return err
}

// setDelivering flips the dispatcher session between delivering and returning assignments to
// the queue.
func (ss *Session) setDelivering(delivering bool) {
	if ss.stream != nil {
		ss.stream.SetPaused(!delivering)
	}

	if ss.handler != nil {
		ss.handler.SetPaused(!delivering)
	}
}

type closeOpts struct {
	pause bool
}

type CloseOpt func(*closeOpts)

// WithoutPause closes the session without pausing its worker. It is what a session whose
// operator pauses for itself uses: a gRPC operator pauses on its stream before it hangs up,
// and a stream that ends unexpectedly must leave the worker assignable so the operator's next
// connection resumes a worker that can be given work.
func WithoutPause() CloseOpt {
	return func(o *closeOpts) { o.pause = false }
}

// Close ends the session: pause, then drain, then deactivate. Pausing stops the scheduler
// assigning new work, releasing the dispatcher session stops anything further being delivered,
// and the deactivation, fenced on this session's id, marks the worker inactive. A host that
// waits for its operator's in-flight work does so between Pause and Close. A hash refresh the
// notifier still owed is done here, so the row never keeps a pending hash past its session.
//
// The pause, the refresh and the deactivation run detached from ctx because the common exit
// is the operator being gone, at which point ctx is already cancelled. Close runs once; later
// calls return the first result.
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

				if err := ss.pauseWorker(pauseCtx, true); err != nil {
					ss.closeErr = err
				}
			}
		}

		if ss.notifier.stop() {
			if err := ss.refreshActionHash(ctx); err != nil {
				ss.l.Error().Ctx(ctx).Err(err).Msg("could not refresh the worker's action hash on close; the next session on the worker refreshes it")
			}
		}

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

// actionDeltaOpts carries the validation rules for an action set delta of an operator whose
// ids are the action ids workers register, "service:verb".
type actionDeltaOpts struct {
	Add    []string `validate:"dive,actionId"`
	Remove []string `validate:"dive,actionId"`
}

// validateDelta checks a delta's ids against the kind of operator the session belongs to.
// The DAG operator's actions are the orchestrator action ids of the tenant's DAG workflows,
// "<workflow>_orchestrator", which are not action ids: they carry no ":" precisely so that no
// SDK or GRPC worker can register one. A DAG operator is hosted in process only, its row is
// created by the engine and never upserted through a registration, so its kind is the
// engine's own claim and its sessions admit orchestrator ids and nothing else. Every other
// kind admits action ids and nothing else.
func (s *Service) validateDelta(kind sqlcv1.V1OperatorKind, add, remove []string) error {
	if kind != sqlcv1.V1OperatorKindDAG {
		return s.v.Validate(actionDeltaOpts{Add: add, Remove: remove})
	}

	for _, ids := range [][]string{add, remove} {
		for _, id := range ids {
			if !repository.IsDAGOrchestratorActionId(id) {
				return fmt.Errorf("%q is not a dag orchestrator action id", id)
			}
		}
	}

	return nil
}

// applyDelta validates and applies one delta to the worker's action set, as one transaction:
// a delta the caller acknowledges by sequence is committed whole or not at all. The
// per-operator action cap is enforced inside that transaction, against the links every worker
// of the operator holds. It reports whether the set changed.
func (s *Service) applyDelta(ctx context.Context, l *zerolog.Logger, tenantId, workerId uuid.UUID, kind sqlcv1.V1OperatorKind, add, remove []string) (bool, error) {
	if n := len(add) + len(remove); n > MaxActionsPerDelta {
		return false, status.Errorf(codes.InvalidArgument, "actions delta carries %d ids, the limit is %d per message", n, MaxActionsPerDelta)
	}

	if err := s.validateDelta(kind, add, remove); err != nil {
		return false, status.Errorf(codes.InvalidArgument, "invalid actions delta: %s", err.Error())
	}

	maxLinks := s.maxActionsPerOperator

	if maxLinks <= 0 {
		maxLinks = -1
	}

	added, removed, err := s.workers.ApplyWorkerActionsDelta(ctx, tenantId, workerId, add, remove, maxLinks)

	if err != nil {
		var budgetErr *repository.ActionBudgetError

		if errors.As(err, &budgetErr) {
			return false, status.Errorf(codes.ResourceExhausted, "the delta would leave the operator with %d action links across its workers, the limit is %d", budgetErr.Linked, budgetErr.Limit)
		}

		l.Error().Ctx(ctx).Err(err).Msg("could not apply worker actions delta")

		return false, err
	}

	changed := added > 0 || removed > 0

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
//
// A request means a delta changed the worker's links, which cleared the row's action hash;
// the window is the delta sequence, so the hash is refreshed once before the notification
// that ends it, and the scheduler's reload sees the committed set's digest. A refresh that
// fails leaves the row without a hash, which the scheduler reads through the join, and is
// retried by the next window or by the session's close.
type throttledNotifier struct {
	ctx      context.Context
	d        DispatcherBackend
	tenant   *sqlcv1.Tenant
	workerId uuid.UUID
	interval time.Duration
	refresh  func(context.Context) error
	l        *zerolog.Logger

	mu       sync.Mutex
	lastFire time.Time
	timer    *time.Timer
	stopped  bool
	// dirty records that a delta changed the links since the hash was last refreshed
	dirty bool
}

func newThrottledNotifier(ctx context.Context, d DispatcherBackend, tenant *sqlcv1.Tenant, workerId uuid.UUID, interval time.Duration, refresh func(context.Context) error, l *zerolog.Logger) *throttledNotifier {
	return &throttledNotifier{ctx: ctx, d: d, tenant: tenant, workerId: workerId, interval: interval, refresh: refresh, l: l}
}

// request asks for a notification, immediately when the last one is older than the interval and
// otherwise when the window closes.
func (n *throttledNotifier) request() {
	n.mu.Lock()

	if n.stopped {
		n.mu.Unlock()
		return
	}

	n.dirty = true

	if n.timer != nil {
		n.mu.Unlock()
		return
	}

	if wait := n.interval - time.Since(n.lastFire); wait > 0 {
		n.timer = time.AfterFunc(wait, n.fireDeferred)
		n.mu.Unlock()

		return
	}

	n.lastFire = time.Now()
	n.dirty = false
	n.mu.Unlock()

	n.notify(true)
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
	dirty := n.dirty
	n.dirty = false
	n.mu.Unlock()

	n.notify(dirty)
}

func (n *throttledNotifier) fireDeferred() {
	n.mu.Lock()
	n.timer = nil

	if n.stopped {
		n.mu.Unlock()
		return
	}

	n.lastFire = time.Now()
	dirty := n.dirty
	n.dirty = false
	n.mu.Unlock()

	n.notify(dirty)
}

// notify runs outside the lock: the refresh and the dispatcher's publish must not be serialised
// behind the notifier's own bookkeeping. The refresh comes first, so the reload the
// notification causes reads the digest of the committed set.
func (n *throttledNotifier) notify(refresh bool) {
	if refresh {
		if err := n.refresh(n.ctx); err != nil {
			n.l.Error().Ctx(n.ctx).Err(err).Msg("could not refresh the worker's action hash; the scheduler reads its actions through the join until the next refresh")

			n.mu.Lock()
			n.dirty = true
			n.mu.Unlock()
		}
	}

	n.d.NotifyNewWorker(n.ctx, n.tenant, n.workerId)
}

// stop ends the notifier and reports whether a refresh is still owed, so the caller can do it.
// A callback that already passed the stopped check may still publish one notification after
// this returns; the scheduler simply reloads a worker that is on its way out, so stop does not
// wait for it.
func (n *throttledNotifier) stop() bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.stopped = true

	if n.timer != nil {
		n.timer.Stop()
		n.timer = nil
	}

	dirty := n.dirty
	n.dirty = false

	return dirty
}
