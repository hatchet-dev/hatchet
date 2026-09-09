package operatorsvc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

var (
	// ErrRequestInFlight is returned by Send while the invocation is waiting for the ack of a
	// previous request.
	ErrRequestInFlight = errors.New("operatorsvc: durable request already in flight for this invocation")

	// ErrChannelClosed is returned once Close was called on the channel.
	ErrChannelClosed = errors.New("operatorsvc: durable channel closed")

	// ErrSessionEnded is returned by Recv when the engine tore the invocation down without
	// Close being called, which the caller reports as a retryable failure.
	ErrSessionEnded = errors.New("operatorsvc: engine durable session ended")
)

const (
	// tenantContextKey is the context key the dispatcher and admin services read the tenant
	// from; the gRPC auth middleware sets the same one.
	tenantContextKey = "tenant"

	// durableCloseDrainTimeout bounds how long Close waits for the engine to close the
	// response side after the invocation is cancelled.
	durableCloseDrainTimeout = 5 * time.Second

	// HandshakeHoldLimit bounds the invocation responses held while the register-worker ack is
	// awaited. A session that exceeds it is refused rather than buffering without end.
	HandshakeHoldLimit = 256
)

// DurableChannel is one durable invocation's request and response pipe. Send stamps the
// invocation's task id and count on the request and admits one ack-bearing request at a time.
type DurableChannel interface {
	Send(req *v1contracts.DurableTaskRequest) error
	Recv() (*v1contracts.DurableTaskResponse, error)
	Close() error
}

// withTenant puts the tenant on the context the way the gRPC auth middleware does, so the
// dispatcher's handlers see the same value whichever host called them.
func withTenant(ctx context.Context, tenant *sqlcv1.Tenant) context.Context {
	return context.WithValue(ctx, tenantContextKey, tenant) //nolint:staticcheck // key must match the gRPC auth middleware's
}

// durableChannel is one durable invocation's pipe over the dispatcher's channel-backed session,
// the in-engine equivalent of the DurableTask stream. The session lives until Close cancels it;
// the engine then deregisters the invocation and closes the response channel.
//
// At most one ack-bearing request (memo, trigger_runs, wait_for, evict_invocation) may be in
// flight: the engine keys its pending state by (task, invocation), so a second one would clobber
// the first. The slot is released when the ack, or the error that replaces it, is read through
// Recv.
type durableChannel struct {
	reqCh  chan<- *v1contracts.DurableTaskRequest
	respCh <-chan *v1contracts.DurableTaskResponse
	ctx    context.Context
	cancel context.CancelFunc
	closed chan struct{}

	taskExternalId string
	invocation     int32

	// The engine delivers entry_completed for an already satisfied entry as soon as the
	// invocation registers, which on a replay is before the operator has sent the wait_for (or
	// trigger_runs) that names it. The operator expects the ack first, so an entry is held
	// until the ack carrying its ref has been returned; wanted records refs whose ack went out
	// before their entry arrived.
	held   map[entryRef][]*v1contracts.DurableTaskResponse
	wanted map[entryRef]struct{}
	ready  []*v1contracts.DurableTaskResponse

	closeOnce sync.Once
	mu        sync.Mutex
	inflight  bool
}

// entryRef identifies one durable event log entry within the invocation.
type entryRef struct {
	branchId int64
	nodeId   int64
}

// OpenDurable opens one durable invocation's pipe for the session's worker and runs the
// register-worker handshake: the first request names the worker running the invocation, and the
// engine's ack is consumed here so Recv only ever returns invocation traffic. The engine routes
// responses to the task as soon as it is registered, before the ack, so invocation traffic that
// arrives during the handshake (a completion restored for a resumed invocation, for instance) is
// held for the channel, under the same ack-before-entry ordering, up to handshakeHoldLimit
// responses. ctx bounds the handshake; the invocation itself is detached from it and ends on
// Close.
func (ss *Session) OpenDurable(ctx context.Context, taskExternalId uuid.UUID, invocation int32) (DurableChannel, error) {
	sctx, cancel := context.WithCancel(withTenant(context.WithoutCancel(ctx), ss.tenant))

	reqCh, respCh, err := ss.svc.dispatcher.RegisterDurableTask(sctx, taskExternalId)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not register durable task %s: %w", taskExternalId, err)
	}

	ch := &durableChannel{
		reqCh:          reqCh,
		respCh:         respCh,
		ctx:            sctx,
		cancel:         cancel,
		closed:         make(chan struct{}),
		taskExternalId: taskExternalId.String(),
		invocation:     invocation,
		held:           make(map[entryRef][]*v1contracts.DurableTaskResponse),
		wanted:         make(map[entryRef]struct{}),
	}

	register := &v1contracts.DurableTaskRequest{
		Message: &v1contracts.DurableTaskRequest_RegisterWorker{
			RegisterWorker: &v1contracts.DurableTaskRequestRegisterWorker{WorkerId: ss.workerId.String()},
		},
	}

	select {
	case reqCh <- register:
	case <-ctx.Done():
		_ = ch.Close()
		return nil, fmt.Errorf("durable session interrupted before the register worker request was sent: %w", ctx.Err())
	}

	held := 0

	for {
		select {
		case resp, ok := <-respCh:
			if !ok {
				_ = ch.Close()
				return nil, errors.New("durable session closed while waiting for the register worker ack")
			}

			if resp.GetRegisterWorker() != nil {
				return ch, nil
			}

			if e := resp.GetError(); e != nil {
				_ = ch.Close()
				return nil, fmt.Errorf("engine rejected the register worker request: %s", e.ErrorMessage)
			}

			held++

			if held > HandshakeHoldLimit {
				_ = ch.Close()
				return nil, fmt.Errorf("durable handshake for task %s received %d responses before the register worker ack; the limit is %d", taskExternalId, held, HandshakeHoldLimit)
			}

			ch.hold(resp)
		case <-ctx.Done():
			_ = ch.Close()
			return nil, fmt.Errorf("durable session interrupted waiting for the register worker ack: %w", ctx.Err())
		}
	}
}

// hold keeps a response that arrived before the handshake completed: entries wait for the ack
// naming their ref, anything else is queued for the first Recv.
func (c *durableChannel) hold(resp *v1contracts.DurableTaskResponse) {
	if c.deliverable(resp) {
		c.mu.Lock()
		c.ready = append(c.ready, resp)
		c.mu.Unlock()
	}
}

func (c *durableChannel) isClosed() bool {
	select {
	case <-c.closed:
		return true
	default:
		return false
	}
}

// acquire takes the one in-flight slot for ack-bearing requests.
func (c *durableChannel) acquire() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.inflight {
		return ErrRequestInFlight
	}

	c.inflight = true

	return nil
}

func (c *durableChannel) release() {
	c.mu.Lock()
	c.inflight = false
	c.mu.Unlock()
}

// Send stamps the invocation's task id and count on the request, as the durable protocol
// requires, before it is handed to the engine. It blocks while the engine is busy with the
// previous request.
func (c *durableChannel) Send(req *v1contracts.DurableTaskRequest) error {
	if c.isClosed() {
		return ErrChannelClosed
	}

	acked := false

	switch m := req.GetMessage().(type) {
	case *v1contracts.DurableTaskRequest_Memo:
		m.Memo.DurableTaskExternalId, m.Memo.InvocationCount = c.taskExternalId, c.invocation
		acked = true
	case *v1contracts.DurableTaskRequest_TriggerRuns:
		m.TriggerRuns.DurableTaskExternalId, m.TriggerRuns.InvocationCount = c.taskExternalId, c.invocation
		acked = true
	case *v1contracts.DurableTaskRequest_WaitFor:
		m.WaitFor.DurableTaskExternalId, m.WaitFor.InvocationCount = c.taskExternalId, c.invocation
		acked = true
	case *v1contracts.DurableTaskRequest_EvictInvocation:
		m.EvictInvocation.DurableTaskExternalId, m.EvictInvocation.InvocationCount = c.taskExternalId, c.invocation
		acked = true
	case *v1contracts.DurableTaskRequest_CompleteMemo:
		if m.CompleteMemo.Ref == nil {
			return errors.New("complete_memo requires a ref")
		}

		m.CompleteMemo.Ref.DurableTaskExternalId, m.CompleteMemo.Ref.InvocationCount = c.taskExternalId, c.invocation
	case *v1contracts.DurableTaskRequest_RegisterWorker, *v1contracts.DurableTaskRequest_WorkerStatus:
		return errors.New("register_worker and worker_status are owned by the session")
	default:
		return errors.New("unknown durable request")
	}

	if acked {
		if err := c.acquire(); err != nil {
			return err
		}
	}

	select {
	case c.reqCh <- req:
		return nil
	case <-c.ctx.Done():
		if acked {
			c.release()
		}

		return ErrChannelClosed
	}
}

// Recv returns the next response for the invocation. It returns ErrChannelClosed once Close was
// called and ErrSessionEnded when the engine ended the invocation on its own.
func (c *durableChannel) Recv() (*v1contracts.DurableTaskResponse, error) {
	for {
		c.mu.Lock()
		if len(c.ready) > 0 {
			resp := c.ready[0]
			c.ready = c.ready[1:]
			c.mu.Unlock()

			return resp, nil
		}
		c.mu.Unlock()

		select {
		case <-c.closed:
			return nil, ErrChannelClosed
		case resp, ok := <-c.respCh:
			if !ok {
				if c.isClosed() {
					return nil, ErrChannelClosed
				}

				return nil, ErrSessionEnded
			}

			if c.deliverable(resp) {
				return resp, nil
			}
		}
	}
}

// deliverable applies the ack-before-entry ordering: an entry_completed is returned only if the
// ack naming its ref already went out, otherwise it is held; an ack releases the in-flight slot
// and queues any held entries for the refs it names.
func (c *durableChannel) deliverable(resp *v1contracts.DurableTaskResponse) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch m := resp.GetMessage().(type) {
	case *v1contracts.DurableTaskResponse_EntryCompleted:
		ref := entryRef{branchId: m.EntryCompleted.GetRef().GetBranchId(), nodeId: m.EntryCompleted.GetRef().GetNodeId()}

		if _, ok := c.wanted[ref]; ok {
			delete(c.wanted, ref)
			return true
		}

		c.held[ref] = append(c.held[ref], resp)

		return false
	case *v1contracts.DurableTaskResponse_WaitForAck:
		c.inflight = false
		c.expect(entryRef{branchId: m.WaitForAck.GetRef().GetBranchId(), nodeId: m.WaitForAck.GetRef().GetNodeId()})
	case *v1contracts.DurableTaskResponse_TriggerRunsAck:
		c.inflight = false

		for _, entry := range m.TriggerRunsAck.GetRunEntries() {
			c.expect(entryRef{branchId: entry.GetBranchId(), nodeId: entry.GetNodeId()})
		}
	case *v1contracts.DurableTaskResponse_MemoAck, *v1contracts.DurableTaskResponse_EvictionAck, *v1contracts.DurableTaskResponse_Error:
		c.inflight = false
	}

	return true
}

// expect marks ref as acknowledged: a held entry for it is queued behind the ack, a future one
// is returned as it arrives. Must be called with mu held.
func (c *durableChannel) expect(ref entryRef) {
	if entries, ok := c.held[ref]; ok {
		c.ready = append(c.ready, entries...)
		delete(c.held, ref)

		return
	}

	c.wanted[ref] = struct{}{}
}

// Close cancels the invocation, unblocks Send and Recv, and drains responses until the engine
// closes its side, so the invocation is deregistered before Close returns.
func (c *durableChannel) Close() error {
	c.closeOnce.Do(func() {
		close(c.closed)
		c.cancel()

		timeout := time.NewTimer(durableCloseDrainTimeout)
		defer timeout.Stop()

		for {
			select {
			case _, ok := <-c.respCh:
				if !ok {
					return
				}
			case <-timeout.C:
				return
			}
		}
	})

	return nil
}
