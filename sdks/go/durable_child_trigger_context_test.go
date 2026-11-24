//go:build !e2e && !load && !rampup && !integration

package hatchet

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	v0Client "github.com/hatchet-dev/hatchet/pkg/client" //nolint:staticcheck
	pkgWorker "github.com/hatchet-dev/hatchet/pkg/worker"
)

const (
	triggerTestTaskID     = "parent-task"
	triggerTestChildRunID = "child-run"
)

// fakeDurableTaskStream is an in-memory durable task bidi stream: requests the
// listener sends land on sendCh, and responses pushed to recvCh are received.
type fakeDurableTaskStream struct {
	v1.V1Dispatcher_DurableTaskClient

	ctx    context.Context
	sendCh chan *v1.DurableTaskRequest
	recvCh chan *v1.DurableTaskResponse
}

func (s *fakeDurableTaskStream) Send(req *v1.DurableTaskRequest) error {
	select {
	case s.sendCh <- req:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}

func (s *fakeDurableTaskStream) Recv() (*v1.DurableTaskResponse, error) {
	select {
	case resp := <-s.recvCh:
		return resp, nil
	case <-s.ctx.Done():
		return nil, io.EOF
	}
}

// awaitTriggerRuns returns the next TriggerRuns request, skipping worker
// registration and status messages.
func (s *fakeDurableTaskStream) awaitTriggerRuns(t *testing.T) *v1.DurableTaskTriggerRunsRequest {
	t.Helper()

	timeout := time.After(time.Second)
	for {
		select {
		case req := <-s.sendCh:
			if tr := req.GetTriggerRuns(); tr != nil {
				return tr
			}
		case <-timeout:
			t.Fatal("timed out waiting for trigger_runs request")
			return nil
		}
	}
}

func (s *fakeDurableTaskStream) ack(tr *v1.DurableTaskTriggerRunsRequest) {
	entries := make([]*v1.DurableTaskRunAckEntry, len(tr.GetTriggerOpts()))
	for i := range entries {
		runID := triggerTestChildRunID
		if len(entries) > 1 {
			runID = fmt.Sprintf("%s-%d", triggerTestChildRunID, i)
		}
		entries[i] = &v1.DurableTaskRunAckEntry{NodeId: int64(i + 1), WorkflowRunExternalId: runID}
	}

	s.recvCh <- &v1.DurableTaskResponse{
		Message: &v1.DurableTaskResponse_TriggerRunsAck{
			TriggerRunsAck: &v1.DurableTaskEventTriggerRunsAckResponse{
				DurableTaskExternalId: tr.GetDurableTaskExternalId(),
				InvocationCount:       tr.GetInvocationCount(),
				RunEntries:            entries,
			},
		},
	}
}

// fakeDurableContext is the part of a durable task context that durable child
// spawning uses, backed by a real DurableTaskListener.
type fakeDurableContext struct {
	pkgWorker.HatchetContext

	ctx        context.Context
	listener   *v0Client.DurableTaskListener
	childIndex int
}

func (f *fakeDurableContext) Deadline() (time.Time, bool)       { return f.ctx.Deadline() }
func (f *fakeDurableContext) Done() <-chan struct{}             { return f.ctx.Done() }
func (f *fakeDurableContext) Err() error                        { return f.ctx.Err() }
func (f *fakeDurableContext) Value(key any) any                 { return f.ctx.Value(key) }
func (f *fakeDurableContext) GetContext() context.Context       { return f.ctx }
func (f *fakeDurableContext) StepRunId() string                 { return triggerTestTaskID }
func (f *fakeDurableContext) WorkflowRunId() string             { return "parent-run" }
func (f *fakeDurableContext) DurableTaskInvocationCount() int32 { return 1 }
func (f *fakeDurableContext) CurChildIndex() int                { return f.childIndex }
func (f *fakeDurableContext) IncChildIndex()                    { f.childIndex++ }

func (f *fakeDurableContext) DurableTaskListener() *v0Client.DurableTaskListener {
	return f.listener
}

func (f *fakeDurableContext) DurableEvictionSupported() bool { return true }

type triggerContextHarness struct {
	stream       *fakeDurableTaskStream
	hctx         *fakeDurableContext
	cancelParent context.CancelFunc
}

func newTriggerContextHarness(t *testing.T) *triggerContextHarness {
	t.Helper()

	streamCtx, cancelStream := context.WithCancel(context.Background())
	stream := &fakeDurableTaskStream{
		ctx:    streamCtx,
		sendCh: make(chan *v1.DurableTaskRequest, 16),
		recvCh: make(chan *v1.DurableTaskResponse, 16),
	}

	logger := zerolog.Nop()
	listener := v0Client.NewDurableTaskListener(
		"test-worker",
		func(context.Context) (v1.V1Dispatcher_DurableTaskClient, error) { return stream, nil },
		&logger,
	)
	listener.Start(streamCtx)
	t.Cleanup(func() {
		cancelStream()
		listener.Stop()
	})

	parentCtx, cancelParent := context.WithCancel(context.Background())
	t.Cleanup(cancelParent)

	return &triggerContextHarness{
		stream:       stream,
		hctx:         &fakeDurableContext{ctx: parentCtx, listener: listener},
		cancelParent: cancelParent,
	}
}

type runNoWaitResult struct {
	ref *WorkflowRunRef
	err error
}

func spawners() map[string]func(ctx context.Context, opts ...RunOptFunc) (*WorkflowRunRef, error) {
	client := &Client{legacyClient: &stubV0Client{}}
	wf := client.NewWorkflow("child")
	task := client.NewStandaloneTask("child-task", func(ctx Context, in any) (any, error) { return in, nil })

	return map[string]func(ctx context.Context, opts ...RunOptFunc) (*WorkflowRunRef, error){
		"Workflow.RunNoWait": func(ctx context.Context, opts ...RunOptFunc) (*WorkflowRunRef, error) {
			return wf.RunNoWait(ctx, map[string]any{}, opts...)
		},
		"StandaloneTask.RunNoWait": func(ctx context.Context, opts ...RunOptFunc) (*WorkflowRunRef, error) {
			return task.RunNoWait(ctx, map[string]any{}, opts...)
		},
		"Client.RunNoWait": func(ctx context.Context, opts ...RunOptFunc) (*WorkflowRunRef, error) {
			return client.RunNoWait(ctx, "child", map[string]any{}, opts...)
		},
	}
}

func startRunNoWait(
	h *triggerContextHarness,
	spawn func(ctx context.Context, opts ...RunOptFunc) (*WorkflowRunRef, error),
	opts ...RunOptFunc,
) <-chan runNoWaitResult {
	done := make(chan runNoWaitResult, 1)
	go func() {
		ref, err := spawn(h.hctx, opts...)
		done <- runNoWaitResult{ref: ref, err: err}
	}()
	return done
}

func awaitResult[T any](t *testing.T, ch <-chan T) T {
	t.Helper()

	select {
	case res := <-ch:
		return res
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for spawn to return")
		var zero T
		return zero
	}
}

func assertPending[T any](t *testing.T, ch <-chan T) {
	t.Helper()

	select {
	case <-ch:
		t.Fatal("spawn returned before its trigger was acknowledged")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestDurableChildTrigger_ParentCancelledBeforeAck_DefaultReturnsCanceled(t *testing.T) {
	for name, spawn := range spawners() {
		t.Run(name, func(t *testing.T) {
			h := newTriggerContextHarness(t)

			done := startRunNoWait(h, spawn)
			h.stream.awaitTriggerRuns(t)
			h.cancelParent()

			res := awaitResult(t, done)
			require.ErrorIs(t, res.err, context.Canceled)
			assert.Nil(t, res.ref)
		})
	}
}

func TestDurableChildTrigger_ParentCancelledBeforeAck_TriggerContextReturnsRunID(t *testing.T) {
	for name, spawn := range spawners() {
		t.Run(name, func(t *testing.T) {
			h := newTriggerContextHarness(t)

			ackCtx, cancelAck := context.WithTimeout(context.WithoutCancel(h.hctx.GetContext()), 10*time.Second)
			defer cancelAck()

			done := startRunNoWait(h, spawn, WithTriggerContext(ackCtx))
			tr := h.stream.awaitTriggerRuns(t)
			h.cancelParent()
			assertPending(t, done)
			h.stream.ack(tr)

			res := awaitResult(t, done)
			require.NoError(t, res.err)
			require.NotNil(t, res.ref)
			assert.Equal(t, triggerTestChildRunID, res.ref.RunId)

			// The result wait still follows the parent task's context.
			_, err := res.ref.Result()
			require.ErrorIs(t, err, context.Canceled)
		})
	}
}

func TestDurableChildTrigger_TriggerContextCanceledBeforeAck(t *testing.T) {
	for name, spawn := range spawners() {
		t.Run(name, func(t *testing.T) {
			h := newTriggerContextHarness(t)

			ackCtx, cancelAck := context.WithCancel(context.Background())
			defer cancelAck()

			done := startRunNoWait(h, spawn, WithTriggerContext(ackCtx))
			h.stream.awaitTriggerRuns(t)
			cancelAck()

			res := awaitResult(t, done)
			require.ErrorIs(t, res.err, context.Canceled)
			assert.Nil(t, res.ref)
			require.NoError(t, h.hctx.GetContext().Err())
		})
	}
}

type runManyResult struct {
	refs []WorkflowRunRef
	err  error
}

func runManySpawners() map[string]func(ctx context.Context, inputs []RunManyOpt) ([]WorkflowRunRef, error) {
	client := &Client{legacyClient: &stubV0Client{}}
	wf := client.NewWorkflow("child")

	return map[string]func(ctx context.Context, inputs []RunManyOpt) ([]WorkflowRunRef, error){
		"Workflow.RunMany": wf.RunMany,
		"Client.RunMany": func(ctx context.Context, inputs []RunManyOpt) ([]WorkflowRunRef, error) {
			return client.RunMany(ctx, "child", inputs)
		},
	}
}

func startRunMany(
	h *triggerContextHarness,
	spawn func(ctx context.Context, inputs []RunManyOpt) ([]WorkflowRunRef, error),
	inputs []RunManyOpt,
) <-chan runManyResult {
	done := make(chan runManyResult, 1)
	go func() {
		refs, err := spawn(h.hctx, inputs)
		done <- runManyResult{refs: refs, err: err}
	}()
	return done
}

func TestDurableChildTrigger_RunManyDefaultReturnsCanceled(t *testing.T) {
	for name, spawn := range runManySpawners() {
		t.Run(name, func(t *testing.T) {
			h := newTriggerContextHarness(t)

			done := startRunMany(h, spawn, []RunManyOpt{{Input: 1}, {Input: 2}})
			h.stream.awaitTriggerRuns(t)
			h.cancelParent()

			res := awaitResult(t, done)
			require.ErrorIs(t, res.err, context.Canceled)
			assert.Nil(t, res.refs)
		})
	}
}

func TestDurableChildTrigger_RunManyTriggerContextReturnsRunIDs(t *testing.T) {
	for name, spawn := range runManySpawners() {
		t.Run(name, func(t *testing.T) {
			h := newTriggerContextHarness(t)

			ackCtx, cancelAck := context.WithTimeout(context.WithoutCancel(h.hctx.GetContext()), 10*time.Second)
			defer cancelAck()

			// Items without the option share the batch's trigger context.
			done := startRunMany(h, spawn, []RunManyOpt{
				{Input: 1, Opts: []RunOptFunc{WithTriggerContext(ackCtx)}},
				{Input: 2},
			})
			tr := h.stream.awaitTriggerRuns(t)
			h.cancelParent()
			assertPending(t, done)
			h.stream.ack(tr)

			res := awaitResult(t, done)
			require.NoError(t, res.err)
			require.Len(t, res.refs, 2)
			assert.Equal(t, triggerTestChildRunID+"-0", res.refs[0].RunId)
			assert.Equal(t, triggerTestChildRunID+"-1", res.refs[1].RunId)

			_, err := res.refs[0].Result()
			require.ErrorIs(t, err, context.Canceled)
		})
	}
}
