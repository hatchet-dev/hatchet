//go:build !e2e && !load && !rampup && !integration

package durable

import (
	"context"
	"errors"
	"fmt"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/operator/safeclient"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/contract"
)

const (
	eventually = 5 * time.Second
	testSecret = "test-signing-secret-0123456789abcdef"
	testTaskId = "11111111-1111-1111-1111-111111111111"
)

// scenario is what the fake endpoint does once the socket is up and the first frame was
// read. Whatever it leaves unread is read afterwards to capture the relay's close code.
type scenario func(t *testing.T, conn *websocket.Conn, first *v1.ServerlessFirstFrame)

// fakeEndpoint is an in-process websocket endpoint that verifies the signed upgrade the way
// a real endpoint SDK would: endpoint id, timestamp age, nonce replay and HMAC.
type fakeEndpoint struct {
	srv       *httptest.Server
	now       func() time.Time
	run       scenario
	nonces    map[string]bool
	closeCode chan int
	rejected  chan int
	upgrader  websocket.Upgrader
	mu        sync.Mutex
}

func newFakeEndpoint(t *testing.T, run scenario) *fakeEndpoint {
	skipWithoutWebsocketHandshake(t)

	t.Helper()

	ep := &fakeEndpoint{
		now:       time.Now,
		run:       run,
		nonces:    map[string]bool{},
		closeCode: make(chan int, 8),
		rejected:  make(chan int, 8),
	}

	ep.srv = httptest.NewServer(http.HandlerFunc(ep.serve))
	t.Cleanup(ep.srv.Close)

	return ep
}

func (ep *fakeEndpoint) reject(w http.ResponseWriter, status int) {
	ep.rejected <- status
	w.WriteHeader(status)
}

func (ep *fakeEndpoint) serve(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(contract.EndpointIdHeader) != "ep-1" {
		ep.reject(w, http.StatusForbidden)
		return
	}

	ts, err := strconv.ParseInt(r.Header.Get(contract.TimestampHeader), 10, 64)

	if err != nil || ep.now().Sub(time.Unix(ts, 0)) > contract.UpgradeMaxAge {
		ep.reject(w, http.StatusUnauthorized)
		return
	}

	nonce := r.Header.Get(contract.NonceHeader)

	ep.mu.Lock()
	seen := ep.nonces[nonce]
	ep.nonces[nonce] = true
	ep.mu.Unlock()

	if nonce == "" || seen {
		ep.reject(w, http.StatusUnauthorized)
		return
	}

	payload := contract.UpgradeSigningPayload(
		r.Header.Get(contract.EndpointIdHeader),
		r.Header.Get(contract.TimestampHeader),
		nonce,
		r.Header.Get(contract.TaskIdHeader),
		r.Header.Get(contract.InvocationHeader),
	)

	if !contract.Verify(payload, testSecret, r.Header.Get(contract.SignatureHeader)) {
		ep.reject(w, http.StatusUnauthorized)
		return
	}

	conn, err := ep.upgrader.Upgrade(w, r, nil)

	if err != nil {
		return
	}

	defer conn.Close()

	_, data, err := conn.ReadMessage()

	if err != nil {
		ep.closeCode <- closeCodeOf(err)
		return
	}

	frame, err := contract.UnmarshalFrame(data)

	if err != nil {
		panic(err)
	}

	first := frame.GetFirst()

	if first == nil {
		panic("first frame is not a first frame")
	}

	// Scenarios run under the test's *testing.T; a failed assertion there fails the test.
	ep.run(currentT(), conn, first)

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			ep.closeCode <- closeCodeOf(err)
			return
		}
	}
}

func closeCodeOf(err error) int {
	var closeErr *websocket.CloseError

	if errors.As(err, &closeErr) {
		return closeErr.Code
	}

	return -1
}

// closed returns the close code the endpoint saw, or -1 for a socket that ended without
// a close frame.
func (ep *fakeEndpoint) closed(t *testing.T) int {
	t.Helper()

	select {
	case code := <-ep.closeCode:
		return code
	case <-time.After(eventually):
		t.Fatal("endpoint never saw the socket close")
		return 0
	}
}

// testT hands the scenario the running test; scenarios are per-test so one slot suffices.
var (
	testTMu sync.Mutex
	testT   *testing.T
)

func setT(t *testing.T) {
	testTMu.Lock()
	testT = t
	testTMu.Unlock()
}

func currentT() *testing.T {
	testTMu.Lock()
	defer testTMu.Unlock()

	return testT
}

// fakeChannel is the engine side: Send records requests (or fails through onSend) and
// Recv yields what the test replies.
type fakeChannel struct {
	onSend     func(*v1.DurableTaskRequest) error
	sent       chan *v1.DurableTaskRequest
	recv       chan recvItem
	closed     chan struct{}
	once       sync.Once
	closeCount int
	mu         sync.Mutex
}

type recvItem struct {
	resp *v1.DurableTaskResponse
	err  error
}

func newFakeChannel() *fakeChannel {
	return &fakeChannel{
		sent:   make(chan *v1.DurableTaskRequest, 64),
		recv:   make(chan recvItem, 512),
		closed: make(chan struct{}),
	}
}

func (f *fakeChannel) Send(_ context.Context, req *v1.DurableTaskRequest) error {
	if f.onSend != nil {
		if err := f.onSend(req); err != nil {
			return err
		}
	}

	f.sent <- req

	return nil
}

func (f *fakeChannel) Recv(ctx context.Context) (*v1.DurableTaskResponse, error) {
	select {
	case item := <-f.recv:
		return item.resp, item.err
	case <-f.closed:
		return nil, operator.ErrChannelClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ExpectEntry is a no-op: the relay never registers entries, the endpoint's requests carry
// their own acks.
func (f *fakeChannel) ExpectEntry(_, _ int64) error { return nil }

func (f *fakeChannel) Close() error {
	f.mu.Lock()
	f.closeCount++
	f.mu.Unlock()

	f.once.Do(func() { close(f.closed) })

	return nil
}

func (f *fakeChannel) closes() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.closeCount
}

func (f *fakeChannel) reply(resp *v1.DurableTaskResponse) {
	f.recv <- recvItem{resp: resp}
}

func (f *fakeChannel) next(t *testing.T) *v1.DurableTaskRequest {
	t.Helper()

	select {
	case req := <-f.sent:
		return req
	case <-time.After(eventually):
		t.Fatal("relay forwarded nothing")
		return nil
	}
}

type loopbackDialer struct{}

func (loopbackDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, network, addr)
}

func testAction() *contracts.AssignedAction {
	inv := int32(3)

	return &contracts.AssignedAction{
		ActionType:                 contracts.ActionType_START_STEP_RUN,
		TaskRunExternalId:          testTaskId,
		ActionId:                   "ns_svc:run",
		JobName:                    "ns_wf",
		ActionPayload:              `{"input":1}`,
		DurableTaskInvocationCount: &inv,
	}
}

func testParams(ep *fakeEndpoint, ch operator.DurableChannel) Params {
	return Params{
		Dialer:             loopbackDialer{},
		Channel:            ch,
		Action:             testAction(),
		TriggerURL:         ep.srv.URL + "/trigger",
		Secret:             testSecret,
		EndpointId:         "ep-1",
		TaskId:             testTaskId,
		PingInterval:       time.Second,
		InlineWaitBudgetMs: 5000,
		Invocation:         3,
		Insecure:           true,
	}
}

// run executes the relay on its own goroutine and returns the outcome through a channel.
func run(ctx context.Context, p Params) <-chan Outcome {
	out := make(chan Outcome, 1)

	go func() { out <- Run(ctx, p) }()

	return out
}

func await(t *testing.T, out <-chan Outcome) Outcome {
	t.Helper()

	select {
	case o := <-out:
		return o
	case <-time.After(eventually):
		t.Fatal("relay did not finish")
		return Outcome{}
	}
}

func writeFrame(t *testing.T, conn *websocket.Conn, frame string) {
	t.Helper()
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(frame)))
}

func readFrame(t *testing.T, conn *websocket.Conn) *v1.ServerlessDurableFrame {
	t.Helper()

	_, data, err := conn.ReadMessage()
	require.NoError(t, err)

	frame, err := contract.UnmarshalFrame(data)
	require.NoError(t, err)

	return frame
}

// readResponse reads the next frame and requires it to carry an engine response.
func readResponse(t *testing.T, conn *websocket.Conn) *v1.DurableTaskResponse {
	t.Helper()

	resp := readFrame(t, conn).GetResponse()
	require.NotNil(t, resp, "expected a response frame")

	return resp
}

func TestRelayCompletesAndDropsLateFrames(t *testing.T) {
	setT(t)

	ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, first *v1.ServerlessFirstFrame) {
		assert.Equal(t, int32(3), first.InvocationCount)
		assert.Equal(t, int32(5000), first.InlineWaitBudgetMs)

		action := first.GetAction()
		require.NotNil(t, action)
		assert.Equal(t, "ns_svc:run", action.ActionId)
		assert.Equal(t, int32(3), action.GetDurableTaskInvocationCount())

		writeFrame(t, conn, `{"done":{"output":"{\"ok\":true}"}}`)
		// A frame after done is dropped, not forwarded.
		writeFrame(t, conn, `{"id":9,"request":{"memo":{"key":"aw=="}}}`)
	})

	ch := newFakeChannel()
	out := await(t, run(context.Background(), testParams(ep, ch)))

	assert.Equal(t, KindCompleted, out.Kind)
	assert.JSONEq(t, `{"ok":true}`, string(out.Output))
	assert.Equal(t, CloseNormal, ep.closed(t))
	assert.Empty(t, ch.sent)
	assert.GreaterOrEqual(t, ch.closes(), 1, "the channel is closed on exit")
}

func TestRelayForwardsRequestsAndResponses(t *testing.T) {
	setT(t)

	ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
		// Ids left empty are stamped; a zero invocation means unset.
		writeFrame(t, conn, `{"id":1,"request":{"memo":{"key":"aw=="}}}`)

		resp := readResponse(t, conn)
		require.NotNil(t, resp.GetMemoAck())
		assert.True(t, resp.GetMemoAck().MemoAlreadyExisted)

		// The wait_for ack and its entry_completed both arrive as response frames.
		writeFrame(t, conn, fmt.Sprintf(`{"id":2,"request":{"waitFor":{"durableTaskExternalId":%q,"invocationCount":3}}}`, testTaskId))

		ack := readResponse(t, conn)
		require.NotNil(t, ack.GetWaitForAck())

		completed := readResponse(t, conn)
		require.NotNil(t, completed.GetEntryCompleted())
		assert.Equal(t, `{"slept":1}`, string(completed.GetEntryCompleted().Payload))

		writeFrame(t, conn, `{"done":{"output":"{\"ok\":1}"}}`)
	})

	ch := newFakeChannel()
	out := run(context.Background(), testParams(ep, ch))

	memo := ch.next(t).GetMemo()
	require.NotNil(t, memo)
	assert.Equal(t, testTaskId, memo.DurableTaskExternalId)
	assert.Equal(t, int32(3), memo.InvocationCount)
	assert.Equal(t, []byte("k"), memo.Key)

	ch.reply(&v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_MemoAck{
		MemoAck: &v1.DurableTaskEventMemoAckResponse{Ref: &v1.DurableEventLogEntryRef{NodeId: 1}, MemoAlreadyExisted: true},
	}})

	waitFor := ch.next(t).GetWaitFor()
	require.NotNil(t, waitFor)

	ch.reply(&v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_WaitForAck{
		WaitForAck: &v1.DurableTaskEventWaitForAckResponse{Ref: &v1.DurableEventLogEntryRef{NodeId: 2}},
	}})
	ch.reply(&v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_EntryCompleted{
		EntryCompleted: &v1.DurableTaskEventLogEntryCompletedResponse{Ref: &v1.DurableEventLogEntryRef{NodeId: 2}, Payload: []byte(`{"slept":1}`)},
	}})

	o := await(t, out)
	assert.Equal(t, KindCompleted, o.Kind)
	assert.Equal(t, CloseNormal, ep.closed(t))
}

func TestRelayUpgradeRejected(t *testing.T) {
	setT(t)

	ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
		writeFrame(t, conn, `{"done":{"output":"{}"}}`)
	})

	t.Run("bad hmac", func(t *testing.T) {
		p := testParams(ep, newFakeChannel())
		p.Secret = "wrong-signing-secret-0123456789abcdef"

		out := await(t, run(context.Background(), p))
		assert.Equal(t, KindFailed, out.Kind)
		assert.False(t, out.Retry)
		assert.Contains(t, out.Error, "status 401")
		assert.Equal(t, http.StatusUnauthorized, <-ep.rejected)
	})

	t.Run("stale timestamp", func(t *testing.T) {
		p := testParams(ep, newFakeChannel())
		p.now = func() time.Time { return time.Now().Add(-10 * time.Minute) }

		out := await(t, run(context.Background(), p))
		assert.Equal(t, KindFailed, out.Kind)
		assert.False(t, out.Retry)
		assert.Equal(t, http.StatusUnauthorized, <-ep.rejected)
	})

	t.Run("replayed nonce", func(t *testing.T) {
		p := testParams(ep, newFakeChannel())
		p.nonce = func() (string, error) { return "fixed-nonce", nil }

		first := await(t, run(context.Background(), p))
		require.Equal(t, KindCompleted, first.Kind)
		assert.Equal(t, CloseNormal, ep.closed(t))

		p.Channel = newFakeChannel()
		second := await(t, run(context.Background(), p))
		assert.Equal(t, KindFailed, second.Kind)
		assert.False(t, second.Retry)
		assert.Equal(t, http.StatusUnauthorized, <-ep.rejected)
	})

	t.Run("wrong endpoint id", func(t *testing.T) {
		p := testParams(ep, newFakeChannel())
		p.EndpointId = "ep-2"

		out := await(t, run(context.Background(), p))
		assert.Equal(t, KindFailed, out.Kind)
		assert.False(t, out.Retry)
		assert.Equal(t, http.StatusForbidden, <-ep.rejected)
	})

	t.Run("5xx is retryable", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer srv.Close()

		p := testParams(ep, newFakeChannel())
		p.TriggerURL = srv.URL

		out := await(t, run(context.Background(), p))
		assert.Equal(t, KindFailed, out.Kind)
		assert.True(t, out.Retry)
	})

	t.Run("no secret", func(t *testing.T) {
		p := testParams(ep, newFakeChannel())
		p.Secret = ""

		out := await(t, run(context.Background(), p))
		assert.Equal(t, KindFailed, out.Kind)
		assert.False(t, out.Retry)
		assert.Contains(t, out.Error, "no signing secret")
	})
}

func TestRelaySchemeRules(t *testing.T) {
	ch := newFakeChannel()

	p := Params{Dialer: loopbackDialer{}, Channel: ch, Action: testAction(), TriggerURL: "http://example.test/trigger", Secret: testSecret}
	out := await(t, run(context.Background(), p))
	assert.Equal(t, KindFailed, out.Kind)
	assert.False(t, out.Retry)
	assert.Contains(t, out.Error, safeclient.ErrBadScheme.Error())

	target, err := upgradeURL("https://ep.example/trigger?x=1", false)
	require.NoError(t, err)
	assert.Equal(t, "wss://ep.example/trigger?x=1", target)

	target, err = upgradeURL("http://localhost:8787/trigger", true)
	require.NoError(t, err)
	assert.Equal(t, "ws://localhost:8787/trigger", target)

	_, err = upgradeURL("ftp://ep.example/", true)
	assert.ErrorIs(t, err, safeclient.ErrBadScheme)

	_, err = upgradeURL("https://user:pw@ep.example/", false)
	assert.ErrorIs(t, err, safeclient.ErrBlockedDestination)
}

func TestRelayProtocolViolations(t *testing.T) {
	tests := []struct {
		name     string
		frame    string
		onSend   func(*v1.DurableTaskRequest) error
		wantCode int
	}{
		{
			name:     "second request in flight",
			frame:    `{"id":1,"request":{"memo":{"key":"aw=="}}}`,
			onSend:   func(*v1.DurableTaskRequest) error { return operator.ErrRequestInFlight },
			wantCode: CloseRequestInFlight,
		},
		{
			name:     "task id mismatch",
			frame:    `{"id":1,"request":{"memo":{"durableTaskExternalId":"other","key":"aw=="}}}`,
			wantCode: CloseInvocationMismatch,
		},
		{
			name:     "invocation mismatch",
			frame:    `{"id":1,"request":{"waitFor":{"invocationCount":2}}}`,
			wantCode: CloseInvocationMismatch,
		},
		{
			name:     "register_worker forbidden",
			frame:    `{"id":1,"request":{"registerWorker":{"workerId":"w"}}}`,
			wantCode: CloseForbiddenMessage,
		},
		{
			name:     "worker_status forbidden",
			frame:    `{"id":1,"request":{"workerStatus":{"workerId":"w"}}}`,
			wantCode: CloseForbiddenMessage,
		},
		{
			name:     "unknown frame",
			frame:    `{"hello":1}`,
			wantCode: CloseForbiddenMessage,
		},
		{
			name:     "malformed frame",
			frame:    `not json`,
			wantCode: CloseForbiddenMessage,
		},
		{
			name:     "complete_memo without ref",
			frame:    `{"id":1,"request":{"completeMemo":{"payload":"e30="}}}`,
			wantCode: CloseForbiddenMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setT(t)

			ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
				writeFrame(t, conn, tt.frame)
			})

			ch := newFakeChannel()
			ch.onSend = tt.onSend

			out := await(t, run(context.Background(), testParams(ep, ch)))
			assert.Equal(t, KindFailed, out.Kind)
			assert.False(t, out.Retry, "protocol violations are endpoint bugs, not retried")
			assert.Equal(t, tt.wantCode, out.CloseCode)
			assert.Equal(t, tt.wantCode, ep.closed(t))
		})
	}
}

func TestRelayBackpressure(t *testing.T) {
	setT(t)

	ep := newFakeEndpoint(t, func(*testing.T, *websocket.Conn, *v1.ServerlessFirstFrame) {})

	ch := newFakeChannel()
	p := testParams(ep, ch)
	// The writer never gets to write, so the queue fills from the engine side alone.
	p.writeGate = make(chan struct{})

	out := run(context.Background(), p)

	// The writer holds one frame while parked on the gate, so the queue overflows on the
	// second frame past its size.
	for i := 0; i < sendQueueSize+2; i++ {
		ch.reply(&v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_MemoAck{
			MemoAck: &v1.DurableTaskEventMemoAckResponse{Ref: &v1.DurableEventLogEntryRef{NodeId: int64(i)}},
		}})
	}

	o := await(t, out)
	assert.Equal(t, KindFailed, o.Kind)
	assert.True(t, o.Retry)
	assert.Equal(t, CloseBackpressure, o.CloseCode)
	assert.Equal(t, CloseBackpressure, ep.closed(t))
}

func TestRelayDoneMapping(t *testing.T) {
	tests := []struct {
		name   string
		done   string
		kind   Kind
		output string
		errMsg string
		retry  bool
	}{
		{name: "output", done: `{"output":"[1,2]"}`, kind: KindCompleted, output: `[1,2]`},
		{name: "empty output", done: `{}`, kind: KindCompleted, output: `{}`},
		{name: "empty output string", done: `{"output":""}`, kind: KindCompleted, output: `{}`},
		{name: "error retry", done: `{"error":"boom","retry":true}`, kind: KindFailed, errMsg: "boom", retry: true},
		{name: "error no retry", done: `{"error":"bad input","retry":false}`, kind: KindFailed, errMsg: "bad input"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setT(t)

			ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
				writeFrame(t, conn, `{"done":`+tt.done+`}`)
			})

			out := await(t, run(context.Background(), testParams(ep, newFakeChannel())))
			assert.Equal(t, tt.kind, out.Kind)
			assert.Equal(t, tt.retry, out.Retry)
			assert.Equal(t, tt.errMsg, out.Error)

			if tt.output != "" {
				assert.JSONEq(t, tt.output, string(out.Output))
			}

			assert.Equal(t, CloseNormal, ep.closed(t))
		})
	}
}

func TestRelayDoneNonJSONOutput(t *testing.T) {
	setT(t)

	ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
		writeFrame(t, conn, `{"done":{"output":"not json"}}`)
	})

	out := await(t, run(context.Background(), testParams(ep, newFakeChannel())))
	assert.Equal(t, KindFailed, out.Kind)
	assert.False(t, out.Retry)
	assert.Equal(t, CloseForbiddenMessage, out.CloseCode)
	assert.Equal(t, CloseForbiddenMessage, ep.closed(t))
}

func TestRelayCloseWithoutDone(t *testing.T) {
	t.Run("close frame", func(t *testing.T) {
		setT(t)

		ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
			msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
			require.NoError(t, conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(time.Second)))
		})

		out := await(t, run(context.Background(), testParams(ep, newFakeChannel())))
		assert.Equal(t, KindFailed, out.Kind)
		assert.True(t, out.Retry)
		assert.Contains(t, out.Error, "without a done frame")
	})

	t.Run("tcp reset", func(t *testing.T) {
		setT(t)

		ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
			require.NoError(t, conn.NetConn().Close())
		})

		out := await(t, run(context.Background(), testParams(ep, newFakeChannel())))
		assert.Equal(t, KindFailed, out.Kind)
		assert.True(t, out.Retry)
	})

	t.Run("after eviction ack is evicted", func(t *testing.T) {
		setT(t)

		ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
			writeFrame(t, conn, `{"id":1,"request":{"evictInvocation":{}}}`)
			require.NotNil(t, readFrame(t, conn).GetResponse())
			require.NoError(t, conn.NetConn().Close())
		})

		ch := newFakeChannel()
		out := run(context.Background(), testParams(ep, ch))

		require.NotNil(t, ch.next(t).GetEvictInvocation())
		ch.reply(&v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_EvictionAck{
			EvictionAck: &v1.DurableTaskEvictionAckResponse{InvocationCount: 3, DurableTaskExternalId: testTaskId},
		}})

		o := await(t, out)
		assert.Equal(t, KindEvicted, o.Kind)
		assert.Equal(t, EvictionSourceEndpoint, o.EvictionSource)
	})

	t.Run("after engine error is permanent", func(t *testing.T) {
		setT(t)

		ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
			writeFrame(t, conn, `{"id":1,"request":{"memo":{"key":"aw=="}}}`)

			body := readFrame(t, conn).GetError()
			require.NotNil(t, body, "expected an error frame")
			assert.Equal(t, contract.ErrorCodeNonDeterminism, body.Code)
			assert.Equal(t, "replay diverged", body.Message)

			require.NoError(t, conn.NetConn().Close())
		})

		ch := newFakeChannel()
		out := run(context.Background(), testParams(ep, ch))

		ch.next(t)
		ch.reply(&v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_Error{
			Error: &v1.DurableTaskErrorResponse{
				ErrorType:    v1.DurableTaskErrorType_DURABLE_TASK_ERROR_TYPE_NONDETERMINISM,
				ErrorMessage: "replay diverged",
			},
		}})

		o := await(t, out)
		assert.Equal(t, KindFailed, o.Kind)
		assert.False(t, o.Retry)
		assert.Equal(t, "replay diverged", o.Error)
	})
}

func TestRelayEndpointEviction(t *testing.T) {
	setT(t)

	ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
		writeFrame(t, conn, `{"id":1,"request":{"waitFor":{}}}`)
		require.NotNil(t, readResponse(t, conn).GetWaitForAck())

		// The inline budget elapsed without entry_completed: evict and finish.
		writeFrame(t, conn, `{"id":2,"request":{"evictInvocation":{"reason":"budget"}}}`)

		ack := readResponse(t, conn)
		require.NotNil(t, ack.GetEvictionAck())

		writeFrame(t, conn, `{"done":{"status":"evicted"}}`)
	})

	ch := newFakeChannel()
	out := run(context.Background(), testParams(ep, ch))

	require.NotNil(t, ch.next(t).GetWaitFor())
	ch.reply(&v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_WaitForAck{
		WaitForAck: &v1.DurableTaskEventWaitForAckResponse{Ref: &v1.DurableEventLogEntryRef{NodeId: 1}},
	}})

	evict := ch.next(t).GetEvictInvocation()
	require.NotNil(t, evict)
	assert.Equal(t, "budget", evict.GetReason())
	assert.Equal(t, int32(3), evict.InvocationCount)

	ch.reply(&v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_EvictionAck{
		EvictionAck: &v1.DurableTaskEvictionAckResponse{InvocationCount: 3, DurableTaskExternalId: testTaskId},
	}})

	o := await(t, out)
	assert.Equal(t, KindEvicted, o.Kind)
	assert.Equal(t, EvictionSourceEndpoint, o.EvictionSource)
	assert.Equal(t, CloseNormal, ep.closed(t))
}

// parkOnWait has the endpoint ask the engine for a durable wait and read its ack, which
// leaves the invocation parked on that wait.
func parkOnWait(t *testing.T, conn *websocket.Conn) {
	t.Helper()

	writeFrame(t, conn, `{"id":1,"request":{"waitFor":{}}}`)
	require.NotNil(t, readResponse(t, conn).GetWaitForAck())
}

// ackWaitFor answers the endpoint's wait_for on the engine side.
func ackWaitFor(t *testing.T, ch *fakeChannel) {
	t.Helper()

	require.NotNil(t, ch.next(t).GetWaitFor())
	ch.reply(&v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_WaitForAck{
		WaitForAck: &v1.DurableTaskEventWaitForAckResponse{Ref: &v1.DurableEventLogEntryRef{NodeId: 1}},
	}})
}

// awaitSignal waits for the scenario step the test orders itself against.
func awaitSignal(t *testing.T, signal <-chan struct{}, what string) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(eventually):
		t.Fatalf("the endpoint never %s", what)
	}
}

// evictionAck is the engine's answer to an eviction request.
func evictionAck() *v1.DurableTaskResponse {
	return &v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_EvictionAck{
		EvictionAck: &v1.DurableTaskEvictionAckResponse{InvocationCount: 3, DurableTaskExternalId: testTaskId},
	}}
}

// TestRelayParkedExitIsEvicted covers the exception to the crash rule: a socket that ends
// while the invocation is parked on a durable wait the engine acked is evicted by the
// operator on the task's behalf, so the engine re-invokes the task when the awaited entry is
// satisfied instead of the task spending a retry.
func TestRelayParkedExitIsEvicted(t *testing.T) {
	t.Run("request timeout while parked", func(t *testing.T) {
		setT(t)

		parked := make(chan struct{})

		ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
			parkOnWait(t, conn)
			close(parked)
		})

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		ch := newFakeChannel()
		out := run(ctx, testParams(ep, ch))

		ackWaitFor(t, ch)
		awaitSignal(t, parked, "read the wait_for ack")

		evict := ch.next(t).GetEvictInvocation()
		require.NotNil(t, evict)
		assert.Equal(t, testTaskId, evict.DurableTaskExternalId)
		assert.Equal(t, int32(3), evict.InvocationCount)
		assert.Contains(t, evict.GetReason(), "timed out")
		assert.Contains(t, evict.GetReason(), "evicted by the operator")

		ch.reply(evictionAck())

		o := await(t, out)
		assert.Equal(t, KindEvicted, o.Kind, "kind=%d retry=%v error=%q", o.Kind, o.Retry, o.Error)
		assert.Equal(t, EvictionSourceOperator, o.EvictionSource)
		assert.Equal(t, CloseTimeout, o.CloseCode)
		assert.Equal(t, CloseTimeout, ep.closed(t))
	})

	t.Run("peer close without done while parked", func(t *testing.T) {
		setT(t)

		ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
			parkOnWait(t, conn)
			require.NoError(t, conn.NetConn().Close())
		})

		ch := newFakeChannel()
		out := run(context.Background(), testParams(ep, ch))

		ackWaitFor(t, ch)

		require.NotNil(t, ch.next(t).GetEvictInvocation())
		ch.reply(evictionAck())

		o := await(t, out)
		assert.Equal(t, KindEvicted, o.Kind, "kind=%d retry=%v error=%q", o.Kind, o.Retry, o.Error)
		assert.Equal(t, EvictionSourceOperator, o.EvictionSource)
	})

	t.Run("timeout while computing stays a retryable failure", func(t *testing.T) {
		setT(t)

		answered := make(chan struct{})

		ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
			writeFrame(t, conn, `{"id":1,"request":{"memo":{"key":"aw=="}}}`)
			require.NotNil(t, readResponse(t, conn).GetMemoAck())
			close(answered)
		})

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		ch := newFakeChannel()
		out := run(ctx, testParams(ep, ch))

		require.NotNil(t, ch.next(t).GetMemo())
		ch.reply(&v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_MemoAck{
			MemoAck: &v1.DurableTaskEventMemoAckResponse{Ref: &v1.DurableEventLogEntryRef{NodeId: 1}},
		}})

		awaitSignal(t, answered, "read the memo ack")

		o := await(t, out)
		assert.Equal(t, KindFailed, o.Kind)
		assert.True(t, o.Retry)
		assert.Contains(t, o.Error, "timed out")
		assert.Empty(t, ch.sent, "no eviction is requested for a task that was computing")
		assert.Equal(t, CloseTimeout, ep.closed(t))
	})

	t.Run("engine refuses the eviction", func(t *testing.T) {
		setT(t)

		ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
			parkOnWait(t, conn)
			require.NoError(t, conn.NetConn().Close())
		})

		ch := newFakeChannel()
		out := run(context.Background(), testParams(ep, ch))

		ackWaitFor(t, ch)

		require.NotNil(t, ch.next(t).GetEvictInvocation())
		ch.reply(&v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_Error{
			Error: &v1.DurableTaskErrorResponse{ErrorMessage: "task not found"},
		}})

		o := await(t, out)
		assert.Equal(t, KindFailed, o.Kind)
		assert.True(t, o.Retry)
		assert.Contains(t, o.Error, "without a done frame")
	})

	t.Run("entry completed before the cut", func(t *testing.T) {
		setT(t)

		ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
			parkOnWait(t, conn)
			require.NotNil(t, readResponse(t, conn).GetEntryCompleted())
			require.NoError(t, conn.NetConn().Close())
		})

		ch := newFakeChannel()
		out := run(context.Background(), testParams(ep, ch))

		ackWaitFor(t, ch)
		ch.reply(&v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_EntryCompleted{
			EntryCompleted: &v1.DurableTaskEventLogEntryCompletedResponse{Ref: &v1.DurableEventLogEntryRef{NodeId: 1}, Payload: []byte(`{}`)},
		}})

		o := await(t, out)
		assert.Equal(t, KindFailed, o.Kind)
		assert.True(t, o.Retry)
		assert.Empty(t, ch.sent, "the task was computing again when the socket ended")
	})
}

func TestRelayServerEvict(t *testing.T) {
	setT(t)

	ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
		notice := readResponse(t, conn)
		require.NotNil(t, notice.GetServerEvict())
		assert.Equal(t, "superseded", notice.GetServerEvict().Reason)
	})

	ch := newFakeChannel()
	out := run(context.Background(), testParams(ep, ch))

	ch.reply(&v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_ServerEvict{
		ServerEvict: &v1.DurableTaskServerEvictNotice{DurableTaskExternalId: testTaskId, InvocationCount: 3, Reason: "superseded"},
	}})

	o := await(t, out)
	assert.Equal(t, KindEvicted, o.Kind)
	assert.Equal(t, EvictionSourceServer, o.EvictionSource)
	assert.Equal(t, CloseEvicted, o.CloseCode)
	assert.Equal(t, CloseEvicted, ep.closed(t))
}

func TestRelayContextEnds(t *testing.T) {
	t.Run("engine cancel", func(t *testing.T) {
		setT(t)

		ready := make(chan struct{})
		ep := newFakeEndpoint(t, func(*testing.T, *websocket.Conn, *v1.ServerlessFirstFrame) { close(ready) })

		ctx, cancel := context.WithCancel(context.Background())
		p := testParams(ep, newFakeChannel())
		p.Cancelled = func() bool { return true }

		out := run(ctx, p)
		<-ready
		cancel()

		o := await(t, out)
		assert.Equal(t, KindCancelled, o.Kind)
		assert.Equal(t, CloseCancelled, ep.closed(t))
	})

	t.Run("shutdown", func(t *testing.T) {
		setT(t)

		ready := make(chan struct{})
		ep := newFakeEndpoint(t, func(*testing.T, *websocket.Conn, *v1.ServerlessFirstFrame) { close(ready) })

		ctx, cancel := context.WithCancel(context.Background())
		out := run(ctx, testParams(ep, newFakeChannel()))
		<-ready
		cancel()

		o := await(t, out)
		assert.Equal(t, KindShutdown, o.Kind)
		assert.Equal(t, CloseShuttingDown, ep.closed(t))
	})

	t.Run("timeout", func(t *testing.T) {
		setT(t)

		ep := newFakeEndpoint(t, func(*testing.T, *websocket.Conn, *v1.ServerlessFirstFrame) {})

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		o := await(t, run(ctx, testParams(ep, newFakeChannel())))
		assert.Equal(t, KindFailed, o.Kind)
		assert.True(t, o.Retry)
		assert.Contains(t, o.Error, "timed out")
		assert.Equal(t, CloseTimeout, ep.closed(t))
	})

	t.Run("cancel before dial", func(t *testing.T) {
		setT(t)

		ep := newFakeEndpoint(t, func(*testing.T, *websocket.Conn, *v1.ServerlessFirstFrame) {})

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		o := await(t, run(ctx, testParams(ep, newFakeChannel())))
		assert.Equal(t, KindShutdown, o.Kind)
	})
}

func TestRelayPingTimeout(t *testing.T) {
	setT(t)

	ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
		// Swallow pings so no pong ever goes back.
		conn.SetPingHandler(func(string) error { return nil })
	})

	p := testParams(ep, newFakeChannel())
	p.PingInterval = 20 * time.Millisecond

	o := await(t, run(context.Background(), p))
	assert.Equal(t, KindFailed, o.Kind)
	assert.True(t, o.Retry)
	assert.Contains(t, o.Error, "missed 2 pings")
	assert.Equal(t, CloseUnresponsive, ep.closed(t))
}

func TestRelayPongsKeepTheSocketOpen(t *testing.T) {
	setT(t)

	ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
		// The default ping handler answers with pongs while the endpoint reads.
		deadline := time.Now().Add(150 * time.Millisecond)

		for time.Now().Before(deadline) {
			require.NoError(t, conn.SetReadDeadline(deadline))

			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}

		require.NoError(t, conn.SetReadDeadline(time.Time{}))
		writeFrame(t, conn, `{"done":{"output":"{}"}}`)
	})

	p := testParams(ep, newFakeChannel())
	p.PingInterval = 20 * time.Millisecond

	o := await(t, run(context.Background(), p))
	assert.Equal(t, KindCompleted, o.Kind)
}

func TestRelayFrameCap(t *testing.T) {
	setT(t)

	ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
		writeFrame(t, conn, `{"id":1,"request":{"memo":{"key":"`+strings.Repeat("A", 2048)+`"}}}`)
	})

	p := testParams(ep, newFakeChannel())
	p.MaxFrameBytes = 1024

	o := await(t, run(context.Background(), p))
	assert.Equal(t, KindFailed, o.Kind)
	assert.False(t, o.Retry)
	assert.Contains(t, o.Error, "exceeded the 1024 byte limit")
	assert.Equal(t, websocket.CloseMessageTooBig, ep.closed(t))

	// The outbound action frame is capped too, before anything is dialed.
	big := testParams(ep, newFakeChannel())
	big.MaxFrameBytes = 64

	o = await(t, run(context.Background(), big))
	assert.Equal(t, KindFailed, o.Kind)
	assert.False(t, o.Retry)
	assert.Contains(t, o.Error, "action frame")
}

func TestRelayLinkFailure(t *testing.T) {
	setT(t)

	ep := newFakeEndpoint(t, func(*testing.T, *websocket.Conn, *v1.ServerlessFirstFrame) {})

	ch := newFakeChannel()
	out := run(context.Background(), testParams(ep, ch))

	ch.recv <- recvItem{err: errors.New("connection reset")}

	o := await(t, out)
	assert.Equal(t, KindFailed, o.Kind)
	assert.True(t, o.Retry)
	assert.Contains(t, o.Error, "engine link failed")
	assert.Equal(t, CloseInternalError, ep.closed(t))
}

func TestSignedUpgradeHeaders(t *testing.T) {
	now := time.Unix(1700000000, 0)

	h, err := signedUpgradeHeaders(testSecret, "ep-1", testTaskId, 4, now, "nonce")
	require.NoError(t, err)

	assert.Equal(t, "ep-1", h.Get(contract.EndpointIdHeader))
	assert.Equal(t, "1700000000", h.Get(contract.TimestampHeader))
	assert.Equal(t, "nonce", h.Get(contract.NonceHeader))
	assert.Equal(t, testTaskId, h.Get(contract.TaskIdHeader))
	assert.Equal(t, "4", h.Get(contract.InvocationHeader))

	expected, err := contract.Sign("ep-1.1700000000.nonce."+testTaskId+".4", testSecret)
	require.NoError(t, err)
	assert.Equal(t, expected, h.Get(contract.SignatureHeader))

	nonce, err := newNonce()
	require.NoError(t, err)
	assert.Len(t, nonce, 22, "16 bytes base64url without padding")

	other, err := newNonce()
	require.NoError(t, err)
	assert.NotEqual(t, nonce, other)
}

// TestRelayBackpressureBytes is the performance F11 regression: the send queue is bounded
// by retained bytes as well as by frame count, so a slow endpoint cannot hold sendQueueSize
// large frames in memory.
func TestRelayBackpressureBytes(t *testing.T) {
	setT(t)

	ep := newFakeEndpoint(t, func(*testing.T, *websocket.Conn, *v1.ServerlessFirstFrame) {})

	ch := newFakeChannel()
	p := testParams(ep, ch)
	p.writeGate = make(chan struct{})
	p.MaxQueuedBytes = 64 * 1024

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := run(ctx, p)

	// Eight 16 KiB frames: far under the frame count limit, over the byte budget (the writer
	// holds one frame on the gate, so the queue retains the rest).
	payload := strings.Repeat("A", 16*1024)

	for i := 0; i < 8; i++ {
		ch.reply(&v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_EntryCompleted{
			EntryCompleted: &v1.DurableTaskEventLogEntryCompletedResponse{
				Ref:     &v1.DurableEventLogEntryRef{NodeId: int64(i)},
				Payload: []byte(payload),
			},
		}})
	}

	o := await(t, out)
	assert.Equal(t, KindFailed, o.Kind)
	assert.True(t, o.Retry)
	assert.Equal(t, CloseBackpressure, o.CloseCode)
	assert.Contains(t, o.Error, "bytes behind")
	assert.Equal(t, CloseBackpressure, ep.closed(t))
}

// A socket error names the endpoint host and the stage, never the address the socket was
// connected to.
func TestTransportMessageRedactsAddresses(t *testing.T) {
	err := &net.OpError{Op: "read", Net: "tcp", Addr: &net.TCPAddr{IP: net.IPv4(203, 0, 113, 9), Port: 443}, Err: errors.New("connection reset by peer")}

	msg := transportMessage("https://endpoint.example.test/trigger?token=marker", err)
	assert.Contains(t, msg, "endpoint.example.test")
	assert.NotContains(t, msg, "203.0.113.9")
	assert.NotContains(t, msg, "marker")
}

// skipWithoutWebsocketHandshake skips a test that upgrades a websocket: the RFC 6455
// handshake is specified over SHA-1, which GODEBUG=fips140=only refuses.
func skipWithoutWebsocketHandshake(t *testing.T) {
	t.Helper()

	if !encryption.SHA1Allowed() {
		t.Skip("websocket handshakes need SHA-1, which fips140=only refuses")
	}
}
