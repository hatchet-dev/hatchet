//go:build !e2e && !load && !rampup && !integration

package serverlessoperator

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/operator"
)

// fakeRunsStream is one engine SubscribeToWorkflowRuns stream: subscriptions land on sent,
// events pushed on recv are what the mux receives, fail ends it.
type fakeRunsStream struct {
	sent   chan string
	recv   chan proto.Message
	failed chan error
	closed chan struct{}
	once   sync.Once
}

func newFakeRunsStream() *fakeRunsStream {
	return &fakeRunsStream{
		sent:   make(chan string, 64),
		recv:   make(chan proto.Message, 64),
		failed: make(chan error, 1),
		closed: make(chan struct{}),
	}
}

func (f *fakeRunsStream) Send(_ context.Context, msg proto.Message) error {
	select {
	case <-f.closed:
		return operator.ErrChannelClosed
	default:
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

// fakeRunsOpener hands out engine streams in order and counts the opens.
type fakeRunsOpener struct {
	streams []*fakeRunsStream
	opens   int
	openErr error
	mu      sync.Mutex
}

func (o *fakeRunsOpener) open(_ context.Context) (operator.RunStream, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.opens++

	if o.openErr != nil {
		return nil, o.openErr
	}

	s := newFakeRunsStream()
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
	m := newRunMux(&l, opener.open)
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
