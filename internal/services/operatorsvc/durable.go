package operatorsvc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// The channel's errors are the contract's, so an operator sees the same sentinels whichever
// host opened its channel.
var (
	ErrRequestInFlight = operator.ErrRequestInFlight
	ErrChannelClosed   = operator.ErrChannelClosed
	ErrSessionEnded    = operator.ErrSessionEnded
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

// DurableChannel is the contract's channel; the in-process implementation is what OpenDurable
// returns.
type DurableChannel = operator.DurableChannel

// WithTenant puts the tenant on the context the way the gRPC auth middleware does, so the
// dispatcher's and the admin service's handlers see the same value whichever host called them.
func WithTenant(ctx context.Context, tenant *sqlcv1.Tenant) context.Context {
	return context.WithValue(ctx, tenantContextKey, tenant) //nolint:staticcheck // key must match the gRPC auth middleware's
}

// durableChannel is one durable invocation's pipe over the dispatcher's channel-backed session,
// the in-engine equivalent of the DurableTask stream. The session lives until Close cancels it;
// the engine then deregisters the invocation and closes the response channel.
//
// The engine delivers responses on one goroutine that blocks on each delivery before it reads
// the next request, so an operator that sends while a response is undelivered would deadlock
// against it. A pump goroutine therefore reads the engine's responses as they come and queues
// them for Recv; Send only ever waits for the engine to take the request.
//
// At most one ack-bearing request (memo, trigger_runs, wait_for, evict_invocation) may be in
// flight: the engine keys its pending state by (task, invocation), so a second one would clobber
// the first. The slot is released when the ack, or the error that replaces it, arrives.
type durableChannel struct {
	reqCh  chan<- *v1contracts.DurableTaskRequest
	respCh <-chan *v1contracts.DurableTaskResponse
	ctx    context.Context
	cancel context.CancelFunc
	closed chan struct{}

	taskExternalId string
	invocation     int32
	workerId       string

	// The engine delivers entry_completed for an already satisfied entry as soon as the
	// invocation registers, which on a replay is before the operator has sent the wait_for (or
	// trigger_runs) that names it. The operator expects the ack first, so an entry is held
	// until the ack carrying its ref has been queued; wanted records refs whose ack went out
	// before their entry arrived.
	held   map[entryRef][]*v1contracts.DurableTaskResponse
	wanted map[entryRef]struct{}
	ready  []*v1contracts.DurableTaskResponse

	// notify wakes a Recv waiting for the queue to grow or the pump to end.
	notify   chan struct{}
	pumpDone chan struct{}

	closeOnce sync.Once
	mu        sync.Mutex
	inflight  bool
	// ended records that the engine closed its side; the pump sets it.
	ended bool
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
// held for the channel, under the same ack-before-entry ordering, up to HandshakeHoldLimit
// responses. ctx bounds the handshake; the invocation itself is detached from it and ends on
// Close.
func (ss *Session) OpenDurable(ctx context.Context, taskExternalId uuid.UUID, invocation int32) (DurableChannel, error) {
	sctx, cancel := context.WithCancel(WithTenant(context.WithoutCancel(ctx), ss.tenant))

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
		workerId:       ss.workerId.String(),
		held:           make(map[entryRef][]*v1contracts.DurableTaskResponse),
		wanted:         make(map[entryRef]struct{}),
		notify:         make(chan struct{}, 1),
		pumpDone:       make(chan struct{}),
	}

	register := &v1contracts.DurableTaskRequest{
		Message: &v1contracts.DurableTaskRequest_RegisterWorker{
			RegisterWorker: &v1contracts.DurableTaskRequestRegisterWorker{WorkerId: ch.workerId},
		},
	}

	select {
	case reqCh <- register:
	case <-ctx.Done():
		ch.abandon()
		return nil, fmt.Errorf("durable session interrupted before the register worker request was sent: %w", ctx.Err())
	}

	held := 0

	for {
		select {
		case resp, ok := <-respCh:
			if !ok {
				ch.abandon()
				return nil, errors.New("durable session closed while waiting for the register worker ack")
			}

			if resp.GetRegisterWorker() != nil {
				go ch.pump()
				return ch, nil
			}

			if e := resp.GetError(); e != nil {
				ch.abandon()
				return nil, fmt.Errorf("engine rejected the register worker request: %s", e.ErrorMessage)
			}

			held++

			if held > HandshakeHoldLimit {
				ch.abandon()
				return nil, fmt.Errorf("durable handshake for task %s received %d responses before the register worker ack; the limit is %d", taskExternalId, held, HandshakeHoldLimit)
			}

			ch.ingest(resp)
		case <-ctx.Done():
			ch.abandon()
			return nil, fmt.Errorf("durable session interrupted waiting for the register worker ack: %w", ctx.Err())
		}
	}
}

// abandon tears down a channel whose handshake failed: the pump never started, so the engine's
// side is drained here until it closes.
func (c *durableChannel) abandon() {
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
}

// pump reads the engine's responses for the life of the invocation and queues the deliverable
// ones for Recv. It runs until the engine closes its side, which Close forces by cancelling the
// invocation.
func (c *durableChannel) pump() {
	defer close(c.pumpDone)

	for resp := range c.respCh {
		c.ingest(resp)
	}

	c.mu.Lock()
	c.ended = true
	c.mu.Unlock()

	c.wake()
}

// ingest applies the ack-before-entry ordering to one response and queues what is deliverable:
// an entry_completed is queued only if the ack naming its ref already went out, otherwise it is
// held; an ack releases the in-flight slot and queues any held entries for the refs it names.
func (c *durableChannel) ingest(resp *v1contracts.DurableTaskResponse) {
	c.mu.Lock()

	switch m := resp.GetMessage().(type) {
	case *v1contracts.DurableTaskResponse_EntryCompleted:
		ref := entryRef{branchId: m.EntryCompleted.GetRef().GetBranchId(), nodeId: m.EntryCompleted.GetRef().GetNodeId()}

		if _, ok := c.wanted[ref]; ok {
			delete(c.wanted, ref)
			c.ready = append(c.ready, resp)
		} else {
			c.held[ref] = append(c.held[ref], resp)
		}
	case *v1contracts.DurableTaskResponse_WaitForAck:
		c.inflight = false
		c.ready = append(c.ready, resp)
		c.expect(entryRef{branchId: m.WaitForAck.GetRef().GetBranchId(), nodeId: m.WaitForAck.GetRef().GetNodeId()})
	case *v1contracts.DurableTaskResponse_TriggerRunsAck:
		c.inflight = false
		c.ready = append(c.ready, resp)

		for _, entry := range m.TriggerRunsAck.GetRunEntries() {
			c.expect(entryRef{branchId: entry.GetBranchId(), nodeId: entry.GetNodeId()})
		}
	case *v1contracts.DurableTaskResponse_MemoAck, *v1contracts.DurableTaskResponse_EvictionAck, *v1contracts.DurableTaskResponse_Error:
		c.inflight = false
		c.ready = append(c.ready, resp)
	default:
		c.ready = append(c.ready, resp)
	}

	c.mu.Unlock()
	c.wake()
}

// expect marks ref as acknowledged: held entries for it are queued behind the ack, a future one
// is queued as it arrives. Must be called with mu held.
func (c *durableChannel) expect(ref entryRef) {
	if entries, ok := c.held[ref]; ok {
		c.ready = append(c.ready, entries...)
		delete(c.held, ref)

		return
	}

	c.wanted[ref] = struct{}{}
}

// ExpectEntry implements the contract: the ref counts as acknowledged, so a completion held for
// it is queued now and one that arrives later is queued as it comes.
func (c *durableChannel) ExpectEntry(branchId, nodeId int64) error {
	if c.isClosed() {
		return ErrChannelClosed
	}

	c.mu.Lock()
	c.expect(entryRef{branchId: branchId, nodeId: nodeId})
	c.mu.Unlock()

	c.wake()

	return nil
}

// wake signals a waiting Recv. The signal is coalesced: a Recv that wakes drains everything
// queued before it waits again.
func (c *durableChannel) wake() {
	select {
	case c.notify <- struct{}{}:
	default:
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
// requires, before it is handed to the engine. It waits for the engine to take the request,
// bounded by ctx.
func (c *durableChannel) Send(ctx context.Context, req *v1contracts.DurableTaskRequest) error {
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
	case *v1contracts.DurableTaskRequest_WorkerStatus:
		// the operator reports the entries it is blocked on; the worker is the session's
		m.WorkerStatus.WorkerId = c.workerId
	case *v1contracts.DurableTaskRequest_RegisterWorker:
		return errors.New("register_worker is owned by the session")
	default:
		return errors.New("unknown durable request")
	}

	if acked {
		if err := c.acquire(); err != nil {
			return err
		}
	}

	// a context that already ended never sends, whatever the engine's readiness
	if err := ctx.Err(); err != nil {
		if acked {
			c.release()
		}

		return err
	}

	select {
	case c.reqCh <- req:
		return nil
	case <-c.ctx.Done():
		if acked {
			c.release()
		}

		return ErrChannelClosed
	case <-ctx.Done():
		if acked {
			c.release()
		}

		return ctx.Err()
	}
}

// Recv returns the next response for the invocation. It returns ErrChannelClosed once Close was
// called, ErrSessionEnded when the engine ended the invocation on its own, and ctx's error when
// ctx ends first.
func (c *durableChannel) Recv(ctx context.Context) (*v1contracts.DurableTaskResponse, error) {
	for {
		c.mu.Lock()

		if len(c.ready) > 0 {
			resp := c.ready[0]
			c.ready[0] = nil
			c.ready = c.ready[1:]
			c.mu.Unlock()

			return resp, nil
		}

		ended := c.ended
		c.mu.Unlock()

		if c.isClosed() {
			return nil, ErrChannelClosed
		}

		if ended {
			return nil, ErrSessionEnded
		}

		select {
		case <-c.notify:
		case <-c.closed:
			return nil, ErrChannelClosed
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// Close cancels the invocation, unblocks Send and Recv, and waits for the engine to close its
// side, so the invocation is deregistered before Close returns.
func (c *durableChannel) Close() error {
	c.closeOnce.Do(func() {
		close(c.closed)
		c.cancel()

		timeout := time.NewTimer(durableCloseDrainTimeout)
		defer timeout.Stop()

		select {
		case <-c.pumpDone:
		case <-timeout.C:
		}
	})

	return nil
}
