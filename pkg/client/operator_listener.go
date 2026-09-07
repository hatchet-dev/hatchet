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
	// Listen opened, start sent, replay done) may take, on top of the
	// constructor ctx.
	operatorRegisterTimeout = 30 * time.Second

	// operatorCloseFlushTimeout bounds the flush Close performs before it
	// tears the session down.
	operatorCloseFlushTimeout = 2 * time.Second
)

var errOperatorActionsStarted = errors.New("operator session actions already started")

// operatorListenClient is one Listen stream plus the cancel for its own
// context. The stream context is a child of the reconnecting stream's
// lifecycle context so that Close releases every stream; cancel is only
// invoked directly when the handshake fails and the stream is never
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

// operatorSession is the OperatorSession implementation. It owns one
// reconnecting Listen stream whose constructor performs the Register call and
// the start handshake, a heartbeat goroutine that sends through retrySend so a
// dead stream reconnects, the receive loop driven by listenStream, and the
// action delta queue (operator_actions.go).
// NOTE: field order follows govet fieldalignment (enforced by the pre-commit
// autofixer); mu guards register, reg, durables, loopCancel, and closed. Lock
// order is stream.sendMu → mu, never reverse: mu is only taken in short
// critical sections that do no stream I/O.
type operatorSession struct {
	client     v1.OperatorServiceClient
	admin      v1.AdminServiceClient
	ctxLoader  *contextLoader
	l          *zerolog.Logger
	stream     *reconnectingStream[*operatorListenClient]
	register   *v1.OperatorRegisterRequest
	loopCancel context.CancelFunc
	durables   []*DurableTaskListener
	reg        OperatorRegistration
	actions    *actionDeltaQueue

	heartbeatInterval time.Duration

	wg           sync.WaitGroup
	mu           sync.Mutex
	resumeWorker bool
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

	s := &operatorSession{
		client:            client,
		admin:             admin,
		ctxLoader:         ctxLoader,
		l:                 &sl,
		register:          register,
		resumeWorker:      resumeWorker,
		heartbeatInterval: operatorHeartbeatInterval,
	}

	s.stream = newReconnectingStreamWithLifecycle(
		context.Background(),
		&sl,
		"operator listener",
		s.openListenStream,
		func(c *operatorListenClient) error {
			return c.CloseSend()
		},
		nil,
	)

	s.actions = newActionDeltaQueue(&sl, s.stream, actionDeltaFlushInterval, maxActionsPerDelta)

	return s
}

// openListenStream is the reconnecting stream constructor: it calls Register
// (resuming the previous worker when enabled), opens Listen, sends the start
// message and, when the registration did not resume the previous worker,
// replays the desired action set before returning. The handshake is bounded
// by ctx and operatorRegisterTimeout; the stream itself outlives ctx and is
// bound to the lifecycle context.
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

	// A new worker starts with no actions, so the engine only learns the
	// desired set through a replay. A resumed worker kept its set, and any
	// delta that was in flight when the previous stream died is retried by
	// the flusher.
	if !registered.Resumed {
		if err := s.actions.replay(listen); err != nil {
			return fail(fmt.Errorf("could not replay operator actions: %w", err))
		}
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
	if s.loopCancel != nil {
		s.mu.Unlock()
		return nil, nil, errOperatorActionsStarted
	}
	loopCtx, cancel := context.WithCancel(ctx)
	s.loopCancel = cancel
	s.mu.Unlock()

	ch := make(chan *dispatchercontracts.AssignedAction)
	errCh := make(chan error, 1)

	s.l.Debug().Ctx(ctx).Msg("starting operator listener")

	s.wg.Add(2)
	go func() {
		defer s.wg.Done()
		s.heartbeatLoop(loopCtx)
	}()
	go func() {
		defer s.wg.Done()
		s.actionLoop(loopCtx, cancel, ch, errCh)
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

// actionLoop runs the receive loop. Leaving it, for any reason, ends the
// session's stream and the heartbeat loop: a session without a receive loop
// cannot run actions, so keeping the worker alive would only attract work.
func (s *operatorSession) actionLoop(ctx context.Context, cancelLoops context.CancelFunc, ch chan<- *dispatchercontracts.AssignedAction, errCh chan<- error) {
	defer close(ch)
	defer close(errCh)
	defer func() {
		cancelLoops()
		if err := s.stream.Close(); err != nil {
			s.l.Error().Ctx(ctx).Err(err).Msg("failed to close operator listener stream")
		}
	}()

	// Recv is bound to the stream's lifecycle context, not ctx, so cancelling
	// ctx has to end the stream for the loop to observe it and exit.
	stopOnCancel := context.AfterFunc(ctx, func() {
		_ = s.stream.Close()
	})
	defer stopOnCancel()

	classify := newStreamClassifier(func(ctx context.Context) bool {
		return ctx.Err() == nil
	})

	err := listenStream(ctx, s.stream,
		func(c *operatorListenClient) (*dispatchercontracts.AssignedAction, error) {
			return c.Recv()
		},
		func(action *dispatchercontracts.AssignedAction) error {
			s.l.Debug().Ctx(ctx).
				Str("action_type", action.ActionType.String()).
				Str("action_id", action.ActionId).
				Msg("received operator action")

			select {
			case ch <- action:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
		classify,
	)
	if err != nil && ctx.Err() == nil {
		sendListenerError(ctx, errCh, err)
	}
}

func (s *operatorSession) SendStepActionEvent(ctx context.Context, in *dispatchercontracts.StepActionEvent) (*dispatchercontracts.ActionEventResponse, error) {
	if in.WorkerId == "" {
		in.WorkerId = s.Registration().WorkerId
	}

	return s.client.SendStepActionEvent(s.opCtx(ctx), in)
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
	s.durables = append(s.durables, listener)
	s.mu.Unlock()

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

func (s *operatorSession) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	cancel := s.loopCancel
	durables := s.durables
	s.mu.Unlock()

	// Deltas queued before Close still belong to the worker the engine will
	// deactivate, so they are given a short window to land before the stream
	// goes away; a flush that does not make it is logged, not fatal.
	flushCtx, cancelFlush := context.WithTimeout(context.Background(), operatorCloseFlushTimeout)
	if err := s.actions.flush(flushCtx); err != nil {
		s.l.Warn().Err(err).Msg("operator action deltas were not flushed before close")
	}
	cancelFlush()

	if cancel != nil {
		cancel()
	}

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
