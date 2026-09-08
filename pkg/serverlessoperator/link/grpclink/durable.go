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

	// sendQueueSize bounds requests waiting for the hub's pump to hand them to the shared
	// listener. A Send past it blocks, selecting on the invocation's close, so a full queue
	// (an unreachable engine) never holds a closing invocation.
	sendQueueSize = 256

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
//
// Requests reach the listener through the hub's bounded outbound queue and one pump
// goroutine: the listener's own enqueue has no cancellation, so only the pump ever blocks on
// it, while every channel's Send selects on its own close signal.
type durableHub struct {
	listener *client.DurableTaskListener //nolint:staticcheck // see import
	channels map[channelKey]*durableChannel
	outbound chan outboundRequest
	stopped  chan struct{}
	pumpDone chan struct{}
	stopOnce sync.Once
	mu       sync.Mutex
	closed   bool

	// pumpCtx bounds the pump's hand-off to the listener; closeAll cancels it so a pump
	// blocked on the listener's full queue returns.
	pumpCtx    context.Context
	pumpCancel context.CancelFunc
}

// outboundRequest is one queued request and the channel it belongs to; the pump drops
// requests of channels closed while they waited.
type outboundRequest struct {
	req *v1.DurableTaskRequest
	ch  *durableChannel
}

func newDurableHub(session client.OperatorSession) *durableHub { //nolint:staticcheck // see import
	listener := session.NewDurableTaskListener()
	hub := newDurableHubOver(listener)

	listener.SetServerEvictCallback(hub.onServerEvict)
	// The session stops the listener when it closes; Start's context only bounds the
	// listener's own reconnect loop.
	listener.Start(context.Background())

	return hub
}

// newDurableHubOver builds a hub over a listener the caller starts and stops.
func newDurableHubOver(listener *client.DurableTaskListener) *durableHub { //nolint:staticcheck // see import
	pumpCtx, pumpCancel := context.WithCancel(context.Background())

	hub := &durableHub{
		listener:   listener,
		channels:   map[channelKey]*durableChannel{},
		outbound:   make(chan outboundRequest, sendQueueSize),
		stopped:    make(chan struct{}),
		pumpDone:   make(chan struct{}),
		pumpCtx:    pumpCtx,
		pumpCancel: pumpCancel,
	}

	go hub.pump()

	return hub
}

// pump hands queued requests to the shared listener. A listener that is no longer running
// would never drain its queue, so requests are dropped instead of blocking on it; the
// channels they belong to are being closed with the registration. The hand-off itself is
// bounded by pumpCtx and by the listener's own stop, so a full listener queue (an
// unreachable engine) cannot hold the pump past closeAll.
func (h *durableHub) pump() {
	defer close(h.pumpDone)

	for {
		select {
		case <-h.stopped:
			return
		case item := <-h.outbound:
			if item.ch.isClosed() || !h.listener.IsRunning() {
				continue
			}

			if err := h.listener.SendRequest(h.pumpCtx, item.req); err != nil && h.pumpCtx.Err() != nil {
				return
			}
		}
	}
}

// enqueue queues req for the listener, or reports the channel closed. A full queue blocks
// until there is room or the channel or hub closes.
func (h *durableHub) enqueue(ch *durableChannel, req *v1.DurableTaskRequest) error {
	select {
	case <-ch.closed:
		return link.ErrChannelClosed
	case <-h.stopped:
		return link.ErrChannelClosed
	default:
	}

	select {
	case h.outbound <- outboundRequest{req: req, ch: ch}:
		return nil
	case <-ch.closed:
		return link.ErrChannelClosed
	case <-h.stopped:
		return link.ErrChannelClosed
	}
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

// closeAll closes every open channel and stops the pump; the listener itself is stopped by
// the session.
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

	h.stopOnce.Do(func() {
		close(h.stopped)
		h.pumpCancel()
	})
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
//
// Every goroutine the channel starts is refused once closing begins and joined before the
// invocation's pending state is dropped from the listener, so no registration can land
// after the cleanup.
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
	goroutines sync.WaitGroup
	mu         sync.Mutex
	inflight   bool
	closing    bool
}

// spawn runs fn on a goroutine the channel joins on Close; nothing is started once closing.
func (c *durableChannel) spawn(fn func()) {
	c.mu.Lock()

	if c.closing {
		c.mu.Unlock()
		return
	}

	c.goroutines.Add(1)
	c.mu.Unlock()

	go func() {
		defer c.goroutines.Done()
		fn()
	}()
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

		return c.hub.enqueue(c, req)
	case *v1.DurableTaskRequest_RegisterWorker, *v1.DurableTaskRequest_WorkerStatus:
		return errors.New("register_worker and worker_status are link-internal")
	default:
		return errors.New("unknown durable request")
	}
}

// sendAcked registers the event ack, sends, and hands the ack to handle on its own goroutine.
// A send refused because the channel closed undoes the registration and the in-flight slot.
func (c *durableChannel) sendAcked(req *v1.DurableTaskRequest, handle func(*v1.DurableTaskResponse)) error {
	if err := c.acquire(); err != nil {
		return err
	}

	ackCh := c.listener.AddPendingEventAck(c.ackKey())

	if err := c.hub.enqueue(c, req); err != nil {
		c.listener.CleanupTaskState(c.taskId, c.invocation)
		c.release()

		return err
	}

	c.spawn(func() {
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
	})

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

	c.awaitEntryAsync(ref.GetBranchId(), ref.GetNodeId())
}

// awaitTriggerRunsAck registers for every spawned run's completion, as the SDK does when
// the caller awaits a child result.
func (c *durableChannel) awaitTriggerRunsAck(resp *v1.DurableTaskResponse) {
	for _, entry := range resp.GetTriggerRunsAck().GetRunEntries() {
		c.awaitEntryAsync(entry.GetBranchId(), entry.GetNodeId())
	}
}

// awaitEntryAsync waits for the entry on a goroutine the channel joins on Close.
func (c *durableChannel) awaitEntryAsync(branchId, nodeId int64) {
	c.spawn(func() { c.awaitEntry(branchId, nodeId) })
}

// awaitEntry registers for the entry's completion and delivers it, unless the channel is
// already closing: a registration after the invocation's cleanup would outlive it.
func (c *durableChannel) awaitEntry(branchId, nodeId int64) {
	if c.isClosed() {
		return
	}

	key := client.PendingCallbackKey{ //nolint:staticcheck // see import
		TaskID:    c.taskId,
		SignalKey: int64(c.invocation),
		NodeID:    nodeId,
		BranchID:  branchId,
	}

	cbCh := c.listener.AddPendingCallback(key)

	select {
	case <-c.closed:
		// Close may have cleaned the invocation's state up before this registration
		// landed; only a goroutine started outside spawn can get here, so drop it again.
		c.listener.CleanupTaskState(c.taskId, c.invocation)
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

	if err := c.hub.enqueue(c, req); err != nil {
		c.listener.CleanupTaskState(c.taskId, c.invocation)
		c.release()

		return err
	}

	c.spawn(func() {
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
	})

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

// Close implements link.DurableChannel: unblocks Send, Recv and the ack goroutines, joins
// every goroutine the channel started so none can register state afterwards, and then drops
// the invocation's pending state on the listener.
func (c *durableChannel) Close() error {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closing = true
		c.mu.Unlock()

		close(c.closed)
		c.goroutines.Wait()
		c.hub.remove(c)
		c.listener.CleanupTaskState(c.taskId, c.invocation)
	})

	return nil
}
