package v1

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
