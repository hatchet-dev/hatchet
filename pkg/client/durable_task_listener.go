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
)

const (
	defaultReconnectInterval = 2 * time.Second
	evictionAckTimeout       = 30 * time.Second
)

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
	l                     *zerolog.Logger
	connectFn             func(ctx context.Context) (v1.V1Dispatcher_DurableTaskClient, error)
	cancel                context.CancelFunc
	requestQueue          chan *v1.DurableTaskRequest
	bufferedCompletions   map[PendingCallbackKey]*v1.DurableTaskResponse
	pendingEventAcks      map[PendingAckKey]chan EventAckResult
	pendingCallbacks      map[PendingCallbackKey]chan CallbackResult
	workerID              string
	streamSeq             int
	reconnectInterval     time.Duration
	evictionAckTTL        time.Duration
	onServerEvictMu       sync.RWMutex
	streamMu              sync.Mutex
	bufferedCompletionsMu sync.Mutex
	pendingCallbacksMu    sync.Mutex
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
	dtl := &DurableTaskListener{
		workerID:            workerID,
		l:                   l,
		reconnectInterval:   defaultReconnectInterval,
		evictionAckTTL:      evictionAckTimeout,
		connectFn:           connectFn,
		pendingEventAcks:    make(map[PendingAckKey]chan EventAckResult),
		pendingEvictionAcks: make(map[PendingAckKey]chan error),
		pendingCallbacks:    make(map[PendingCallbackKey]chan CallbackResult),
		bufferedCompletions: make(map[PendingCallbackKey]*v1.DurableTaskResponse),
		requestQueue:        make(chan *v1.DurableTaskRequest, 100),
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
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.running {
		return
	}

	listenerCtx, cancel := context.WithCancel(ctx)
	l.cancel = cancel
	l.running = true
	go l.receiveLoop(listenerCtx)
}

// Stop halts the listener.
func (l *DurableTaskListener) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.running = false
	if l.cancel != nil {
		l.cancel()
		l.cancel = nil
	}
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

	l.bufferedCompletionsMu.Lock()
	buffered, hasBuffered := l.bufferedCompletions[key]
	if hasBuffered {
		delete(l.bufferedCompletions, key)
	}
	l.bufferedCompletionsMu.Unlock()

	if hasBuffered {
		ch <- CallbackResult{Resp: buffered}
		return ch
	}

	l.pendingCallbacksMu.Lock()
	defer l.pendingCallbacksMu.Unlock()
	l.pendingCallbacks[key] = ch
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
	l.pendingCallbacksMu.Lock()
	defer l.pendingCallbacksMu.Unlock()
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
	l.bufferedCompletionsMu.Lock()
	defer l.bufferedCompletionsMu.Unlock()
	return len(l.bufferedCompletions)
}

// CleanupTaskState removes pending callbacks, acks, and buffered completions for any
// invocation <= invocationCount of the given task. Used after server-side eviction so
// that stale waiters from a prior invocation don't leak.
func (l *DurableTaskListener) CleanupTaskState(taskExternalID string, invocationCount int32) {
	cancelErr := fmt.Errorf("state cleaned up after eviction of task %s invocation %d", taskExternalID, invocationCount)

	l.pendingCallbacksMu.Lock()
	for k, ch := range l.pendingCallbacks {
		if k.TaskID == taskExternalID && k.SignalKey <= int64(invocationCount) {
			delete(l.pendingCallbacks, k)
			select {
			case ch <- CallbackResult{Err: cancelErr}:
			default:
			}
		}
	}
	l.pendingCallbacksMu.Unlock()

	l.pendingEventAcksMu.Lock()
	for k, ch := range l.pendingEventAcks {
		if k.TaskID == taskExternalID && k.SignalKey <= int64(invocationCount) {
			delete(l.pendingEventAcks, k)
			select {
			case ch <- EventAckResult{Err: cancelErr}:
			default:
			}
		}
	}
	l.pendingEventAcksMu.Unlock()

	l.bufferedCompletionsMu.Lock()
	for k := range l.bufferedCompletions {
		if k.TaskID == taskExternalID && k.SignalKey <= int64(invocationCount) {
			delete(l.bufferedCompletions, k)
		}
	}
	l.bufferedCompletionsMu.Unlock()
}

func (l *DurableTaskListener) receiveLoop(ctx context.Context) {
	defer func() {
		l.mu.Lock()
		l.running = false
		l.mu.Unlock()
		l.failPendingAcks(errors.New("durable task listener stopped"))
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		stream, err := l.connect(ctx)
		if err != nil {
			if isCancelled(ctx) {
				return
			}
			if l.l != nil {
				l.l.Error().Err(err).Msg("DurableTaskListener: connection failed, retrying")
			}
			time.Sleep(l.reconnectInterval)
			continue
		}

		l.streamMu.Lock()
		l.streamSeq++
		l.streamMu.Unlock()

		err = l.handleStream(ctx, stream)
		if err != nil {
			if isCancelled(ctx) || isGRPCCancelled(err) {
				return
			}
			l.failPendingAcks(fmt.Errorf("connection reset: %w", err))
			if l.l != nil {
				l.l.Warn().Err(err).Msg("DurableTaskListener: stream ended, reconnecting")
			}
			time.Sleep(l.reconnectInterval)
			continue
		}

		l.failPendingAcks(errors.New("connection reset: stream ended"))
		if isCancelled(ctx) {
			return
		}
		time.Sleep(l.reconnectInterval)
	}
}

func (l *DurableTaskListener) connect(ctx context.Context) (v1.V1Dispatcher_DurableTaskClient, error) {
	return l.connectFn(ctx)
}

func (l *DurableTaskListener) handleStream(ctx context.Context, stream v1.V1Dispatcher_DurableTaskClient) error {
	if err := stream.Send(&v1.DurableTaskRequest{
		Message: &v1.DurableTaskRequest_RegisterWorker{
			RegisterWorker: &v1.DurableTaskRequestRegisterWorker{
				WorkerId: l.workerID,
			},
		},
	}); err != nil {
		return fmt.Errorf("failed to register worker on durable task stream: %w", err)
	}

	if l.l != nil {
		l.l.Debug().Str("worker_id", l.workerID).Msg("DurableTaskListener: registered worker on stream")
	}

	streamCtx, cancelStream := context.WithCancel(ctx)
	defer cancelStream()

	sendDone := make(chan error, 1)
	go func() {
		for {
			select {
			case <-streamCtx.Done():
				sendDone <- streamCtx.Err()
				return
			case req := <-l.requestQueue:
				if err := stream.Send(req); err != nil {
					sendDone <- err
					return
				}
			}
		}
	}()

	for {
		select {
		case err := <-sendDone:
			return err
		default:
		}

		resp, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		l.dispatchResponse(resp)
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
	l.pendingCallbacksMu.Lock()
	defer l.pendingCallbacksMu.Unlock()
	delete(l.pendingCallbacks, key)
}

// SendWaitForRequest registers a durable wait-for on the engine over the bidi DurableTask
// stream. It mirrors the Python SDK's `listener.send_event(WaitForEvent(...))` + wait-for-callback
// flow: the listener first sends a WaitFor request and blocks for the WaitForAck (which carries
// the server-assigned node_id/branch_id), then blocks until the corresponding EntryCompleted
// response arrives (or ctx is cancelled). The returned payload is the raw JSON result bytes
// which the caller can unmarshal with the same shape as the legacy RegisterDurableEvent path.
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

	if l.l != nil {
		l.l.Debug().
			Str("step_run_id", taskExternalID).
			Int32("invocation_count", invocationCount).
			Msg("DurableTaskListener: sending wait_for request")
	}

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

	cbKey := PendingCallbackKey{
		TaskID:    taskExternalID,
		SignalKey: int64(invocationCount),
		BranchID:  ref.GetBranchId(),
		NodeID:    ref.GetNodeId(),
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
			return nil, fmt.Errorf("wait_for completion missing entry_completed for task %s", taskExternalID)
		}
		return completed.GetPayload(), nil
	}
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
// Mirrors the Python SDK's MemoEvent send_event flow.
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

		l.pendingCallbacksMu.Lock()
		ch, ok := l.pendingCallbacks[key]
		if ok {
			delete(l.pendingCallbacks, key)
		}
		l.pendingCallbacksMu.Unlock()

		if ok {
			select {
			case ch <- CallbackResult{Resp: resp}:
			default:
			}
			return
		}

		l.bufferedCompletionsMu.Lock()
		l.bufferedCompletions[key] = resp
		l.bufferedCompletionsMu.Unlock()
	case *v1.DurableTaskResponse_Error:
		l.dispatchError(msg.Error)
	case *v1.DurableTaskResponse_ServerEvict:
		evict := msg.ServerEvict
		taskID := evict.GetDurableTaskExternalId()
		invCount := evict.GetInvocationCount()
		reason := evict.GetReason()

		if l.l != nil {
			l.l.Info().
				Str("task_id", taskID).
				Int32("invocation_count", invCount).
				Str("reason", reason).
				Msg("DurableTaskListener: received server eviction notice")
		}

		l.CleanupTaskState(taskID, invCount)

		l.onServerEvictMu.RLock()
		cb := l.onServerEvict
		l.onServerEvictMu.RUnlock()
		if cb != nil {
			cb(taskID, invCount, reason)
		}
	default:
		if l.l != nil {
			l.l.Warn().Msg("DurableTaskListener: unknown response type")
		}
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
	if ch, ok := l.pendingEventAcks[ackKey]; ok {
		delete(l.pendingEventAcks, ackKey)
		select {
		case ch <- EventAckResult{Err: err}:
		default:
		}
	}
	l.pendingEventAcksMu.Unlock()

	cbKey := PendingCallbackKey{
		TaskID:    ref.GetDurableTaskExternalId(),
		SignalKey: int64(ref.GetInvocationCount()),
		BranchID:  ref.GetBranchId(),
		NodeID:    ref.GetNodeId(),
	}

	l.pendingCallbacksMu.Lock()
	if ch, ok := l.pendingCallbacks[cbKey]; ok {
		delete(l.pendingCallbacks, cbKey)
		select {
		case ch <- CallbackResult{Err: err}:
		default:
		}
	}
	l.pendingCallbacksMu.Unlock()

	l.pendingEvictionAcksMu.Lock()
	if ch, ok := l.pendingEvictionAcks[ackKey]; ok {
		delete(l.pendingEvictionAcks, ackKey)
		select {
		case ch <- err:
		default:
		}
	}
	l.pendingEvictionAcksMu.Unlock()
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

func isCancelled(ctx context.Context) bool {
	return ctx.Err() != nil
}

func isGRPCCancelled(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	return st.Code() == codes.Canceled
}
