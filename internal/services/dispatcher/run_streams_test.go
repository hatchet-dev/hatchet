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

	reqCh, respCh, err := registerBidiRunStream(tenantContext(), nil, func(ctx context.Context, receive func() (*contracts.ListenForDurableEventRequest, error), sender *rpcstream.Sender[contracts.DurableEvent]) error {
		return d.listenForDurableEvent(ctx, receive, sender)
	})
	if err != nil {
		t.Fatalf("registerBidiRunStream returned error: %v", err)
	}

	close(reqCh)

	select {
	case _, open := <-respCh:
		if open {
			t.Fatal("expected respCh to be closed once the request channel closed")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("respCh was not closed after the request channel closed")
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

	if _, _, err := registerBidiRunStream(tenantContext(), wrong, handler); err == nil {
		t.Fatal("expected a wrongly typed opening message to be refused")
	}

	ctx, cancel := context.WithCancel(tenantContext())

	_, respCh, err := registerBidiRunStream(ctx, nil, handler)
	if err != nil {
		t.Fatalf("registerBidiRunStream returned error: %v", err)
	}

	cancel()

	select {
	case _, open := <-respCh:
		if open {
			t.Fatal("expected respCh to be closed after cancel")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("respCh was not closed after cancel")
	}
}
