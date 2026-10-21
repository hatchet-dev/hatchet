package operator

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// ErrStreamEnded is returned by RunStream.Recv once the engine has finished the stream on its
// own: the server stream sent its last message, or the engine hung the stream up. Recv after
// Close returns ErrChannelClosed.
var ErrStreamEnded = errors.New("operator: engine stream ended")

// RunStreamKind names one of the engine's run observation streams a session opens on behalf
// of its tenant. Every kind is one procedure; Request and Response are its message types.
type RunStreamKind int

const (
	// RunStreamWorkflowRuns is Dispatcher.SubscribeToWorkflowRuns, a bidi stream: every
	// SubscribeToWorkflowRunsRequest sent names a run whose terminal WorkflowRunEvent the
	// engine sends once, with the results of its tasks.
	RunStreamWorkflowRuns RunStreamKind = iota + 1

	// RunStreamWorkflowEvents is Dispatcher.SubscribeToWorkflowEvents, a server stream: the
	// SubscribeToWorkflowEventsRequest names one run, or an additional metadata pair, and the
	// engine sends every WorkflowEvent of the matching runs until the run finishes.
	RunStreamWorkflowEvents

	// RunStreamDurableEvents is V1Dispatcher.ListenForDurableEvent, a bidi stream: every
	// ListenForDurableEventRequest names a task and a signal key, and the engine sends the
	// DurableEvent that completes it.
	RunStreamDurableEvents
)

// runStreamKinds is every kind, keyed by procedure.
var runStreamKinds = map[string]RunStreamKind{
	"/Dispatcher/SubscribeToWorkflowRuns":    RunStreamWorkflowRuns,
	"/Dispatcher/SubscribeToWorkflowEvents":  RunStreamWorkflowEvents,
	"/v1.V1Dispatcher/ListenForDurableEvent": RunStreamDurableEvents,
}

// RunStreamKindOf resolves a fully qualified procedure name to its kind; ok is false for a
// procedure no session opens.
func RunStreamKindOf(procedure string) (kind RunStreamKind, ok bool) {
	kind, ok = runStreamKinds[procedure]
	return kind, ok
}

// Procedure is the kind's fully qualified procedure name.
func (k RunStreamKind) Procedure() string {
	for procedure, kind := range runStreamKinds {
		if kind == k {
			return procedure
		}
	}

	return fmt.Sprintf("RunStreamKind(%d)", int(k))
}

func (k RunStreamKind) String() string {
	return k.Procedure()
}

// Bidi reports whether the client sends messages after the first one.
func (k RunStreamKind) Bidi() bool {
	return k != RunStreamWorkflowEvents
}

// NewRequest returns an empty request message of the kind.
func (k RunStreamKind) NewRequest() proto.Message {
	switch k {
	case RunStreamWorkflowRuns:
		return &contracts.SubscribeToWorkflowRunsRequest{}
	case RunStreamWorkflowEvents:
		return &contracts.SubscribeToWorkflowEventsRequest{}
	case RunStreamDurableEvents:
		return &v1.ListenForDurableEventRequest{}
	default:
		return nil
	}
}

// NewResponse returns an empty response message of the kind.
func (k RunStreamKind) NewResponse() proto.Message {
	switch k {
	case RunStreamWorkflowRuns:
		return &contracts.WorkflowRunEvent{}
	case RunStreamWorkflowEvents:
		return &contracts.WorkflowEvent{}
	case RunStreamDurableEvents:
		return &v1.DurableEvent{}
	default:
		return nil
	}
}

// RunStream is one open engine stream. Messages are the kind's request and response types.
// Send is refused on a server stream (ErrNotSupported): its one request is the message the
// stream was opened with. Recv returns ErrStreamEnded once the engine finished the stream,
// ErrChannelClosed after Close, and the transport's error when the stream failed. Close ends
// the stream in both directions and may be called more than once.
type RunStream interface {
	Send(ctx context.Context, msg proto.Message) error
	Recv(ctx context.Context) (proto.Message, error)
	Close() error
}
