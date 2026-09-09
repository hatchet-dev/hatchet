package enginelink

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/link"
)

// ErrFlowControl is returned by HandleAction when the registration's action buffer is full,
// which only happens when the core has stopped reading. The dispatcher treats it like a
// stream in flow control: the assignment fails and the task is requeued.
var ErrFlowControl = errors.New("serverless registration action buffer is full, flow control is active")

// ErrClosed is returned once the registration's action stream has ended.
var ErrClosed = errors.New("serverless registration is closed")

const (
	// heartbeatInterval matches the SDK's and the operator manager's cadence.
	heartbeatInterval = 4 * time.Second

	// heartbeatTimeout bounds one heartbeat write.
	heartbeatTimeout = 5 * time.Second

	// deactivateTimeout bounds the detached deactivation write on Close.
	deactivateTimeout = 20 * time.Second
)

// registration is one in-engine registration: the link.Registration the core drives, and
// through sessionOperator the operator.Operator the dispatcher delivers to.
type registration struct {
	link   *Link
	l      *zerolog.Logger
	tenant *sqlcv1.Tenant

	// actions is the bounded buffer between the dispatcher's HandleAction and the core's
	// action loop; errCh never carries an error because the in-process link cannot fail, but
	// the Registration contract closes it with the stream.
	actions chan *contracts.AssignedAction
	errCh   chan error

	// done is closed when the stream ends so the Actions watcher goroutine exits.
	done chan struct{}

	// release removes the dispatcher session; set by Open after AddOperatorSession.
	release func()

	heartbeatCancel context.CancelFunc
	heartbeatDone   chan struct{}

	// mu guards ended and started. HandleAction sends under the read lock so the stream can
	// be closed under the write lock without racing a send.
	mu      sync.RWMutex
	ended   bool
	started bool

	closeOnce sync.Once
	closeErr  error

	workerId uuid.UUID

	// sessionId is the listener session recorded on the worker row by Open; Close deactivates
	// the worker fenced on it.
	sessionId uuid.UUID
}

func newRegistration(l *Link, tenant *sqlcv1.Tenant, workerId uuid.UUID, sessionId uuid.UUID, buffer int, logger *zerolog.Logger) *registration {
	return &registration{
		link:      l,
		l:         logger,
		tenant:    tenant,
		actions:   make(chan *contracts.AssignedAction, buffer),
		errCh:     make(chan error, 1),
		done:      make(chan struct{}),
		workerId:  workerId,
		sessionId: sessionId,
	}
}

// WorkerId implements link.Registration.
func (r *registration) WorkerId() string {
	return r.workerId.String()
}

// sessionOperator is the operator.Operator face of a registration. It exists because the
// two interfaces both declare WorkerId with different return types; everything else is the
// registration's own method.
type sessionOperator struct {
	*registration
}

// WorkerId implements operator.Operator.
func (s sessionOperator) WorkerId() uuid.UUID {
	return s.workerId
}

// HandleAction implements operator.Operator: START_STEP_RUN and CANCEL_STEP_RUN are pushed
// onto the action buffer for the core's loop; it never blocks the dispatcher. Other action
// types have no meaning for a serverless worker and are dropped.
func (r *registration) HandleAction(_ context.Context, action *contracts.AssignedAction) error {
	switch action.ActionType {
	case contracts.ActionType_START_STEP_RUN, contracts.ActionType_CANCEL_STEP_RUN:
	default:
		r.l.Debug().Str("action_type", action.ActionType.String()).Msg("dropping unsupported action type for serverless worker")
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.ended {
		return ErrClosed
	}

	select {
	case r.actions <- action:
		return nil
	default:
		return ErrFlowControl
	}
}

// Start implements operator.Operator. The link opens the session itself; the dispatcher
// never calls it.
func (r *registration) Start(context.Context, operator.Session) error { return nil }

// Drain implements operator.Operator. The core drains in-flight deliveries itself, so there
// is nothing to do here; the dispatcher never calls it for sessions it does not own.
func (r *registration) Drain(context.Context) {}

// Actions implements link.Registration. The channels close when ctx is cancelled or the
// registration is closed.
func (r *registration) Actions(ctx context.Context) (<-chan *contracts.AssignedAction, <-chan error, error) {
	r.mu.Lock()

	if r.started {
		r.mu.Unlock()
		return nil, nil, errors.New("Actions may only be called once per registration")
	}

	r.started = true
	r.mu.Unlock()

	go func() {
		select {
		case <-ctx.Done():
			r.endStream()
		case <-r.done:
		}
	}()

	return r.actions, r.errCh, nil
}

// endStream closes the action channels once. Later HandleAction calls return ErrClosed so the
// dispatcher requeues instead of delivering into a closed buffer.
func (r *registration) endStream() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ended {
		return
	}

	r.ended = true

	close(r.actions)
	close(r.errCh)
	close(r.done)
}

// PutWorkflow implements link.Registration: put the workflow through the admin service with
// the tenant on the context and return its derived action ids. The worker's action set is
// untouched; the core adds the ids with AddActions.
func (r *registration) PutWorkflow(ctx context.Context, wf *v1.CreateWorkflowVersionRequest) ([]string, error) {
	derived, err := workflowActions(wf)

	if err != nil {
		return nil, err
	}

	if _, err := r.link.admin.PutWorkflow(withTenant(ctx, r.tenant), wf); err != nil {
		return nil, fmt.Errorf("could not put workflow %s: %w", wf.Name, err)
	}

	return derived, nil
}

// AddActions implements link.Registration: the ids are linked to the worker in chunks, and
// the scheduler is notified once when the set grew. The write is synchronous, so Flush has
// nothing to wait for.
func (r *registration) AddActions(ctx context.Context, ids []string) error {
	return r.applyDelta(ctx, ids, r.link.workers.AddWorkerActions, "add")
}

// RemoveActions implements link.Registration; see AddActions.
func (r *registration) RemoveActions(ctx context.Context, ids []string) error {
	return r.applyDelta(ctx, ids, r.link.workers.RemoveWorkerActions, "remove")
}

// Flush implements link.Registration. Deltas are applied synchronously by AddActions and
// RemoveActions, so there is never anything pending.
func (r *registration) Flush(context.Context) error {
	return nil
}

type deltaFn func(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, actionIds []string) (int, error)

// applyDelta validates ids and applies them with fn in chunks of maxActionsPerDelta. The
// scheduler reloads the worker's action set on notify, so one notification covers every
// chunk and is skipped when nothing changed.
func (r *registration) applyDelta(ctx context.Context, ids []string, fn deltaFn, verb string) error {
	ids = unionActions(ids)

	if len(ids) == 0 {
		return nil
	}

	if err := r.link.v.Validate(registerOpts{Name: r.link.name, Actions: ids}); err != nil {
		return fmt.Errorf("invalid actions: %w", err)
	}

	tctx := withTenant(ctx, r.tenant)
	changed := 0

	for start := 0; start < len(ids); start += maxActionsPerDelta {
		end := min(start+maxActionsPerDelta, len(ids))

		n, err := fn(tctx, r.tenant.ID, r.workerId, ids[start:end])

		if err != nil {
			return fmt.Errorf("could not %s actions for worker %s: %w", verb, r.workerId, err)
		}

		changed += n
	}

	if changed > 0 {
		r.link.dispatcher.NotifyNewWorker(tctx, r.tenant, r.workerId)
	}

	r.l.Debug().Str("op", verb).Int("ids", len(ids)).Int("changed", changed).Msg("applied serverless worker actions delta")

	return nil
}

// SendStepActionEvent implements link.Registration. The event is handled exactly like an SDK
// worker's; the worker id is filled in when the caller left it empty.
func (r *registration) SendStepActionEvent(ctx context.Context, ev *contracts.StepActionEvent) error {
	if ev.WorkerId == "" {
		ev.WorkerId = r.workerId.String()
	}

	_, err := r.link.dispatcher.SendStepActionEvent(withTenant(ctx, r.tenant), ev)

	return err
}

// Close implements link.Registration: end the action stream, remove the dispatcher session,
// stop heartbeats and deactivate the worker, fenced on the session id so a newer session on
// the same worker id is never clobbered. A superseded session (pgx.ErrNoRows) has nothing to
// do.
func (r *registration) Close() error {
	r.closeOnce.Do(func() {
		r.endStream()

		if r.release != nil {
			r.release()
		}

		r.stopHeartbeats()

		ctx, cancel := context.WithTimeout(context.Background(), deactivateTimeout)
		defer cancel()

		_, err := r.link.workers.DeactivateWorkerListener(ctx, r.tenant.ID, r.workerId, r.sessionId)

		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			r.closeErr = fmt.Errorf("could not deactivate serverless worker %s: %w", r.workerId, err)
		}

		r.l.Info().Msg("serverless worker deregistered from engine")
	})

	return r.closeErr
}

// startHeartbeats writes the worker heartbeat every heartbeatInterval, as the SDK does over
// its stream and the operator manager does for in-engine operators. The loop is detached
// from any request context and ends on Close.
func (r *registration) startHeartbeats() {
	ctx, cancel := context.WithCancel(context.Background())

	r.heartbeatCancel = cancel
	r.heartbeatDone = make(chan struct{})

	interval := r.link.heartbeatInterval

	if interval <= 0 {
		interval = heartbeatInterval
	}

	go func() {
		defer close(r.heartbeatDone)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				writeCtx, writeCancel := context.WithTimeout(ctx, heartbeatTimeout)

				err := r.link.workers.UpdateWorkerHeartbeat(writeCtx, r.tenant.ID, r.workerId, time.Now().UTC())

				writeCancel()

				if err != nil && ctx.Err() == nil {
					r.l.Error().Err(err).Msg("could not update serverless worker heartbeat")
				}
			}
		}
	}()
}

func (r *registration) stopHeartbeats() {
	if r.heartbeatCancel == nil {
		return
	}

	r.heartbeatCancel()
	<-r.heartbeatDone
}

// OpenDurable implements link.Registration over the dispatcher's in-process durable session.
// It performs the handshake the DAG operator does: the first request registers this worker,
// and the session is usable once the engine acks it.
func (r *registration) OpenDurable(ctx context.Context, taskExternalId string, invocation int32) (link.DurableChannel, error) {
	taskId, err := uuid.Parse(taskExternalId)

	if err != nil {
		return nil, fmt.Errorf("invalid task external id %q: %w", taskExternalId, err)
	}

	return openDurable(ctx, r.link.dispatcher, r.tenant, r.workerId, taskId, invocation)
}
