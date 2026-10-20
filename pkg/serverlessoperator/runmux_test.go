//go:build !e2e && !load && !rampup && !integration

package serverlessoperator

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/operator"
)

// fakeRunsStream is one engine SubscribeToWorkflowRuns stream: subscriptions land on sent,
// events pushed on recv are what the mux receives, fail ends it. A stream with a gate holds
// every Send until the gate is closed or the stream is closed, the way a gRPC send blocked
// on flow control is held: the send's context argument is ignored, as the real client
// ignores it.
type fakeRunsStream struct {
	sent   chan string
	recv   chan proto.Message
	failed chan error
	closed chan struct{}
	gate   chan struct{}
	// gated receives one token for every Send held at the gate.
	gated chan struct{}
	once  sync.Once
}

func newFakeRunsStream() *fakeRunsStream {
	return &fakeRunsStream{
		sent:   make(chan string, 64),
		recv:   make(chan proto.Message, 64),
		failed: make(chan error, 1),
		closed: make(chan struct{}),
		gated:  make(chan struct{}, 64),
	}
}

func (f *fakeRunsStream) Send(_ context.Context, msg proto.Message) error {
	select {
	case <-f.closed:
		return operator.ErrChannelClosed
	default:
	}

	if f.gate != nil {
		f.gated <- struct{}{}

		select {
		case <-f.gate:
		case <-f.closed:
			return operator.ErrChannelClosed
		}
	}

	f.sent <- msg.(*contracts.SubscribeToWorkflowRunsRequest).WorkflowRunId

	return nil
}

func (f *fakeRunsStream) Recv(ctx context.Context) (proto.Message, error) {
	select {
	case msg := <-f.recv:
		return msg, nil
	case err := <-f.failed:
		return nil, err
	case <-f.closed:
		return nil, operator.ErrChannelClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (f *fakeRunsStream) Close() error {
	f.once.Do(func() { close(f.closed) })
	return nil
}

func (f *fakeRunsStream) isClosed() bool {
	select {
	case <-f.closed:
		return true
	default:
		return false
	}
}

func (f *fakeRunsStream) subscription(t *testing.T) string {
	t.Helper()

	select {
	case runId := <-f.sent:
		return runId
	case <-time.After(eventually):
		t.Fatal("no subscription reached the engine stream")
		return ""
	}
}

func finished(runId string) *contracts.WorkflowRunEvent {
	return &contracts.WorkflowRunEvent{WorkflowRunId: runId, EventType: contracts.WorkflowRunEventType_WORKFLOW_RUN_EVENT_TYPE_FINISHED}
}

// fakeRunsOpener hands out engine streams in order and counts the opens; gate, when set, is
// the gate of every stream it opens, and openGate, when set, holds every open until it is
// closed, so a test can record subscriptions before the first stream exists and have the
// replay, not the subscriber, send them.
type fakeRunsOpener struct {
	streams  []*fakeRunsStream
	opens    int
	openErr  error
	gate     chan struct{}
	openGate chan struct{}
	mu       sync.Mutex
}

func (o *fakeRunsOpener) open(ctx context.Context) (operator.RunStream, error) {
	if o.openGate != nil {
		select {
		case <-o.openGate:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	o.opens++

	if o.openErr != nil {
		return nil, o.openErr
	}

	s := newFakeRunsStream()
	s.gate = o.gate
	o.streams = append(o.streams, s)

	return s, nil
}

func (o *fakeRunsOpener) stream(t *testing.T, i int) *fakeRunsStream {
	t.Helper()

	require.Eventually(t, func() bool {
		o.mu.Lock()
		defer o.mu.Unlock()

		return len(o.streams) > i
	}, eventually, time.Millisecond, "engine stream %d was never opened", i)

	o.mu.Lock()
	defer o.mu.Unlock()

	return o.streams[i]
}

func (o *fakeRunsOpener) openCount() int {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.opens
}

func newTestMux(opener *fakeRunsOpener) *runMux {
	l := zerolog.Nop()
	m := newRunMux(&l, DefaultWSMaxRunsPerStream, opener.open)
	m.backoff = func(ctx context.Context, _ int) error { return ctx.Err() }

	return m
}

func recvEvent(t *testing.T, h operator.RunStream) *contracts.WorkflowRunEvent {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), eventually)
	defer cancel()

	msg, err := h.Recv(ctx)
	require.NoError(t, err)

	return msg.(*contracts.WorkflowRunEvent)
}

func subscribe(runId string) *contracts.SubscribeToWorkflowRunsRequest {
	return &contracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: runId}
}

// One engine stream serves every subscriber: each run id is subscribed once per subscriber,
// its terminal event reaches every subscriber that asked for it, and a subscriber that closes
// drops its runs.
func TestRunMuxFansOutOneStream(t *testing.T) {
	opener := &fakeRunsOpener{}
	m := newTestMux(opener)
	t.Cleanup(m.close)

	a, err := m.Subscribe(context.Background(), subscribe("run-1"))
	require.NoError(t, err)

	b, err := m.Subscribe(context.Background(), nil)
	require.NoError(t, err)
	require.NoError(t, b.Send(context.Background(), subscribe("run-1")))
	require.NoError(t, b.Send(context.Background(), subscribe("run-2")))

	// The engine sees each run id at least once: a run subscribed before the stream opened is
	// replayed once by the open, one subscribed after it is sent as it comes.
	stream := opener.stream(t, 0)
	seen := map[string]struct{}{}

	for len(seen) < 2 {
		seen[stream.subscription(t)] = struct{}{}
	}

	assert.Equal(t, map[string]struct{}{"run-1": {}, "run-2": {}}, seen)
	assert.Equal(t, 1, opener.openCount(), "one engine stream serves both subscribers")

	stream.recv <- finished("run-1")

	assert.Equal(t, "run-1", recvEvent(t, a).WorkflowRunId)
	assert.Equal(t, "run-1", recvEvent(t, b).WorkflowRunId)

	// A run nobody awaits is dropped; b still gets run-2.
	stream.recv <- finished("run-9")
	stream.recv <- finished("run-2")
	assert.Equal(t, "run-2", recvEvent(t, b).WorkflowRunId)

	require.NoError(t, b.Close())
	_, err = b.Recv(context.Background())
	assert.ErrorIs(t, err, operator.ErrChannelClosed)

	m.mu.Lock()
	assert.Empty(t, m.subs, "terminal events and the close left no interest behind")
	m.mu.Unlock()
}

// A failed engine stream is reopened and the runs still awaited are subscribed again on it;
// a subscription made while no stream is open is sent by the reopen.
func TestRunMuxReconnectsAndReplays(t *testing.T) {
	opener := &fakeRunsOpener{}
	m := newTestMux(opener)
	t.Cleanup(m.close)

	a, err := m.Subscribe(context.Background(), subscribe("run-1"))
	require.NoError(t, err)

	first := opener.stream(t, 0)
	assert.Equal(t, "run-1", first.subscription(t))

	first.failed <- errors.New("transport broke")

	second := opener.stream(t, 1)
	assert.Equal(t, "run-1", second.subscription(t), "the awaited run is subscribed again")
	assert.True(t, first.isClosed())

	second.recv <- finished("run-1")
	assert.Equal(t, "run-1", recvEvent(t, a).WorkflowRunId)

	// An open that fails is retried.
	opener.mu.Lock()
	opener.openErr = errors.New("engine down")
	opener.mu.Unlock()

	second.failed <- errors.New("transport broke again")

	require.NoError(t, a.Send(context.Background(), subscribe("run-2")))

	require.Eventually(t, func() bool { return opener.openCount() >= 4 }, eventually, time.Millisecond, "the open is retried")

	opener.mu.Lock()
	opener.openErr = nil
	opener.mu.Unlock()

	third := opener.stream(t, 2)
	assert.Equal(t, "run-2", third.subscription(t), "the subscription made while disconnected is sent on the new stream")
}

// Closing the mux closes the engine stream and every subscriber, and refuses new ones.
func TestRunMuxClose(t *testing.T) {
	opener := &fakeRunsOpener{}
	m := newTestMux(opener)

	a, err := m.Subscribe(context.Background(), subscribe("run-1"))
	require.NoError(t, err)

	stream := opener.stream(t, 0)
	stream.subscription(t)

	m.close()

	assert.True(t, stream.isClosed())

	_, err = a.Recv(context.Background())
	assert.ErrorIs(t, err, operator.ErrChannelClosed)

	_, err = m.Subscribe(context.Background(), nil)
	assert.ErrorIs(t, err, operator.ErrSessionClosed)

	m.close()
}

// Closing the mux while the replay on a fresh engine stream is blocked by flow control
// returns promptly: the stream is not published until its replay is done and a blocked send
// is released only by the stream's own Close, so the close must close the stream being
// replayed on and never wait behind the send for the loop.
func TestRunMuxCloseDuringBlockedReplay(t *testing.T) {
	openGate := make(chan struct{})
	opener := &fakeRunsOpener{gate: make(chan struct{}), openGate: openGate}
	m := newTestMux(opener)

	// The first subscription is recorded before the stream opens, so the replay of run-1 is
	// what blocks on the gate.
	a, err := m.Subscribe(context.Background(), subscribe("run-1"))
	require.NoError(t, err)

	close(openGate)

	stream := opener.stream(t, 0)

	// Wait for the replay send to be held at the gate, so the close meets a stream that is
	// open but not published.
	select {
	case <-stream.gated:
	case <-time.After(eventually):
		t.Fatal("the replay never reached the engine stream")
	}

	closed := make(chan struct{})

	go func() {
		defer close(closed)
		m.close()
	}()

	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("mux close waited behind the replay blocked on the engine stream")
	}

	assert.True(t, stream.isClosed())

	_, err = a.Recv(context.Background())
	assert.ErrorIs(t, err, operator.ErrChannelClosed)
}

// A subscriber awaits at most the cap's worth of distinct runs at once: a run it already
// awaits is not counted again, one past the cap is refused with ResourceExhausted, and a run
// that finishes frees its place.
func TestRunMuxCapsRunsPerStream(t *testing.T) {
	opener := &fakeRunsOpener{}
	m := newTestMux(opener)
	m.maxRuns = 2
	t.Cleanup(m.close)

	a, err := m.Subscribe(context.Background(), subscribe("run-1"))
	require.NoError(t, err)
	require.NoError(t, a.Send(context.Background(), subscribe("run-2")))
	require.NoError(t, a.Send(context.Background(), subscribe("run-1")), "a run already awaited is not counted again")

	err = a.Send(context.Background(), subscribe("run-3"))
	require.Error(t, err)
	assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))

	// The cap is per subscriber, not per tenant.
	b, err := m.Subscribe(context.Background(), subscribe("run-3"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = b.Close() })

	stream := opener.stream(t, 0)
	seen := map[string]struct{}{}

	for len(seen) < 3 {
		seen[stream.subscription(t)] = struct{}{}
	}

	stream.recv <- finished("run-1")
	assert.Equal(t, "run-1", recvEvent(t, a).WorkflowRunId)

	require.NoError(t, a.Send(context.Background(), subscribe("run-4")), "the finished run freed its place")
	assert.Equal(t, "run-4", stream.subscription(t))
}

// A subscriber that closes while it still awaits runs nobody else holds retires the engine
// stream, since the engine keeps polling every run it was told of until the stream ends; the
// replacement is subscribed to the runs still awaited only. A close that leaves every run
// with another holder retires nothing.
func TestRunMuxRetiresOrphanedRunsOnHandleClose(t *testing.T) {
	opener := &fakeRunsOpener{}
	m := newTestMux(opener)
	t.Cleanup(m.close)

	a, err := m.Subscribe(context.Background(), subscribe("run-1"))
	require.NoError(t, err)

	b, err := m.Subscribe(context.Background(), subscribe("run-1"))
	require.NoError(t, err)
	require.NoError(t, b.Send(context.Background(), subscribe("run-2")))

	first := opener.stream(t, 0)
	seen := map[string]struct{}{}

	for len(seen) < 2 {
		seen[first.subscription(t)] = struct{}{}
	}

	// b still awaits run-1, so a's close orphans nothing.
	require.NoError(t, a.Close())
	assert.False(t, first.isClosed())
	assert.Equal(t, 1, opener.openCount())

	// Nobody else awaits run-2: b's close retires the stream; the replacement replays only
	// the runs still awaited, which is none.
	require.NoError(t, b.Close())
	assert.True(t, first.isClosed())

	second := opener.stream(t, 1)

	c, err := m.Subscribe(context.Background(), subscribe("run-3"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })

	assert.Equal(t, "run-3", second.subscription(t))

	select {
	case runId := <-second.sent:
		t.Fatalf("the replacement stream was subscribed to %s, which nobody awaits", runId)
	default:
	}

	second.recv <- finished("run-3")
	assert.Equal(t, "run-3", recvEvent(t, c).WorkflowRunId)
	assert.Equal(t, 2, opener.openCount())
}

// A subscriber that closes while the replay on a fresh stream is in flight, after its run was
// sent, retires that stream before it is published; the next one carries the live runs only.
func TestRunMuxRetiresStreamOrphanedDuringReplay(t *testing.T) {
	gate := make(chan struct{})
	openGate := make(chan struct{})
	opener := &fakeRunsOpener{gate: gate, openGate: openGate}
	m := newTestMux(opener)
	t.Cleanup(m.close)

	a, err := m.Subscribe(context.Background(), subscribe("run-1"))
	require.NoError(t, err)

	close(openGate)

	// The replay of run-1 on the first stream blocks on the gate; a closes meanwhile.
	first := opener.stream(t, 0)
	require.NoError(t, a.Close())

	b, err := m.Subscribe(context.Background(), subscribe("run-2"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = b.Close() })

	close(gate)

	assert.Equal(t, "run-1", first.subscription(t))
	require.Eventually(t, first.isClosed, eventually, time.Millisecond, "the stream that carries the orphaned run is retired")

	second := opener.stream(t, 1)
	assert.Equal(t, "run-2", second.subscription(t))

	select {
	case runId := <-second.sent:
		t.Fatalf("the replacement stream was subscribed to %s, which nobody awaits", runId)
	case <-time.After(50 * time.Millisecond):
	}

	second.recv <- finished("run-2")
	assert.Equal(t, "run-2", recvEvent(t, b).WorkflowRunId)
}
