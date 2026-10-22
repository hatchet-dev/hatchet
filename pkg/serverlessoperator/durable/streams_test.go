//go:build !e2e && !load && !rampup && !integration

package durable

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator"
)

// fakeRunStream is one engine stream the fake opener handed out: Send records, Recv yields
// what the test pushes, Close is counted.
type fakeRunStream struct {
	kind   operator.RunStreamKind
	first  proto.Message
	sent   chan proto.Message
	recv   chan streamItem
	closed chan struct{}
	once   sync.Once
	closes int
	mu     sync.Mutex
}

type streamItem struct {
	msg proto.Message
	err error
}

func newFakeRunStream(kind operator.RunStreamKind, first proto.Message) *fakeRunStream {
	return &fakeRunStream{
		kind:   kind,
		first:  first,
		sent:   make(chan proto.Message, 16),
		recv:   make(chan streamItem, 512),
		closed: make(chan struct{}),
	}
}

func (f *fakeRunStream) Send(_ context.Context, msg proto.Message) error {
	select {
	case <-f.closed:
		return operator.ErrChannelClosed
	default:
	}

	f.sent <- msg

	return nil
}

func (f *fakeRunStream) Recv(ctx context.Context) (proto.Message, error) {
	select {
	case item := <-f.recv:
		return item.msg, item.err
	case <-f.closed:
		return nil, operator.ErrChannelClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (f *fakeRunStream) Close() error {
	f.mu.Lock()
	f.closes++
	f.mu.Unlock()

	f.once.Do(func() { close(f.closed) })

	return nil
}

func (f *fakeRunStream) isClosed() bool {
	select {
	case <-f.closed:
		return true
	default:
		return false
	}
}

func (f *fakeRunStream) push(msg proto.Message) { f.recv <- streamItem{msg: msg} }
func (f *fakeRunStream) end(err error)          { f.recv <- streamItem{err: err} }

// fakeOpener records every open and hands out fake streams; openErr refuses them.
type fakeOpener struct {
	openErr error
	streams []*fakeRunStream
	mu      sync.Mutex
}

func (o *fakeOpener) OpenRunStream(_ context.Context, kind operator.RunStreamKind, first proto.Message) (operator.RunStream, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.openErr != nil {
		return nil, o.openErr
	}

	s := newFakeRunStream(kind, first)
	o.streams = append(o.streams, s)

	return s, nil
}

// stream waits for the i-th stream to be opened.
func (o *fakeOpener) stream(t *testing.T, i int) *fakeRunStream {
	t.Helper()

	require.Eventually(t, func() bool {
		o.mu.Lock()
		defer o.mu.Unlock()

		return len(o.streams) > i
	}, eventually, time.Millisecond, "stream %d was never opened", i)

	o.mu.Lock()
	defer o.mu.Unlock()

	return o.streams[i]
}

func (o *fakeOpener) count() int {
	o.mu.Lock()
	defer o.mu.Unlock()

	return len(o.streams)
}

// socketParams are the params of a non-durable task's socket: no channel, a stream opener.
func socketParams(ep *fakeEndpoint, opener StreamOpener) Params {
	p := testParams(ep, nil)
	p.Action.DurableTaskInvocationCount = nil
	p.Invocation = 1
	p.Streams = opener

	return p
}

// readStreamFrame reads the next frame and requires it to be a stream frame with the id.
func readStreamFrame(t *testing.T, conn *websocket.Conn) *v1.ServerlessDurableFrame {
	t.Helper()

	frame := readFrame(t, conn)
	require.True(t, frame.GetStreamMessage() != nil || frame.GetStreamClose() != nil, "expected a stream frame, got %s", frame.String())

	return frame
}

func readStreamClose(t *testing.T, conn *websocket.Conn, id string) *v1.ServerlessStreamClose {
	t.Helper()

	closeFrame := readStreamFrame(t, conn).GetStreamClose()
	require.NotNil(t, closeFrame, "expected a stream_close frame")
	assert.Equal(t, id, closeFrame.GetId())

	return closeFrame
}

// A flagged task's socket gets the same first frame as a durable one, with invocation 1;
// the endpoint opens a child run stream, receives the engine's message, sees the stream
// finish with code 0 when the engine ends it, and completes with done.
func TestSocketStreamOpenReceiveAndEngineEnd(t *testing.T) {
	setT(t)

	opener := &fakeOpener{}

	ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, first *v1.ServerlessFirstFrame) {
		assert.Equal(t, int32(1), first.InvocationCount)
		assert.Nil(t, first.GetAction().DurableTaskInvocationCount)

		writeFrame(t, conn, `{"streamOpen":{"id":"s1","procedure":"/Dispatcher/SubscribeToWorkflowRuns","request":"{\"workflowRunId\":\"run-1\"}"}}`)

		rs := opener.stream(t, 0)
		assert.Equal(t, operator.RunStreamWorkflowRuns, rs.kind)
		assert.Equal(t, "run-1", rs.first.(*contracts.SubscribeToWorkflowRunsRequest).WorkflowRunId)

		output := "{}"
		rs.push(&contracts.WorkflowRunEvent{WorkflowRunId: "run-1", Results: []*contracts.StepRunResult{{TaskName: "child", Output: &output}}})

		msg := readStreamFrame(t, conn).GetStreamMessage()
		require.NotNil(t, msg)
		assert.Equal(t, "s1", msg.GetId())

		event := &contracts.WorkflowRunEvent{}
		require.NoError(t, unmarshalStream.Unmarshal([]byte(msg.GetMessage()), event))
		assert.Equal(t, "run-1", event.WorkflowRunId)
		assert.Equal(t, "child", event.Results[0].TaskName)

		rs.end(operator.ErrStreamEnded)

		closeFrame := readStreamClose(t, conn, "s1")
		assert.Equal(t, int32(0), closeFrame.GetCode())

		writeFrame(t, conn, `{"done":{"output":"{\"child\":\"ok\"}"}}`)
	})

	out := await(t, run(context.Background(), socketParams(ep, opener)))

	assert.Equal(t, KindCompleted, out.Kind)
	assert.JSONEq(t, `{"child":"ok"}`, string(out.Output))
	assert.Equal(t, CloseNormal, ep.closed(t))
	assert.True(t, opener.stream(t, 0).isClosed())
}

// Messages flow both ways on a bidi stream, and a stream_close from the endpoint tears the
// engine stream down without a frame back.
func TestSocketStreamMessageAndEndpointClose(t *testing.T) {
	setT(t)

	opener := &fakeOpener{}

	ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
		writeFrame(t, conn, `{"streamOpen":{"id":"d","procedure":"/v1.V1Dispatcher/ListenForDurableEvent","request":"{\"taskId\":\"t1\",\"signalKey\":\"k1\"}"}}`)

		rs := opener.stream(t, 0)
		assert.Equal(t, "k1", rs.first.(*v1.ListenForDurableEventRequest).SignalKey)

		writeFrame(t, conn, `{"streamMessage":{"id":"d","message":"{\"taskId\":\"t1\",\"signalKey\":\"k2\"}"}}`)

		select {
		case sent := <-rs.sent:
			assert.Equal(t, "k2", sent.(*v1.ListenForDurableEventRequest).SignalKey)
		case <-time.After(eventually):
			t.Fatal("the message never reached the engine stream")
		}

		writeFrame(t, conn, `{"streamClose":{"id":"d"}}`)

		require.Eventually(t, rs.isClosed, eventually, time.Millisecond, "the endpoint's close must close the engine stream")

		// A message for the closed stream is dropped, not answered.
		writeFrame(t, conn, `{"streamMessage":{"id":"d","message":"{}"}}`)
		writeFrame(t, conn, `{"done":{"output":"{}"}}`)
	})

	out := await(t, run(context.Background(), socketParams(ep, opener)))

	assert.Equal(t, KindCompleted, out.Kind)
	assert.Equal(t, CloseNormal, ep.closed(t))
}

// The operator closes a stream with the engine's connect code when the stream fails, when
// the message does not decode, and when a server stream is sent a second message.
func TestSocketStreamOperatorCloseCodes(t *testing.T) {
	setT(t)

	opener := &fakeOpener{}

	ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
		// An engine failure carries its code.
		writeFrame(t, conn, `{"streamOpen":{"id":"a","procedure":"/Dispatcher/SubscribeToWorkflowRuns","request":"{\"workflowRunId\":\"run-1\"}"}}`)
		opener.stream(t, 0).end(connect.NewError(connect.CodeNotFound, errors.New("no such run")))

		closeFrame := readStreamClose(t, conn, "a")
		assert.Equal(t, int32(connect.CodeNotFound), closeFrame.GetCode())
		assert.Equal(t, "not_found: the engine found no such resource", closeFrame.GetMessage(), "the engine's own text stays with the operator")

		// A message that does not decode closes the stream as invalid.
		writeFrame(t, conn, `{"streamOpen":{"id":"b","procedure":"/Dispatcher/SubscribeToWorkflowRuns","request":"{\"workflowRunId\":\"run-2\"}"}}`)
		opener.stream(t, 1)
		writeFrame(t, conn, `{"streamMessage":{"id":"b","message":"not json"}}`)

		closeFrame = readStreamClose(t, conn, "b")
		assert.Equal(t, int32(connect.CodeInvalidArgument), closeFrame.GetCode())
		assert.True(t, opener.stream(t, 1).isClosed())

		// A server stream takes no further message.
		writeFrame(t, conn, `{"streamOpen":{"id":"c","procedure":"/Dispatcher/SubscribeToWorkflowEvents","request":"{\"workflowRunId\":\"run-3\"}"}}`)
		opener.stream(t, 2)
		writeFrame(t, conn, `{"streamMessage":{"id":"c","message":"{}"}}`)

		closeFrame = readStreamClose(t, conn, "c")
		assert.Equal(t, int32(connect.CodeInvalidArgument), closeFrame.GetCode())

		// An open the engine refuses is closed with its code and holds no slot; a transport
		// diagnostic from the engine link is not what the endpoint is told.
		opener.mu.Lock()
		opener.openErr = status.Error(codes.Unavailable, "connection error: dial tcp 10.1.2.3:7070: connect: connection refused")
		opener.mu.Unlock()

		writeFrame(t, conn, `{"streamOpen":{"id":"e","procedure":"/Dispatcher/SubscribeToWorkflowRuns","request":"{\"workflowRunId\":\"run-4\"}"}}`)

		closeFrame = readStreamClose(t, conn, "e")
		assert.Equal(t, int32(connect.CodeUnavailable), closeFrame.GetCode())
		assert.Equal(t, "unavailable: the engine stream was closed", closeFrame.GetMessage())
		assert.NotContains(t, closeFrame.GetMessage(), "10.1.2.3")

		writeFrame(t, conn, `{"done":{"output":"{}"}}`)
	})

	out := await(t, run(context.Background(), socketParams(ep, opener)))

	assert.Equal(t, KindCompleted, out.Kind)
	assert.Equal(t, CloseNormal, ep.closed(t))
}

// A procedure outside the allow-list, a request that does not decode and a socket without
// an opener are answered with a stream_close before any engine call; the socket stays up.
func TestSocketStreamOpenRefused(t *testing.T) {
	setT(t)

	opener := &fakeOpener{}

	ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
		writeFrame(t, conn, `{"streamOpen":{"id":"x","procedure":"/v1.V1Dispatcher/DurableTask","request":"{}"}}`)

		closeFrame := readStreamClose(t, conn, "x")
		assert.Equal(t, int32(connect.CodeUnimplemented), closeFrame.GetCode())

		writeFrame(t, conn, `{"streamOpen":{"id":"y","procedure":"/Dispatcher/SubscribeToWorkflowRuns","request":"nope"}}`)

		closeFrame = readStreamClose(t, conn, "y")
		assert.Equal(t, int32(connect.CodeInvalidArgument), closeFrame.GetCode())

		writeFrame(t, conn, `{"done":{"output":"{}"}}`)
	})

	out := await(t, run(context.Background(), socketParams(ep, opener)))

	assert.Equal(t, KindCompleted, out.Kind)
	assert.Equal(t, 0, opener.count(), "nothing was opened on the engine")

	// Without an opener every open is unimplemented.
	setT(t)

	ep = newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
		writeFrame(t, conn, `{"streamOpen":{"id":"z","procedure":"/Dispatcher/SubscribeToWorkflowRuns","request":"{}"}}`)

		closeFrame := readStreamClose(t, conn, "z")
		assert.Equal(t, int32(connect.CodeUnimplemented), closeFrame.GetCode())

		writeFrame(t, conn, `{"done":{"output":"{}"}}`)
	})

	out = await(t, run(context.Background(), socketParams(ep, nil)))
	assert.Equal(t, KindCompleted, out.Kind)
}

// Past the cap an open is refused with resource exhausted; a closed stream frees its slot.
func TestSocketStreamCap(t *testing.T) {
	setT(t)

	opener := &fakeOpener{}

	ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
		for i := 0; i < 2; i++ {
			writeFrame(t, conn, fmt.Sprintf(`{"streamOpen":{"id":"s%d","procedure":"/Dispatcher/SubscribeToWorkflowRuns","request":"{\"workflowRunId\":\"run-%d\"}"}}`, i, i))
			opener.stream(t, i)
		}

		writeFrame(t, conn, `{"streamOpen":{"id":"s2","procedure":"/Dispatcher/SubscribeToWorkflowRuns","request":"{\"workflowRunId\":\"run-2\"}"}}`)

		closeFrame := readStreamClose(t, conn, "s2")
		assert.Equal(t, int32(connect.CodeResourceExhausted), closeFrame.GetCode())
		assert.Equal(t, 2, opener.count())

		writeFrame(t, conn, `{"streamClose":{"id":"s0"}}`)
		require.Eventually(t, opener.stream(t, 0).isClosed, eventually, time.Millisecond)

		writeFrame(t, conn, `{"streamOpen":{"id":"s3","procedure":"/Dispatcher/SubscribeToWorkflowRuns","request":"{\"workflowRunId\":\"run-3\"}"}}`)
		opener.stream(t, 2)

		writeFrame(t, conn, `{"done":{"output":"{}"}}`)
	})

	p := socketParams(ep, opener)
	p.MaxStreams = 2

	out := await(t, run(context.Background(), p))
	assert.Equal(t, KindCompleted, out.Kind)
}

// Streams open when the socket ends are closed with it, whichever way it ends.
func TestSocketCloseTearsStreamsDown(t *testing.T) {
	setT(t)

	opener := &fakeOpener{}

	ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
		writeFrame(t, conn, `{"streamOpen":{"id":"a","procedure":"/Dispatcher/SubscribeToWorkflowRuns","request":"{\"workflowRunId\":\"run-1\"}"}}`)
		writeFrame(t, conn, `{"streamOpen":{"id":"b","procedure":"/v1.V1Dispatcher/ListenForDurableEvent","request":"{\"taskId\":\"t\",\"signalKey\":\"k\"}"}}`)
		opener.stream(t, 1)

		// The endpoint crashes: the socket ends without done.
		_ = conn.NetConn().Close()
	})

	out := await(t, run(context.Background(), socketParams(ep, opener)))

	assert.Equal(t, KindFailed, out.Kind)
	assert.True(t, out.Retry)
	assert.True(t, opener.stream(t, 0).isClosed())
	assert.True(t, opener.stream(t, 1).isClosed())

	// A done frame also ends what is still open.
	setT(t)

	opener = &fakeOpener{}

	ep = newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
		writeFrame(t, conn, `{"streamOpen":{"id":"a","procedure":"/Dispatcher/SubscribeToWorkflowRuns","request":"{\"workflowRunId\":\"run-1\"}"}}`)
		opener.stream(t, 0)
		writeFrame(t, conn, `{"done":{"output":"{}"}}`)
	})

	out = await(t, run(context.Background(), socketParams(ep, opener)))
	assert.Equal(t, KindCompleted, out.Kind)
	assert.True(t, opener.stream(t, 0).isClosed())
}

// Stream frames count against the queued-bytes budget: an endpoint that stops reading
// while the engine keeps sending is closed with backpressure.
func TestSocketStreamFramesCountAgainstQueuedBytes(t *testing.T) {
	setT(t)

	opener := &fakeOpener{}
	gate := make(chan struct{})

	ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
		writeFrame(t, conn, `{"streamOpen":{"id":"a","procedure":"/Dispatcher/SubscribeToWorkflowRuns","request":"{\"workflowRunId\":\"run-1\"}"}}`)

		rs := opener.stream(t, 0)
		payload := strings.Repeat("x", 1024)

		for i := 0; i < 8; i++ {
			rs.push(&contracts.WorkflowRunEvent{WorkflowRunId: "run-1", Results: []*contracts.StepRunResult{{Output: &payload}}})
		}
	})

	p := socketParams(ep, opener)
	p.MaxQueuedBytes = 4 * 1024
	p.writeGate = gate

	out := await(t, run(context.Background(), p))

	assert.Equal(t, KindFailed, out.Kind)
	assert.Contains(t, out.Error, "bytes behind")
	assert.Equal(t, CloseBackpressure, ep.closed(t))
	close(gate)
}

// A durable request on the socket of a non-durable task is a protocol violation; a durable
// socket keeps its request frames and can open streams beside them.
func TestSocketRequestFrameRules(t *testing.T) {
	setT(t)

	ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
		writeFrame(t, conn, `{"id":1,"request":{"memo":{"key":"aw=="}}}`)
	})

	out := await(t, run(context.Background(), socketParams(ep, &fakeOpener{})))

	assert.Equal(t, KindFailed, out.Kind)
	assert.False(t, out.Retry)
	assert.Contains(t, out.Error, "non-durable task")
	assert.Equal(t, CloseForbiddenMessage, ep.closed(t))

	setT(t)

	opener := &fakeOpener{}

	ep = newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
		writeFrame(t, conn, `{"streamOpen":{"id":"a","procedure":"/Dispatcher/SubscribeToWorkflowRuns","request":"{\"workflowRunId\":\"run-1\"}"}}`)
		opener.stream(t, 0)
		writeFrame(t, conn, `{"id":1,"request":{"memo":{"key":"aw=="}}}`)
		require.NotNil(t, readResponse(t, conn).GetMemoAck())
		writeFrame(t, conn, `{"done":{"output":"{}"}}`)
	})

	ch := newFakeChannel()
	p := testParams(ep, ch)
	p.Streams = opener

	pending := run(context.Background(), p)

	ch.next(t)
	ch.reply(&v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_MemoAck{MemoAck: &v1.DurableTaskEventMemoAckResponse{}}})

	assert.Equal(t, KindCompleted, await(t, pending).Kind)
	assert.True(t, opener.stream(t, 0).isClosed())
}

// The endpoint is told an engine-side error's code with the fixed message for that code: the
// error's own text, whatever its type, never reaches the stream_close frame.
func TestPublicStreamErrorRedactsEngineText(t *testing.T) {
	const secret = "dial tcp 10.1.2.3:7070"

	for name, tc := range map[string]struct {
		err  error
		code connect.Code
	}{
		"connect error":        {err: connect.NewError(connect.CodeNotFound, errors.New(secret)), code: connect.CodeNotFound},
		"wrapped connect":      {err: fmt.Errorf("open: %w", connect.NewError(connect.CodePermissionDenied, errors.New(secret))), code: connect.CodePermissionDenied},
		"grpc status":          {err: status.Error(codes.Unavailable, secret), code: connect.CodeUnavailable},
		"grpc unknown":         {err: status.Error(codes.Unknown, secret), code: connect.CodeUnknown},
		"grpc invalid":         {err: status.Error(codes.Code(99), secret), code: connect.CodeUnknown},
		"closed channel":       {err: fmt.Errorf("%w: %s", operator.ErrChannelClosed, secret), code: connect.CodeUnavailable},
		"closed session":       {err: operator.ErrSessionClosed, code: connect.CodeUnavailable},
		"canceled":             {err: context.Canceled, code: connect.CodeUnavailable},
		"untyped":              {err: errors.New(secret), code: connect.CodeUnknown},
		"connect invalid code": {err: connect.NewError(connect.Code(99), errors.New(secret)), code: connect.CodeUnknown},
	} {
		t.Run(name, func(t *testing.T) {
			cerr := publicStreamError(tc.err)

			assert.Equal(t, tc.code, cerr.Code())
			assert.Equal(t, tc.code.String()+": "+publicStreamMessages[tc.code], cerr.Message())
			assert.NotContains(t, cerr.Message(), secret)
		})
	}

	for code := connect.CodeCanceled; code <= connect.CodeUnauthenticated; code++ {
		assert.Contains(t, publicStreamMessages, code, "every connect code has a public message")
	}
}

// A topic subscription (V1Streams.Subscribe) is a server stream over the socket: its request
// decodes with no run to check, the engine's StreamMessage frames reach the endpoint under the
// stream id, a second message from the endpoint is refused, and the engine refusing the
// subscription, as it does for a tenant without the durable streams entitlement, is told as
// code 7 whether the refusal ends the stream or the open itself.
func TestSocketDurableStreamSubscribe(t *testing.T) {
	setT(t)

	opener := &fakeOpener{}

	ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
		writeFrame(t, conn, `{"streamOpen":{"id":"t","procedure":"/v1.V1Streams/Subscribe","request":"{\"namespace\":\"ns\",\"topic\":\"orders\"}"}}`)

		rs := opener.stream(t, 0)
		assert.Equal(t, operator.RunStreamDurableStreams, rs.kind)

		req := rs.first.(*v1.SubscribeStreamRequest)
		assert.Equal(t, "ns", req.Namespace)
		assert.Equal(t, "orders", req.Topic)
		assert.Nil(t, req.Cursor)

		rs.push(&v1.StreamMessage{Cursor: "c2", Entries: []*v1.StreamEntry{
			{Payload: []byte(`{"n":1}`), Cursor: "c1"},
			{Payload: []byte(`{"n":2}`), Cursor: "c2"},
		}})

		msg := readStreamFrame(t, conn).GetStreamMessage()
		require.NotNil(t, msg)
		assert.Equal(t, "t", msg.GetId())

		got := &v1.StreamMessage{}
		require.NoError(t, unmarshalStream.Unmarshal([]byte(msg.GetMessage()), got))
		require.Len(t, got.Entries, 2)
		assert.Equal(t, `{"n":1}`, string(got.Entries[0].Payload))
		assert.Equal(t, `{"n":2}`, string(got.Entries[1].Payload))
		assert.Equal(t, "c2", got.Cursor)

		// A server stream takes no message after the request.
		writeFrame(t, conn, `{"streamMessage":{"id":"t","message":"{}"}}`)

		closeFrame := readStreamClose(t, conn, "t")
		assert.Equal(t, int32(connect.CodeInvalidArgument), closeFrame.GetCode())
		assert.True(t, rs.isClosed())

		// The engine ending the stream with PermissionDenied, as over gRPC.
		writeFrame(t, conn, `{"streamOpen":{"id":"u","procedure":"/v1.V1Streams/Subscribe","request":"{\"namespace\":\"ns\",\"topic\":\"orders\"}"}}`)
		opener.stream(t, 1).end(status.Error(codes.PermissionDenied, "durable streams are not enabled for this tenant"))

		closeFrame = readStreamClose(t, conn, "u")
		assert.Equal(t, int32(connect.CodePermissionDenied), closeFrame.GetCode())
		assert.Equal(t, "permission_denied: the engine denied the request", closeFrame.GetMessage())

		// The open itself refused with a connect error, as in process.
		opener.mu.Lock()
		opener.openErr = connect.NewError(connect.CodePermissionDenied, errors.New("durable streams are not enabled for this tenant"))
		opener.mu.Unlock()

		writeFrame(t, conn, `{"streamOpen":{"id":"v","procedure":"/v1.V1Streams/Subscribe","request":"{\"namespace\":\"ns\",\"topic\":\"orders\"}"}}`)

		closeFrame = readStreamClose(t, conn, "v")
		assert.Equal(t, int32(connect.CodePermissionDenied), closeFrame.GetCode())
		assert.Equal(t, 2, opener.count(), "a refused open holds no stream")

		writeFrame(t, conn, `{"done":{"output":"{}"}}`)
	})

	out := await(t, run(context.Background(), socketParams(ep, opener)))

	assert.Equal(t, KindCompleted, out.Kind)
	assert.Equal(t, CloseNormal, ep.closed(t))
}

// A topic subscription takes a slot like every other stream: past the cap it is refused, and
// a closed stream of another kind frees the slot for it.
func TestSocketDurableStreamCountsAgainstCap(t *testing.T) {
	setT(t)

	opener := &fakeOpener{}

	ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
		writeFrame(t, conn, `{"streamOpen":{"id":"a","procedure":"/Dispatcher/SubscribeToWorkflowRuns","request":"{\"workflowRunId\":\"run-1\"}"}}`)
		opener.stream(t, 0)

		writeFrame(t, conn, `{"streamOpen":{"id":"t","procedure":"/v1.V1Streams/Subscribe","request":"{\"namespace\":\"ns\",\"topic\":\"orders\"}"}}`)

		closeFrame := readStreamClose(t, conn, "t")
		assert.Equal(t, int32(connect.CodeResourceExhausted), closeFrame.GetCode())
		assert.Equal(t, 1, opener.count())

		writeFrame(t, conn, `{"streamClose":{"id":"a"}}`)
		require.Eventually(t, opener.stream(t, 0).isClosed, eventually, time.Millisecond)

		writeFrame(t, conn, `{"streamOpen":{"id":"t","procedure":"/v1.V1Streams/Subscribe","request":"{\"namespace\":\"ns\",\"topic\":\"orders\"}"}}`)
		assert.Equal(t, operator.RunStreamDurableStreams, opener.stream(t, 1).kind)

		writeFrame(t, conn, `{"done":{"output":"{}"}}`)
	})

	p := socketParams(ep, opener)
	p.MaxStreams = 1

	out := await(t, run(context.Background(), p))
	assert.Equal(t, KindCompleted, out.Kind)
	assert.True(t, opener.stream(t, 1).isClosed(), "the done frame closes what is still open")
}
