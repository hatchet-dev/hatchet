package grpclink

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client" //nolint:staticcheck // OperatorService's client lives in the legacy client package
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/link"
)

const (
	// recvQueueSize bounds responses waiting for the relay to Recv them. Pushes from
	// per-request goroutines block when it is full; the server-evict push never blocks
	// the listener's receive loop (it is spawned).
	recvQueueSize = 64

	// evictionAckTimeout mirrors the SDK's bound on an eviction ack.
	evictionAckTimeout = 30 * time.Second

	// ackFailureGrace is how long an ack failure waits for a server-evict notice before it
	// is reported as a link failure. The listener fails pending acks (CleanupTaskState) a
	// moment before it invokes the evict callback; within the grace the eviction wins and
	// the failure is dropped, so a superseded invocation is reported as evicted, not failed.
	ackFailureGrace = 100 * time.Millisecond
)

type channelKey struct {
	taskId     string
	invocation int32
}

// durableHub owns the registration's single DurableTaskListener and the channels open on
// it, keyed by (task, invocation). It is created on the first OpenDurable.
type durableHub struct {
	listener *client.DurableTaskListener //nolint:staticcheck // see import
	channels map[channelKey]*durableChannel
	mu       sync.Mutex
	closed   bool
}

func newDurableHub(session client.OperatorSession) *durableHub { //nolint:staticcheck // see import
	hub := &durableHub{channels: map[channelKey]*durableChannel{}}

	listener := session.NewDurableTaskListener()
	listener.SetServerEvictCallback(hub.onServerEvict)
	// The session stops the listener when it closes; Start's context only bounds the
	// listener's own reconnect loop.
	listener.Start(context.Background())

	hub.listener = listener

	return hub
}

func (h *durableHub) open(taskId string, invocation int32) (*durableChannel, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return nil, errors.New("registration closed")
	}

	key := channelKey{taskId: taskId, invocation: invocation}

	if _, ok := h.channels[key]; ok {
		return nil, fmt.Errorf("durable channel for task %s invocation %d is already open", taskId, invocation)
	}

	ch := &durableChannel{
		hub:        h,
		listener:   h.listener,
		taskId:     taskId,
		invocation: invocation,
		queue:      make(chan recvItem, recvQueueSize),
		closed:     make(chan struct{}),
		evictedCh:  make(chan struct{}),
	}

	h.channels[key] = ch

	return ch, nil
}

func (h *durableHub) remove(ch *durableChannel) {
	h.mu.Lock()
	defer h.mu.Unlock()

	key := channelKey{taskId: ch.taskId, invocation: ch.invocation}

	if h.channels[key] == ch {
		delete(h.channels, key)
	}
}

// onServerEvict runs on the listener's receive loop. The notice supersedes the named
// invocation and every older one of the task; each open channel affected yields a
// server_evict response and the relay closes its socket.
func (h *durableHub) onServerEvict(taskId string, invocation int32, reason string) {
	h.mu.Lock()
	affected := make([]*durableChannel, 0, 1)

	for key, ch := range h.channels {
		if key.taskId == taskId && key.invocation <= invocation {
			affected = append(affected, ch)
		}
	}

	h.mu.Unlock()

	for _, ch := range affected {
		ch.serverEvicted(invocation, reason)
	}
}

// closeAll closes every open channel; the listener itself is stopped by the session.
func (h *durableHub) closeAll() {
	h.mu.Lock()
	h.closed = true
	channels := make([]*durableChannel, 0, len(h.channels))

	for _, ch := range h.channels {
		channels = append(channels, ch)
	}

	h.mu.Unlock()

	for _, ch := range channels {
		_ = ch.Close()
	}
}

type recvItem struct {
	resp *v1.DurableTaskResponse
	err  error
}

// durableChannel is one invocation's pipe over the shared listener. Send registers the
// pending ack the request kind needs and a goroutine per ack turns the listener's result
// into a response on the queue that Recv drains, exactly as the SDK's durable context does
// in process: memo and trigger_runs deliver their ack; wait_for delivers its ack and then
// the awaited entry_completed; trigger_runs also delivers entry_completed for every spawned
// run; evict_invocation delivers an eviction_ack; complete_memo is fire-and-forget.
type durableChannel struct {
	hub        *durableHub
	listener   *client.DurableTaskListener //nolint:staticcheck // see import
	queue      chan recvItem
	closed     chan struct{}
	evictedCh  chan struct{}
	taskId     string
	invocation int32
	closeOnce  sync.Once
	evictOnce  sync.Once
	mu         sync.Mutex
	inflight   bool
}

func (c *durableChannel) ackKey() client.PendingAckKey { //nolint:staticcheck // see import
	return client.PendingAckKey{TaskID: c.taskId, SignalKey: int64(c.invocation)} //nolint:staticcheck // see import
}

// acquire takes the one in-flight slot for ack-bearing requests.
func (c *durableChannel) acquire() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.inflight {
		return link.ErrRequestInFlight
	}

	c.inflight = true

	return nil
}

func (c *durableChannel) release() {
	c.mu.Lock()
	c.inflight = false
	c.mu.Unlock()
}

func (c *durableChannel) isClosed() bool {
	select {
	case <-c.closed:
		return true
	default:
		return false
	}
}

// Send implements link.DurableChannel.
func (c *durableChannel) Send(req *v1.DurableTaskRequest) error {
	if c.isClosed() {
		return link.ErrChannelClosed
	}

	switch m := req.Message.(type) {
	case *v1.DurableTaskRequest_Memo:
		m.Memo.DurableTaskExternalId, m.Memo.InvocationCount = c.taskId, c.invocation
		return c.sendAcked(req, c.awaitMemoAck)
	case *v1.DurableTaskRequest_TriggerRuns:
		m.TriggerRuns.DurableTaskExternalId, m.TriggerRuns.InvocationCount = c.taskId, c.invocation
		return c.sendAcked(req, c.awaitTriggerRunsAck)
	case *v1.DurableTaskRequest_WaitFor:
		m.WaitFor.DurableTaskExternalId, m.WaitFor.InvocationCount = c.taskId, c.invocation
		return c.sendAcked(req, c.awaitWaitForAck)
	case *v1.DurableTaskRequest_EvictInvocation:
		m.EvictInvocation.DurableTaskExternalId, m.EvictInvocation.InvocationCount = c.taskId, c.invocation
		return c.sendEviction(req)
	case *v1.DurableTaskRequest_CompleteMemo:
		if m.CompleteMemo.Ref == nil {
			return errors.New("complete_memo requires a ref")
		}

		m.CompleteMemo.Ref.DurableTaskExternalId, m.CompleteMemo.Ref.InvocationCount = c.taskId, c.invocation
		c.listener.SendRequest(req)

		return nil
	case *v1.DurableTaskRequest_RegisterWorker, *v1.DurableTaskRequest_WorkerStatus:
		return errors.New("register_worker and worker_status are link-internal")
	default:
		return errors.New("unknown durable request")
	}
}

// sendAcked registers the event ack, sends, and hands the ack to handle on its own goroutine.
func (c *durableChannel) sendAcked(req *v1.DurableTaskRequest, handle func(*v1.DurableTaskResponse)) error {
	if err := c.acquire(); err != nil {
		return err
	}

	ackCh := c.listener.AddPendingEventAck(c.ackKey())
	c.listener.SendRequest(req)

	go func() {
		select {
		case <-c.closed:
			return
		case ack := <-ackCh:
			c.release()

			if ack.Err != nil {
				c.failed(ack.Err)
				return
			}

			c.push(ack.Resp)
			handle(ack.Resp)
		}
	}()

	return nil
}

func (c *durableChannel) awaitMemoAck(*v1.DurableTaskResponse) {}

// awaitWaitForAck registers for the awaited entry, keyed like the SDK's WaitForCallback:
// the current invocation with the ack's branch and node.
func (c *durableChannel) awaitWaitForAck(resp *v1.DurableTaskResponse) {
	ref := resp.GetWaitForAck().GetRef()

	if ref == nil {
		return
	}

	go c.awaitEntry(ref.GetBranchId(), ref.GetNodeId())
}

// awaitTriggerRunsAck registers for every spawned run's completion, as the SDK does when
// the caller awaits a child result.
func (c *durableChannel) awaitTriggerRunsAck(resp *v1.DurableTaskResponse) {
	for _, entry := range resp.GetTriggerRunsAck().GetRunEntries() {
		go c.awaitEntry(entry.GetBranchId(), entry.GetNodeId())
	}
}

func (c *durableChannel) awaitEntry(branchId, nodeId int64) {
	key := client.PendingCallbackKey{ //nolint:staticcheck // see import
		TaskID:    c.taskId,
		SignalKey: int64(c.invocation),
		NodeID:    nodeId,
		BranchID:  branchId,
	}

	cbCh := c.listener.AddPendingCallback(key)

	select {
	case <-c.closed:
	case result := <-cbCh:
		if result.Err != nil {
			c.failed(result.Err)
			return
		}

		c.push(result.Resp)
	}
}

// sendEviction registers the eviction ack and reconstructs the ack message from it, since
// the listener only reports success or failure.
func (c *durableChannel) sendEviction(req *v1.DurableTaskRequest) error {
	if err := c.acquire(); err != nil {
		return err
	}

	ackCh := c.listener.AddPendingEvictionAck(c.ackKey())
	c.listener.SendRequest(req)

	go func() {
		timer := time.NewTimer(evictionAckTimeout)
		defer timer.Stop()

		select {
		case <-c.closed:
		case err := <-ackCh:
			c.release()

			if err != nil {
				c.failed(err)
				return
			}

			c.push(&v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_EvictionAck{
				EvictionAck: &v1.DurableTaskEvictionAckResponse{
					InvocationCount:       c.invocation,
					DurableTaskExternalId: c.taskId,
				},
			}})
		case <-timer.C:
			c.release()
			c.pushErr(fmt.Errorf("eviction ack timed out after %s", evictionAckTimeout))
		}
	}()

	return nil
}

// failed turns a listener-reported failure into what Recv yields: a non-determinism error
// becomes an error response the relay forwards to the endpoint; anything else is a link
// failure, unless a server-evict notice explains it within the grace.
func (c *durableChannel) failed(err error) {
	var nonDet *client.NonDeterminismError //nolint:staticcheck // see import

	if errors.As(err, &nonDet) {
		c.push(&v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_Error{
			Error: &v1.DurableTaskErrorResponse{
				Ref: &v1.DurableEventLogEntryRef{
					DurableTaskExternalId: nonDet.TaskExternalID,
					InvocationCount:       nonDet.InvocationCount,
					NodeId:                nonDet.NodeID,
				},
				ErrorType:    v1.DurableTaskErrorType_DURABLE_TASK_ERROR_TYPE_NONDETERMINISM,
				ErrorMessage: nonDet.Message,
			},
		}})

		return
	}

	timer := time.NewTimer(ackFailureGrace)
	defer timer.Stop()

	select {
	case <-c.closed:
	case <-c.evictedCh:
	case <-timer.C:
		c.pushErr(err)
	}
}

func (c *durableChannel) push(resp *v1.DurableTaskResponse) {
	select {
	case c.queue <- recvItem{resp: resp}:
	case <-c.closed:
	}
}

func (c *durableChannel) pushErr(err error) {
	select {
	case c.queue <- recvItem{err: err}:
	case <-c.closed:
	}
}

func (c *durableChannel) serverEvicted(invocation int32, reason string) {
	c.evictOnce.Do(func() {
		close(c.evictedCh)

		notice := &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_ServerEvict{
			ServerEvict: &v1.DurableTaskServerEvictNotice{
				DurableTaskExternalId: c.taskId,
				InvocationCount:       invocation,
				Reason:                reason,
			},
		}}

		// Off the listener's receive loop so a full queue cannot stall other channels.
		go c.push(notice)
	})
}

// Recv implements link.DurableChannel.
func (c *durableChannel) Recv() (*v1.DurableTaskResponse, error) {
	select {
	case item := <-c.queue:
		return item.resp, item.err
	case <-c.closed:
		return nil, link.ErrChannelClosed
	}
}

// Close implements link.DurableChannel: drops the invocation's pending state on the
// listener and unblocks Recv and the ack goroutines.
func (c *durableChannel) Close() error {
	c.closeOnce.Do(func() {
		close(c.closed)
		c.hub.remove(c)
		c.listener.CleanupTaskState(c.taskId, c.invocation)
	})

	return nil
}
