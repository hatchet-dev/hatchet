package streams

import (
	"context"
	"testing"
	"time"
)

func TestRegistryCancelAll(t *testing.T) {
	r := NewRegistry()

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())

	deregister1 := r.Register(cancel1)
	defer deregister1()
	deregister2 := r.Register(cancel2)
	defer deregister2()

	r.CancelAll()

	for _, ctx := range []context.Context{ctx1, ctx2} {
		select {
		case <-ctx.Done():
		case <-time.After(time.Second):
			t.Fatal("expected context to be cancelled by CancelAll")
		}
	}
}

func TestRegistryDeregister(t *testing.T) {
	r := NewRegistry()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	deregister := r.Register(cancel)
	deregister()

	r.CancelAll()

	select {
	case <-ctx.Done():
		t.Fatal("deregistered session should not be cancelled by CancelAll")
	default:
	}
}

func TestRegistryCancelAllResets(t *testing.T) {
	r := NewRegistry()

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	r.Register(cancel)
	r.CancelAll()

	if len(r.sessions) != 0 {
		t.Fatalf("expected empty registry after CancelAll, got %d sessions", len(r.sessions))
	}
}

func TestRegistryRegisterAfterCancelAll(t *testing.T) {
	r := NewRegistry()
	r.CancelAll()

	ctx, cancel := context.WithCancel(context.Background())
	deregister := r.Register(cancel)
	defer deregister()

	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("session registered after CancelAll should be cancelled immediately")
	}
}
