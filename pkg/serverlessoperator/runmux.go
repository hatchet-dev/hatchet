package serverlessoperator

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/client/retry"
	"github.com/hatchet-dev/hatchet/pkg/operator"
)

// runMuxQueueSize bounds the events queued for one subscriber that reads slower than the
// engine sends; the fan-out blocks on it, which the relay's own backpressure resolves by
// closing the socket, and with it the subscriber.
const runMuxQueueSize = 64

// runMux multiplexes one SubscribeToWorkflowRuns stream per tenant session over every socket
// of the tenant that asks for a run's result, the way the SDK's child listener shares one
// stream per process. The engine stream is opened on the first subscriber and reopened with
// backoff when it fails, with every run still awaited subscribed again; a run's terminal
// event is delivered to every subscriber that asked for it and ends the interest in the run.
//
// A subscriber is an operator.RunStream: Send subscribes a run id, Recv yields the events of
// the runs it subscribed, Close forgets its runs. The mux itself is closed with the
// registration that owns it.
type runMux struct {
	open    func(ctx context.Context) (operator.RunStream, error)
	l       *zerolog.Logger
	backoff func(ctx context.Context, attempt int) error

	ctx    context.Context
	cancel context.CancelFunc

	// mu guards everything below. stream is the current engine stream, nil while none is
	// open; subs is every awaited run and the handles that await it.
	mu      sync.Mutex
	stream  operator.RunStream
	subs    map[string]map[*runMuxHandle]struct{}
	loop    chan struct{}
	started bool
	closed  bool
}

func newRunMux(l *zerolog.Logger, open func(ctx context.Context) (operator.RunStream, error)) *runMux {
	ctx, cancel := context.WithCancel(context.Background())

	return &runMux{
		open:    open,
		l:       l,
		backoff: retry.SleepStreamBackoff,
		ctx:     ctx,
		cancel:  cancel,
		subs:    map[string]map[*runMuxHandle]struct{}{},
		loop:    make(chan struct{}),
	}
}

// Subscribe hands out a subscriber; first, when set, is its first subscription.
func (m *runMux) Subscribe(ctx context.Context, first proto.Message) (operator.RunStream, error) {
	m.mu.Lock()

	if m.closed {
		m.mu.Unlock()
		return nil, operator.ErrSessionClosed
	}

	if !m.started {
		m.started = true

		go m.run()
	}

	m.mu.Unlock()

	h := &runMuxHandle{
		mux:    m,
		queue:  make(chan proto.Message, runMuxQueueSize),
		closed: make(chan struct{}),
		runs:   map[string]struct{}{},
	}

	if first != nil {
		if err := h.Send(ctx, first); err != nil {
			_ = h.Close()
			return nil, err
		}
	}

	return h, nil
}

// run keeps one engine stream open while the mux lives: it opens the stream with backoff,
// subscribes every awaited run on it, and fans its events out until it fails.
func (m *runMux) run() {
	defer close(m.loop)

	for attempt := 0; ; attempt++ {
		if attempt > 0 {
			if err := m.backoff(m.ctx, attempt-1); err != nil {
				return
			}
		}

		if m.ctx.Err() != nil {
			return
		}

		stream, err := m.open(m.ctx)

		if err != nil {
			if m.ctx.Err() != nil {
				return
			}

			m.l.Warn().Err(err).Int("attempt", attempt+1).Msg("could not open the workflow runs stream; retrying")

			continue
		}

		if err := m.install(stream); err != nil {
			_ = stream.Close()

			if m.ctx.Err() != nil {
				return
			}

			m.l.Warn().Err(err).Int("attempt", attempt+1).Msg("could not resubscribe on the workflow runs stream; reopening")

			continue
		}

		attempt = m.serve(stream, attempt)

		if m.ctx.Err() != nil {
			return
		}
	}
}

// install publishes stream as the current one and replays every awaited run on it.
func (m *runMux) install(stream operator.RunStream) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return operator.ErrSessionClosed
	}

	for runId := range m.subs {
		if err := stream.Send(m.ctx, &contracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: runId}); err != nil {
			return err
		}
	}

	m.stream = stream

	return nil
}

// serve fans the stream's events out until it fails, then retires it. It returns the
// attempt count the next open starts from: zero once the stream delivered anything.
func (m *runMux) serve(stream operator.RunStream, attempt int) int {
	for {
		msg, err := stream.Recv(m.ctx)

		if err != nil {
			m.mu.Lock()

			if m.stream == stream {
				m.stream = nil
			}

			m.mu.Unlock()
			_ = stream.Close()

			if m.ctx.Err() == nil {
				m.l.Warn().Err(err).Msg("workflow runs stream ended; reopening")
			}

			return attempt
		}

		attempt = 0

		event, ok := msg.(*contracts.WorkflowRunEvent)

		if !ok {
			continue
		}

		m.deliver(event)
	}
}

// deliver hands the event to every handle awaiting its run and, on the terminal event, forgets
// the run. The push blocks on a full handle until the handle closes or the mux ends.
func (m *runMux) deliver(event *contracts.WorkflowRunEvent) {
	m.mu.Lock()
	handles := make([]*runMuxHandle, 0, len(m.subs[event.GetWorkflowRunId()]))

	for h := range m.subs[event.GetWorkflowRunId()] {
		handles = append(handles, h)
	}

	if event.GetEventType() == contracts.WorkflowRunEventType_WORKFLOW_RUN_EVENT_TYPE_FINISHED {
		delete(m.subs, event.GetWorkflowRunId())

		for _, h := range handles {
			h.forget(event.GetWorkflowRunId())
		}
	}

	m.mu.Unlock()

	for _, h := range handles {
		h.push(event)
	}
}

// subscribe records the handle's interest in the run and sends the subscription on the
// current stream; without one the install of the next stream sends it.
func (m *runMux) subscribe(ctx context.Context, h *runMuxHandle, runId string) error {
	m.mu.Lock()

	if m.closed {
		m.mu.Unlock()
		return operator.ErrSessionClosed
	}

	handles, ok := m.subs[runId]

	if !ok {
		handles = map[*runMuxHandle]struct{}{}
		m.subs[runId] = handles
	}

	handles[h] = struct{}{}
	h.remember(runId)

	stream := m.stream
	m.mu.Unlock()

	if stream == nil {
		return nil
	}

	if err := stream.Send(ctx, &contracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: runId}); err != nil {
		// the stream is failing; serve retires it and the reopen replays the subscription
		m.l.Debug().Err(err).Str("workflow_run_id", runId).Msg("subscription send failed; it is replayed on the next stream")
	}

	return nil
}

// unsubscribe forgets the handle's interest in every run it awaited.
func (m *runMux) unsubscribe(h *runMuxHandle) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for runId := range h.runs {
		delete(m.subs[runId], h)

		if len(m.subs[runId]) == 0 {
			delete(m.subs, runId)
		}
	}
}

// close ends the mux: the engine stream is closed, the loop joined, every handle closed.
func (m *runMux) close() {
	m.mu.Lock()

	if m.closed {
		m.mu.Unlock()
		return
	}

	m.closed = true
	started := m.started
	stream := m.stream
	m.stream = nil

	handles := map[*runMuxHandle]struct{}{}

	for _, hs := range m.subs {
		for h := range hs {
			handles[h] = struct{}{}
		}
	}

	m.mu.Unlock()

	m.cancel()

	if stream != nil {
		_ = stream.Close()
	}

	if started {
		<-m.loop
	}

	for h := range handles {
		_ = h.Close()
	}
}

// runMuxHandle is one subscriber of the mux.
type runMuxHandle struct {
	mux    *runMux
	queue  chan proto.Message
	closed chan struct{}
	// runs is the handle's awaited runs; guarded by the mux's mu.
	runs      map[string]struct{}
	closeOnce sync.Once
}

func (h *runMuxHandle) remember(runId string) { h.runs[runId] = struct{}{} }
func (h *runMuxHandle) forget(runId string)   { delete(h.runs, runId) }

func (h *runMuxHandle) push(event *contracts.WorkflowRunEvent) {
	select {
	case h.queue <- event:
	case <-h.closed:
	case <-h.mux.ctx.Done():
	}
}

// Send implements operator.RunStream: the message is a subscription for one run.
func (h *runMuxHandle) Send(ctx context.Context, msg proto.Message) error {
	select {
	case <-h.closed:
		return operator.ErrChannelClosed
	default:
	}

	req, ok := msg.(*contracts.SubscribeToWorkflowRunsRequest)

	if !ok {
		return fmt.Errorf("%s takes a SubscribeToWorkflowRunsRequest, not a %T", operator.RunStreamWorkflowRuns, msg)
	}

	if req.GetWorkflowRunId() == "" {
		return errors.New("a subscription names a workflow run")
	}

	return h.mux.subscribe(ctx, h, req.GetWorkflowRunId())
}

// Recv implements operator.RunStream. A closed handle reports so before anything else, so
// the mux closing (which closes every handle) reads the same whichever signal wins.
func (h *runMuxHandle) Recv(ctx context.Context) (proto.Message, error) {
	select {
	case <-h.closed:
		return nil, operator.ErrChannelClosed
	default:
	}

	select {
	case event := <-h.queue:
		return event, nil
	case <-h.closed:
		return nil, operator.ErrChannelClosed
	case <-h.mux.ctx.Done():
		return nil, operator.ErrSessionClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close implements operator.RunStream.
func (h *runMuxHandle) Close() error {
	h.closeOnce.Do(func() {
		close(h.closed)
		h.mux.unsubscribe(h)
	})

	return nil
}
