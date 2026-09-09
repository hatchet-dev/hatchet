package client

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

const (
	// operatorHeartbeatInterval matches the cadence the OperatorService
	// protocol expects; a stale heartbeat makes the scheduler skip the worker.
	operatorHeartbeatInterval = 4 * time.Second

	// operatorRegisterTimeout caps how long one handshake (Register call,
	// Listen opened, start sent) may take, on top of the constructor ctx.
	operatorRegisterTimeout = 30 * time.Second

	// operatorCloseFlushTimeout bounds the flush Close performs before it
	// tears the session down.
	operatorCloseFlushTimeout = 2 * time.Second

	// operatorCloseDrainTimeout bounds how long Close waits for the actions
	// already handed to the consumer to be reported before it hangs up.
	operatorCloseDrainTimeout = 30 * time.Second
)

var errOperatorActionsStarted = errors.New("operator session actions already started")

// operatorListenClient is one Listen stream plus the cancel for its own
// context. The stream context is a child of the reconnecting stream's
// lifecycle context so that Close releases every stream; cancel is invoked
// directly when the handshake or the replay fails and the stream is never
// published. Retired streams are half-closed with CloseSend and left to the
// server to end, so the receive loop hands off on EOF like the other
// listeners in this package.
type operatorListenClient struct {
	v1.OperatorService_ListenClient
	cancel context.CancelFunc
}

// operatorDurableTaskClient adapts the OperatorService durable stream to the
// V1Dispatcher one DurableTaskListener expects and rewrites the worker id on
// the register message to the session's current registration, so a listener
// built before a reconnect still registers the worker the engine now knows.
type operatorDurableTaskClient struct {
	v1.OperatorService_DurableTaskClient
	workerId func() string
}

func (c *operatorDurableTaskClient) Send(req *v1.DurableTaskRequest) error {
	if register := req.GetRegisterWorker(); register != nil {
		register.WorkerId = c.workerId()
	}

	return c.OperatorService_DurableTaskClient.Send(req)
}

// actionInbox buffers assigned actions between the session's receive loop
// and the consumer Actions starts. The receive loop must never block on a
// consumer: delta acks share the stream with actions, and Flush may wait on
// an ack before Actions has been called.
type actionInbox struct {
	items []*dispatchercontracts.AssignedAction
	ready chan struct{}
	mu    sync.Mutex
}

func newActionInbox() *actionInbox {
	return &actionInbox{ready: make(chan struct{}, 1)}
}

func (b *actionInbox) push(action *dispatchercontracts.AssignedAction) {
	b.mu.Lock()
	b.items = append(b.items, action)
	b.mu.Unlock()

	select {
	case b.ready <- struct{}{}:
	default:
	}
}

// pop returns the oldest buffered action, or nil when the inbox is empty.
func (b *actionInbox) pop() *dispatchercontracts.AssignedAction {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.items) == 0 {
		return nil
	}

	action := b.items[0]
	b.items[0] = nil
	b.items = b.items[1:]

	if len(b.items) == 0 {
		b.items = nil
	}

	return action
}

// operatorSession is the OperatorSession implementation. It owns one
// reconnecting Listen stream whose constructor performs the Register call and
// the start handshake and whose replay callback resends unacknowledged
// action deltas, a heartbeat goroutine that sends through retrySend so a
// dead stream reconnects, the receive loop that demultiplexes assigned
// actions and delta acks, and the action delta queue (operator_actions.go).
// The receive and heartbeat loops run from connect until Close so that
// Flush can observe acks before Actions is called; Actions only attaches a
// consumer to the inbox.
// NOTE: field order follows govet fieldalignment (enforced by the pre-commit
// autofixer); mu guards register, reg, durables, loopErr, inflight, idle,
// started, consuming and closed. Lock order is stream.sendMu → mu, never reverse: mu is only
// taken in short critical sections that do no stream I/O.
type operatorSession struct {
	client     v1.OperatorServiceClient
	admin      v1.AdminServiceClient
	ctxLoader  *contextLoader
	l          *zerolog.Logger
	stream     *reconnectingStream[*operatorListenClient]
	register   *v1.OperatorRegisterRequest
	inbox      *actionInbox
	loopCtx    context.Context
	loopCancel context.CancelFunc
	loopDone   chan struct{}
	loopErr    error
	durables   []*DurableTaskListener
	reg        OperatorRegistration
	actions    *actionDeltaQueue

	// inflight holds the task runs handed to the consumer that have not been
	// reported as finished, and idle is closed the moment the last one is, so
	// Close can drain. Both are guarded by mu.
	inflight map[string]struct{}
	idle     chan struct{}

	heartbeatInterval time.Duration

	wg           sync.WaitGroup
	mu           sync.Mutex
	resumeWorker bool
	started      bool
	consuming    bool
	closed       bool
}

func newOperatorSession(
	client v1.OperatorServiceClient,
	admin v1.AdminServiceClient,
	ctxLoader *contextLoader,
	l *zerolog.Logger,
	register *v1.OperatorRegisterRequest,
	resumeWorker bool,
) *operatorSession {
	sl := l.With().Str("operator", register.Name).Logger()

	loopCtx, loopCancel := context.WithCancel(context.Background())

	s := &operatorSession{
		client:            client,
		admin:             admin,
		ctxLoader:         ctxLoader,
		l:                 &sl,
		register:          register,
		resumeWorker:      resumeWorker,
		heartbeatInterval: operatorHeartbeatInterval,
		inbox:             newActionInbox(),
		loopCtx:           loopCtx,
		loopCancel:        loopCancel,
		loopDone:          make(chan struct{}),
		inflight:          map[string]struct{}{},
	}

	s.stream = newReconnectingStreamWithLifecycle(
		context.Background(),
		&sl,
		"operator listener",
		s.openListenStream,
		func(c *operatorListenClient) error {
			return c.CloseSend()
		},
		s.replayActions,
	)

	s.actions = newActionDeltaQueue(&sl, s.stream, actionDeltaFlushInterval, maxActionsPerDelta)

	return s
}

// connect opens the first stream and starts the session loops.
func (s *operatorSession) connect(ctx context.Context) error {
	if err := s.stream.connectSync(ctx); err != nil {
		return err
	}

	s.start()

	return nil
}

// start launches the receive and heartbeat loops once. The WaitGroup is
// incremented under mu, in the same critical section that checks closed, so
// Close never waits on a count that a concurrent start is still adding to.
func (s *operatorSession) start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.started {
		return
	}

	s.started = true
	s.wg.Add(2)

	go func() {
		defer s.wg.Done()
		s.heartbeatLoop(s.loopCtx)
	}()

	go func() {
		defer s.wg.Done()
		s.receiveLoop(s.loopCtx)
	}()
}

// openListenStream is the reconnecting stream constructor: it calls Register
// (resuming the previous worker when enabled), opens Listen and sends the
// start message. The handshake is bounded by ctx and
// operatorRegisterTimeout; the stream itself outlives ctx and is bound to
// the lifecycle context. Action deltas are replayed by replayActions once
// the stream is constructed.
func (s *operatorSession) openListenStream(ctx context.Context) (*operatorListenClient, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	handshakeCtx, cancelHandshake := context.WithTimeout(ctx, operatorRegisterTimeout)
	defer cancelHandshake()

	registered, err := s.client.Register(s.ctxLoader.newContext(handshakeCtx), s.registerRequest())
	if err != nil {
		return nil, fmt.Errorf("could not register operator: %w", err)
	}

	s.setRegistration(OperatorRegistration{
		TenantId:   registered.TenantId,
		OperatorId: registered.OperatorId,
		WorkerId:   registered.WorkerId,
		Resumed:    registered.Resumed,
	})

	streamCtx, cancelStream := context.WithCancel(s.stream.lifecycleContext())
	stopAfter := context.AfterFunc(handshakeCtx, cancelStream)

	fail := func(err error) (*operatorListenClient, error) {
		stopAfter()
		cancelStream()
		if herr := handshakeCtx.Err(); herr != nil {
			return nil, fmt.Errorf("operator handshake did not complete: %w", herr)
		}
		return nil, err
	}

	listen, err := s.client.Listen(s.opCtx(streamCtx), grpc_retry.Disable())
	if err != nil {
		return fail(err)
	}

	if err := listen.Send(&v1.OperatorListenRequest{
		Message: &v1.OperatorListenRequest_Start{Start: &v1.OperatorListenStart{WorkerId: registered.WorkerId}},
	}); err != nil {
		return fail(fmt.Errorf("could not send operator start message: %w", err))
	}

	if !stopAfter() {
		// The handshake deadline fired between the last send and here, so the
		// stream context is already cancelled and the stream is unusable.
		cancelStream()
		return nil, fmt.Errorf("operator handshake did not complete: %w", handshakeCtx.Err())
	}

	s.l.Debug().Ctx(ctx).
		Str("operator_id", registered.OperatorId).
		Str("worker_id", registered.WorkerId).
		Bool("resumed", registered.Resumed).
		Msg("operator registered")

	return &operatorListenClient{OperatorService_ListenClient: listen, cancel: cancelStream}, nil
}

// replayActions is the reconnecting stream's replay callback: it runs under
// sendMu on every new stream before the stream is published and brings the
// engine's view of the action set up to date (see actionDeltaQueue.replay).
// A stream whose replay fails is cancelled here because it is never
// published.
func (s *operatorSession) replayActions(_ context.Context, c *operatorListenClient) error {
	if err := s.actions.replay(c, s.Registration().Resumed); err != nil {
		c.cancel()
		return fmt.Errorf("could not replay operator actions: %w", err)
	}

	return nil
}

// registerRequest snapshots the register template and, when resuming, sets
// the worker id from the most recent registration.
func (s *operatorSession) registerRequest() *v1.OperatorRegisterRequest {
	s.mu.Lock()
	defer s.mu.Unlock()

	req := &v1.OperatorRegisterRequest{
		Name:        s.register.Name,
		SlotConfig:  s.register.SlotConfig,
		Labels:      s.register.Labels,
		RuntimeInfo: s.register.RuntimeInfo,
	}

	if s.resumeWorker && s.reg.WorkerId != "" {
		workerId := s.reg.WorkerId
		req.WorkerId = &workerId
	}

	return req
}

func (s *operatorSession) setRegistration(reg OperatorRegistration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reg = reg
}

func (s *operatorSession) Registration() OperatorRegistration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reg
}

// opCtx attaches the bearer token and the hatchet-operator-id metadata every
// RPC after Register requires.
func (s *operatorSession) opCtx(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(
		s.ctxLoader.newContext(ctx),
		operatorIdMetadataKey, s.Registration().OperatorId,
	)
}

func (s *operatorSession) Actions(ctx context.Context) (<-chan *dispatchercontracts.AssignedAction, <-chan error, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, nil, errListenerClosed
	}
	if s.consuming {
		s.mu.Unlock()
		return nil, nil, errOperatorActionsStarted
	}
	s.consuming = true
	// the count is added under mu so a concurrent Close either sees the
	// consumer or is seen by it
	s.wg.Add(1)
	s.mu.Unlock()

	ch := make(chan *dispatchercontracts.AssignedAction)
	errCh := make(chan error, 1)

	s.l.Debug().Ctx(ctx).Msg("starting operator action consumer")

	go func() {
		defer s.wg.Done()
		s.deliverLoop(ctx, ch, errCh)
	}()

	return ch, errCh, nil
}

func (s *operatorSession) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(s.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		err := s.stream.retrySend(ctx, func(c *operatorListenClient) error {
			return c.Send(&v1.OperatorListenRequest{
				Message: &v1.OperatorListenRequest_Heartbeat{
					Heartbeat: &v1.OperatorHeartbeat{HeartbeatAt: timestamppb.Now()},
				},
			})
		})

		if err == nil || ctx.Err() != nil {
			continue
		}

		s.l.Error().Ctx(ctx).Err(err).Str("worker_id", s.Registration().WorkerId).Msg("could not send operator heartbeat")

		if errors.Is(err, errListenerClosed) {
			return
		}
	}
}

// receiveLoop runs the stream's receive side for the life of the session:
// acks go to the delta queue and actions to the inbox. Leaving it, for any
// reason, ends the session's stream and the heartbeat loop: a session
// without a receive loop cannot run actions or confirm deltas, so keeping
// the worker alive would only attract work.
func (s *operatorSession) receiveLoop(ctx context.Context) {
	defer close(s.loopDone)
	defer func() {
		s.loopCancel()
		s.actions.stop()
		if err := s.stream.Close(); err != nil {
			s.l.Error().Ctx(ctx).Err(err).Msg("failed to close operator listener stream")
		}
	}()

	classify := newStreamClassifier(func(ctx context.Context) bool {
		return ctx.Err() == nil
	})

	err := listenStream(ctx, s.stream,
		func(c *operatorListenClient) (*v1.OperatorListenResponse, error) {
			return c.Recv()
		},
		func(resp *v1.OperatorListenResponse) error {
			switch msg := resp.Message.(type) {
			case *v1.OperatorListenResponse_Ack:
				s.actions.ack(msg.Ack.Sequence)
			case *v1.OperatorListenResponse_Action:
				s.l.Debug().Ctx(ctx).
					Str("action_type", msg.Action.ActionType.String()).
					Str("action_id", msg.Action.ActionId).
					Msg("received operator action")

				s.inbox.push(msg.Action)
			default:
				s.l.Warn().Ctx(ctx).Msgf("ignoring unexpected message on the operator listener stream: %T", msg)
			}

			return nil
		},
		classify,
	)
	if err != nil && ctx.Err() == nil {
		s.mu.Lock()
		s.loopErr = err
		s.mu.Unlock()
	}
}

// terminalErr reports why the receive loop ended, if it ended on an error.
func (s *operatorSession) terminalErr() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loopErr
}

// deliverLoop hands buffered actions to the consumer until ctx ends, the
// session closes, or the receive loop exits. Cancelling ctx ends the
// session's stream, as a consumer that has stopped taking actions must not
// keep its worker active.
func (s *operatorSession) deliverLoop(ctx context.Context, ch chan<- *dispatchercontracts.AssignedAction, errCh chan<- error) {
	defer close(ch)
	defer close(errCh)

	stopOnCancel := context.AfterFunc(ctx, func() {
		s.loopCancel()
		_ = s.stream.Close()
	})
	defer stopOnCancel()

	for {
		if action := s.inbox.pop(); action != nil {
			select {
			case ch <- action:
				s.startAction(action)
				continue
			case <-ctx.Done():
				return
			case <-s.loopDone:
			}
		}

		select {
		case <-s.inbox.ready:
			continue
		case <-ctx.Done():
			return
		case <-s.loopDone:
		}

		if err := s.terminalErr(); err != nil && ctx.Err() == nil {
			sendListenerError(ctx, errCh, err)
		}

		return
	}
}

func (s *operatorSession) SendStepActionEvent(ctx context.Context, in *dispatchercontracts.StepActionEvent) (*dispatchercontracts.ActionEventResponse, error) {
	if in.WorkerId == "" {
		in.WorkerId = s.Registration().WorkerId
	}

	resp, err := s.client.SendStepActionEvent(s.opCtx(ctx), in)

	// the report is what tells the session the action is done, whether or not the engine
	// accepted it: a report that failed will not be retried by the caller either
	if isTerminalStepEvent(in.EventType) {
		s.finishAction(in.TaskRunExternalId)
	}

	return resp, err
}

// isTerminalStepEvent reports whether an event ends the caller's work on a task run. Started
// and acknowledged events do not.
func isTerminalStepEvent(eventType dispatchercontracts.StepActionEventType) bool {
	switch eventType {
	case dispatchercontracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED,
		dispatchercontracts.StepActionEventType_STEP_EVENT_TYPE_FAILED,
		dispatchercontracts.StepActionEventType_STEP_EVENT_TYPE_CANCELLED:
		return true
	default:
		return false
	}
}

// startAction records that a task run was handed to the consumer and is not finished. Cancels
// carry no work of their own and are never reported, so they are not tracked. A task run is
// tracked once: a retry of the same run replaces the entry rather than adding one, since the
// caller reports the run, not the attempt.
func (s *operatorSession) startAction(action *dispatchercontracts.AssignedAction) {
	if action.ActionType == dispatchercontracts.ActionType_CANCEL_STEP_RUN {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.inflight[action.TaskRunExternalId] = struct{}{}
}

// finishAction records that the caller reported the task run's outcome.
func (s *operatorSession) finishAction(taskRunExternalId string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.inflight[taskRunExternalId]; !ok {
		return
	}

	delete(s.inflight, taskRunExternalId)

	if len(s.inflight) == 0 && s.idle != nil {
		close(s.idle)
		s.idle = nil
	}
}

// idleCh returns a channel that is closed once nothing is in flight.
func (s *operatorSession) idleCh() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.inflight) == 0 {
		closed := make(chan struct{})
		close(closed)

		return closed
	}

	if s.idle == nil {
		s.idle = make(chan struct{})
	}

	return s.idle
}

func (s *operatorSession) Pause(ctx context.Context) error {
	return s.setPaused(ctx, true)
}

func (s *operatorSession) Resume(ctx context.Context) error {
	return s.setPaused(ctx, false)
}

func (s *operatorSession) setPaused(ctx context.Context, paused bool) error {
	workerId := s.Registration().WorkerId

	if workerId == "" {
		return fmt.Errorf("operator session has no worker to pause")
	}

	_, err := s.client.PauseWorker(s.opCtx(ctx), &v1.OperatorPauseWorkerRequest{
		WorkerId: workerId,
		Paused:   paused,
	})

	return err
}

// drain pauses the worker and waits for the actions already handed to the
// consumer to be reported. The pause is what makes the wait terminate: without
// it the scheduler keeps assigning. A pause that fails is logged and the wait
// still runs, bounded by timeout, so in-flight work gets its chance to finish.
func (s *operatorSession) drain(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := s.Pause(ctx); err != nil {
		s.l.Warn().Ctx(ctx).Err(err).Msg("could not pause the operator worker before draining")
	}

	select {
	case <-s.idleCh():
	case <-ctx.Done():
		s.mu.Lock()
		outstanding := len(s.inflight)
		s.mu.Unlock()

		s.l.Warn().Int("in_flight", outstanding).Msg("operator session closed with actions still in flight")
	}
}

func (s *operatorSession) PutWorkflow(ctx context.Context, wf *v1.CreateWorkflowVersionRequest) (*v1.CreateWorkflowVersionResponse, []string, error) {
	actions, err := actionsForWorkflow(wf)
	if err != nil {
		return nil, nil, err
	}

	// The admin service authenticates with the bearer token alone; the
	// operator id metadata is only meaningful to OperatorService.
	resp, err := s.admin.PutWorkflow(s.ctxLoader.newContext(ctx), wf)
	if err != nil {
		return nil, nil, err
	}

	return resp, actions, nil
}

func (s *operatorSession) AddActions(ids ...string) {
	s.actions.add(ids)
}

func (s *operatorSession) RemoveActions(ids ...string) {
	s.actions.remove(ids)
}

func (s *operatorSession) Flush(ctx context.Context) error {
	return s.actions.flush(ctx)
}

// NewDurableTaskListener builds a listener bound to this session's worker. A
// listener created after Close is stopped before it is returned: the
// session no longer has a worker for it to register.
func (s *operatorSession) NewDurableTaskListener(opts ...DurableTaskListenerOpt) *DurableTaskListener {
	currentWorkerId := func() string {
		return s.Registration().WorkerId
	}

	listener := NewDurableTaskListener(
		currentWorkerId(),
		func(ctx context.Context) (v1.V1Dispatcher_DurableTaskClient, error) {
			stream, err := s.client.DurableTask(s.opCtx(ctx), grpc_retry.Disable())
			if err != nil {
				return nil, err
			}
			return &operatorDurableTaskClient{OperatorService_DurableTaskClient: stream, workerId: currentWorkerId}, nil
		},
		s.l,
		opts...,
	)

	s.mu.Lock()
	closed := s.closed
	if !closed {
		s.durables = append(s.durables, listener)
	}
	s.mu.Unlock()

	if closed {
		listener.Stop()
	}

	return listener
}

// CloseListenStream half-closes the current Listen stream without closing the
// session, so the automatic reconnect registers the worker again. It exists
// to exercise reconnect against a live engine; production callers do not
// need it.
func (s *operatorSession) CloseListenStream() error {
	return s.stream.closeStream()
}

// ForgetWorker clears the remembered worker id so the next reconnect
// registers a new worker instead of resuming the previous one. Like
// CloseListenStream it exists to exercise the non-resume path against a live
// engine.
func (s *operatorSession) ForgetWorker() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reg.WorkerId = ""
}

// Close is pause then drain: the worker is paused so the scheduler stops
// assigning, the actions already handed to the consumer are given the drain
// timeout to be reported, and only then is the stream ended and the worker
// deactivated. WithoutDrain hangs up at once instead.
func (s *operatorSession) Close(fs ...CloseOpt) error {
	o := &closeOpts{drain: true, drainTimeout: operatorCloseDrainTimeout}

	for _, f := range fs {
		f(o)
	}

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	durables := s.durables
	s.mu.Unlock()

	// The loops are still running here, so the consumer can finish the work it
	// holds and report it while the drain waits.
	if o.drain {
		s.drain(o.drainTimeout)
	}

	// Deltas queued before Close still belong to the worker the engine will
	// deactivate, so they are given a short window to be acknowledged before
	// the stream goes away; a flush that does not make it is logged, not
	// fatal. The receive loop is still running here, so acks are observed.
	flushCtx, cancelFlush := context.WithTimeout(context.Background(), operatorCloseFlushTimeout)
	if err := s.actions.flush(flushCtx); err != nil {
		s.l.Warn().Err(err).Msg("operator action deltas were not flushed before close")
	}
	cancelFlush()

	s.loopCancel()

	// Close cancels the lifecycle context, which releases every stream context
	// and unblocks any in-flight Send or Recv before the loops are awaited.
	err := s.stream.Close()

	s.actions.stop()
	s.wg.Wait()

	for _, listener := range durables {
		listener.Stop()
	}

	return err
}
