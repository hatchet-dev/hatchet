package operatorclient

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
	"github.com/hatchet-dev/hatchet/pkg/client/streaming"
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

// listenClient is one Listen stream plus the cancel for its own
// context. The stream context is a child of the reconnecting stream's
// lifecycle context so that Close releases every stream; cancel is invoked
// directly when the handshake or the replay fails and the stream is never
// published. Retired streams are half-closed with CloseSend and left to the
// server to end, so the receive loop hands off on EOF like the other
// listeners in this package.
type listenClient struct {
	v1.OperatorService_ListenClient
	cancel context.CancelFunc
}

// durableTaskClient adapts the OperatorService durable stream to the
// V1Dispatcher one a durable task listener expects and rewrites the worker id
// on the register message to the session's current registration, so a
// listener built before a reconnect still registers the worker the engine
// now knows.
type durableTaskClient struct {
	v1.OperatorService_DurableTaskClient
	workerId func() string
}

func (c *durableTaskClient) Send(req *v1.DurableTaskRequest) error {
	if register := req.GetRegisterWorker(); register != nil {
		register.WorkerId = c.workerId()
	}

	return c.OperatorService_DurableTaskClient.Send(req)
}

// actionInbox buffers assigned actions between the session's receive loop
// and the consumer Actions starts. The receive loop must never block on a
// consumer: delta acks share the stream with actions, and Flush may wait on
// an ack before Actions has been called. The inbox itself has no bound; what
// bounds it is the consumer draining the Actions channel eagerly (the gRPC
// host reads it as fast as it arrives and queues starts behind a bounded
// queue of its own, refusing what does not fit), so the engine's slots, not
// the consumer's pace, size it.
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

// session is the Session implementation. It owns one reconnecting Listen
// stream whose constructor performs the Register call and the start
// handshake and whose replay callback resends unacknowledged action deltas,
// a heartbeat goroutine that sends through RetrySend so a dead stream
// reconnects, the receive loop that demultiplexes assigned actions and delta
// acks, and the action delta queue (actions.go). The receive and heartbeat
// loops run from connect until Close so that Flush can observe acks before
// Actions is called; Actions only attaches a consumer to the inbox.
// NOTE: field order follows govet fieldalignment (enforced by the pre-commit
// autofixer); mu guards register, reg, loopErr, inflight, idle, started,
// consuming and closed. Lock order is the stream's send lock → mu, never
// reverse: mu is only taken in short critical sections that do no stream I/O.
type session struct {
	client     v1.OperatorServiceClient
	admin      v1.AdminServiceClient
	md         *callMetadata
	l          *zerolog.Logger
	stream     *streaming.ReconnectingStream[*listenClient]
	register   *v1.OperatorRegisterRequest
	inbox      *actionInbox
	loopCtx    context.Context
	loopCancel context.CancelFunc
	loopDone   chan struct{}
	loopErr    error
	reg        Registration
	actions    *actionDeltaQueue

	// paused is the pause state the session wants on its worker, resent on every new stream
	// (see replayActions); pauseAck is where the receive loop hands the next pause ack to the
	// setPaused call waiting for it. Both are guarded by mu; pauseMu serialises the calls.
	pauseAck chan bool
	pauseMu  sync.Mutex
	paused   bool

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

func newSession(
	client v1.OperatorServiceClient,
	admin v1.AdminServiceClient,
	md *callMetadata,
	l *zerolog.Logger,
	register *v1.OperatorRegisterRequest,
	resumeWorker bool,
) *session {
	sl := l.With().Str("operator", register.Name).Logger()

	loopCtx, loopCancel := context.WithCancel(context.Background())

	s := &session{
		client:            client,
		admin:             admin,
		md:                md,
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

	s.stream = streaming.NewReconnectingStreamWithLifecycle(
		context.Background(),
		&sl,
		"operator listener",
		s.openListenStream,
		func(c *listenClient) error {
			return c.CloseSend()
		},
		s.replayActions,
	)

	s.actions = newActionDeltaQueue(&sl, s.stream, actionDeltaFlushInterval, maxActionsPerDelta)

	return s
}

// connect opens the first stream and starts the session loops.
func (s *session) connect(ctx context.Context) error {
	if err := s.stream.ConnectSync(ctx); err != nil {
		return err
	}

	s.start()

	return nil
}

// start launches the receive and heartbeat loops once. The WaitGroup is
// incremented under mu, in the same critical section that checks closed, so
// Close never waits on a count that a concurrent start is still adding to.
func (s *session) start() {
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
func (s *session) openListenStream(ctx context.Context) (*listenClient, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	handshakeCtx, cancelHandshake := context.WithTimeout(ctx, operatorRegisterTimeout)
	defer cancelHandshake()

	registered, err := s.client.Register(s.md.context(handshakeCtx), s.registerRequest())
	if err != nil {
		return nil, fmt.Errorf("could not register operator: %w", err)
	}

	s.setRegistration(Registration{
		TenantId:   registered.TenantId,
		OperatorId: registered.OperatorId,
		WorkerId:   registered.WorkerId,
		Resumed:    registered.Resumed,
	})

	streamCtx, cancelStream := context.WithCancel(s.stream.LifecycleContext())
	stopAfter := context.AfterFunc(handshakeCtx, cancelStream)

	fail := func(err error) (*listenClient, error) {
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

	return &listenClient{OperatorService_ListenClient: listen, cancel: cancelStream}, nil
}

// replayActions is the reconnecting stream's replay callback: it runs under
// the send lock on every new stream before the stream is published and brings the
// engine's view of the action set up to date (see actionDeltaQueue.replay).
// A stream whose replay fails is cancelled here because it is never
// published.
func (s *session) replayActions(_ context.Context, c *listenClient) error {
	// the pause belongs to the stream and Register clears it on a resumed worker, so a paused
	// session restores it first, before any delta and before the stream is published
	if s.isPaused() {
		if err := c.Send(pauseMessage(true)); err != nil {
			c.cancel()
			return fmt.Errorf("could not replay operator pause: %w", err)
		}
	}

	if err := s.actions.replay(c, s.Registration().Resumed); err != nil {
		c.cancel()
		return fmt.Errorf("could not replay operator actions: %w", err)
	}

	return nil
}

func (s *session) isPaused() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.paused
}

func pauseMessage(paused bool) *v1.OperatorListenRequest {
	return &v1.OperatorListenRequest{
		Message: &v1.OperatorListenRequest_Pause{Pause: &v1.OperatorPause{Paused: paused}},
	}
}

// deliverPauseAck hands a pause ack to the setPaused call waiting for one. An ack nobody waits
// for (a replayed pause's, or one that arrived after its waiter gave up) is dropped.
func (s *session) deliverPauseAck(paused bool) {
	s.mu.Lock()
	ack := s.pauseAck
	s.mu.Unlock()

	if ack == nil {
		return
	}

	select {
	case ack <- paused:
	default:
	}
}

// registerRequest snapshots the register template and, when resuming, sets
// the worker id from the most recent registration.
func (s *session) registerRequest() *v1.OperatorRegisterRequest {
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

func (s *session) setRegistration(reg Registration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reg = reg
}

func (s *session) Registration() Registration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reg
}

// opCtx attaches the bearer token and the hatchet-operator-id metadata every
// RPC after Register requires.
func (s *session) opCtx(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(
		s.md.context(ctx),
		operatorIdMetadataKey, s.Registration().OperatorId,
	)
}

func (s *session) Actions(ctx context.Context) (<-chan *dispatchercontracts.AssignedAction, <-chan error, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, nil, streaming.ErrListenerClosed
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

func (s *session) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(s.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		err := s.stream.RetrySend(ctx, func(c *listenClient) error {
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

		if errors.Is(err, streaming.ErrListenerClosed) {
			return
		}
	}
}

// receiveLoop runs the stream's receive side for the life of the session:
// acks go to the delta queue and actions to the inbox. Leaving it, for any
// reason, ends the session's stream and the heartbeat loop: a session
// without a receive loop cannot run actions or confirm deltas, so keeping
// the worker alive would only attract work.
func (s *session) receiveLoop(ctx context.Context) {
	defer close(s.loopDone)
	defer func() {
		s.loopCancel()
		s.actions.stop()
		if err := s.stream.Close(); err != nil {
			s.l.Error().Ctx(ctx).Err(err).Msg("failed to close operator listener stream")
		}
	}()

	classify := streaming.NewClassifier(func(ctx context.Context) bool {
		return ctx.Err() == nil
	})

	err := streaming.Listen(ctx, s.stream,
		func(c *listenClient) (*v1.OperatorListenResponse, error) {
			return c.Recv()
		},
		func(resp *v1.OperatorListenResponse) error {
			switch msg := resp.Message.(type) {
			case *v1.OperatorListenResponse_Ack:
				s.actions.ack(msg.Ack.Sequence)
			case *v1.OperatorListenResponse_PauseAck:
				s.deliverPauseAck(msg.PauseAck.Paused)
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
func (s *session) terminalErr() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loopErr
}

// deliverLoop hands buffered actions to the consumer until ctx ends, the
// session closes, or the receive loop exits. Cancelling ctx ends the
// session's stream, as a consumer that has stopped taking actions must not
// keep its worker active.
func (s *session) deliverLoop(ctx context.Context, ch chan<- *dispatchercontracts.AssignedAction, errCh chan<- error) {
	defer close(ch)
	defer close(errCh)
	// once delivery ends there is nobody left to report the runs the consumer was handed, so a
	// Close that is draining must stop waiting for them
	defer s.abandonInflight()

	stopOnCancel := context.AfterFunc(ctx, func() {
		s.loopCancel()
		_ = s.stream.Close()
	})
	defer stopOnCancel()

	for {
		if action := s.inbox.pop(); action != nil {
			// The action counts as in flight from the moment this loop commits to delivering
			// it, not once the consumer has taken it: a consumer that reports the outcome the
			// instant it receives the action would otherwise clear an entry that is not there
			// yet, and the entry added afterwards would never be cleared.
			s.startAction(action)

			select {
			case ch <- action:
				continue
			case <-ctx.Done():
				s.finishAction(action.TaskRunExternalId)
				return
			case <-s.loopDone:
				s.finishAction(action.TaskRunExternalId)
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
			streaming.SendListenerError(ctx, errCh, err)
		}

		return
	}
}

func (s *session) SendStepActionEvent(ctx context.Context, in *dispatchercontracts.StepActionEvent) (*dispatchercontracts.ActionEventResponse, error) {
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
func (s *session) startAction(action *dispatchercontracts.AssignedAction) {
	if action.ActionType == dispatchercontracts.ActionType_CANCEL_STEP_RUN {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.inflight[action.TaskRunExternalId] = struct{}{}
}

// finishAction records that the caller reported the task run's outcome.
func (s *session) finishAction(taskRunExternalId string) {
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

// abandonInflight forgets every task run handed to the consumer. Delivery has ended, so the
// consumer that would have reported them is gone; the engine retries whatever it was holding
// once those tasks time out.
func (s *session) abandonInflight() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.inflight) == 0 {
		return
	}

	s.inflight = map[string]struct{}{}

	if s.idle != nil {
		close(s.idle)
		s.idle = nil
	}
}

// idleCh returns a channel that is closed once nothing is in flight.
func (s *session) idleCh() <-chan struct{} {
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

func (s *session) Pause(ctx context.Context) error {
	return s.setPaused(ctx, true)
}

func (s *session) Resume(ctx context.Context) error {
	return s.setPaused(ctx, false)
}

// setPaused sends the pause on the Listen stream and waits for the engine's ack, which is what
// makes the pause a promise: after it nothing is delivered on the stream. The desired state is
// recorded first, so a reconnect that happens while the send is in flight replays it, and a
// send that fails is retried through the reconnecting stream like a heartbeat. Calls are
// serialised, and the wait is for an ack carrying the state asked for: an ack of the previous
// state, or of a replayed pause, is skipped.
func (s *session) setPaused(ctx context.Context, paused bool) error {
	s.pauseMu.Lock()
	defer s.pauseMu.Unlock()

	ack := make(chan bool, 1)

	s.mu.Lock()
	s.paused = paused
	s.pauseAck = ack
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		if s.pauseAck == ack {
			s.pauseAck = nil
		}
		s.mu.Unlock()
	}()

	err := s.stream.RetrySend(ctx, func(c *listenClient) error {
		return c.Send(pauseMessage(paused))
	})

	if err != nil {
		return fmt.Errorf("could not send operator pause: %w", err)
	}

	for {
		select {
		case got := <-ack:
			if got == paused {
				return nil
			}
		case <-s.loopDone:
			return streaming.ErrListenerClosed
		case <-ctx.Done():
			return fmt.Errorf("operator pause was not acknowledged: %w", ctx.Err())
		}
	}
}

// drain pauses the worker and waits for the actions already handed to the
// consumer to be reported. The pause's ack is what makes the wait terminate:
// without it the engine keeps delivering. A pause that fails or is not
// acknowledged in time is logged and the wait still runs, bounded by timeout,
// so in-flight work gets its chance to finish.
func (s *session) drain(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := s.Pause(ctx); err != nil {
		s.l.Warn().Ctx(ctx).Err(err).Msg("could not pause the operator worker before draining")
	}

	select {
	case <-s.idleCh():
		return
	case <-ctx.Done():
	}

	s.mu.Lock()
	outstanding := len(s.inflight)
	s.mu.Unlock()

	if outstanding == 0 {
		return
	}

	s.l.Warn().Int("in_flight", outstanding).Msg("operator session closed with actions still in flight")
}

func (s *session) PutWorkflow(ctx context.Context, wf *v1.CreateWorkflowVersionRequest) (*v1.CreateWorkflowVersionResponse, []string, error) {
	actions, err := actionsForWorkflow(wf)
	if err != nil {
		return nil, nil, err
	}

	// The admin service authenticates with the bearer token alone; the
	// operator id metadata is only meaningful to OperatorService.
	resp, err := s.admin.PutWorkflow(s.md.context(ctx), wf)
	if err != nil {
		return nil, nil, err
	}

	return resp, actions, nil
}

func (s *session) AddActions(ids ...string) {
	s.actions.add(ids)
}

func (s *session) RemoveActions(ids ...string) {
	s.actions.remove(ids)
}

func (s *session) Flush(ctx context.Context) error {
	return s.actions.flush(ctx)
}

// OpenDurableTaskStream opens the OperatorService durable task stream with
// the operator metadata and returns it shaped as the V1Dispatcher stream a
// durable task listener expects; the register message's worker id is
// rewritten to the current registration on every send.
func (s *session) OpenDurableTaskStream(ctx context.Context) (v1.V1Dispatcher_DurableTaskClient, error) {
	stream, err := s.client.DurableTask(s.opCtx(ctx), grpc_retry.Disable())
	if err != nil {
		return nil, err
	}

	return &durableTaskClient{
		OperatorService_DurableTaskClient: stream,
		workerId: func() string {
			return s.Registration().WorkerId
		},
	}, nil
}

// CloseListenStream half-closes the current Listen stream without closing the
// session, so the automatic reconnect registers the worker again. It exists
// to exercise reconnect against a live engine; production callers do not
// need it.
func (s *session) CloseListenStream() error {
	return s.stream.CloseStream()
}

// ForgetWorker clears the remembered worker id so the next reconnect
// registers a new worker instead of resuming the previous one. Like
// CloseListenStream it exists to exercise the non-resume path against a live
// engine.
func (s *session) ForgetWorker() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reg.WorkerId = ""
}

// Close is pause then drain: the worker is paused so the scheduler stops
// assigning, the actions already handed to the consumer are given the drain
// timeout to be reported, and only then is the stream ended and the worker
// deactivated. WithoutDrain hangs up at once instead.
func (s *session) Close(fs ...CloseOpt) error {
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

	return err
}
