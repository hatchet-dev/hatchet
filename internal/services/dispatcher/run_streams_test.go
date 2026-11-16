package dispatcher

import (
	"context"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/services/shared/rpcstream"
)

// A channel-backed bidi stream ends, and closes its response channel, when the caller closes
// the request channel, the way a gRPC client's half-close ends the handler.
func TestRegisterBidiRunStreamEndsOnRequestClose(t *testing.T) {
	d := newTestDispatcher()

	cs, err := rpcstream.ServeBidi(tenantContext(), nil, func(ctx context.Context, receive func() (*contracts.ListenForDurableEventRequest, error), sender *rpcstream.Sender[contracts.DurableEvent]) error {
		return d.listenForDurableEvent(ctx, receive, sender)
	})
	if err != nil {
		t.Fatalf("ServeBidi returned error: %v", err)
	}

	close(cs.Requests)

	select {
	case _, open := <-cs.Responses:
		if open {
			t.Fatal("expected Responses to be closed once the request channel closed")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Responses was not closed after the request channel closed")
	}

	if err := cs.Err; err != nil {
		t.Fatalf("a half-closed stream ends without error, got %v", err)
	}
}

// Cancelling the stream's context ends the handler; an opening message of the wrong type is
// refused before anything runs.
func TestRegisterBidiRunStreamCancelAndBadFirst(t *testing.T) {
	d := newTestDispatcher()

	handler := func(ctx context.Context, receive func() (*contracts.ListenForDurableEventRequest, error), sender *rpcstream.Sender[contracts.DurableEvent]) error {
		return d.listenForDurableEvent(ctx, receive, sender)
	}

	var wrong proto.Message = &contracts.DurableEvent{}

	if _, err := rpcstream.ServeBidi(tenantContext(), wrong, handler); err == nil {
		t.Fatal("expected a wrongly typed opening message to be refused")
	}

	ctx, cancel := context.WithCancel(tenantContext())

	cs, err := rpcstream.ServeBidi(ctx, nil, handler)
	if err != nil {
		t.Fatalf("ServeBidi returned error: %v", err)
	}

	cancel()

	select {
	case _, open := <-cs.Responses:
		if open {
			t.Fatal("expected Responses to be closed after cancel")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Responses was not closed after cancel")
	}
}
