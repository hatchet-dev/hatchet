package hostgrpc

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client/operatorclient"
	"github.com/hatchet-dev/hatchet/pkg/operator"
)

// refusalReportTimeout bounds the failure report for an action the handler refused.
const refusalReportTimeout = 30 * time.Second

// session adapts an operatorclient.Session to operator.Session and runs the deliver loop that
// hands the client's assigned actions to the handler. hub is created by the first OpenDurable
// and holds the session's one DurableTaskListener.
type session struct {
	cs      operatorclient.Session
	reg     operator.Registration
	handler operator.ActionHandler
	l       *zerolog.Logger

	// ctx bounds the deliver loop; cancelling it after the client session closed lets the loop
	// return without ending the stream early.
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	hub    *durableHub
	mu     sync.Mutex
	closed bool
}

func newSession(cs operatorclient.Session, reg operator.Registration, handler operator.ActionHandler, l *zerolog.Logger) *session {
	ctx, cancel := context.WithCancel(context.Background())

	sl := l.With().Str("worker_id", reg.WorkerId.String()).Logger()

	return &session{cs: cs, reg: reg, handler: handler, l: &sl, ctx: ctx, cancel: cancel}
}

// startDelivery attaches the handler to the client's action stream.
func (s *session) startDelivery() error {
	actions, errCh, err := s.cs.Actions(s.ctx)

	if err != nil {
		return err
	}

	s.wg.Add(2)

	go func() {
		defer s.wg.Done()
		s.deliver(actions)
	}()

	go func() {
		defer s.wg.Done()

		for err := range errCh {
			s.l.Error().Err(err).Msg("operator session stream failed")
		}
	}()

	return nil
}

// deliver hands each assigned action to the handler, one at a time, the way the dispatcher does
// in process. In process a refused action is requeued by the dispatcher; over gRPC there is no
// requeue, so a refusal is reported as a retryable failure and the engine applies the task's
// retry policy.
func (s *session) deliver(actions <-chan *contracts.AssignedAction) {
	for action := range actions {
		err := s.handler.HandleAction(s.ctx, action)

		if err == nil || action.ActionType == contracts.ActionType_CANCEL_STEP_RUN {
			continue
		}

		s.l.Warn().Err(err).Str("task_run_external_id", action.TaskRunExternalId).Msg("operator refused an assigned action; reporting a retryable failure")

		ctx, cancel := context.WithTimeout(context.Background(), refusalReportTimeout)

		retryCount := action.RetryCount

		if _, reportErr := s.cs.SendStepActionEvent(ctx, &contracts.StepActionEvent{
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
			s.l.Error().Err(reportErr).Str("task_run_external_id", action.TaskRunExternalId).Msg("could not report the refused action")
		}

		cancel()
	}
}

func (s *session) Registration() operator.Registration {
	return s.reg
}

func (s *session) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.closed
}

// PutWorkflow implements operator.Session over the admin service; the client derives the
// action ids without touching the streamed action set.
func (s *session) PutWorkflow(ctx context.Context, wf *v1.CreateWorkflowVersionRequest) ([]string, error) {
	if s.isClosed() {
		return nil, operator.ErrSessionClosed
	}

	_, actions, err := s.cs.PutWorkflow(ctx, wf)

	if err != nil {
		return nil, err
	}

	return actions, nil
}

// AddActions implements operator.Session. The client coalesces and chunks the delta onto the
// Listen stream; nothing is sent until its flusher runs, so the call never blocks.
func (s *session) AddActions(_ context.Context, ids []string) error {
	if s.isClosed() {
		return operator.ErrSessionClosed
	}

	s.cs.AddActions(ids...)

	return nil
}

// RemoveActions implements operator.Session; see AddActions.
func (s *session) RemoveActions(_ context.Context, ids []string) error {
	if s.isClosed() {
		return operator.ErrSessionClosed
	}

	s.cs.RemoveActions(ids...)

	return nil
}

// Flush implements operator.Session: it waits until the client has sent every queued delta and
// the engine acknowledged it, and reports the last send failure.
func (s *session) Flush(ctx context.Context) error {
	if s.isClosed() {
		return operator.ErrSessionClosed
	}

	return s.cs.Flush(ctx)
}

// SendStepActionEvent implements operator.Session. The client fills an empty worker id; the
// engine refuses another operator's worker.
func (s *session) SendStepActionEvent(ctx context.Context, ev *contracts.StepActionEvent) error {
	if s.isClosed() {
		return operator.ErrSessionClosed
	}

	_, err := s.cs.SendStepActionEvent(ctx, ev)

	return err
}

// OpenDurable implements operator.Session over one DurableTaskListener per session, opened
// on the client session's durable task stream and multiplexed by task external id and
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

// Pause implements operator.Session through the unary PauseWorker RPC, which works while the
// Listen stream is reconnecting.
func (s *session) Pause(ctx context.Context) error {
	if s.isClosed() {
		return operator.ErrSessionClosed
	}

	return s.cs.Pause(ctx)
}

// Close implements operator.Session: open durable channels are closed and the durable
// listener stopped, then the client session is closed, which pauses the worker, waits for the
// actions already handed to the handler to be reported, flushes pending deltas and ends the
// stream, and only then is the deliver loop released. Close runs once; later calls return nil.
func (s *session) Close(context.Context) error {
	s.mu.Lock()

	if s.closed {
		s.mu.Unlock()
		return nil
	}

	s.closed = true
	hub := s.hub
	s.mu.Unlock()

	if hub != nil {
		hub.closeAll()
	}

	err := s.cs.Close()

	s.cancel()
	s.wg.Wait()

	return err
}
