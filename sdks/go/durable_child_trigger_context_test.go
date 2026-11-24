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
func (f *fakeDurableContext) WorkflowRunId() string             { return "parent-workflow-run" }
func (f *fakeDurableContext) DurableTaskInvocationCount() int32 { return 1 }
func (f *fakeDurableContext) CurChildIndex() int                { return f.childIndex }
func (f *fakeDurableContext) IncChildIndex()                    { f.childIndex++ }
func (f *fakeDurableContext) NextChildIndex() int {
	idx := f.childIndex
	f.childIndex++
	return idx
}
func (f *fakeDurableContext) DurableTaskListener() *v0Client.DurableTaskListener { return f.listener }
func (f *fakeDurableContext) DurableEvictionSupported() bool                     { return true }

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
		"worker-1",
		func(context.Context) (v1.V1Dispatcher_DurableTaskClient, error) { return stream, nil },
		&logger,
	)
	listener.Start(context.Background())
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

func awaitResult[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for spawn to return")
		var zero T
		return zero
	}
}

func TestDurableChildTrigger_TriggerContextReturnsRunID(t *testing.T) {
	for name, spawn := range spawners() {
		t.Run(name, func(t *testing.T) {
			h := newTriggerContextHarness(t)

			ackCtx, cancelAck := context.WithTimeout(context.WithoutCancel(h.hctx.GetContext()), 10*time.Second)
			defer cancelAck()

			done := make(chan runNoWaitResult, 1)
			go func() {
				ref, err := spawn(h.hctx, WithTriggerContext(ackCtx))
				done <- runNoWaitResult{ref: ref, err: err}
			}()

			tr := h.stream.awaitTriggerRuns(t)
			h.cancelParent()
			h.stream.ack(tr)

			res := awaitResult(t, done)
			require.NoError(t, res.err)
			require.NotNil(t, res.ref)
			assert.Equal(t, triggerTestChildRunID, res.ref.RunId)

			_, err := res.ref.Result()
			require.ErrorIs(t, err, context.Canceled)
		})
	}
}

func TestDurableChildTrigger_TriggerContextCanceledBeforeAck(t *testing.T) {
	h := newTriggerContextHarness(t)
	spawn := spawners()["Workflow.RunNoWait"]

	ackCtx, cancelAck := context.WithCancel(context.Background())
	defer cancelAck()

	done := make(chan runNoWaitResult, 1)
	go func() {
		ref, err := spawn(h.hctx, WithTriggerContext(ackCtx))
		done <- runNoWaitResult{ref: ref, err: err}
	}()

	h.stream.awaitTriggerRuns(t)
	cancelAck()

	res := awaitResult(t, done)
	require.ErrorIs(t, res.err, context.Canceled)
	assert.Nil(t, res.ref)
	require.NoError(t, h.hctx.GetContext().Err())
}

type runManyResult struct {
	refs []WorkflowRunRef
	err  error
}

func TestDurableChildTrigger_RunManyTriggerContext(t *testing.T) {
	client := &Client{legacyClient: &stubV0Client{}}
	wf := client.NewWorkflow("child")

	runManySpawners := map[string]func(ctx context.Context, inputs []RunManyOpt) ([]WorkflowRunRef, error){
		"Workflow.RunMany": wf.RunMany,
		"Client.RunMany": func(ctx context.Context, inputs []RunManyOpt) ([]WorkflowRunRef, error) {
			return client.RunMany(ctx, "child", inputs)
		},
	}

	for name, spawn := range runManySpawners {
		t.Run(name, func(t *testing.T) {
			h := newTriggerContextHarness(t)

			ackCtx, cancelAck := context.WithTimeout(context.WithoutCancel(h.hctx.GetContext()), 10*time.Second)
			defer cancelAck()

			// A second cancelled context verifies the first non-nil trigger context governs the batch.
			ignoredCtx, cancelIgnored := context.WithCancel(context.Background())
			cancelIgnored()

			done := make(chan runManyResult, 1)
			go func() {
				refs, err := spawn(h.hctx, []RunManyOpt{
					{Input: 1, Opts: []RunOptFunc{WithTriggerContext(ackCtx)}},
					{Input: 2, Opts: []RunOptFunc{WithTriggerContext(ignoredCtx)}},
				})
				done <- runManyResult{refs: refs, err: err}
			}()

			tr := h.stream.awaitTriggerRuns(t)
			h.cancelParent()
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
