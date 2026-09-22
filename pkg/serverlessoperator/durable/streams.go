package durable

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"connectrpc.com/connect"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/contract"
)

// defaultMaxStreams is the cap on the engine streams one socket may hold open at once.
const defaultMaxStreams = 16

// StreamOpener opens the engine streams a socket asks for. operator.Session satisfies it; the
// core wraps the session so SubscribeToWorkflowRuns goes through its per-tenant multiplexer.
type StreamOpener interface {
	OpenRunStream(ctx context.Context, kind operator.RunStreamKind, first proto.Message) (operator.RunStream, error)
}

// unmarshalStream is the decoder for endpoint stream messages: unknown fields are ignored, as
// for every other endpoint message.
var unmarshalStream = protojson.UnmarshalOptions{DiscardUnknown: true}

// relayStream is one engine stream open on the socket, keyed by the endpoint-chosen id.
type relayStream struct {
	rs     operator.RunStream
	id     string
	kind   operator.RunStreamKind
	ctx    context.Context
	cancel context.CancelFunc
	// endpointClosed records that the endpoint sent stream_close, so the reader's exit needs
	// no stream_close back.
	endpointClosed bool
}

// streamTable is the socket's open streams. Frames from the endpoint are handled on the read
// loop, one at a time; each open stream has a reader goroutine of its own that forwards the
// engine's messages to the send queue, so the table is locked around every change.
type streamTable struct {
	r    *relay
	open map[string]*relayStream
	wg   sync.WaitGroup
	mu   sync.Mutex
	// closed refuses opens once the socket is ending.
	closed bool
}

func newStreamTable(r *relay) *streamTable {
	return &streamTable{r: r, open: map[string]*relayStream{}}
}

// allowRun is the run filter: whether the task on this socket may observe the run. The socket
// is authorized for one task run of the tenant, and the operator's engine calls carry the
// operator's own credentials, so today the task may observe any run of its tenant. When
// tokens become namespace-scoped this is where the namespace of the run is checked, without a
// change to the frames.
func (r *relay) allowRun(tenantId, namespace, runId string) bool {
	_, _, _ = tenantId, namespace, runId
	return true
}

// runsOf lists the runs a message names, for allowRun: the workflow run of a subscription,
// the task of a durable event listener. A message naming no run (an additional-metadata
// subscription spans runs) yields none and is admitted by the tenant-wide filter.
func runsOf(msg proto.Message) []string {
	switch m := msg.(type) {
	case *contracts.SubscribeToWorkflowRunsRequest:
		return []string{m.GetWorkflowRunId()}
	case *contracts.SubscribeToWorkflowEventsRequest:
		if m.WorkflowRunId != nil {
			return []string{m.GetWorkflowRunId()}
		}

		return nil
	case *v1.ListenForDurableEventRequest:
		return []string{m.GetTaskId()}
	default:
		return nil
	}
}

// decodeStreamMessage parses a protojson request of the kind and checks the runs it names
// against the run filter. The error carries the connect code the stream is closed with.
func (r *relay) decodeStreamMessage(kind operator.RunStreamKind, raw string) (proto.Message, *connect.Error) {
	msg := kind.NewRequest()

	if err := unmarshalStream.Unmarshal([]byte(raw), msg); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("request is not a %s request: %v", kind, err))
	}

	for _, runId := range runsOf(msg) {
		if !r.allowRun(r.p.Action.GetTenantId(), r.p.Namespace, runId) {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("run %s is not observable from this task", runId))
		}
	}

	return msg, nil
}

// handleStreamOpen opens the engine stream a stream_open frame asks for. A frame the contract
// forbids (no id, an id already open) ends the socket; anything the stream layer can answer
// (a procedure not allowed, a request that does not decode, the cap, an engine refusal) is
// answered with a stream_close carrying a connect code and leaves the socket up.
func (t *streamTable) handleStreamOpen(f *v1.ServerlessStreamOpen) bool {
	if f.GetId() == "" {
		t.r.violation(CloseForbiddenMessage, "endpoint sent stream_open without an id")
		return false
	}

	kind, ok := operator.RunStreamKindOf(f.GetProcedure())

	if !ok {
		t.closeStream(f.GetId(), connect.NewError(connect.CodeUnimplemented, fmt.Errorf("procedure %q is not served over the socket", f.GetProcedure())))
		return true
	}

	first, cerr := t.r.decodeStreamMessage(kind, f.GetRequest())

	if cerr != nil {
		t.closeStream(f.GetId(), cerr)
		return true
	}

	if t.r.p.Streams == nil {
		t.closeStream(f.GetId(), connect.NewError(connect.CodeUnimplemented, errors.New("this socket cannot open engine streams")))
		return true
	}

	t.mu.Lock()

	if t.closed {
		t.mu.Unlock()
		return false
	}

	if _, dup := t.open[f.GetId()]; dup {
		t.mu.Unlock()
		t.r.violation(CloseForbiddenMessage, fmt.Sprintf("endpoint sent stream_open for stream %q, which is already open", f.GetId()))

		return false
	}

	if len(t.open) >= t.r.p.MaxStreams {
		t.mu.Unlock()
		t.closeStream(f.GetId(), connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("the socket already holds %d streams, the limit", t.r.p.MaxStreams)))

		return true
	}

	// The slot is taken before the open so two opens in flight cannot both pass the cap; the
	// open itself runs outside the lock.
	sctx, cancel := context.WithCancel(t.r.ctx)
	s := &relayStream{id: f.GetId(), kind: kind, ctx: sctx, cancel: cancel}
	t.open[s.id] = s
	t.mu.Unlock()

	rs, err := t.r.p.Streams.OpenRunStream(sctx, kind, first)

	if err != nil {
		t.remove(s)
		cancel()
		t.closeStream(s.id, streamError(err))

		return true
	}

	t.mu.Lock()

	if t.closed {
		// the socket ended during the open; the stream is closed with the others
		t.mu.Unlock()
		_ = rs.Close()

		return false
	}

	s.rs = rs
	t.wg.Add(1)
	t.mu.Unlock()

	go t.pump(s)

	return true
}

// handleStreamMessage forwards a message from the endpoint on an open bidi stream.
func (t *streamTable) handleStreamMessage(f *v1.ServerlessStreamMessage) bool {
	s := t.lookup(f.GetId())

	if s == nil {
		// a message for a stream the operator already closed is expected to cross the close
		t.r.l.Debug().Str("task_id", t.r.p.TaskId).Str("stream_id", f.GetId()).Msg("dropping stream_message for a stream that is not open")
		return true
	}

	if !s.kind.Bidi() {
		t.endStream(s, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%s takes no message after the request", s.kind)))
		return true
	}

	msg, cerr := t.r.decodeStreamMessage(s.kind, f.GetMessage())

	if cerr != nil {
		t.endStream(s, cerr)
		return true
	}

	if err := s.rs.Send(s.ctx, msg); err != nil {
		t.endStream(s, streamError(err))
	}

	return true
}

// handleStreamClose ends a stream the endpoint is done with. No frame goes back: the endpoint
// already forgot the id.
func (t *streamTable) handleStreamClose(f *v1.ServerlessStreamClose) bool {
	s := t.lookup(f.GetId())

	if s == nil {
		return true
	}

	t.mu.Lock()
	s.endpointClosed = true
	t.mu.Unlock()

	t.remove(s)
	s.cancel()
	_ = s.rs.Close()

	return true
}

func (t *streamTable) lookup(id string) *relayStream {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.open[id]
}

// remove drops the stream from the table; it reports whether it was still there.
func (t *streamTable) remove(s *relayStream) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.open[s.id] != s {
		return false
	}

	delete(t.open, s.id)

	return true
}

// endStream closes a stream on the operator's side and tells the endpoint why.
func (t *streamTable) endStream(s *relayStream, cerr *connect.Error) {
	if !t.remove(s) {
		return
	}

	s.cancel()
	_ = s.rs.Close()
	t.closeStream(s.id, cerr)
}

// closeStream queues the stream_close frame for a stream id. A nil error is the engine
// finishing the stream (code 0).
func (t *streamTable) closeStream(id string, cerr *connect.Error) {
	closeFrame := &v1.ServerlessStreamClose{Id: id}

	if cerr != nil {
		closeFrame.Code = int32(cerr.Code()) // nolint:gosec // connect codes are 1 to 16
		closeFrame.Message = cerr.Message()
	}

	frame, err := contract.MarshalFrame(&v1.ServerlessDurableFrame{
		Frame: &v1.ServerlessDurableFrame_StreamClose{StreamClose: closeFrame},
	})

	if err != nil {
		t.r.linkFailed(err)
		return
	}

	t.r.enqueue(frame)
}

// pump forwards the engine's messages on one stream to the endpoint until the stream ends,
// then reports the end unless the endpoint closed the stream itself or the socket is ending.
func (t *streamTable) pump(s *relayStream) {
	defer t.wg.Done()

	var end error

	for {
		msg, err := s.rs.Recv(s.ctx)

		if err != nil {
			end = err
			break
		}

		raw, err := protojson.Marshal(msg)

		if err != nil {
			end = err
			break
		}

		frame, err := contract.MarshalFrame(&v1.ServerlessDurableFrame{
			Frame: &v1.ServerlessDurableFrame_StreamMessage{StreamMessage: &v1.ServerlessStreamMessage{Id: s.id, Message: string(raw)}},
		})

		if err != nil {
			end = err
			break
		}

		if !t.r.enqueue(frame) {
			return
		}
	}

	if !t.remove(s) {
		// closed by the endpoint or by the socket's teardown
		return
	}

	s.cancel()
	_ = s.rs.Close()

	if t.r.stopped() {
		return
	}

	if errors.Is(end, operator.ErrStreamEnded) {
		t.closeStream(s.id, nil)
		return
	}

	t.closeStream(s.id, streamError(end))
}

// closeAll ends every open stream when the socket ends and waits for their readers.
func (t *streamTable) closeAll() {
	t.mu.Lock()
	t.closed = true
	streams := make([]*relayStream, 0, len(t.open))

	for _, s := range t.open {
		streams = append(streams, s)
	}

	t.open = map[string]*relayStream{}
	t.mu.Unlock()

	for _, s := range streams {
		s.cancel()

		if s.rs != nil {
			_ = s.rs.Close()
		}
	}

	t.wg.Wait()
}

// streamError maps an engine-side error to the connect error the endpoint is told about. A
// connect error keeps its code, a gRPC status its code (the two share numbering), a closed
// channel or session is unavailable, anything else unknown.
func streamError(err error) *connect.Error {
	var cerr *connect.Error

	if errors.As(err, &cerr) {
		return cerr
	}

	if st, ok := status.FromError(err); ok && st.Code() != codes.Unknown {
		return connect.NewError(connect.Code(st.Code()), errors.New(st.Message()))
	}

	if errors.Is(err, operator.ErrChannelClosed) || errors.Is(err, operator.ErrSessionClosed) || errors.Is(err, context.Canceled) {
		return connect.NewError(connect.CodeUnavailable, errors.New("the engine stream was closed"))
	}

	return connect.NewError(connect.CodeUnknown, err)
}
