package dispatcher

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/services/shared/streams"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// fakeDurableTaskServer stubs the bidi stream; the embedded interface is left nil
// because the handler only uses Context and Recv on these test paths.
type fakeDurableTaskServer struct {
	contracts.V1Dispatcher_DurableTaskServer

	ctx  context.Context
	recv func() (*contracts.DurableTaskRequest, error)
}

func (f *fakeDurableTaskServer) Context() context.Context { return f.ctx }

func (f *fakeDurableTaskServer) Recv() (*contracts.DurableTaskRequest, error) { return f.recv() }

func newTestDispatcher() *DispatcherServiceImpl {
	l := zerolog.Nop()

	return &DispatcherServiceImpl{
		l:              &l,
		streamSessions: streams.NewRegistry(),
	}
}

func tenantContext() context.Context {
	//nolint:staticcheck // the handler reads the tenant with a plain string key
	return context.WithValue(context.Background(), "tenant", &sqlcv1.Tenant{ID: uuid.New()})
}

func runDurableTask(d *DispatcherServiceImpl, server contracts.V1Dispatcher_DurableTaskServer) <-chan error {
	done := make(chan error, 1)

	go func() {
		done <- d.DurableTask(server)
	}()

	return done
}

// A shutdown drain (CancelStreamSessions) must unblock the handler promptly and
// return nil even while a Recv is pending; this is the property the graceful
// shutdown work relies on.
func TestDurableTaskReturnsNilOnSessionCancel(t *testing.T) {
	d := newTestDispatcher()

	release := make(chan struct{})
	t.Cleanup(func() { close(release) })

	recvStarted := make(chan struct{})

	server := &fakeDurableTaskServer{
		ctx: tenantContext(),
		recv: func() (*contracts.DurableTaskRequest, error) {
			close(recvStarted)
			<-release
			return nil, io.EOF
		},
	}

	done := runDurableTask(d, server)

	// only cancel once Recv is pending, so the test pins the drain-while-blocked
	// behavior instead of passing via the registry's late-register immediate cancel
	select {
	case <-recvStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("Recv was never called")
	}

	d.CancelStreamSessions()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected nil error on session cancel, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("DurableTask did not return after session cancel; Recv is still pending")
	}
}

func TestDurableTaskSurfacesRecvError(t *testing.T) {
	d := newTestDispatcher()
	recvErr := errors.New("transport broke")

	server := &fakeDurableTaskServer{
		ctx:  tenantContext(),
		recv: func() (*contracts.DurableTaskRequest, error) { return nil, recvErr },
	}

	select {
	case err := <-runDurableTask(d, server):
		if !errors.Is(err, recvErr) {
			t.Fatalf("expected recv error to be surfaced, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("DurableTask did not return after recv error")
	}
}

// RegisterDurableTask gives operators the channel-based equivalent of the DurableTask
// stream: the task is registered up front so async responses route to the returned channel,
// and cancelling the context tears the session down (deregisters the invocation, closes the
// response channel, and makes further sends error rather than panic).
func TestRegisterDurableTaskBridgesResponsesAndCleansUp(t *testing.T) {
	d := newTestDispatcher()
	ctx, cancel := context.WithCancel(tenantContext())
	t.Cleanup(cancel)

	externalId := uuid.New()

	_, respCh, err := d.RegisterDurableTask(ctx, externalId)
	if err != nil {
		t.Fatalf("RegisterDurableTask returned error: %v", err)
	}

	inv, ok := d.durableInvocations.Load(externalId)
	if !ok {
		t.Fatal("expected invocation to be registered for externalId up front")
	}

	// An async response routed through the invocation (as the dispatcher does when a
	// wait-for is satisfied) must reach the response channel.
	want := &contracts.DurableTaskResponse{
		Message: &contracts.DurableTaskResponse_RegisterWorker{
			RegisterWorker: &contracts.DurableTaskResponseRegisterWorker{WorkerId: "w1"},
		},
	}

	sendErr := make(chan error, 1)
	go func() { sendErr <- inv.send(want) }()

	select {
	case got := <-respCh:
		if got != want {
			t.Fatalf("unexpected response delivered: %v", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("response was not delivered to the channel")
	}

	if err := <-sendErr; err != nil {
		t.Fatalf("send returned error: %v", err)
	}

	// Cancelling tears the session down.
	cancel()

	select {
	case _, open := <-respCh:
		if open {
			t.Fatal("expected respCh to be closed after cancel")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("respCh was not closed after cancel")
	}

	// The close happens after the deregister in the same teardown, so by now it's gone.
	if _, ok := d.durableInvocations.Load(externalId); ok {
		t.Fatal("expected invocation to be deregistered after cancel")
	}

	// Sends after teardown return an error rather than panicking on the closed channel.
	if err := inv.send(want); err == nil {
		t.Fatal("expected send after teardown to return an error")
	}
}

func TestDurableTaskReturnsNilOnEOF(t *testing.T) {
	d := newTestDispatcher()

	server := &fakeDurableTaskServer{
		ctx:  tenantContext(),
		recv: func() (*contracts.DurableTaskRequest, error) { return nil, io.EOF },
	}

	select {
	case err := <-runDurableTask(d, server):
		if err != nil {
			t.Fatalf("expected nil error on EOF, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("DurableTask did not return after EOF")
	}
}
