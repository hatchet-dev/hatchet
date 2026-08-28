// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

// Invariant tests for the reconnect handoff, driven through the public
// listener API (the internals are a blackbox)
//
// The properties under test:
//
//	Every AddWorkflowRun that returns no error has its run ID sent to the server.
//	A successful registration remains receivable after a clean stream stop.

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
)

// recordingStream is a mock subscribe stream that records every run ID it
// accepts, so tests can assert which subscriptions each stream instance
// received. It can also deliver server events (deliver) and models the ways
// a real stream dies:
//
//   - breakRecv: server hangup. Recv fails with Unavailable, but the
//     half-dead stream still accepts local Sends without error.
//   - breakAll: full transport failure. Sends fail too.
//   - stopClean: server ends the stream with codes.Canceled, which the
//     client classifies as a clean stop rather than a failure.
type recordingStream struct {
	onSend   func(s *recordingStream, runID string)
	recvDead chan struct{}
	events   chan *dispatchercontracts.WorkflowRunEvent
	recvErr  error
	sent     map[string]int
	id       int
	mu       sync.Mutex
	recvOnce sync.Once
	sendDead atomic.Bool
}

func (s *recordingStream) Send(req *dispatchercontracts.SubscribeToWorkflowRunsRequest) error {
	if s.sendDead.Load() {
		return status.Error(codes.Unavailable, "send on broken stream")
	}
	if s.onSend != nil {
		s.onSend(s, req.WorkflowRunId)
	}
	s.mu.Lock()
	s.sent[req.WorkflowRunId]++
	s.mu.Unlock()
	return nil
}

func (s *recordingStream) Recv() (*dispatchercontracts.WorkflowRunEvent, error) {
	select {
	case ev := <-s.events:
		return ev, nil
	case <-s.recvDead:
		return nil, s.recvErr
	}
}

// deliver hands a server event to the stream's next Recv.
func (s *recordingStream) deliver(ev *dispatchercontracts.WorkflowRunEvent) {
	s.events <- ev
}

// breakRecvWith ends the receive side exactly once with the given error;
// later calls (including CloseSend) keep the first error.
func (s *recordingStream) breakRecvWith(err error) {
	s.recvOnce.Do(func() {
		s.recvErr = err
		close(s.recvDead)
	})
}

func (s *recordingStream) breakRecv() {
	s.breakRecvWith(status.Error(codes.Unavailable, "recv on broken stream"))
}

func (s *recordingStream) stopClean() {
	s.breakRecvWith(status.Error(codes.Canceled, "server ended the stream cleanly"))
}

func (s *recordingStream) breakAll() {
	s.sendDead.Store(true)
	s.breakRecv()
}

func (s *recordingStream) saw(runID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sent[runID] > 0
}

// CloseSend marks the stream retired; unblocking Recv mirrors the server
// closing its side after the client half-closes.
func (s *recordingStream) CloseSend() error {
	s.breakRecv()
	return nil
}

func (s *recordingStream) Header() (metadata.MD, error)  { return nil, nil }
func (s *recordingStream) Trailer() metadata.MD          { return nil }
func (s *recordingStream) Context() context.Context      { return context.Background() }
func (s *recordingStream) SendMsg(msg interface{}) error { return nil }
func (s *recordingStream) RecvMsg(msg interface{}) error { return nil }

// recordingStreamFactory hands out numbered recordingStreams from the
// listener's constructor seam and keeps every instance for later inspection.
type recordingStreamFactory struct {
	onSend  func(s *recordingStream, runID string)
	streams []*recordingStream
	mu      sync.Mutex
}

func (f *recordingStreamFactory) constructor(context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := &recordingStream{
		id:       len(f.streams),
		onSend:   f.onSend,
		recvDead: make(chan struct{}),
		events:   make(chan *dispatchercontracts.WorkflowRunEvent, 1),
		sent:     map[string]int{},
	}
	f.streams = append(f.streams, s)
	return s, nil
}

func (f *recordingStreamFactory) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.streams)
}

func (f *recordingStreamFactory) last() *recordingStream {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.streams) == 0 {
		return nil
	}
	return f.streams[len(f.streams)-1]
}

func (f *recordingStreamFactory) get(i int) *recordingStream {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.streams[i]
}

func nopWorkflowRunHandler(WorkflowRunEvent) error { return nil }

func waitOrFatal(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

// TestRegistrationRemainsReceivableAfterCleanStreamStop asserts that a
// successful AddWorkflowRun remains receivable after the current receive
// stream stops cleanly: the listener must open a replacement stream, replay
// the run ID, and deliver the run's event to the registered handler.
//
// codes.Canceled exercises the clean-stop classification (the path where a
// pre-fix listener exited and orphaned its registrations); this does not
// claim that this exact race caused #4711.
func TestRegistrationRemainsReceivableAfterCleanStreamStop(t *testing.T) {
	logger := zerolog.Nop()
	factory := &recordingStreamFactory{}
	listener := newTestWorkflowRunsListener(t, &logger, factory.constructor, nil)

	received := make(chan WorkflowRunEvent, 1)
	require.NoError(t, listener.AddWorkflowRun("run-1", "session-1", func(ev WorkflowRunEvent) error {
		received <- ev
		return nil
	}))
	require.Equal(t, 1, factory.count())
	require.True(t, factory.get(0).saw("run-1"))

	// The registration has succeeded; only now does the server stop the
	// stream cleanly, so the ordering is deterministic.
	factory.get(0).stopClean()

	// Registered work must keep the listener alive: a replacement stream
	// appears and the run ID is replayed onto it.
	require.Eventually(t, func() bool {
		return factory.count() >= 2 && factory.get(1).saw("run-1")
	}, 5*time.Second, time.Millisecond,
		"listener stopped after a clean stream stop instead of reconnecting for its registered run")

	// The event delivered on the replacement stream reaches the handler
	// registered before the stop.
	factory.get(1).deliver(&dispatchercontracts.WorkflowRunEvent{WorkflowRunId: "run-1"})
	select {
	case ev := <-received:
		assert.Equal(t, "run-1", ev.WorkflowRunId)
	case <-time.After(5 * time.Second):
		t.Fatal("handler registered before the clean stop never received its event")
	}

	require.NoError(t, listener.Close())
}

// TestAddWorkflowRunDuringReconnectLandsOnReplacement holds a reconnect open
// mid-replay (at the stream boundary: the replacement's first Send blocks)
// and registers a run while the handoff window is open. However the
// registration interleaves with the handoff, it must end up on the
// replacement stream.
func TestAddWorkflowRunDuringReconnectLandsOnReplacement(t *testing.T) {
	logger := zerolog.Nop()
	replayStarted := make(chan struct{})
	releaseReplay := make(chan struct{})
	var replayOnce sync.Once

	factory := &recordingStreamFactory{
		// Stream 1 is the replacement; park its first send (the replay)
		// until releaseReplay closes.
		onSend: func(s *recordingStream, runID string) {
			if s.id == 1 {
				replayOnce.Do(func() { close(replayStarted) })
				<-releaseReplay
			}
		},
	}
	listener := newTestWorkflowRunsListener(t, &logger, factory.constructor, nil)

	// First registration creates stream 0 and starts the background run loop.
	require.NoError(t, listener.AddWorkflowRun("existing-run", "existing-session", nopWorkflowRunHandler))
	require.Equal(t, 1, factory.count())
	require.True(t, factory.get(0).saw("existing-run"))

	// Server hangup on stream 0: the run loop reconnects and replays onto
	// stream 1, where the first Send parks. Sends on stream 0 still succeed.
	factory.get(0).breakRecv()
	waitOrFatal(t, replayStarted, "replay to reach the replacement stream")

	// The handoff is now provably mid-replay; register a run inside it.
	addErr := make(chan error, 1)
	go func() {
		addErr <- listener.AddWorkflowRun("late-run", "late-session", nopWorkflowRunHandler)
	}()

	// Give the registration time to reach the window. If it misses, it
	// sends on the installed replacement and the assertions below still
	// legitimately hold, so this can never cause a false failure.
	time.Sleep(20 * time.Millisecond)
	close(releaseReplay)

	select {
	case err := <-addErr:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for AddWorkflowRun to return")
	}

	replacement := factory.get(1)
	assert.True(t, replacement.saw("existing-run"), "replay must re-send existing registrations on the replacement")
	assert.True(t, replacement.saw("late-run"), "registration made during the reconnect handoff must reach the replacement stream")

	require.NoError(t, listener.Close())
}

// TestWorkflowRunsListenerReconnectChaosPreservesRegistrations drives many
// concurrent registrations against randomized stream breaks and asserts the
// quiescence invariant. Schedules are seeded; a failure logs the seed and can
// be replayed with HATCHET_CHAOS_SEED=<seed> go test -run ReconnectChaos.
func TestWorkflowRunsListenerReconnectChaosPreservesRegistrations(t *testing.T) {
	seed := time.Now().UnixNano()
	if env := os.Getenv("HATCHET_CHAOS_SEED"); env != "" {
		parsed, err := strconv.ParseInt(env, 10, 64)
		require.NoError(t, err, "HATCHET_CHAOS_SEED must be an int64")
		seed = parsed
	}
	t.Logf("chaos seed: %d (replay with HATCHET_CHAOS_SEED=%d)", seed, seed)

	const (
		adders       = 4
		runsPerAdder = 10
	)

	// Break the stream when registration progress crosses these fractions,
	// so handoffs always overlap in-flight registrations. The last handoff
	// matters most: a registration lost in an earlier one is healed by the
	// next replay, so only a race against the final handoff stays visible.
	breakAt := []int32{
		int32(0.2 * adders * runsPerAdder),
		int32(0.5 * adders * runsPerAdder),
		int32(0.9 * adders * runsPerAdder),
	}

	logger := zerolog.Nop()

	// Every send pays a seeded random latency, as on a real network. This
	// widens the handoff window (replay is a sequence of sends) enough for
	// registrations to actually interleave with it.
	latencyRng := rand.New(rand.NewSource(seed + 2000)) // nolint: gosec // deterministic schedule exploration, not crypto
	var latencyMu sync.Mutex
	factory := &recordingStreamFactory{
		onSend: func(*recordingStream, string) {
			latencyMu.Lock()
			d := time.Duration(latencyRng.Intn(200)) * time.Microsecond
			latencyMu.Unlock()
			time.Sleep(d)
		},
	}
	listener := newTestWorkflowRunsListener(t, &logger, factory.constructor, nil)

	var wg sync.WaitGroup
	var issued atomic.Int32
	addErrs := make([][]error, adders)
	for g := 0; g < adders; g++ {
		addErrs[g] = make([]error, runsPerAdder)
		rng := rand.New(rand.NewSource(seed + int64(g))) // nolint: gosec // deterministic schedule exploration, not crypto
		wg.Add(1)
		go func(g int, rng *rand.Rand) {
			defer wg.Done()
			for i := 0; i < runsPerAdder; i++ {
				time.Sleep(time.Duration(rng.Intn(2000)) * time.Microsecond)
				runID := fmt.Sprintf("run-%d-%d", g, i)
				issued.Add(1)
				addErrs[g][i] = listener.AddWorkflowRun(runID, runID+"-session", nopWorkflowRunHandler)
			}
		}(g, rng)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		rng := rand.New(rand.NewSource(seed + 1000)) // nolint: gosec // deterministic schedule exploration, not crypto
		for _, threshold := range breakAt {
			for issued.Load() < threshold {
				time.Sleep(100 * time.Microsecond)
			}
			time.Sleep(time.Duration(rng.Intn(500)) * time.Microsecond)
			s := factory.last()
			if s == nil {
				continue
			}
			if rng.Intn(10) < 7 {
				s.breakRecv() // server hangup: sends still succeed locally
			} else {
				s.breakAll() // full transport failure
			}
		}
	}()

	wg.Wait()

	// Quiesce and identify the current stream from the outside: a sentinel
	// registration lands on the installed stream; the state is stable once no
	// new stream appeared during the sentinel and the newest stream took it.
	var current *recordingStream
	sentinels := 0
	require.Eventually(t, func() bool {
		sentinels++
		before := factory.count()
		sentinel := fmt.Sprintf("sentinel-%d", sentinels)
		if err := listener.AddWorkflowRun(sentinel, sentinel+"-session", nopWorkflowRunHandler); err != nil {
			return false
		}
		newest := factory.last()
		if factory.count() != before || !newest.saw(sentinel) {
			return false
		}
		current = newest
		return true
	}, 5*time.Second, time.Millisecond, "listener never quiesced onto a stable current stream")

	require.True(t, listener.isListening(), "background run loop must survive the chaos")
	for g := 0; g < adders; g++ {
		for i := 0; i < runsPerAdder; i++ {
			runID := fmt.Sprintf("run-%d-%d", g, i)
			require.NoErrorf(t, addErrs[g][i], "AddWorkflowRun(%s) failed (seed %d)", runID, seed)
			assert.Truef(t, current.saw(runID),
				"registration %s missing from current stream %d after %d reconnects (seed %d)",
				runID, current.id, factory.count()-1, seed)
		}
	}

	require.NoError(t, listener.Close())
}
