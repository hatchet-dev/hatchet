package dispatcher

import (
	"context"
	"io"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/services/shared/rpcstream"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type statusEntry struct {
	task       string
	branch     int64
	node       int64
	invocation int32
}

func statusOf(entries ...statusEntry) *contracts.DurableTaskWorkerStatusRequest {
	req := &contracts.DurableTaskWorkerStatusRequest{}

	for _, e := range entries {
		req.WaitingEntries = append(req.WaitingEntries, &contracts.DurableTaskAwaitedCompletedEntry{
			DurableTaskExternalId: e.task,
			BranchId:              e.branch,
			NodeId:                e.node,
			InvocationCount:       e.invocation,
		})
	}

	return req
}

func entriesOf(req *contracts.DurableTaskWorkerStatusRequest) []statusEntry {
	if req == nil {
		return nil
	}

	out := make([]statusEntry, 0, len(req.WaitingEntries))

	for _, e := range req.WaitingEntries {
		out = append(out, statusEntry{task: e.DurableTaskExternalId, branch: e.BranchId, node: e.NodeId, invocation: e.InvocationCount})
	}

	return out
}

func TestWorkerStatusCoalescerMerge(t *testing.T) {
	tests := []struct {
		name     string
		statuses [][]statusEntry
		want     []statusEntry
	}{
		{
			name: "union_across_statuses",
			statuses: [][]statusEntry{
				{{"a", 1, 1, 1}},
				{{"a", 1, 2, 1}, {"b", 1, 1, 1}},
			},
			want: []statusEntry{{"a", 1, 1, 1}, {"a", 1, 2, 1}, {"b", 1, 1, 1}},
		},
		{
			name: "dedupe_by_branch_and_node",
			statuses: [][]statusEntry{
				{{"a", 1, 1, 1}, {"a", 2, 1, 1}},
				{{"a", 1, 1, 1}, {"a", 1, 1, 1}},
			},
			want: []statusEntry{{"a", 1, 1, 1}, {"a", 2, 1, 1}},
		},
		{
			name: "higher_invocation_replaces_only_its_task",
			statuses: [][]statusEntry{
				{{"a", 1, 1, 1}, {"a", 1, 2, 1}, {"b", 1, 1, 1}},
				{{"a", 1, 3, 2}},
			},
			want: []statusEntry{{"a", 1, 3, 2}, {"b", 1, 1, 1}},
		},
		{
			name: "lower_invocation_is_dropped",
			statuses: [][]statusEntry{
				{{"a", 1, 1, 2}},
				{{"a", 1, 2, 1}},
			},
			want: []statusEntry{{"a", 1, 1, 2}},
		},
		{
			name: "mixed_invocations_within_one_status",
			statuses: [][]statusEntry{
				{{"a", 1, 1, 1}, {"a", 1, 2, 3}, {"a", 1, 3, 2}},
			},
			want: []statusEntry{{"a", 1, 2, 3}},
		},
		{
			name:     "empty_status_yields_nothing",
			statuses: [][]statusEntry{{}},
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newWorkerStatusCoalescer()

			for _, status := range tt.statuses {
				c.merge(statusOf(status...))
			}

			got := c.next()

			if tt.want == nil {
				assert.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			assert.ElementsMatch(t, tt.want, entriesOf(got))
		})
	}
}

func TestWorkerStatusCoalescerRunnerLifecycle(t *testing.T) {
	c := newWorkerStatusCoalescer()

	assert.False(t, c.merge(statusOf()), "an empty status does not start a runner")
	assert.False(t, c.merge(&contracts.DurableTaskWorkerStatusRequest{WaitingEntries: []*contracts.DurableTaskAwaitedCompletedEntry{nil}}), "a status of nil entries does not start a runner")
	assert.False(t, c.merge(nil), "a nil status does not start a runner")

	assert.True(t, c.merge(statusOf(statusEntry{"a", 1, 1, 1})), "first merge with entries starts the runner")
	assert.False(t, c.merge(statusOf(statusEntry{"a", 1, 2, 1})), "merge while running does not start another runner")

	assert.ElementsMatch(t, []statusEntry{{"a", 1, 1, 1}, {"a", 1, 2, 1}}, entriesOf(c.next()))

	// a merge that lands while the runner is handling the previous batch is picked up by next
	assert.False(t, c.merge(statusOf(statusEntry{"b", 1, 1, 1})))
	assert.ElementsMatch(t, []statusEntry{{"b", 1, 1, 1}}, entriesOf(c.next()))

	assert.Nil(t, c.next(), "next returns nil once drained")
	assert.Nil(t, c.next(), "draining twice stays drained")

	assert.True(t, c.merge(statusOf(statusEntry{"c", 1, 1, 1})), "a merge after the runner finished starts a new one")
	assert.ElementsMatch(t, []statusEntry{{"c", 1, 1, 1}}, entriesOf(c.next()))
}

// TestWorkerStatusCoalescerConcurrentHandoff races one producer, as on the stream loop, against
// the runners it starts, so merges land while a runner is handling a batch and while it is
// going idle. Every merged entry must be handled and batches must never be handled in parallel.
func TestWorkerStatusCoalescerConcurrentHandoff(t *testing.T) {
	const statuses = 2_000

	c := newWorkerStatusCoalescer()

	var (
		runners   sync.WaitGroup
		mu        sync.Mutex
		seen      = make(map[statusEntry]struct{}, statuses)
		handling  atomic.Int32
		overlaps  atomic.Int32
		runnerRun atomic.Int32
	)

	runner := func() {
		defer runners.Done()

		runnerRun.Add(1)

		for {
			batch := c.next()
			if batch == nil {
				return
			}

			if handling.Add(1) > 1 {
				overlaps.Add(1)
			}

			mu.Lock()
			for _, e := range entriesOf(batch) {
				seen[e] = struct{}{}
			}
			mu.Unlock()

			runtime.Gosched()

			handling.Add(-1)
		}
	}

	want := make([]statusEntry, 0, statuses)

	for i := range statuses {
		e := statusEntry{task: strconv.Itoa(i), branch: 1, node: 1, invocation: 1}
		want = append(want, e)

		if c.merge(statusOf(e)) {
			runners.Add(1)
			go runner()
		}

		if i%7 == 0 {
			runtime.Gosched()
		}
	}

	runners.Wait()

	assert.Zero(t, overlaps.Load(), "batches were handled in parallel")
	assert.Positive(t, runnerRun.Load())

	mu.Lock()
	defer mu.Unlock()

	assert.Len(t, seen, statuses, "every merged entry is handled")

	for _, e := range want {
		assert.Contains(t, seen, e)
	}

	assert.Nil(t, c.next(), "nothing is left pending")
}

func TestWorkerStatusCoalescerCapsBatchesAtWholeTasks(t *testing.T) {
	// 5 tasks of 4_000 entries is over the cap in total, and any 3 of them are too, so a batch
	// holds exactly 2 whole tasks
	const tasks, perTask = 5, 4_000

	c := newWorkerStatusCoalescer()

	for i := range tasks {
		entries := make([]statusEntry, perTask)
		for n := range entries {
			entries[n] = statusEntry{task: string(rune('a' + i)), branch: 1, node: int64(n), invocation: 1}
		}

		c.merge(statusOf(entries...))
	}

	var batchSizes []int

	seen := make(map[statusEntry]struct{}, tasks*perTask)

	for batch := c.next(); batch != nil; batch = c.next() {
		got := entriesOf(batch)
		batchSizes = append(batchSizes, len(got))

		perTaskCount := map[string]int{}

		for _, e := range got {
			perTaskCount[e.task]++
			seen[e] = struct{}{}
		}

		for task, n := range perTaskCount {
			assert.Equal(t, perTask, n, "task %s must go out whole", task)
		}
	}

	assert.Equal(t, []int{8_000, 8_000, 4_000}, batchSizes)
	assert.Len(t, seen, tasks*perTask)

	assert.True(t, c.merge(statusOf(statusEntry{"z", 1, 1, 1})), "running is cleared once next returned nil")
}

func TestWorkerStatusCoalescerOversizedTaskGoesOutWhole(t *testing.T) {
	c := newWorkerStatusCoalescer()

	entries := make([]statusEntry, maxWorkerStatusEntriesPerBatch+5)
	for n := range entries {
		entries[n] = statusEntry{task: "big", branch: 1, node: int64(n), invocation: 1}
	}

	c.merge(statusOf(entries...))
	c.merge(statusOf(statusEntry{"small", 1, 1, 1}))

	first := entriesOf(c.next())
	second := entriesOf(c.next())

	require.Nil(t, c.next())

	// whichever comes first, the big task is never split and never shares a batch with the small one
	bySize := map[int]string{}

	for _, batch := range [][]statusEntry{first, second} {
		for _, e := range batch {
			assert.Equal(t, batch[0].task, e.task)
		}

		bySize[len(batch)] = batch[0].task
	}

	assert.Equal(t, map[int]string{len(entries): "big", 1: "small"}, bySize)
}

// fakeSatisfiedEvents records every GetSatisfiedDurableEvents call and returns one satisfied
// row per call. It ignores the inserted-at range on purpose: the row is always returned so the
// handler sends it and reaches the blocked stream.
type fakeSatisfiedEvents struct {
	v1.DurableEventsRepository

	calls [][]v1.TaskExternalIdNodeIdBranchId
	mu    sync.Mutex
}

func (f *fakeSatisfiedEvents) GetSatisfiedDurableEvents(_ context.Context, _ uuid.UUID, events []v1.TaskExternalIdNodeIdBranchId, _ v1.TaskInsertedAtRange) ([]*v1.SatisfiedEventWithPayload, error) {
	f.mu.Lock()
	f.calls = append(f.calls, append([]v1.TaskExternalIdNodeIdBranchId(nil), events...))
	f.mu.Unlock()

	return []*v1.SatisfiedEventWithPayload{{
		TaskExternalId:  events[0].TaskExternalId,
		NodeID:          events[0].NodeId,
		BranchID:        events[0].BranchId,
		InvocationCount: 1,
		Result:          []byte("payload"),
	}}, nil
}

func (f *fakeSatisfiedEvents) snapshot() [][]v1.TaskExternalIdNodeIdBranchId {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([][]v1.TaskExternalIdNodeIdBranchId(nil), f.calls...)
}

type fakeNoTasks struct{ v1.TaskRepository }

func (fakeNoTasks) FlattenExternalIds(context.Context, uuid.UUID, []uuid.UUID) ([]*sqlcv1.FlattenExternalIdsRow, error) {
	return nil, nil
}

type fakeWorkerStatusRepo struct {
	v1.Repository
	events *fakeSatisfiedEvents
}

func (f fakeWorkerStatusRepo) Tasks() v1.TaskRepository                  { return fakeNoTasks{} }
func (f fakeWorkerStatusRepo) DurableEvents() v1.DurableEventsRepository { return f.events }

// blockingSendServer is a stream whose peer has stopped reading: Send blocks until released.
type blockingSendServer struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
	sends   atomic.Int32
}

func newBlockingSendServer() *blockingSendServer {
	return &blockingSendServer{entered: make(chan struct{}, 1), release: make(chan struct{})}
}

func (s *blockingSendServer) Send(*contracts.DurableTaskResponse) error {
	s.sends.Add(1)

	select {
	case s.entered <- struct{}{}:
	default:
	}

	<-s.release

	return nil
}

func (s *blockingSendServer) unblock() { s.once.Do(func() { close(s.release) }) }

// statusSession drives durableTask with a scripted receive side.
type statusSession struct {
	d      *DispatcherServiceImpl
	events *fakeSatisfiedEvents
	stream *blockingSendServer
	cancel context.CancelFunc
	in     chan *contracts.DurableTaskRequest
	done   chan error
	// closed once the durableTask goroutine has fully returned
	finished chan struct{}
	// Receive call n+1 starts only after message n was handed to the loop, and the loop takes
	// message n only after it finished processing message n-1.
	recvCalls atomic.Int32
	pushed    int32
}

func startStatusSession(t *testing.T) *statusSession {
	t.Helper()

	s := &statusSession{
		d:      newTestDispatcher(),
		events: &fakeSatisfiedEvents{},
		stream: newBlockingSendServer(),
		in:     make(chan *contracts.DurableTaskRequest),
		done:   make(chan error, 1),

		finished: make(chan struct{}),
	}
	s.d.repo = fakeWorkerStatusRepo{events: s.events}

	var ctx context.Context
	ctx, s.cancel = context.WithCancel(tenantContext())

	t.Cleanup(func() {
		s.stream.unblock()
		s.cancel()

		select {
		case <-s.finished:
		case <-time.After(5 * time.Second):
			t.Error("durableTask goroutine outlived the test")
		}
	})

	receive := func() (*contracts.DurableTaskRequest, error) {
		s.recvCalls.Add(1)

		select {
		case req := <-s.in:
			return req, nil
		case <-ctx.Done():
			return nil, io.EOF
		}
	}

	go func() {
		defer close(s.finished)

		sender := rpcstream.NewSender[contracts.DurableTaskResponse](context.Background(), s.stream)
		defer sender.Close()

		s.done <- s.d.durableTask(ctx, receive, sender)
	}()

	return s
}

func (s *statusSession) push(t *testing.T, req *contracts.DurableTaskRequest) {
	t.Helper()

	select {
	case s.in <- req:
		s.pushed++
	case <-time.After(5 * time.Second):
		t.Fatal("durableTask stopped receiving")
	}
}

func (s *statusSession) pushStatus(t *testing.T, taskId uuid.UUID, node int64) {
	t.Helper()

	s.push(t, &contracts.DurableTaskRequest{
		Message: &contracts.DurableTaskRequest_WorkerStatus{
			WorkerStatus: statusOf(statusEntry{task: taskId.String(), branch: 1, node: node, invocation: 1}),
		},
	})
}

// settle pushes a message with no payload and waits until the loop has taken it, so every
// message pushed before it has been fully processed by the loop goroutine.
func (s *statusSession) settle(t *testing.T) {
	t.Helper()

	s.push(t, &contracts.DurableTaskRequest{})

	require.Eventually(t, func() bool { return s.recvCalls.Load() > s.pushed }, 5*time.Second, time.Millisecond)
}

// A stream whose peer stopped reading must not accumulate one handler per worker status.
func TestDurableTaskWorkerStatusIsSingleFlightAndMerged(t *testing.T) {
	const total = 50

	s := startStatusSession(t)

	taskIds := make([]uuid.UUID, total)
	for i := range taskIds {
		taskIds[i] = uuid.New()
	}

	s.pushStatus(t, taskIds[0], 0)

	select {
	case <-s.stream.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("first status never reached Send")
	}

	for i := 1; i < total; i++ {
		s.pushStatus(t, taskIds[i], int64(i))
	}

	s.settle(t)

	calls := s.events.snapshot()
	require.Len(t, calls, 1, "statuses arriving while the stream is blocked must not start more handlers")
	assert.Len(t, calls[0], 1)

	s.stream.unblock()

	require.Eventually(t, func() bool { return len(s.events.snapshot()) == 2 }, 5*time.Second, time.Millisecond)

	calls = s.events.snapshot()

	got := make(map[v1.TaskExternalIdNodeIdBranchId]struct{}, total-1)
	for _, w := range calls[1] {
		got[w] = struct{}{}
	}

	want := make(map[v1.TaskExternalIdNodeIdBranchId]struct{}, total-1)
	for i := 1; i < total; i++ {
		want[v1.TaskExternalIdNodeIdBranchId{TaskExternalId: taskIds[i], BranchId: 1, NodeId: int64(i)}] = struct{}{}
	}

	assert.Len(t, calls[1], total-1)
	assert.Equal(t, want, got)

	// the merged batch is the last one: the runner exits once nothing is pending
	require.Eventually(t, func() bool { return s.stream.sends.Load() == 2 }, 5*time.Second, time.Millisecond)

	assert.Len(t, s.events.snapshot(), 2)
}

// Cancelling the session while a send is stuck must end the session without handling the
// statuses that were still pending.
func TestDurableTaskWorkerStatusRunnerStopsOnSessionCancel(t *testing.T) {
	s := startStatusSession(t)

	s.pushStatus(t, uuid.New(), 0)

	select {
	case <-s.stream.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("first status never reached Send")
	}

	for i := 1; i < 5; i++ {
		s.pushStatus(t, uuid.New(), int64(i))
	}

	s.settle(t)

	s.cancel()
	s.stream.unblock()

	select {
	case err := <-s.done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("durableTask did not return after cancel and unblocked send")
	}

	assert.Len(t, s.events.snapshot(), 1, "pending statuses must not be handled after the session ended")
}
