package operatorclient

import (
	"context"
	"net"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client/streaming"
)

// droppingDeltaServer is an OperatorService whose first Listen stream ends with Unavailable
// right after it receives a delta, before applying it, and whose later streams apply and
// acknowledge deltas. Register resumes the same worker from the second call on.
type droppingDeltaServer struct {
	v1.UnimplementedOperatorServiceServer

	starts  atomic.Int32
	applied atomic.Int32
	// dropped is closed when the first stream has discarded its delta
	dropped chan struct{}

	mu    sync.Mutex
	seen  [][]string
	acked []uint64
}

func newDroppingDeltaServer() *droppingDeltaServer {
	return &droppingDeltaServer{dropped: make(chan struct{})}
}

func (d *droppingDeltaServer) Register(context.Context, *v1.OperatorRegisterRequest) (*v1.OperatorRegisterResponse, error) {
	return registeredAs("same-worker", d.starts.Load() > 0), nil
}

func (d *droppingDeltaServer) Listen(stream v1.OperatorService_ListenServer) error {
	n := d.starts.Add(1)

	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}

		// the pause the session's Close sends is acknowledged like the engine would
		if pause := req.GetPause(); pause != nil {
			if err := stream.Send(&v1.OperatorListenResponse{
				Message: &v1.OperatorListenResponse_PauseAck{PauseAck: &v1.OperatorPauseAck{Paused: pause.Paused}},
			}); err != nil {
				return err
			}

			continue
		}

		delta := req.GetActions()
		if delta == nil {
			continue
		}

		if n == 1 {
			close(d.dropped)
			return status.Error(codes.Unavailable, "connection lost before the delta was committed")
		}

		d.applied.Add(int32(len(delta.Add))) // nolint: gosec

		d.mu.Lock()
		d.seen = append(d.seen, delta.Add)
		d.acked = append(d.acked, delta.Sequence)
		d.mu.Unlock()

		if err := stream.Send(&v1.OperatorListenResponse{
			Message: &v1.OperatorListenResponse_Ack{Ack: &v1.OperatorActionsAck{Sequence: delta.Sequence}},
		}); err != nil {
			return err
		}
	}
}

func newBufconnOperatorSession(t *testing.T, srv v1.OperatorServiceServer) *session {
	t.Helper()

	lis := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	v1.RegisterOperatorServiceServer(server, srv)

	go func() { _ = server.Serve(lis) }()

	conn, err := grpc.NewClient(
		"passthrough:///operator",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
	)
	require.NoError(t, err)

	l := zerolog.Nop()
	s := newSession(v1.NewOperatorServiceClient(conn), nil, newCallMetadata("token", nil), &l, &v1.OperatorRegisterRequest{Name: "acks"}, true)
	s.heartbeatInterval = 10 * time.Millisecond
	s.actions.interval = time.Millisecond
	s.stream.SetSleep(func(context.Context, int) error { return nil })

	t.Cleanup(func() {
		_ = s.Close()
		conn.Close()
		server.Stop()
	})

	return s
}

// A delta the stream accepted but the engine never committed is not lost when the worker is
// resumed: the client keeps it until it is acknowledged and resends it on the new stream.
func TestOperatorSessionResendsUnacknowledgedDeltaAfterResume(t *testing.T) {
	srv := newDroppingDeltaServer()
	s := newBufconnOperatorSession(t, srv)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	require.NoError(t, s.connect(ctx))

	s.AddActions("svc:run")
	require.NoError(t, s.Flush(ctx), "Flush returns once the resent delta is acknowledged")

	select {
	case <-srv.dropped:
	case <-ctx.Done():
		t.Fatal("the first stream never received the delta")
	}

	assert.EqualValues(t, 2, srv.starts.Load(), "the delta drop reconnected the stream")
	assert.EqualValues(t, 1, srv.applied.Load(), "the resumed stream applied the resent delta")
	assert.Empty(t, s.actions.unackedSequences(), "the acknowledged delta is no longer retained")

	srv.mu.Lock()
	defer srv.mu.Unlock()
	assert.Equal(t, [][]string{{"svc:run"}}, srv.seen)
	assert.Equal(t, []uint64{1}, srv.acked, "the resent delta keeps its sequence")
}

// A resumed reconnect resends every unacknowledged delta in order and drops them once the
// new stream acknowledges them.
func TestOperatorSessionResumedReplayResendsUnackedChunks(t *testing.T) {
	client := &fakeOperatorServiceClient{
		registrations: []*v1.OperatorRegisterResponse{registeredAs("worker-1", false), registeredAs("worker-1", true)},
		noAck:         true,
	}
	s, _ := connectTestOperatorSession(t, client, true)
	s.actions.maxChunk = 2

	s.AddActions("svc:a", "svc:b", "svc:c")
	waitFor(t, func() bool { return len(client.stream(0).deltas()) == 2 }, "chunks were not sent")

	// the first chunk is acknowledged, the second is not when the stream dies
	client.stream(0).ack(1)
	waitFor(t, func() bool { return len(s.actions.unackedSequences()) == 1 }, "ack was not observed")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	flushErr := make(chan error, 1)
	go func() { flushErr <- s.Flush(ctx) }()

	client.stream(0).breakRecv()

	waitFor(t, func() bool { return client.streamCount() >= 2 && len(client.stream(1).deltas()) == 1 }, "unacked chunk was not replayed")
	second := client.stream(1)
	replayed := second.deltas()[0]
	assert.Equal(t, []string{"svc:c"}, replayed.Add)
	assert.EqualValues(t, 2, replayed.Sequence, "a replayed chunk keeps its sequence")

	select {
	case err := <-flushErr:
		t.Fatalf("Flush returned %v before the replayed chunk was acknowledged", err)
	case <-time.After(50 * time.Millisecond):
	}

	second.ack(2)

	select {
	case err := <-flushErr:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal("Flush did not return after the ack")
	}

	assert.Empty(t, s.actions.unackedSequences())
}

// A non-resumed reconnect replaces the queue with the desired set snapshot: a removal that
// runs while the snapshot is on the wire is queued relative to the snapshot instead of being
// coalesced away with the add it never sent.
func TestOperatorSessionFreshWorkerReplayPreservesConcurrentRemoval(t *testing.T) {
	q := &actionDeltaQueue{
		desired:  map[string]struct{}{},
		pending:  map[string]actionDeltaOp{},
		wake:     make(chan struct{}, 1),
		idle:     make(chan struct{}),
		done:     make(chan struct{}),
		maxChunk: 1000,
	}

	q.add([]string{"svc:run"})

	stream := &removingReplayStream{q: q}

	require.NoError(t, q.replay(stream, false))
	assert.Equal(t, []string{"svc:run"}, stream.added, "the fresh worker receives the snapshot")

	staged := q.takeChunk()
	require.NotNil(t, staged, "the removal that raced the replay must still be sent")
	assert.Equal(t, []string{"svc:run"}, staged.delta.Remove)
	assert.Empty(t, staged.delta.Add)
	assert.Empty(t, q.desiredSet())
}

// removingReplayStream removes every replayed action from the queue while the replay send is
// in progress, modelling a RemoveActions call that races the replay.
type removingReplayStream struct {
	v1.OperatorService_ListenClient

	q     *actionDeltaQueue
	added []string
}

func (s *removingReplayStream) Send(req *v1.OperatorListenRequest) error {
	s.added = append(s.added, req.GetActions().Add...)
	s.q.remove(req.GetActions().Add)

	return nil
}

// A delta that waits too long for its ack hangs the stream up so the reconnect replays it.
func TestOperatorSessionAckTimeoutReconnectsAndReplays(t *testing.T) {
	client := &fakeOperatorServiceClient{
		registrations: []*v1.OperatorRegisterResponse{registeredAs("worker-1", false), registeredAs("worker-1", true)},
		noAck:         true,
	}
	s, _ := connectTestOperatorSession(t, client, true)
	s.actions.ackTimeout = 20 * time.Millisecond

	s.AddActions("svc:a")

	waitFor(t, func() bool { return client.streamCount() >= 2 && len(client.stream(1).deltas()) == 1 }, "the unacknowledged delta was not replayed on a new stream")
	assert.Equal(t, []string{"svc:a"}, client.stream(1).deltas()[0].Add)

	client.stream(1).ack(1)
	flushed(t, s)
}

// Actions and Close may race: the consumer's WaitGroup count is added under the same lock
// Close takes, so Close never waits on a count that is still being added.
func TestOperatorSessionActionsAndCloseConcurrently(t *testing.T) {
	for i := 0; i < 200; i++ {
		client := &fakeOperatorServiceClient{registrations: []*v1.OperatorRegisterResponse{registeredAs("worker", false)}}
		s, _ := newTestOperatorSession(t, client, true)
		require.NoError(t, s.connect(context.Background()))

		gate := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			<-gate
			_, _, _ = s.Actions(context.Background())
		}()

		go func() {
			defer wg.Done()
			<-gate
			_ = s.Close()
		}()

		close(gate)
		wg.Wait()

		_, _, err := s.Actions(context.Background())
		assert.ErrorIs(t, err, streaming.ErrListenerClosed, "Actions after Close is refused")
	}
}

// setStream is a Listen stream that applies deltas to a set the way the engine does and
// acknowledges them at once, recording the order sequences went out in.
type setStream struct {
	v1.OperatorService_ListenClient

	q    *actionDeltaQueue
	have map[string]bool
	seqs []uint64
}

func newSetStream(q *actionDeltaQueue) *setStream {
	return &setStream{q: q, have: map[string]bool{}}
}

func (s *setStream) Send(req *v1.OperatorListenRequest) error {
	d := req.GetActions()
	s.seqs = append(s.seqs, d.Sequence)

	for _, id := range d.Add {
		s.have[id] = true
	}

	for _, id := range d.Remove {
		delete(s.have, id)
	}

	s.q.ack(d.Sequence)

	return nil
}

func (s *setStream) set() []string {
	out := make([]string, 0, len(s.have))

	for id := range s.have {
		out = append(out, id)
	}

	sort.Strings(out)

	return out
}

func newStagedQueue(desired ...string) *actionDeltaQueue {
	q := &actionDeltaQueue{
		desired:  map[string]struct{}{},
		pending:  map[string]actionDeltaOp{},
		wake:     make(chan struct{}, 1),
		idle:     make(chan struct{}),
		done:     make(chan struct{}),
		maxChunk: 1000,
	}

	for _, id := range desired {
		q.desired[id] = struct{}{}
	}

	return q
}

// sendStaged is the flusher's send of a chunk it staged with takeChunk, as run performs it
// under the send lock: a chunk staged under an older stream generation is not sent.
func sendStaged(q *actionDeltaQueue, stream v1.OperatorService_ListenClient, staged *stagedChunk) error {
	if !q.currentGeneration(staged.generation) {
		q.finishChunk(nil)
		return nil
	}

	err := stream.Send(&v1.OperatorListenRequest{Message: &v1.OperatorListenRequest_Actions{Actions: staged.delta}})
	q.finishChunk(err)

	return err
}

// A chunk the flusher staged before a fresh-worker replay is superseded by the replay's
// snapshot: sending it afterwards would undo the snapshot. The remove staged here must not be
// sent after the snapshot that re-adds the action.
func TestOperatorSessionFreshReplayDropsStagedRemoval(t *testing.T) {
	q := newStagedQueue("svc:run")

	q.remove([]string{"svc:run"})
	staged := q.takeChunk()
	require.NotNil(t, staged)
	require.Equal(t, []string{"svc:run"}, staged.delta.Remove)

	// the action is wanted again before the reconnect; the flusher has not sent its removal
	q.add([]string{"svc:run"})

	stream := newSetStream(q)
	require.NoError(t, q.replay(stream, false))
	require.NoError(t, sendStaged(q, stream, staged))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	require.NoError(t, q.flush(ctx))
	assert.Equal(t, q.desiredSet(), stream.set(), "the engine's set is the desired set")
	assert.Equal(t, []uint64{2}, stream.seqs, "only the snapshot went out")
}

// The reverse direction: an add staged before the replay must not be sent after the snapshot
// that omits the action, since it was removed in between.
func TestOperatorSessionFreshReplayDropsStagedAdd(t *testing.T) {
	q := newStagedQueue()

	q.add([]string{"svc:run"})
	staged := q.takeChunk()
	require.NotNil(t, staged)
	require.Equal(t, []string{"svc:run"}, staged.delta.Add)

	q.remove([]string{"svc:run"})

	stream := newSetStream(q)
	require.NoError(t, q.replay(stream, false))
	require.NoError(t, sendStaged(q, stream, staged))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	require.NoError(t, q.flush(ctx))
	assert.Empty(t, stream.set(), "the removed action never reached the engine")
	assert.Empty(t, stream.seqs, "an empty desired set replays nothing and the staged add is dropped")
}

// A resumed replay resends the staged chunk itself: the flusher's own send of it is skipped, so
// the chunk goes out exactly once.
func TestOperatorSessionResumedReplaySendsStagedChunkOnce(t *testing.T) {
	q := newStagedQueue()

	q.add([]string{"svc:run"})
	staged := q.takeChunk()
	require.NotNil(t, staged)

	stream := newSetStream(q)
	require.NoError(t, q.replay(stream, true))
	require.NoError(t, sendStaged(q, stream, staged))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	require.NoError(t, q.flush(ctx))
	assert.Equal(t, []string{"svc:run"}, stream.set())
	assert.Equal(t, []uint64{1}, stream.seqs, "the staged chunk went out once, by the replay")
}
