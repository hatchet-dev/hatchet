package enginelink

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/link"
)

// errSessionEnded is returned by Recv when the engine tore the session down without Close
// being called, which the relay reports as a retryable link failure.
var errSessionEnded = errors.New("engine durable session ended")

// durableCloseDrainTimeout bounds how long Close waits for the engine to close the response
// side after the session is cancelled.
const durableCloseDrainTimeout = 5 * time.Second

// durableChannel is one durable invocation's pipe over the dispatcher's channel-backed
// session (the in-engine equivalent of the DurableTask stream). The session lives until Close
// cancels it; the engine then deregisters the invocation and closes the response channel.
//
// Like grpclink, at most one ack-bearing request (memo, trigger_runs, wait_for,
// evict_invocation) may be in flight: the engine keys its pending state by (task,
// invocation), so a second one would clobber the first. The slot is released when the ack,
// or the error that replaces it, is read through Recv.
type durableChannel struct {
	reqCh  chan<- *v1.DurableTaskRequest
	respCh <-chan *v1.DurableTaskResponse
	ctx    context.Context
	cancel context.CancelFunc
	closed chan struct{}

	taskExternalId string
	invocation     int32

	closeOnce sync.Once
	mu        sync.Mutex
	inflight  bool
}

// openDurable registers the session and runs the register-worker handshake: the first request
// names the worker running the invocation, and the engine's ack is consumed here so Recv only
// ever returns invocation traffic. ctx bounds the handshake; the session itself is detached
// from it and ends on Close.
func openDurable(ctx context.Context, d Dispatcher, tenant *sqlcv1.Tenant, workerId uuid.UUID, taskId uuid.UUID, invocation int32) (link.DurableChannel, error) {
	sctx, cancel := context.WithCancel(withTenant(context.WithoutCancel(ctx), tenant))

	reqCh, respCh, err := d.RegisterDurableTask(sctx, taskId)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not register durable task %s: %w", taskId, err)
	}

	ch := &durableChannel{
		reqCh:          reqCh,
		respCh:         respCh,
		ctx:            sctx,
		cancel:         cancel,
		closed:         make(chan struct{}),
		taskExternalId: taskId.String(),
		invocation:     invocation,
	}

	register := &v1.DurableTaskRequest{
		Message: &v1.DurableTaskRequest_RegisterWorker{
			RegisterWorker: &v1.DurableTaskRequestRegisterWorker{WorkerId: workerId.String()},
		},
	}

	select {
	case reqCh <- register:
	case <-ctx.Done():
		_ = ch.Close()
		return nil, fmt.Errorf("durable session interrupted before the register worker request was sent: %w", ctx.Err())
	}

	select {
	case resp, ok := <-respCh:
		if !ok {
			_ = ch.Close()
			return nil, errors.New("durable session closed while waiting for the register worker ack")
		}

		if resp.GetRegisterWorker() == nil {
			_ = ch.Close()

			if e := resp.GetError(); e != nil {
				return nil, fmt.Errorf("engine rejected the register worker request: %s", e.ErrorMessage)
			}

			return nil, fmt.Errorf("unexpected first durable response %T, want the register worker ack", resp.GetMessage())
		}
	case <-ctx.Done():
		_ = ch.Close()
		return nil, fmt.Errorf("durable session interrupted waiting for the register worker ack: %w", ctx.Err())
	}

	return ch, nil
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

// Send implements link.DurableChannel. The invocation's task id and count are stamped on the
// request, as the websocket protocol requires, before it is handed to the engine. It blocks
// while the engine is busy with the previous request.
func (c *durableChannel) Send(req *v1.DurableTaskRequest) error {
	if c.isClosed() {
		return link.ErrChannelClosed
	}

	acked := false

	switch m := req.GetMessage().(type) {
	case *v1.DurableTaskRequest_Memo:
		m.Memo.DurableTaskExternalId, m.Memo.InvocationCount = c.taskExternalId, c.invocation
		acked = true
	case *v1.DurableTaskRequest_TriggerRuns:
		m.TriggerRuns.DurableTaskExternalId, m.TriggerRuns.InvocationCount = c.taskExternalId, c.invocation
		acked = true
	case *v1.DurableTaskRequest_WaitFor:
		m.WaitFor.DurableTaskExternalId, m.WaitFor.InvocationCount = c.taskExternalId, c.invocation
		acked = true
	case *v1.DurableTaskRequest_EvictInvocation:
		m.EvictInvocation.DurableTaskExternalId, m.EvictInvocation.InvocationCount = c.taskExternalId, c.invocation
		acked = true
	case *v1.DurableTaskRequest_CompleteMemo:
		if m.CompleteMemo.Ref == nil {
			return errors.New("complete_memo requires a ref")
		}

		m.CompleteMemo.Ref.DurableTaskExternalId, m.CompleteMemo.Ref.InvocationCount = c.taskExternalId, c.invocation
	case *v1.DurableTaskRequest_RegisterWorker, *v1.DurableTaskRequest_WorkerStatus:
		return errors.New("register_worker and worker_status are link-internal")
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

		return link.ErrChannelClosed
	}
}

// Recv implements link.DurableChannel. It returns link.ErrChannelClosed once Close was
// called and errSessionEnded when the engine ended the session on its own.
func (c *durableChannel) Recv() (*v1.DurableTaskResponse, error) {
	select {
	case <-c.closed:
		return nil, link.ErrChannelClosed
	case resp, ok := <-c.respCh:
		if !ok {
			if c.isClosed() {
				return nil, link.ErrChannelClosed
			}

			return nil, errSessionEnded
		}

		switch resp.GetMessage().(type) {
		case *v1.DurableTaskResponse_MemoAck,
			*v1.DurableTaskResponse_TriggerRunsAck,
			*v1.DurableTaskResponse_WaitForAck,
			*v1.DurableTaskResponse_EvictionAck,
			*v1.DurableTaskResponse_Error:
			c.release()
		}

		return resp, nil
	}
}

// Close implements link.DurableChannel: it cancels the session, unblocks Send and Recv, and
// drains responses until the engine closes its side, so the invocation is deregistered
// before Close returns.
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
