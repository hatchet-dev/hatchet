package hostgrpc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client/operatorclient"
	"github.com/hatchet-dev/hatchet/pkg/client/retry"
	"github.com/hatchet-dev/hatchet/pkg/operator"
)

const (
	// refusalReportTimeout bounds the failure report for an action the handler refused.
	refusalReportTimeout = 30 * time.Second

	// minStartQueueSize is the least number of starts a session queues ahead of its handler.
	minStartQueueSize = 64

	// defaultSlotUnits is what the engine gives a worker that asks for no slot config.
	defaultSlotUnits = 100
)

// startQueueSizeFor sizes the start queue from the worker's slot config: twice the slots the
// engine can assign at once, so the queue absorbs a burst of reassignments on top of the
// starts the handler holds, and never less than minStartQueueSize.
func startQueueSizeFor(slotConfig map[string]int32) int {
	units := 0

	for _, n := range slotConfig {
		units += int(n)
	}

	if len(slotConfig) == 0 {
		units = defaultSlotUnits
	}

	return max(minStartQueueSize, 2*units)
}

// reconnectFunc opens a replacement client session for the same identity, verified the way
// Open verifies the first one. It is the host's; a session built without one is not
// supervised.
type reconnectFunc func(ctx context.Context) (operatorclient.Session, operator.Registration, func(), error)

// session adapts an operatorclient.Session to operator.Session and runs the delivery that
// hands the client's assigned actions to the handler.
//
// Delivery reads the client's action channel eagerly, so the channel never backs up behind a
// handler that blocks: a cancel is handed to the handler on its own goroutine at once, and a
// start is queued for the one goroutine that calls the handler serially, as the dispatcher does
// in process. The queue is bounded (startQueueSizeFor); a start that arrives when it is full is
// refused with a retryable failure report rather than held, so the engine reassigns it once
// the handler has caught up.
//
// The session supervises its client session. A client session ends on its own when its stream
// fails for good, which the client reports on its error channel before closing its action
// channel; a revoked token is the usual cause. The session then opens a new client session
// through the host, which asks the token source again, restores the action set it advertised
// and resumes delivery under the new registration. It gives up only on a failure no retry can
// fix (ErrNoToken, a tenant mismatch, ErrNotSupported), reported through Done and Err.
//
// The worker the session reports under is the client's current one: the client registers a
// fresh worker exactly when the engine cannot resume the previous one, so a report stamped with
// a worker the session used to be is rewritten to the current worker. The worker is recorded
// per delivered start so the choice is explicit (see SendStepActionEvent).
//
// hub is created by the first OpenDurable and holds the session's one DurableTaskListener; a
// replacement client session gets a fresh hub on its first OpenDurable.
type session struct {
	handler operator.ActionHandler
	l       *zerolog.Logger

	// ctx bounds delivery and supervision; Close cancels it after the client session closed.
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// reconnect, backoff and startQueueSize are set by the host before startDelivery.
	reconnect      reconnectFunc
	backoff        func(ctx context.Context, attempt int) error
	startQueueSize int

	// starts is the bounded queue between the eager reader and the serial start worker.
	starts chan *contracts.AssignedAction

	// mu guards everything below.
	mu sync.Mutex

	cs  operatorclient.Session
	reg operator.Registration

	// release gives the client reference back to the host; nil once released.
	release func()

	// desired is the action set the session advertised, restored on a new client session.
	desired map[string]struct{}

	// workers is every worker id the session registered as; attempts records the worker
	// current when each start was delivered, by task run, until its terminal report.
	workers  map[string]struct{}
	attempts map[string]string

	hub    *durableHub
	closed bool

	// done closes once the session stopped serving, with err set when the session gave up
	// rather than being closed.
	done     chan struct{}
	doneOnce sync.Once
	err      error
}

func newSession(cs operatorclient.Session, reg operator.Registration, handler operator.ActionHandler, l *zerolog.Logger) *session {
	ctx, cancel := context.WithCancel(context.Background())

	sl := l.With().Str("tenant_id", reg.TenantId.String()).Logger()

	return &session{
		cs:             cs,
		reg:            reg,
		handler:        handler,
		l:              &sl,
		ctx:            ctx,
		cancel:         cancel,
		backoff:        retry.SleepStreamBackoff,
		startQueueSize: minStartQueueSize,
		desired:        map[string]struct{}{},
		workers:        map[string]struct{}{reg.WorkerId.String(): {}},
		attempts:       map[string]string{},
		done:           make(chan struct{}),
	}
}

// startDelivery starts the serial start worker and attaches the current client session.
func (s *session) startDelivery() error {
	s.starts = make(chan *contracts.AssignedAction, s.startQueueSize)

	s.wg.Add(1)

	go func() {
		defer s.wg.Done()
		s.serveStarts()
	}()

	return s.attach(s.client())
}

// attach starts reading cs's assigned actions and supervises the client session.
func (s *session) attach(cs operatorclient.Session) error {
	actions, errCh, err := cs.Actions(s.ctx)

	if err != nil {
		return err
	}

	s.wg.Add(1)

	go func() {
		defer s.wg.Done()
		s.supervise(cs, actions, errCh)
	}()

	return nil
}

// supervise reads cs until its action channel closes, then, unless the session is closing,
// replaces the client session.
func (s *session) supervise(cs operatorclient.Session, actions <-chan *contracts.AssignedAction, errCh <-chan error) {
	var (
		causeMu sync.Mutex
		cause   error
	)

	errsDone := make(chan struct{})

	go func() {
		defer close(errsDone)

		for err := range errCh {
			causeMu.Lock()
			cause = err
			causeMu.Unlock()

			s.l.Error().Err(err).Str("worker_id", s.workerId()).Msg("operator session stream failed")
		}
	}()

	s.read(actions)
	<-errsDone

	s.mu.Lock()
	closing := s.closed || s.ctx.Err() != nil
	current := s.cs == cs
	s.mu.Unlock()

	if closing || !current || s.reconnect == nil {
		return
	}

	causeMu.Lock()
	reason := cause
	causeMu.Unlock()

	if reason == nil {
		reason = errors.New("the client session ended its action stream")
	}

	s.recover(cs, reason)
}

// read hands cs's actions on: cancels to the handler at once, starts to the bounded queue.
func (s *session) read(actions <-chan *contracts.AssignedAction) {
	for action := range actions {
		if action.ActionType == contracts.ActionType_CANCEL_STEP_RUN {
			s.wg.Add(1)

			go func() {
				defer s.wg.Done()

				// a refused cancel has nothing to report
				_ = s.handler.HandleAction(s.ctx, action)
			}()

			continue
		}

		s.recordAttempt(action)

		select {
		case s.starts <- action:
		default:
			s.wg.Add(1)

			go func() {
				defer s.wg.Done()
				s.refuse(action, fmt.Errorf("the operator's start queue of %d is full", cap(s.starts)))
			}()
		}
	}
}

// serveStarts calls the handler for queued starts, one at a time, until the session ends.
func (s *session) serveStarts() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case action := <-s.starts:
			if err := s.handler.HandleAction(s.ctx, action); err != nil {
				s.refuse(action, err)
			}
		}
	}
}

// refuse reports an action the handler did not take as a retryable failure: in process a
// refused action is requeued by the dispatcher; over gRPC there is no requeue, so the engine
// applies the task's retry policy instead.
func (s *session) refuse(action *contracts.AssignedAction, err error) {
	s.l.Warn().Err(err).Str("worker_id", s.workerId()).Str("task_run_external_id", action.TaskRunExternalId).Msg("operator refused an assigned action; reporting a retryable failure")

	ctx, cancel := context.WithTimeout(context.Background(), refusalReportTimeout)
	defer cancel()

	retryCount := action.RetryCount

	if reportErr := s.SendStepActionEvent(ctx, &contracts.StepActionEvent{
		JobId:             action.JobId,
		JobRunId:          action.JobRunId,
		TaskId:            action.TaskId,
		TaskRunExternalId: action.TaskRunExternalId,
		ActionId:          action.ActionId,
		EventTimestamp:    timestamppb.Now(),
		EventType:         contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED,
		EventPayload:      err.Error(),
		RetryCount:        &retryCount,
	}); reportErr != nil {
		s.l.Error().Err(reportErr).Str("worker_id", s.workerId()).Str("task_run_external_id", action.TaskRunExternalId).Msg("could not report the refused action")
	}
}

// recover replaces a client session that ended: the durable hub over it is closed (open
// channels fail and their invocations retry), the client session is closed, and a new one is
// opened with backoff until it works or the failure is one no retry fixes.
func (s *session) recover(old operatorclient.Session, cause error) {
	s.mu.Lock()
	hub := s.hub
	s.hub = nil
	s.mu.Unlock()

	if hub != nil {
		hub.closeAll()
	}

	_ = old.Close(operatorclient.WithoutDrain())

	s.l.Warn().Err(cause).Str("worker_id", s.workerId()).Msg("operator session ended; opening a new one")

	for attempt := 0; ; attempt++ {
		if attempt > 0 {
			if err := s.backoff(s.ctx, attempt-1); err != nil {
				return
			}
		}

		if s.ctx.Err() != nil {
			return
		}

		cs, reg, release, err := s.reconnect(s.ctx)

		if err != nil {
			if s.ctx.Err() != nil {
				return
			}

			if isPermanent(err) {
				s.giveUp(err)
				return
			}

			s.l.Error().Err(err).Int("attempt", attempt+1).Str("worker_id", s.workerId()).Msg("could not open a new operator session")

			continue
		}

		if err := s.restoreActions(cs); err != nil {
			_ = cs.Close(operatorclient.WithoutDrain())
			release()

			if s.ctx.Err() != nil {
				return
			}

			s.l.Error().Err(err).Int("attempt", attempt+1).Str("worker_id", reg.WorkerId.String()).Msg("could not restore the action set on the new operator session")

			continue
		}

		s.mu.Lock()

		if s.closed {
			// Close ran meanwhile and closed the previous client session; this one is its
			// caller's to close
			s.mu.Unlock()
			_ = cs.Close(operatorclient.WithoutDrain())
			release()

			return
		}

		oldRelease := s.release
		s.cs = cs
		s.reg = reg
		s.release = release
		s.workers[reg.WorkerId.String()] = struct{}{}
		s.mu.Unlock()

		if oldRelease != nil {
			oldRelease()
		}

		if err := s.attach(cs); err != nil {
			s.giveUp(fmt.Errorf("could not start delivery on the new operator session: %w", err))
			return
		}

		s.l.Info().Str("worker_id", reg.WorkerId.String()).Bool("resumed", reg.Resumed).Msg("operator session reopened")

		return
	}
}

// restoreActions advertises the session's action set on a new client session.
func (s *session) restoreActions(cs operatorclient.Session) error {
	s.mu.Lock()
	ids := make([]string, 0, len(s.desired))

	for id := range s.desired {
		ids = append(ids, id)
	}

	s.mu.Unlock()

	if len(ids) == 0 {
		return nil
	}

	cs.AddActions(ids...)

	return cs.Flush(s.ctx)
}

// isPermanent reports whether a reconnect failure is one supervision cannot retry past.
func isPermanent(err error) bool {
	return errors.Is(err, ErrNoToken) || errors.Is(err, errTenantMismatch) || errors.Is(err, operator.ErrNotSupported)
}

// giveUp ends supervision: the session stops serving and reports why through Done and Err.
func (s *session) giveUp(err error) {
	s.mu.Lock()
	s.err = err
	release := s.release
	s.release = nil
	s.mu.Unlock()

	if release != nil {
		release()
	}

	s.l.Error().Err(err).Str("worker_id", s.workerId()).Msg("operator session gave up; the tenant is no longer served by it")

	s.doneOnce.Do(func() { close(s.done) })
}

// currentRegistration is the client's current registration, which the client may have changed
// on its own by registering a fresh worker when the engine could not resume the previous one.
func (s *session) currentRegistration() operator.Registration {
	s.mu.Lock()
	defer s.mu.Unlock()

	reg, err := parseRegistration(s.cs.Registration())

	if err != nil {
		return s.reg
	}

	s.workers[reg.WorkerId.String()] = struct{}{}
	s.reg = reg

	return reg
}

func (s *session) workerId() string {
	return s.currentRegistration().WorkerId.String()
}

func (s *session) Registration() operator.Registration {
	return s.currentRegistration()
}

// Done implements operator.Session.
func (s *session) Done() <-chan struct{} {
	return s.done
}

// Err implements operator.Session.
func (s *session) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.err
}

func (s *session) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.closed
}

// client is the current client session.
func (s *session) client() operatorclient.Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.cs
}

// recordAttempt notes the worker current when a start was delivered.
func (s *session) recordAttempt(action *contracts.AssignedAction) {
	worker := s.workerId()

	s.mu.Lock()
	s.attempts[action.TaskRunExternalId] = worker
	s.mu.Unlock()
}

func (s *session) addDesired(ids []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range ids {
		s.desired[id] = struct{}{}
	}
}

func (s *session) removeDesired(ids []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range ids {
		delete(s.desired, id)
	}
}

// PutWorkflow implements operator.Session over the admin service; the client derives the
// action ids without touching the streamed action set.
func (s *session) PutWorkflow(ctx context.Context, wf *v1.CreateWorkflowVersionRequest) ([]string, error) {
	if s.isClosed() {
		return nil, operator.ErrSessionClosed
	}

	_, actions, err := s.client().PutWorkflow(ctx, wf)

	if err != nil {
		return nil, err
	}

	return actions, nil
}

// AddActions implements operator.Session. The client coalesces and chunks the delta onto the
// Listen stream; nothing is sent until its flusher runs, so the call never blocks. The ids
// join the session's desired set, which a replacement client session is given.
func (s *session) AddActions(_ context.Context, ids []string) error {
	if s.isClosed() {
		return operator.ErrSessionClosed
	}

	s.addDesired(ids)
	s.client().AddActions(ids...)

	return nil
}

// RemoveActions implements operator.Session; see AddActions.
func (s *session) RemoveActions(_ context.Context, ids []string) error {
	if s.isClosed() {
		return operator.ErrSessionClosed
	}

	s.removeDesired(ids)
	s.client().RemoveActions(ids...)

	return nil
}

// Flush implements operator.Session: it waits until the client has sent every queued delta and
// the engine acknowledged it, and reports the last send failure.
func (s *session) Flush(ctx context.Context) error {
	if s.isClosed() {
		return operator.ErrSessionClosed
	}

	return s.client().Flush(ctx)
}

// SendStepActionEvent implements operator.Session. The event's worker is the worker the engine
// knows the session as: an empty WorkerId is filled with the worker recorded when the task's
// start was delivered if that is still the current worker, and with the current worker
// otherwise; a WorkerId naming any worker the session has registered as is rewritten the same
// way. The client only ever registers a fresh worker when the engine cannot resume the
// previous one, so a report under the previous worker could only be refused, while the current
// worker is the one the engine holds the task against. A WorkerId naming a worker that was never
// this session's is passed through for the engine to refuse.
func (s *session) SendStepActionEvent(ctx context.Context, ev *contracts.StepActionEvent) error {
	if s.isClosed() {
		return operator.ErrSessionClosed
	}

	current := s.workerId()

	s.mu.Lock()

	recorded, attempted := s.attempts[ev.TaskRunExternalId]
	_, ours := s.workers[ev.WorkerId]

	if isTerminalStepEvent(ev.EventType) {
		delete(s.attempts, ev.TaskRunExternalId)
	}

	cs := s.cs
	s.mu.Unlock()

	if ev.WorkerId == "" || ours {
		worker := current

		if attempted && recorded == current {
			worker = recorded
		}

		if ev.WorkerId != "" && ev.WorkerId != worker {
			s.l.Debug().Str("worker_id", worker).Str("reported_worker_id", ev.WorkerId).Str("task_run_external_id", ev.TaskRunExternalId).Msg("report rewritten to the session's current worker")
		}

		ev.WorkerId = worker
	}

	_, err := cs.SendStepActionEvent(ctx, ev)

	return err
}

// isTerminalStepEvent reports whether an event ends the session's record of a task run.
func isTerminalStepEvent(eventType contracts.StepActionEventType) bool {
	switch eventType {
	case contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED,
		contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED,
		contracts.StepActionEventType_STEP_EVENT_TYPE_CANCELLED:
		return true
	default:
		return false
	}
}

// OpenDurable implements operator.Session over one DurableTaskListener per client session,
// opened on the client session's durable task stream and multiplexed by task external id and
// invocation.
func (s *session) OpenDurable(_ context.Context, taskExternalId uuid.UUID, invocation int32) (operator.DurableChannel, error) {
	s.mu.Lock()

	if s.closed {
		s.mu.Unlock()
		return nil, operator.ErrSessionClosed
	}

	if s.hub == nil {
		s.hub = newDurableHub(s.cs, s.l)
	}

	hub := s.hub
	s.mu.Unlock()

	return hub.open(taskExternalId.String(), invocation)
}

// Pause implements operator.Session through the pause message on the Listen stream; it returns
// once the engine has acknowledged it, at which point nothing more is delivered.
func (s *session) Pause(ctx context.Context) error {
	if s.isClosed() {
		return operator.ErrSessionClosed
	}

	return s.client().Pause(ctx)
}

// Close implements operator.Session: open durable channels are closed and the durable
// listener stopped, then the client session is closed, which pauses the worker, waits for the
// actions already handed to the handler to be reported, flushes pending deltas and ends the
// stream, and only then are delivery and supervision released and the client reference given
// back. Close runs once; later calls return nil.
func (s *session) Close(context.Context) error {
	s.mu.Lock()

	if s.closed {
		s.mu.Unlock()
		return nil
	}

	s.closed = true
	hub := s.hub
	cs := s.cs
	s.mu.Unlock()

	if hub != nil {
		hub.closeAll()
	}

	err := cs.Close()

	s.cancel()
	s.wg.Wait()

	s.mu.Lock()
	release := s.release
	s.release = nil
	s.attempts = map[string]string{}
	s.mu.Unlock()

	if release != nil {
		release()
	}

	s.doneOnce.Do(func() { close(s.done) })

	return err
}
