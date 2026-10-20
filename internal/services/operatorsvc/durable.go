package operatorsvc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
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

	// RetainedResponseLimit bounds what the receive loop retains for Recv over the life of the
	// invocation: responses queued for delivery plus entry completions held for an ack or an
	// expectation. The handshake bound covers a burst before the operator can read; this one
	// allows for an operator that reads but is momentarily behind a fan-out of completions, and
	// for entries the engine completes that the operator never registers. A channel past it
	// fails with ErrChannelClosed rather than buffer without end inside the engine process.
	RetainedResponseLimit = 4 * HandshakeHoldLimit
)

// DurableChannel is the contract's channel; the in-process implementation is what OpenDurable
// returns.
type DurableChannel = operator.DurableChannel

// WithTenant puts the tenant on the context the way the gRPC auth middleware does, so the
// dispatcher's and the admin service's handlers see the same value whichever host called them.
// In-process callers have no API token, so the analytics tenant id is what becomes the
// PostHog distinct id. Without it, captures such as durable-task:register are rejected.
func WithTenant(ctx context.Context, tenant *sqlcv1.Tenant) context.Context {
	ctx = context.WithValue(ctx, tenantContextKey, tenant) //nolint:staticcheck // key must match the gRPC auth middleware's
	if tenant != nil {
		ctx = context.WithValue(ctx, analytics.TenantIDKey, tenant.ID)
	}
	return ctx
}

// contextWithAnalyticsTenant fills the analytics tenant id when the caller has a tenant but
// no API token. PostHog rejects a capture whose distinct id is empty, and the tenant id is
// the distinct id in that case. An id already on the context is left in place.
func contextWithAnalyticsTenant(ctx context.Context, tenantID *uuid.UUID) context.Context {
	if tenantID == nil || analytics.TenantIDFromContext(ctx) != nil {
		return ctx
	}
	return context.WithValue(ctx, analytics.TenantIDKey, *tenantID)
}

// durableChannel is one durable invocation's pipe over the dispatcher's channel-backed session,
// the in-engine equivalent of the DurableTask stream. The session lives until Close cancels it;
// the engine then deregisters the invocation and closes the response channel.
//
// The engine delivers responses on one goroutine that blocks on each delivery before it reads
// the next request, so an operator that sends while a response is undelivered would deadlock
// against it. A receive loop goroutine therefore reads the engine's responses as they come and
// queues them for Recv; Send only ever waits for the engine to take the request.
//
// At most one ack-bearing request (memo, trigger_runs, wait_for, evict_invocation) may be in
// flight: the engine keys its pending state by (task, invocation), so a second one would clobber
// the first. The slot is released when the ack, or the error that replaces it, arrives.
//
// What the receive loop retains is bounded by RetainedResponseLimit. Past it the channel is failed:
// everything retained is dropped, Send and Recv return ErrChannelClosed, and the receive loop keeps
// discarding what the engine sends until the operator closes the channel, so the engine's
// delivery goroutine is never left blocked on it.
type durableChannel struct {
	reqCh  chan<- *v1contracts.DurableTaskRequest
	respCh <-chan *v1contracts.DurableTaskResponse
	ctx    context.Context
	cancel context.CancelFunc
	closed chan struct{}
	l      zerolog.Logger

	taskExternalId string
	invocation     int32
	workerId       string

	// The engine delivers entry_completed for an already satisfied entry as soon as the
	// invocation registers, which on a replay is before the operator has sent the wait_for (or
	// trigger_runs) that names it. The operator expects the ack first, so an entry is held
	// until the ack carrying its ref has been queued, or until the operator expects the ref
	// (ExpectEntry, for entries no request on the channel acknowledges); wanted records refs
	// whose ack or expectation came before their entry.
	//
	// completions keeps the engine's entry completions in the order it delivered them, and
	// only a prefix of it is ever released: the dispatcher delivers completions in
	// satisfied_order, and an operator that replays a durable log applies them in that order,
	// so a completion held for its ack is never overtaken by a later one whose ref is already
	// acknowledged. A completion is delivered once: queued records the refs waiting in
	// completions and delivered the refs already released, and a repeat (the engine resends
	// satisfied entries the operator reports as awaited) is dropped.
	completions []*v1contracts.DurableTaskResponse
	queued      map[entryRef]struct{}
	wanted      map[entryRef]struct{}
	delivered   map[entryRef]struct{}
	ready       []*v1contracts.DurableTaskResponse

	// retained counts ready plus held, against RetainedResponseLimit; overflowed records that
	// the limit was passed and the channel failed.
	retained   int
	overflowed bool

	// notify wakes a Recv waiting for the queue to grow or the receive loop to end.
	notify       chan struct{}
	recvLoopDone chan struct{}

	closeOnce sync.Once
	mu        sync.Mutex
	inflight  bool
	// ended records that the engine closed its side; the receive loop sets it.
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
		l:              ss.l.With().Str("task_external_id", taskExternalId.String()).Int32("invocation", invocation).Logger(),
		taskExternalId: taskExternalId.String(),
		invocation:     invocation,
		workerId:       ss.workerId.String(),
		queued:         make(map[entryRef]struct{}),
		wanted:         make(map[entryRef]struct{}),
		delivered:      make(map[entryRef]struct{}),
		notify:         make(chan struct{}, 1),
		recvLoopDone:   make(chan struct{}),
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
				go ch.recvLoop()
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

// abandon tears down a channel whose handshake failed: the receive loop never started, so the
// engine's side is drained here until it closes.
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

// recvLoop reads the engine's responses for the life of the invocation and queues the deliverable
// ones for Recv. It runs until the engine closes its side, which Close forces by cancelling the
// invocation; a failed channel keeps reading and discards, so the engine never blocks on it.
func (c *durableChannel) recvLoop() {
	defer close(c.recvLoopDone)

	for resp := range c.respCh {
		c.ingest(resp)
	}

	c.mu.Lock()
	c.ended = true
	c.mu.Unlock()

	c.wake()
}

// ingest applies the ack-before-entry ordering to one response and queues what is deliverable:
// an entry_completed joins the completion queue, whose acknowledged prefix is released; an ack
// releases the in-flight slot and acknowledges the refs it names. Every response retained,
// queued or held, counts against RetainedResponseLimit.
func (c *durableChannel) ingest(resp *v1contracts.DurableTaskResponse) {
	c.mu.Lock()

	if c.overflowed {
		c.mu.Unlock()
		return
	}

	switch m := resp.GetMessage().(type) {
	case *v1contracts.DurableTaskResponse_EntryCompleted:
		// The engine routes responses by task, so a callback delayed from an earlier invocation
		// (or, after a fork, from a branch this invocation does not run) can reach this channel.
		// Nothing here will ever acknowledge its ref, and the completion queue releases only its
		// acknowledged prefix, so admitting it would hold every later completion behind it.
		if r := m.EntryCompleted.GetRef(); r.GetDurableTaskExternalId() != c.taskExternalId || r.GetInvocationCount() != c.invocation {
			c.l.Debug().Str("ref_task", r.GetDurableTaskExternalId()).Int32("ref_invocation", r.GetInvocationCount()).Int64("node_id", r.GetNodeId()).Int64("branch_id", r.GetBranchId()).Msg("dropping entry completion addressed to another invocation")
			c.mu.Unlock()
			return
		}

		ref := entryRef{branchId: m.EntryCompleted.GetRef().GetBranchId(), nodeId: m.EntryCompleted.GetRef().GetNodeId()}

		if _, ok := c.delivered[ref]; ok {
			c.mu.Unlock()
			return
		}

		if _, ok := c.queued[ref]; ok {
			c.mu.Unlock()
			return
		}

		c.queued[ref] = struct{}{}
		c.completions = append(c.completions, resp)
		c.retain(1)
		c.releaseCompletions()
	case *v1contracts.DurableTaskResponse_WaitForAck:
		c.inflight = false
		c.queue(resp)
		c.expect(entryRef{branchId: m.WaitForAck.GetRef().GetBranchId(), nodeId: m.WaitForAck.GetRef().GetNodeId()})
	case *v1contracts.DurableTaskResponse_TriggerRunsAck:
		c.inflight = false
		c.queue(resp)

		for _, entry := range m.TriggerRunsAck.GetRunEntries() {
			c.expect(entryRef{branchId: entry.GetBranchId(), nodeId: entry.GetNodeId()})
		}
	case *v1contracts.DurableTaskResponse_MemoAck, *v1contracts.DurableTaskResponse_EvictionAck, *v1contracts.DurableTaskResponse_Error:
		c.inflight = false
		c.queue(resp)
	default:
		c.queue(resp)
	}

	c.mu.Unlock()
	c.wake()
}

// queue makes resp the next deliverable response. Must be called with mu held.
func (c *durableChannel) queue(resp *v1contracts.DurableTaskResponse) {
	c.ready = append(c.ready, resp)
	c.retain(1)
}

// retain accounts for n more retained responses and fails the channel past the limit: what
// was retained is dropped so the failure frees the memory it protects. Must be called with mu
// held.
func (c *durableChannel) retain(n int) {
	c.retained += n

	if c.retained <= RetainedResponseLimit {
		return
	}

	c.l.Error().Int("retained", c.retained).Int("limit", RetainedResponseLimit).Msg("durable invocation retains more responses than the limit allows; failing the channel")

	c.overflowed = true
	c.retained = 0
	c.ready = nil
	c.completions = nil
	c.queued = make(map[entryRef]struct{})
	c.wanted = make(map[entryRef]struct{})
}

// expect marks ref as acknowledged: a completion for it is released once every completion the
// engine delivered before it has been, a future one as it arrives. A ref whose completion was
// delivered is left alone. Must be called with mu held.
func (c *durableChannel) expect(ref entryRef) {
	if _, ok := c.delivered[ref]; ok {
		return
	}

	c.wanted[ref] = struct{}{}
	c.releaseCompletions()
}

// releaseCompletions queues the acknowledged prefix of the completion queue, in the engine's
// delivery order, stopping at the first completion whose ref is not acknowledged. Must be
// called with mu held.
func (c *durableChannel) releaseCompletions() {
	for len(c.completions) > 0 {
		resp := c.completions[0]
		ref := entryRef{branchId: resp.GetEntryCompleted().GetRef().GetBranchId(), nodeId: resp.GetEntryCompleted().GetRef().GetNodeId()}

		if _, ok := c.wanted[ref]; !ok {
			return
		}

		c.completions[0] = nil
		c.completions = c.completions[1:]
		delete(c.wanted, ref)
		delete(c.queued, ref)
		c.delivered[ref] = struct{}{}
		c.ready = append(c.ready, resp)
	}
}

// ExpectEntry implements the contract: the ref counts as acknowledged, so a completion held for
// it is queued now (behind any earlier completion still held for its own ack) and one that
// arrives later is queued as it comes.
func (c *durableChannel) ExpectEntry(branchId, nodeId int64) error {
	if c.isClosed() {
		return ErrChannelClosed
	}

	c.mu.Lock()

	if c.overflowed {
		c.mu.Unlock()
		return ErrChannelClosed
	}

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

// isOverflowed reports whether the channel failed on its retention limit.
func (c *durableChannel) isOverflowed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.overflowed
}

// Send stamps the invocation's task id and count on the request, as the durable protocol
// requires, before it is handed to the engine. It waits for the engine to take the request,
// bounded by ctx.
func (c *durableChannel) Send(ctx context.Context, req *v1contracts.DurableTaskRequest) error {
	if c.isClosed() || c.isOverflowed() {
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
// called or the channel failed on its retention limit, ErrSessionEnded when the engine ended
// the invocation on its own, and ctx's error when ctx ends first.
func (c *durableChannel) Recv(ctx context.Context) (*v1contracts.DurableTaskResponse, error) {
	for {
		c.mu.Lock()

		if c.overflowed {
			c.mu.Unlock()
			return nil, ErrChannelClosed
		}

		if len(c.ready) > 0 {
			resp := c.ready[0]
			c.ready[0] = nil
			c.ready = c.ready[1:]
			c.retained--
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
		case <-c.recvLoopDone:
		case <-timeout.C:
		}
	})

	return nil
}
