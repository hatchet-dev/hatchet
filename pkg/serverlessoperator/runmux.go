package serverlessoperator

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"connectrpc.com/connect"
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

// runMuxSendQueueSize bounds the subscriptions queued for the engine stream's writer. A
// healthy stream drains it at once; a stream that has not accepted that many subscriptions
// is stalled and is retired.
const runMuxSendQueueSize = 1024

// errRunMuxRetired is install's report that the stream it replayed on already carries an
// interest nobody holds: a subscriber closed during the replay. The loop opens another.
var errRunMuxRetired = errors.New("the workflow runs stream was retired during its replay")

// runMux multiplexes one SubscribeToWorkflowRuns stream per tenant session over every socket
// of the tenant that asks for a run's result, the way the SDK's child listener shares one
// stream per process. The engine stream is opened on the first subscriber and reopened with
// backoff when it fails, with every run still awaited subscribed again; a run's terminal
// event is delivered to every subscriber that asked for it and ends the interest in the run.
//
// A subscriber is an operator.RunStream: Send subscribes a run id, Recv yields the events of
// the runs it subscribed, Close forgets its runs. A subscriber awaits at most maxRuns runs
// at once; past that, a subscription is refused with ResourceExhausted. The engine has no
// unsubscribe message and keeps polling every run it was told of until the stream ends, so
// when a subscriber closes while it still awaits runs nobody else holds, the engine stream is
// retired and reopened with the runs still awaited; closes that overlap share one reopen. The
// mux itself is closed with the registration that owns it.
//
// A subscriber never sends on the engine stream itself: the stream's writer does, from a
// bounded queue, since a send the transport holds on flow control is released only by the
// stream's close, and a subscriber whose socket ended must be able to close meanwhile.
type runMux struct {
	open      func(ctx context.Context) (operator.RunStream, error)
	l         *zerolog.Logger
	backoff   func(ctx context.Context, attempt int) error
	maxRuns   int
	sendQueue int

	ctx    context.Context
	cancel context.CancelFunc

	// mu guards everything below and is never held across a send on the engine stream, so
	// close can always reach the stream whose Close releases a blocked send. stream is the
	// current engine stream, nil while none is open, and writer owns its sends; opening is
	// the stream the loop is replaying on, held here from the moment open returns until it
	// is published as stream or given up, so close finds a stream whose replay is blocked.
	// subs is every awaited run and the handles that await it.
	mu      sync.Mutex
	stream  operator.RunStream
	writer  *runMuxWriter
	opening operator.RunStream
	subs    map[string]map[*runMuxHandle]struct{}
	loop    chan struct{}
	started bool
	closed  bool
}

func newRunMux(l *zerolog.Logger, maxRuns int, open func(ctx context.Context) (operator.RunStream, error)) *runMux {
	ctx, cancel := context.WithCancel(context.Background())

	return &runMux{
		open:      open,
		l:         l,
		backoff:   retry.SleepStreamBackoff,
		maxRuns:   maxRuns,
		sendQueue: runMuxSendQueueSize,
		ctx:       ctx,
		cancel:    cancel,
		subs:      map[string]map[*runMuxHandle]struct{}{},
		loop:      make(chan struct{}),
	}
}

// runMuxWriter sends the subscriptions queued for one published engine stream, in order,
// until the stream is retired or a send fails. It is the stream's only sender once the
// stream is published; the replay before that is the loop's.
type runMuxWriter struct {
	stream operator.RunStream
	queue  chan string
	// stop is closed when the stream is retired; done when the writer has left.
	stop chan struct{}
	done chan struct{}
}

func newRunMuxWriter(stream operator.RunStream, size int) *runMuxWriter {
	return &runMuxWriter{
		stream: stream,
		queue:  make(chan string, size),
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
}

// run drains the queue onto the stream. A send that fails ends the writer: the stream is
// failing, serve retires it on the Recv error that follows, and the reopen replays every
// awaited run, the queued ones among them.
func (w *runMuxWriter) run(m *runMux) {
	defer close(w.done)

	for {
		select {
		case <-w.stop:
			return
		case runId := <-w.queue:
			if err := w.stream.Send(m.ctx, &contracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: runId}); err != nil {
				m.l.Debug().Err(err).Str("workflow_run_id", runId).Msg("subscription send failed; it is replayed on the next stream")
				return
			}
		}
	}
}

// retireLocked takes the current stream out of service, stops its writer and returns it for
// the caller to close outside mu; nil when no stream is current.
func (m *runMux) retireLocked() operator.RunStream {
	stream := m.stream

	if stream == nil {
		return nil
	}

	m.stream = nil
	close(m.writer.stop)
	m.writer = nil

	return stream
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

// run keeps one engine stream open while the mux lives: it opens the stream, backing off
// after consecutive failures, subscribes every awaited run on it, and fans its events out
// until it fails or is retired. A retired stream is replaced at once; a failed one counts
// toward the backoff unless it delivered something first.
func (m *runMux) run() {
	defer close(m.loop)

	failures := 0

	for {
		if failures > 0 {
			if err := m.backoff(m.ctx, failures-1); err != nil {
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

			failures++
			m.l.Warn().Err(err).Int("attempt", failures).Msg("could not open the workflow runs stream; retrying")

			continue
		}

		if err := m.install(stream); err != nil {
			_ = stream.Close()

			if m.ctx.Err() != nil {
				return
			}

			if errors.Is(err, errRunMuxRetired) {
				continue
			}

			failures++
			m.l.Warn().Err(err).Int("attempt", failures).Msg("could not resubscribe on the workflow runs stream; reopening")

			continue
		}

		failed, delivered := m.serve(stream)

		if m.ctx.Err() != nil {
			return
		}

		switch {
		case !failed:
			failures = 0
		case delivered:
			failures = 1
		default:
			failures++
		}
	}
}

// install replays every awaited run on stream, then publishes it as the current one with
// its writer. The sends run outside the lock, since flow control may block them and close
// must be able to close the stream that releases them; runs subscribed during the replay are
// sent before the stream is published, so nothing a subscriber recorded while no stream was
// current is left out, and a run the replay sent that nobody awaits any more retires the
// stream instead of publishing it. The stream is held as opening throughout, so a close
// during the replay closes it; the caller closes the stream on an error.
func (m *runMux) install(stream operator.RunStream) error {
	m.mu.Lock()

	if m.closed {
		m.mu.Unlock()
		return operator.ErrSessionClosed
	}

	m.opening = stream
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()

		if m.opening == stream {
			m.opening = nil
		}

		m.mu.Unlock()
	}()

	sent := map[string]struct{}{}

	for {
		m.mu.Lock()

		if m.closed {
			m.mu.Unlock()
			return operator.ErrSessionClosed
		}

		pending := make([]string, 0, len(m.subs))

		for runId := range m.subs {
			if _, ok := sent[runId]; !ok {
				pending = append(pending, runId)
			}
		}

		if len(pending) == 0 {
			for runId := range sent {
				if _, ok := m.subs[runId]; !ok {
					m.mu.Unlock()
					return errRunMuxRetired
				}
			}

			w := newRunMuxWriter(stream, m.sendQueue)
			m.stream = stream
			m.writer = w
			m.opening = nil
			m.mu.Unlock()

			go w.run(m)

			return nil
		}

		m.mu.Unlock()

		for _, runId := range pending {
			if err := stream.Send(m.ctx, &contracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: runId}); err != nil {
				return err
			}

			sent[runId] = struct{}{}
		}
	}
}

// serve fans the stream's events out until it ends. It reports whether the stream failed on
// its own, as opposed to being retired by a subscriber's close or the mux's, and whether it
// delivered anything first.
func (m *runMux) serve(stream operator.RunStream) (failed, delivered bool) {
	for {
		msg, err := stream.Recv(m.ctx)

		if err != nil {
			m.mu.Lock()
			failed = m.stream == stream

			if failed {
				m.retireLocked()
			}

			m.mu.Unlock()
			_ = stream.Close()

			if failed && m.ctx.Err() == nil {
				m.l.Warn().Err(err).Msg("workflow runs stream ended; reopening")
			}

			return failed, delivered
		}

		delivered = true

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

// subscribe records the handle's interest in the run and queues the subscription for the
// current stream's writer; without a current stream the install of the next one sends it.
// It never waits on the engine stream: a writer that has not drained a full queue is behind
// a stalled stream, which is retired so the reopen replays every awaited run, this one among
// them. A handle at its cap is refused, unless it already awaits the run.
func (m *runMux) subscribe(_ context.Context, h *runMuxHandle, runId string) error {
	m.mu.Lock()

	if m.closed {
		m.mu.Unlock()
		return operator.ErrSessionClosed
	}

	if _, awaited := h.runs[runId]; !awaited && len(h.runs) >= m.maxRuns {
		m.mu.Unlock()
		return connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("the stream already awaits %d runs, the limit", m.maxRuns))
	}

	handles, ok := m.subs[runId]

	if !ok {
		handles = map[*runMuxHandle]struct{}{}
		m.subs[runId] = handles
	}

	handles[h] = struct{}{}
	h.remember(runId)

	w := m.writer
	m.mu.Unlock()

	if w == nil {
		return nil
	}

	select {
	case w.queue <- runId:
		return nil
	default:
	}

	m.mu.Lock()

	var retired operator.RunStream

	if m.writer == w {
		retired = m.retireLocked()
	}

	m.mu.Unlock()

	if retired != nil {
		m.l.Warn().Int("queued", m.sendQueue).Msg("the workflow runs stream has not accepted a full queue of subscriptions; reopening it")
		_ = retired.Close()
	}

	return nil
}

// unsubscribe forgets the handle's interest in every run it awaited. A run left with no
// other holder is one the engine stream still carries, so the stream is retired: serve
// finds it gone and the loop opens another with the runs still awaited. While no stream is
// current, the replay in progress notices the orphan itself.
func (m *runMux) unsubscribe(h *runMuxHandle) {
	m.mu.Lock()

	orphaned := false

	for runId := range h.runs {
		delete(m.subs[runId], h)

		if len(m.subs[runId]) == 0 {
			delete(m.subs, runId)

			orphaned = true
		}
	}

	var retired operator.RunStream

	if orphaned && !m.closed {
		retired = m.retireLocked()
	}

	m.mu.Unlock()

	if retired != nil {
		_ = retired.Close()
	}
}

// close ends the mux: the engine stream is closed, the loop and the writer joined, every
// handle closed. The stream being replayed on is closed with the current one, since a send
// blocked on flow control, the replay's or the writer's, is released by its stream's Close
// alone, not by the mux context; without that neither would ever be joined.
func (m *runMux) close() {
	m.cancel()

	m.mu.Lock()

	if m.closed {
		m.mu.Unlock()
		return
	}

	m.closed = true
	started := m.started
	writer := m.writer
	stream := m.retireLocked()
	opening := m.opening
	m.opening = nil

	handles := map[*runMuxHandle]struct{}{}

	for _, hs := range m.subs {
		for h := range hs {
			handles[h] = struct{}{}
		}
	}

	m.mu.Unlock()

	if stream != nil {
		_ = stream.Close()
	}

	if opening != nil {
		_ = opening.Close()
	}

	if started {
		<-m.loop
	}

	if writer != nil {
		<-writer.done
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

// Send implements operator.RunStream: the message is a subscription for one run. It never
// waits on the engine stream.
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
