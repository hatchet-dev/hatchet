package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client/retry"
)

const (
	defaultReconnectInterval    = 2 * time.Second
	defaultWorkerStatusInterval = time.Second
	defaultCompletionBufferTTL  = 10 * time.Second
	evictionAckTimeout          = 30 * time.Second
)

var errDurableTaskListenerStopped = errors.New("durable task listener stopped")

// DurableTaskCallback is called when a response is received for a durable task.
type DurableTaskCallback func(resp *v1.DurableTaskResponse) error

// PendingAckKey uniquely identifies a pending acknowledgment by task external id
// and invocation count. (The field name SignalKey is retained for backward
// compatibility but really represents the invocation count, matching the Python SDK.)
type PendingAckKey struct {
	TaskID    string
	SignalKey int64
}

// PendingCallbackKey uniquely identifies a pending event-log entry callback.
type PendingCallbackKey struct {
	TaskID    string
	SignalKey int64
	NodeID    int64
	BranchID  int64
}

// EventAckResult carries either a response or an error for a pending event ack.
type EventAckResult struct {
	Resp *v1.DurableTaskResponse
	Err  error
}

// CallbackResult carries the completed event-log entry payload, or an error.
type CallbackResult struct {
	Resp *v1.DurableTaskResponse
	Err  error
}

// TriggerRunAckEntry describes a child workflow spawned through the durable task
// event log.
type TriggerRunAckEntry struct {
	WorkflowRunID string
	NodeID        int64
	BranchID      int64
}

type bufferedCompletion struct {
	resp      *v1.DurableTaskResponse
	expiresAt time.Time
}

type durableTaskStreamResult struct {
	err      error
	terminal bool
}

// NonDeterminismError is returned by the engine when a durable task replay detects
// a non-deterministic mutation (e.g. branching differently from the prior run).
type NonDeterminismError struct {
	TaskExternalID  string
	Message         string
	NodeID          int64
	InvocationCount int32
}

func (e *NonDeterminismError) Error() string {
	return fmt.Sprintf(
		"non-determinism error for task %s (invocation %d, node %d): %s",
		e.TaskExternalID, e.InvocationCount, e.NodeID, e.Message,
	)
}

// ServerEvictCallback is invoked when the engine notifies the worker that a
// durable task invocation should be evicted.
type ServerEvictCallback func(taskExternalID string, invocationCount int32, reason string)

// DurableTaskListener manages a bidirectional gRPC stream for durable task operations.
type DurableTaskListener struct {
	onServerEvict         ServerEvictCallback
	pendingEvictionAcks   map[PendingAckKey]chan error
	l                     zerolog.Logger
	connectFn             func(ctx context.Context) (v1.V1Dispatcher_DurableTaskClient, error)
	cancel                context.CancelFunc
	done                  chan struct{}
	requestQueue          chan *v1.DurableTaskRequest
	statusChanged         chan struct{}
	bufferedCompletions   map[PendingCallbackKey]bufferedCompletion
	pendingEventAcks      map[PendingAckKey]chan EventAckResult
	pendingCallbacks      map[PendingCallbackKey]chan CallbackResult
	callbackTerminalErr   error
	workerID              string
	streamSeq             int
	reconnectInterval     time.Duration
	evictionAckTTL        time.Duration
	onServerEvictMu       sync.RWMutex
	streamMu              sync.Mutex
	callbackStateMu       sync.Mutex
	callbacksTerminal     bool
	pendingEvictionAcksMu sync.Mutex
	pendingEventAcksMu    sync.Mutex
	mu                    sync.Mutex
	running               bool
}

// DurableTaskListenerOpt configures a DurableTaskListener.
type DurableTaskListenerOpt func(*DurableTaskListener)

// WithReconnectInterval overrides the default reconnect interval.
func WithReconnectInterval(d time.Duration) DurableTaskListenerOpt {
	return func(l *DurableTaskListener) {
		l.reconnectInterval = d
	}
}

// WithEvictionAckTimeout overrides the default eviction-ack timeout.
func WithEvictionAckTimeout(d time.Duration) DurableTaskListenerOpt {
	return func(l *DurableTaskListener) {
		l.evictionAckTTL = d
	}
}

// NewDurableTaskListener creates a new listener.
func NewDurableTaskListener(
	workerID string,
	connectFn func(ctx context.Context) (v1.V1Dispatcher_DurableTaskClient, error),
	l *zerolog.Logger,
	opts ...DurableTaskListenerOpt,
) *DurableTaskListener {
	logger := zerolog.Nop()
	if l != nil {
		logger = *l
	}

	dtl := &DurableTaskListener{
		workerID:            workerID,
		l:                   logger,
		reconnectInterval:   defaultReconnectInterval,
		evictionAckTTL:      evictionAckTimeout,
		connectFn:           connectFn,
		pendingEventAcks:    make(map[PendingAckKey]chan EventAckResult),
		pendingEvictionAcks: make(map[PendingAckKey]chan error),
		pendingCallbacks:    make(map[PendingCallbackKey]chan CallbackResult),
		bufferedCompletions: make(map[PendingCallbackKey]bufferedCompletion),
		requestQueue:        make(chan *v1.DurableTaskRequest, 100),
		statusChanged:       make(chan struct{}, 1),
	}

	for _, opt := range opts {
		opt(dtl)
	}

	return dtl
}

// SetServerEvictCallback registers the callback invoked when the server notifies
// the worker to evict an invocation. Pass nil to unregister.
func (l *DurableTaskListener) SetServerEvictCallback(cb ServerEvictCallback) {
	l.onServerEvictMu.Lock()
	defer l.onServerEvictMu.Unlock()
	l.onServerEvict = cb
}

// Start begins the listener loop.
func (l *DurableTaskListener) Start(ctx context.Context) {
	for {
		l.mu.Lock()
		if l.running {
			l.mu.Unlock()
			return
		}
		previousDone := l.done
		l.mu.Unlock()

		if previousDone != nil {
			// A restarted listener cannot share callback state with senders or
			// receivers from the preceding stream generation.
			select {
			case <-ctx.Done():
				return
			case <-previousDone:
				continue
			}
		}

		l.callbackStateMu.Lock()
		l.mu.Lock()
		if l.running || l.done != nil {
			l.mu.Unlock()
			l.callbackStateMu.Unlock()
			continue
		}

		l.callbacksTerminal = false
		l.callbackTerminalErr = nil
		listenerCtx, cancel := context.WithCancel(ctx)
		done := make(chan struct{})
		l.cancel = cancel
		l.done = done
		l.running = true
		l.mu.Unlock()
		l.callbackStateMu.Unlock()

		go l.receiveLoop(listenerCtx, done)
		return
	}
}

// Stop halts the listener.
func (l *DurableTaskListener) Stop() {
	// The callback-state lock makes terminal shutdown atomic with callback
	// registration, completion buffering, and delivery.
	l.callbackStateMu.Lock()
	l.mu.Lock()
	l.running = false
	cancel := l.cancel
	l.cancel = nil
	l.mu.Unlock()
	l.terminateCallbackStateLocked(errDurableTaskListenerStopped)
	l.callbackStateMu.Unlock()

	if cancel != nil {
		cancel()
	}

	l.failPendingAcks(errDurableTaskListenerStopped)
}

// IsRunning returns whether the listener loop is active.
func (l *DurableTaskListener) IsRunning() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.running
}

// StreamSeq returns the number of streams that have been opened.
func (l *DurableTaskListener) StreamSeq() int {
	l.streamMu.Lock()
	defer l.streamMu.Unlock()
	return l.streamSeq
}

// SendRequest queues a DurableTaskRequest for sending on the stream.
func (l *DurableTaskListener) SendRequest(req *v1.DurableTaskRequest) {
	l.requestQueue <- req
}

// AddPendingEventAck registers a pending event ack and returns a channel to wait on.
func (l *DurableTaskListener) AddPendingEventAck(key PendingAckKey) chan EventAckResult {
	l.pendingEventAcksMu.Lock()
	defer l.pendingEventAcksMu.Unlock()
	ch := make(chan EventAckResult, 1)
	l.pendingEventAcks[key] = ch
	return ch
}

// AddPendingEvictionAck registers a pending eviction ack.
func (l *DurableTaskListener) AddPendingEvictionAck(key PendingAckKey) chan error {
	l.pendingEvictionAcksMu.Lock()
	defer l.pendingEvictionAcksMu.Unlock()
	ch := make(chan error, 1)
	l.pendingEvictionAcks[key] = ch
	return ch
}

// AddPendingCallback registers a pending callback. If a completion for this key was
// already buffered (arrived before the caller registered interest), the channel is
// returned pre-populated.
func (l *DurableTaskListener) AddPendingCallback(key PendingCallbackKey) chan CallbackResult {
	ch := make(chan CallbackResult, 1)

	l.callbackStateMu.Lock()
	if l.callbacksTerminal {
		ch <- CallbackResult{Err: l.callbackTerminalErr}
		l.callbackStateMu.Unlock()
		return ch
	}

	l.pruneExpiredCompletionsLocked(time.Now())
	buffered, hasBuffered := l.bufferedCompletions[key]
	if hasBuffered {
		delete(l.bufferedCompletions, key)
		ch <- CallbackResult{Resp: buffered.resp}
	} else {
		l.pendingCallbacks[key] = ch
	}
	l.callbackStateMu.Unlock()

	if hasBuffered {
		return ch
	}

	l.signalStatusChanged()
	return ch
}

// PendingEventAckCount returns the number of pending event acks.
func (l *DurableTaskListener) PendingEventAckCount() int {
	l.pendingEventAcksMu.Lock()
	defer l.pendingEventAcksMu.Unlock()
	return len(l.pendingEventAcks)
}

// PendingCallbackCount returns the number of pending callbacks.
func (l *DurableTaskListener) PendingCallbackCount() int {
	l.callbackStateMu.Lock()
	defer l.callbackStateMu.Unlock()
	return len(l.pendingCallbacks)
}

// PendingEvictionAckCount returns the number of pending eviction acks.
func (l *DurableTaskListener) PendingEvictionAckCount() int {
	l.pendingEvictionAcksMu.Lock()
	defer l.pendingEvictionAcksMu.Unlock()
	return len(l.pendingEvictionAcks)
}

// BufferedCompletionCount returns the number of completions buffered for late consumers.
func (l *DurableTaskListener) BufferedCompletionCount() int {
	l.callbackStateMu.Lock()
	defer l.callbackStateMu.Unlock()
	l.pruneExpiredCompletionsLocked(time.Now())
	return len(l.bufferedCompletions)
}

// CleanupTaskState removes pending callbacks, acks, and buffered completions for any
// invocation <= invocationCount of the given task. Used after server-side eviction so
// that stale waiters from a prior invocation don't leak.
func (l *DurableTaskListener) CleanupTaskState(taskExternalID string, invocationCount int32) {
	cancelErr := fmt.Errorf("state cleaned up after eviction of task %s invocation %d", taskExternalID, invocationCount)

	l.callbackStateMu.Lock()
	for k, ch := range l.pendingCallbacks {
		if k.TaskID == taskExternalID && k.SignalKey <= int64(invocationCount) {
			delete(l.pendingCallbacks, k)
			select {
			case ch <- CallbackResult{Err: cancelErr}:
			default:
			}
		}
	}
	for k := range l.bufferedCompletions {
		if k.TaskID == taskExternalID && k.SignalKey <= int64(invocationCount) {
			delete(l.bufferedCompletions, k)
		}
	}
	l.callbackStateMu.Unlock()

	l.pendingEventAcksMu.Lock()
	eventAcks := make([]chan EventAckResult, 0)
	for k, ch := range l.pendingEventAcks {
		if k.TaskID == taskExternalID && k.SignalKey <= int64(invocationCount) {
			delete(l.pendingEventAcks, k)
			eventAcks = append(eventAcks, ch)
		}
	}
	l.pendingEventAcksMu.Unlock()

	for _, ch := range eventAcks {
		select {
		case ch <- EventAckResult{Err: cancelErr}:
		default:
		}
	}
}

func (l *DurableTaskListener) receiveLoop(ctx context.Context, done chan struct{}) {
	defer func() {
		l.failPendingAcks(errDurableTaskListenerStopped)
		l.terminateCallbackState(errDurableTaskListenerStopped)

		l.mu.Lock()
		l.running = false
		if l.done == done {
			l.done = nil
		}
		close(done)
		l.mu.Unlock()
	}()

	for {
		result := l.handleStream(ctx)
		if result.terminal {
			return
		}

		resetErr := errors.New("connection reset: stream ended")
		if result.err != nil {
			resetErr = fmt.Errorf("connection reset: %w", result.err)
			l.l.Warn().Err(result.err).Msg("DurableTaskListener: stream ended, reconnecting")
		}
		l.failPendingAcks(resetErr)

		if sleepErr := retry.Sleep(ctx, l.reconnectInterval); sleepErr != nil {
			return
		}
	}
}

func (l *DurableTaskListener) connect(ctx context.Context) (v1.V1Dispatcher_DurableTaskClient, error) {
	return l.connectFn(ctx)
}

func (l *DurableTaskListener) handleStream(ctx context.Context) durableTaskStreamResult {
	streamCtx, cancelStream := context.WithCancel(ctx)
	stream, err := l.connect(streamCtx)
	if err != nil {
		cancelStream()
		return durableTaskStreamResult{
			err:      err,
			terminal: ctx.Err() != nil,
		}
	}

	l.streamMu.Lock()
	l.streamSeq++
	l.streamMu.Unlock()

	if err := stream.Send(&v1.DurableTaskRequest{
		Message: &v1.DurableTaskRequest_RegisterWorker{
			RegisterWorker: &v1.DurableTaskRequestRegisterWorker{
				WorkerId: l.workerID,
			},
		},
	}); err != nil {
		cancelStream()
		return durableTaskStreamResult{
			err:      fmt.Errorf("failed to register worker on durable task stream: %w", err),
			terminal: ctx.Err() != nil,
		}
	}

	l.l.Debug().Str("worker_id", l.workerID).Msg("DurableTaskListener: registered worker on stream")

	sendFailed := make(chan error, 1)
	senderStopped := make(chan struct{})
	go l.sendStreamRequests(streamCtx, cancelStream, stream, sendFailed, senderStopped)

	defer func() {
		cancelStream()
		<-senderStopped
	}()

	for {
		resp, err := stream.Recv()
		if err != nil {
			select {
			case sendErr := <-sendFailed:
				return durableTaskStreamResult{
					err:      sendErr,
					terminal: ctx.Err() != nil,
				}
			default:
			}
			if ctx.Err() != nil {
				return durableTaskStreamResult{err: ctx.Err(), terminal: true}
			}
			if errors.Is(err, io.EOF) {
				return durableTaskStreamResult{}
			}
			return durableTaskStreamResult{
				err:      err,
				terminal: isGRPCCancelled(err),
			}
		}

		l.dispatchResponse(resp)
	}
}

func (l *DurableTaskListener) sendStreamRequests(
	ctx context.Context,
	cancelStream context.CancelFunc,
	stream v1.V1Dispatcher_DurableTaskClient,
	sendFailed chan<- error,
	stopped chan<- struct{},
) {
	defer close(stopped)

	ticker := time.NewTicker(defaultWorkerStatusInterval)
	defer ticker.Stop()

	select {
	case <-l.statusChanged:
	default:
	}
	if err := l.sendWorkerStatus(stream); err != nil {
		sendFailed <- err
		cancelStream()
		return
	}

	// The capacity-one status signal stays ready until consumed, so Go's select
	// can service either source without allowing status bursts into the mutation queue.
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-l.requestQueue:
			if err := stream.Send(req); err != nil {
				sendFailed <- err
				cancelStream()
				return
			}
		case <-l.statusChanged:
			if err := l.sendWorkerStatus(stream); err != nil {
				sendFailed <- err
				cancelStream()
				return
			}
		case <-ticker.C:
			if err := l.sendWorkerStatus(stream); err != nil {
				sendFailed <- err
				cancelStream()
				return
			}
		}
	}
}

// SendEvictionRequest sends an eviction request and waits for the ack. If the engine
// does not acknowledge within evictionAckTTL, the pending entry is cleaned up and a
// timeout error is returned so callers (e.g. shutdown) cannot hang indefinitely.
func (l *DurableTaskListener) SendEvictionRequest(ctx context.Context, stepRunID string, invocationCount int) error {
	key := PendingAckKey{TaskID: stepRunID, SignalKey: int64(invocationCount)}
	ackCh := l.AddPendingEvictionAck(key)

	l.SendRequest(&v1.DurableTaskRequest{
		Message: &v1.DurableTaskRequest_EvictInvocation{
			EvictInvocation: &v1.DurableTaskEvictInvocationRequest{
				InvocationCount:       int32(invocationCount), // nolint:gosec
				DurableTaskExternalId: stepRunID,
			},
		},
	})

	timer := time.NewTimer(l.evictionAckTTL)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		l.removePendingEvictionAck(key)
		return ctx.Err()
	case err := <-ackCh:
		return err
	case <-timer.C:
		l.removePendingEvictionAck(key)
		return fmt.Errorf(
			"eviction ack timed out after %s for task %s invocation %d",
			l.evictionAckTTL, stepRunID, invocationCount,
		)
	}
}

func (l *DurableTaskListener) removePendingEvictionAck(key PendingAckKey) {
	l.pendingEvictionAcksMu.Lock()
	defer l.pendingEvictionAcksMu.Unlock()
	delete(l.pendingEvictionAcks, key)
}

func (l *DurableTaskListener) removePendingEventAck(key PendingAckKey) {
	l.pendingEventAcksMu.Lock()
	defer l.pendingEventAcksMu.Unlock()
	delete(l.pendingEventAcks, key)
}

func (l *DurableTaskListener) removePendingCallback(key PendingCallbackKey) {
	l.callbackStateMu.Lock()
	defer l.callbackStateMu.Unlock()
	delete(l.pendingCallbacks, key)
}

// SendTriggerRunsRequest sends child workflow requests and waits for their event-log ack.
func (l *DurableTaskListener) SendTriggerRunsRequest(
	ctx context.Context,
	taskExternalID string,
	invocationCount int32,
	triggerOpts []*v1.TriggerWorkflowRequest,
) ([]TriggerRunAckEntry, error) {
	ackKey := PendingAckKey{TaskID: taskExternalID, SignalKey: int64(invocationCount)}
	ackCh := l.AddPendingEventAck(ackKey)

	l.l.Debug().
		Str("step_run_id", taskExternalID).
		Int32("invocation_count", invocationCount).
		Int("children", len(triggerOpts)).
		Msg("DurableTaskListener: sending trigger_runs request")

	l.SendRequest(&v1.DurableTaskRequest{
		Message: &v1.DurableTaskRequest_TriggerRuns{
			TriggerRuns: &v1.DurableTaskTriggerRunsRequest{
				InvocationCount:       invocationCount,
				DurableTaskExternalId: taskExternalID,
				TriggerOpts:           triggerOpts,
			},
		},
	})

	select {
	case <-ctx.Done():
		l.removePendingEventAck(ackKey)
		return nil, ctx.Err()
	case ack := <-ackCh:
		if ack.Err != nil {
			return nil, ack.Err
		}

		triggerAck := ack.Resp.GetTriggerRunsAck()
		if triggerAck == nil {
			return nil, fmt.Errorf("trigger_runs ack missing for task %s invocation %d", taskExternalID, invocationCount)
		}

		runEntries := triggerAck.GetRunEntries()
		entries := make([]TriggerRunAckEntry, 0, len(runEntries))
		for _, entry := range runEntries {
			entries = append(entries, TriggerRunAckEntry{
				NodeID:        entry.GetNodeId(),
				BranchID:      entry.GetBranchId(),
				WorkflowRunID: entry.GetWorkflowRunExternalId(),
			})
		}

		return entries, nil
	}
}

// WaitForCallback waits for a durable event-log entry to complete and returns
// the raw JSON payload recorded by the engine.
func (l *DurableTaskListener) WaitForCallback(
	ctx context.Context,
	taskExternalID string,
	invocationCount int32,
	branchID int64,
	nodeID int64,
) ([]byte, error) {
	cbKey := PendingCallbackKey{
		TaskID:    taskExternalID,
		SignalKey: int64(invocationCount),
		BranchID:  branchID,
		NodeID:    nodeID,
	}
	cbCh := l.AddPendingCallback(cbKey)

	select {
	case <-ctx.Done():
		l.removePendingCallback(cbKey)
		return nil, ctx.Err()
	case result := <-cbCh:
		if result.Err != nil {
			return nil, result.Err
		}
		completed := result.Resp.GetEntryCompleted()
		if completed == nil {
			return nil, fmt.Errorf("durable callback missing entry_completed for task %s", taskExternalID)
		}
		if completed.GetIsFailure() {
			msg := completed.GetErrorMessage()
			if msg == "" {
				msg = "child task failed"
			}
			return nil, fmt.Errorf("%s", msg)
		}
		return completed.GetPayload(), nil
	}
}

// SendWaitForRequest registers a durable wait-for on the engine over the bidi DurableTask
// stream. It waits for the server-assigned event-log ref before waiting for completion.
func (l *DurableTaskListener) SendWaitForRequest(
	ctx context.Context,
	taskExternalID string,
	invocationCount int32,
	conditions *v1.DurableEventListenerConditions,
	label string,
) ([]byte, error) {
	ackKey := PendingAckKey{TaskID: taskExternalID, SignalKey: int64(invocationCount)}
	ackCh := l.AddPendingEventAck(ackKey)

	var labelPtr *string
	if label != "" {
		labelPtr = &label
	}

	l.l.Debug().
		Str("step_run_id", taskExternalID).
		Int32("invocation_count", invocationCount).
		Msg("DurableTaskListener: sending wait_for request")

	l.SendRequest(&v1.DurableTaskRequest{
		Message: &v1.DurableTaskRequest_WaitFor{
			WaitFor: &v1.DurableTaskWaitForRequest{
				InvocationCount:       invocationCount,
				DurableTaskExternalId: taskExternalID,
				WaitForConditions:     conditions,
				Label:                 labelPtr,
			},
		},
	})

	var ackResp *v1.DurableTaskResponse
	select {
	case <-ctx.Done():
		l.removePendingEventAck(ackKey)
		return nil, ctx.Err()
	case ack := <-ackCh:
		if ack.Err != nil {
			return nil, ack.Err
		}
		ackResp = ack.Resp
	}

	waitAck := ackResp.GetWaitForAck()
	if waitAck == nil || waitAck.GetRef() == nil {
		return nil, fmt.Errorf("wait_for ack missing ref for task %s invocation %d", taskExternalID, invocationCount)
	}
	ref := waitAck.GetRef()

	return l.WaitForCallback(ctx, taskExternalID, invocationCount, ref.GetBranchId(), ref.GetNodeId())
}

// MemoAckResult is what SendMemoRequest returns on a successful ack: the server-assigned
// log entry ref, whether a memo already existed, and (if it did) the cached payload.
type MemoAckResult struct {
	Ref                *v1.DurableEventLogEntryRef
	CachedPayload      []byte
	MemoAlreadyExisted bool
}

// SendMemoRequest sends a memo lookup over the bidi DurableTask stream and blocks
// until the engine acks with the cached payload (if any) and the log entry ref.
func (l *DurableTaskListener) SendMemoRequest(
	ctx context.Context,
	taskExternalID string,
	invocationCount int32,
	memoKey []byte,
) (*MemoAckResult, error) {
	ackKey := PendingAckKey{TaskID: taskExternalID, SignalKey: int64(invocationCount)}
	ackCh := l.AddPendingEventAck(ackKey)

	l.SendRequest(&v1.DurableTaskRequest{
		Message: &v1.DurableTaskRequest_Memo{
			Memo: &v1.DurableTaskMemoRequest{
				InvocationCount:       invocationCount,
				DurableTaskExternalId: taskExternalID,
				Key:                   memoKey,
			},
		},
	})

	select {
	case <-ctx.Done():
		l.removePendingEventAck(ackKey)
		return nil, ctx.Err()
	case ack := <-ackCh:
		if ack.Err != nil {
			return nil, ack.Err
		}
		memoAck := ack.Resp.GetMemoAck()
		if memoAck == nil || memoAck.GetRef() == nil {
			return nil, fmt.Errorf("memo ack missing ref for task %s invocation %d", taskExternalID, invocationCount)
		}
		return &MemoAckResult{
			Ref:                memoAck.GetRef(),
			MemoAlreadyExisted: memoAck.GetMemoAlreadyExisted(),
			CachedPayload:      memoAck.GetMemoResultPayload(),
		}, nil
	}
}

// SendMemoCompleted sends a fire-and-forget completion notification carrying the
// computed memo payload so the engine persists it for future replays.
func (l *DurableTaskListener) SendMemoCompleted(
	ref *v1.DurableEventLogEntryRef,
	memoKey []byte,
	payload []byte,
) {
	l.SendRequest(&v1.DurableTaskRequest{
		Message: &v1.DurableTaskRequest_CompleteMemo{
			CompleteMemo: &v1.DurableTaskCompleteMemoRequest{
				Ref:     ref,
				MemoKey: memoKey,
				Payload: payload,
			},
		},
	})
}

func (l *DurableTaskListener) dispatchResponse(resp *v1.DurableTaskResponse) {
	switch msg := resp.GetMessage().(type) {
	case *v1.DurableTaskResponse_RegisterWorker:
		// register-worker responses are informational; nothing to dispatch.
	case *v1.DurableTaskResponse_EvictionAck:
		ack := msg.EvictionAck
		key := PendingAckKey{
			TaskID:    ack.GetDurableTaskExternalId(),
			SignalKey: int64(ack.GetInvocationCount()),
		}
		l.pendingEvictionAcksMu.Lock()
		ch, ok := l.pendingEvictionAcks[key]
		if ok {
			delete(l.pendingEvictionAcks, key)
		}
		l.pendingEvictionAcksMu.Unlock()
		if ok {
			ch <- nil
		}
	case *v1.DurableTaskResponse_MemoAck:
		ack := msg.MemoAck
		key := PendingAckKey{
			TaskID:    ack.GetRef().GetDurableTaskExternalId(),
			SignalKey: int64(ack.GetRef().GetInvocationCount()),
		}
		l.deliverEventAck(key, resp)
	case *v1.DurableTaskResponse_TriggerRunsAck:
		ack := msg.TriggerRunsAck
		key := PendingAckKey{
			TaskID:    ack.GetDurableTaskExternalId(),
			SignalKey: int64(ack.GetInvocationCount()),
		}
		l.deliverEventAck(key, resp)
	case *v1.DurableTaskResponse_WaitForAck:
		waitAck := msg.WaitForAck
		key := PendingAckKey{
			TaskID:    waitAck.GetRef().GetDurableTaskExternalId(),
			SignalKey: int64(waitAck.GetRef().GetInvocationCount()),
		}
		l.deliverEventAck(key, resp)
	case *v1.DurableTaskResponse_EntryCompleted:
		completed := msg.EntryCompleted
		ref := completed.GetRef()
		key := PendingCallbackKey{
			TaskID:    ref.GetDurableTaskExternalId(),
			SignalKey: int64(ref.GetInvocationCount()),
			BranchID:  ref.GetBranchId(),
			NodeID:    ref.GetNodeId(),
		}

		l.callbackStateMu.Lock()
		if l.callbacksTerminal {
			l.callbackStateMu.Unlock()
			return
		}
		l.pruneExpiredCompletionsLocked(time.Now())
		ch, ok := l.pendingCallbacks[key]
		if ok {
			delete(l.pendingCallbacks, key)
			// Callback channels are buffered, so publishing while locked cannot
			// block and ensures Stop cannot return before this delivery.
			select {
			case ch <- CallbackResult{Resp: resp}:
			default:
			}
		} else {
			l.cacheCompletionLocked(key, resp)
		}
		l.callbackStateMu.Unlock()
	case *v1.DurableTaskResponse_Error:
		l.dispatchError(msg.Error)
	case *v1.DurableTaskResponse_ServerEvict:
		evict := msg.ServerEvict
		taskID := evict.GetDurableTaskExternalId()
		invCount := evict.GetInvocationCount()
		reason := evict.GetReason()

		l.l.Info().
			Str("task_id", taskID).
			Int32("invocation_count", invCount).
			Str("reason", reason).
			Msg("DurableTaskListener: received server eviction notice")

		l.CleanupTaskState(taskID, invCount)

		l.onServerEvictMu.RLock()
		cb := l.onServerEvict
		l.onServerEvictMu.RUnlock()
		if cb != nil {
			cb(taskID, invCount, reason)
		}
	default:
		l.l.Warn().Msg("DurableTaskListener: unknown response type")
	}
}

func (l *DurableTaskListener) deliverEventAck(key PendingAckKey, resp *v1.DurableTaskResponse) {
	l.pendingEventAcksMu.Lock()
	ch, ok := l.pendingEventAcks[key]
	if ok {
		delete(l.pendingEventAcks, key)
	}
	l.pendingEventAcksMu.Unlock()

	if !ok {
		return
	}

	select {
	case ch <- EventAckResult{Resp: resp}:
	default:
	}
}

func (l *DurableTaskListener) dispatchError(errResp *v1.DurableTaskErrorResponse) {
	ref := errResp.GetRef()
	var err error
	switch errResp.GetErrorType() {
	case v1.DurableTaskErrorType_DURABLE_TASK_ERROR_TYPE_NONDETERMINISM:
		err = &NonDeterminismError{
			TaskExternalID:  ref.GetDurableTaskExternalId(),
			InvocationCount: ref.GetInvocationCount(),
			NodeID:          ref.GetNodeId(),
			Message:         errResp.GetErrorMessage(),
		}
	default:
		err = fmt.Errorf("durable task error (type %d): %s", errResp.GetErrorType(), errResp.GetErrorMessage())
	}

	ackKey := PendingAckKey{
		TaskID:    ref.GetDurableTaskExternalId(),
		SignalKey: int64(ref.GetInvocationCount()),
	}

	l.pendingEventAcksMu.Lock()
	eventAckCh, hasEventAck := l.pendingEventAcks[ackKey]
	if hasEventAck {
		delete(l.pendingEventAcks, ackKey)
	}
	l.pendingEventAcksMu.Unlock()
	if hasEventAck {
		select {
		case eventAckCh <- EventAckResult{Err: err}:
		default:
		}
	}

	cbKey := PendingCallbackKey{
		TaskID:    ref.GetDurableTaskExternalId(),
		SignalKey: int64(ref.GetInvocationCount()),
		BranchID:  ref.GetBranchId(),
		NodeID:    ref.GetNodeId(),
	}

	l.callbackStateMu.Lock()
	if !l.callbacksTerminal {
		callbackCh, hasCallback := l.pendingCallbacks[cbKey]
		if hasCallback {
			delete(l.pendingCallbacks, cbKey)
			select {
			case callbackCh <- CallbackResult{Err: err}:
			default:
			}
		}
	}
	l.callbackStateMu.Unlock()

	l.pendingEvictionAcksMu.Lock()
	evictionAckCh, hasEvictionAck := l.pendingEvictionAcks[ackKey]
	if hasEvictionAck {
		delete(l.pendingEvictionAcks, ackKey)
	}
	l.pendingEvictionAcksMu.Unlock()
	if hasEvictionAck {
		select {
		case evictionAckCh <- err:
		default:
		}
	}
}

func (l *DurableTaskListener) signalStatusChanged() {
	// The signal carries no snapshot; the stream sender reads current callback
	// state immediately before Send and coalesces any registration burst.
	select {
	case l.statusChanged <- struct{}{}:
	default:
	}
}

func (l *DurableTaskListener) sendWorkerStatus(stream v1.V1Dispatcher_DurableTaskClient) error {
	req := l.workerStatusRequest()
	if req == nil {
		return nil
	}
	return stream.Send(req)
}

func (l *DurableTaskListener) workerStatusRequest() *v1.DurableTaskRequest {
	l.callbackStateMu.Lock()
	if l.callbacksTerminal || len(l.pendingCallbacks) == 0 {
		l.callbackStateMu.Unlock()
		return nil
	}

	waitingEntries := make([]*v1.DurableTaskAwaitedCompletedEntry, 0, len(l.pendingCallbacks))
	for key := range l.pendingCallbacks {
		waitingEntries = append(waitingEntries, &v1.DurableTaskAwaitedCompletedEntry{
			DurableTaskExternalId: key.TaskID,
			InvocationCount:       int32(key.SignalKey), //nolint:gosec
			BranchId:              key.BranchID,
			NodeId:                key.NodeID,
		})
	}
	l.callbackStateMu.Unlock()

	return &v1.DurableTaskRequest{
		Message: &v1.DurableTaskRequest_WorkerStatus{
			WorkerStatus: &v1.DurableTaskWorkerStatusRequest{
				WorkerId:       l.workerID,
				WaitingEntries: waitingEntries,
			},
		},
	}
}

func (l *DurableTaskListener) cacheCompletionLocked(
	key PendingCallbackKey,
	resp *v1.DurableTaskResponse,
) {
	expiresAt := time.Now().Add(defaultCompletionBufferTTL)
	l.bufferedCompletions[key] = bufferedCompletion{
		resp:      resp,
		expiresAt: expiresAt,
	}
}

func (l *DurableTaskListener) pruneExpiredCompletionsLocked(now time.Time) {
	for key, buffered := range l.bufferedCompletions {
		if !buffered.expiresAt.After(now) {
			delete(l.bufferedCompletions, key)
		}
	}
}

// failPendingAcks fails all pending event acks and eviction acks on disconnect.
// Pending callbacks survive disconnect because the engine re-delivers them on
// reconnection.
func (l *DurableTaskListener) failPendingAcks(err error) {
	l.pendingEventAcksMu.Lock()
	acks := l.pendingEventAcks
	l.pendingEventAcks = make(map[PendingAckKey]chan EventAckResult)
	l.pendingEventAcksMu.Unlock()

	for _, ch := range acks {
		select {
		case ch <- EventAckResult{Err: err}:
		default:
		}
	}

	l.pendingEvictionAcksMu.Lock()
	evictionAcks := l.pendingEvictionAcks
	l.pendingEvictionAcks = make(map[PendingAckKey]chan error)
	l.pendingEvictionAcksMu.Unlock()

	for _, ch := range evictionAcks {
		select {
		case ch <- err:
		default:
		}
	}
}

func (l *DurableTaskListener) terminateCallbackState(err error) {
	l.callbackStateMu.Lock()
	defer l.callbackStateMu.Unlock()
	l.terminateCallbackStateLocked(err)
}

func (l *DurableTaskListener) terminateCallbackStateLocked(err error) {
	l.callbacksTerminal = true
	l.callbackTerminalErr = err
	callbacks := l.pendingCallbacks
	l.pendingCallbacks = make(map[PendingCallbackKey]chan CallbackResult)
	l.bufferedCompletions = make(map[PendingCallbackKey]bufferedCompletion)

	// Keep delivery inside the terminal transition so no detached waiter can
	// receive its shutdown result after Stop returns.
	for _, ch := range callbacks {
		select {
		case ch <- CallbackResult{Err: err}:
		default:
		}
	}
}

func isGRPCCancelled(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	return st.Code() == codes.Canceled
}
