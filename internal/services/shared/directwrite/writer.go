package directwrite

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
)

const (
	resultDirect         = "direct"
	resultFallbackNoSlot = "fallback_no_slot"
	resultFallbackError  = "fallback_error"
	resultDirectAlone    = "direct_alone"
)

// Writer applies task-processing messages in the calling process with the task controller's handler,
// instead of sending them over the durable task queue for the controller to consume. Messages are
// batched by tenant and message id with the same buffer the controller consumes the queue through, so
// the database sees the same batch sizes either way.
type Writer struct {
	mq        msgqueue.MessageQueue
	l         *zerolog.Logger
	handle    msgqueue.MsgHandler
	dst       msgqueue.DstFunc
	slots     chan struct{}
	stopBuf   func() error
	onDrained func()

	mu       sync.Mutex
	closed   bool
	inflight int
	drained  chan struct{}
}

// Opts bounds the database load a Writer adds to its process.
type Opts struct {
	// OnDrained runs in Close once every write has finished, before Close returns. It is where the
	// handler's own resources are released, such as the task controller's publish buffer that the
	// handler's monitoring events go through, since no write can publish into them afterwards.
	OnDrained func()

	// Slots caps how many messages are applied in-process at once; messages beyond it go over the
	// durable task queue. Zero sends every message over the queue, and a negative value removes the cap.
	Slots int
}

// New returns a Writer that applies messages with dst, normally the task controller's handler from
// task.NewMessageHandler, and sends them over mq when they can't be applied in-process. Call Close
// once the gRPC server has stopped and before mq disconnects: writes still in flight can fall back to
// mq, and the handler publishes its own events through it.
func New(dst msgqueue.DstFunc, mq msgqueue.MessageQueue, l *zerolog.Logger, opts Opts) (*Writer, error) {
	w := &Writer{
		mq:        mq,
		l:         l,
		dst:       dst,
		onDrained: opts.OnDrained,
		drained:   make(chan struct{}),
	}

	// An unbuffered channel never has a free slot (nothing receives until a slot is held), so zero
	// slots sends every message over the queue.
	if opts.Slots >= 0 {
		w.slots = make(chan struct{}, opts.Slots)
	}

	// The buffer hands each subscriber message to the handler it registers here, which blocks until the
	// message's batch is flushed and returns the batch's error. Registering it as the subscription turns
	// the consumer-side buffer into an in-process batcher with the controller's flush settings.
	buf := msgqueue.NewSubBufferFromSubscribe(func(preAck, _ msgqueue.MsgHandler) (func() error, error) {
		w.handle = preAck
		return func() error { return nil }, nil
	}, dst)

	stopBuf, err := buf.Start()

	if err != nil {
		return nil, err
	}

	w.stopBuf = stopBuf

	return w, nil
}

// Close stops applying messages in-process and waits, up to ctx, for the writes already in flight.
// Writes that arrive afterwards go straight over the task queue. The batching buffer is only stopped
// once nothing waits on it: it doesn't flush on shutdown, so stopping it early would leave those
// writes blocked forever. If ctx ends first, Close returns its error and leaves the buffer running.
func (w *Writer) Close(ctx context.Context) error {
	w.mu.Lock()

	if !w.closed {
		w.closed = true

		if w.inflight == 0 {
			close(w.drained)
		}
	}

	w.mu.Unlock()

	// checked on its own first, so an already expired ctx still finishes a writer with nothing in flight
	select {
	case <-w.drained:
	default:
		select {
		case <-w.drained:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if w.onDrained != nil {
		w.onDrained()
	}

	return w.stopBuf()
}

// begin registers a write that will use the batching buffer. It fails once Close has started, which
// sends the write over the queue instead.
func (w *Writer) begin() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return false
	}

	w.inflight++

	return true
}

func (w *Writer) end() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.inflight--

	if w.closed && w.inflight == 0 {
		close(w.drained)
	}
}

// Write applies msg in-process and returns once its batch has been handled. When the batch fails it
// runs msg once more on its own, and sends it over the durable task queue only if that fails too, or
// when every slot is taken, so an error only means the queue could not take it either. Sending a message the handler partly applied is safe: the controller
// already handles messages the queue redelivers after a failed attempt.
func (w *Writer) Write(ctx context.Context, msg *msgqueue.Message) error {
	return w.write(ctx, msg)
}

// TryWriteAsync applies msg in-process in the background when a slot is free, and returns false
// without doing anything otherwise. The caller then sends msg its own way, typically through a
// batching publish buffer, which keeps the batching and backpressure that a burst of overflow would
// lose if every message were published on its own. The background write holds its slot until it
// finishes, so slots also bound how many of these goroutines exist. It outlives the caller's context,
// since the caller typically returns before the batch flushes.
func (w *Writer) TryWriteAsync(ctx context.Context, msg *msgqueue.Message) bool {
	// Without a cap nothing would bound the goroutines, so uncapped writers leave async messages to the
	// caller's batching path.
	if w.slots == nil {
		return false
	}

	if !w.begin() {
		return false
	}

	if !w.tryAcquire() {
		w.end()
		w.record(msg.ID, resultFallbackNoSlot, 0)
		return false
	}

	ctx = context.WithoutCancel(ctx)

	go func() {
		defer w.end()

		// the only error apply can return here comes from sendToQueue, which already logs it
		_ = w.apply(ctx, msg)
	}()

	return true
}

func (w *Writer) write(ctx context.Context, msg *msgqueue.Message) error {
	if !w.begin() {
		return w.sendToQueue(ctx, msg)
	}

	defer w.end()

	if !w.tryAcquire() {
		w.record(msg.ID, resultFallbackNoSlot, 0)
		return w.sendToQueue(ctx, msg)
	}

	return w.apply(ctx, msg)
}

func (w *Writer) apply(ctx context.Context, msg *msgqueue.Message) error {
	start := time.Now()
	err := w.handle(msg)

	if err == nil {
		w.release()
		w.record(msg.ID, resultDirect, time.Since(start))
		return nil
	}

	// The buffer merges up to a batch of messages from different callers into one handler call and
	// hands its error to all of them, so one bad message fails its neighbours. Running this message
	// on its own, on the slot it still holds, keeps a healthy one off the queue. It is as safe as the
	// queue's redelivery of a failed batch, which also re-runs messages the handler partly applied.
	batchErr := err

	if err = w.dst(msg.TenantID, msg.ID, msg.Payloads); err == nil {
		w.release()
		w.record(msg.ID, resultDirectAlone, time.Since(start))

		// the batch error belongs to another message in the batch, which reports it when it fails alone
		w.l.Debug().Ctx(ctx).Err(batchErr).
			Str("message_id", msg.ID).
			Str("tenant_id", msg.TenantID.String()).
			Msg("direct task write applied on its own after its batch failed")

		return nil
	}

	w.release()

	w.record(msg.ID, resultFallbackError, time.Since(start))

	w.l.Warn().Ctx(ctx).Err(err).
		Str("message_id", msg.ID).
		Str("tenant_id", msg.TenantID.String()).
		Msg("direct task write failed, sending the message over the task queue")

	return w.sendToQueue(ctx, msg)
}

// fallbackAttempts bounds how often sendToQueue tries the publish. The queue backends don't retry a failed
// publish themselves (rabbitmq only retries acquiring a channel that turned out to be closed).
const fallbackAttempts = 3

// fallbackRetryBackoff is a var so tests can shorten it.
var fallbackRetryBackoff = 100 * time.Millisecond

// sendToQueue publishes on a context detached from the caller's. A handler can commit part of a message
// before it fails (CompleteTasks commits before the internal events are published), and the queue's
// redelivery is then the only thing that finishes it; the worker's RPC may be cancelled by the time the
// handler returns, which would otherwise fail this publish too. Every attempt shares one PUB_TIMEOUT budget,
// so a caller that is still waiting is held for at most that long.
func (w *Writer) sendToQueue(ctx context.Context, msg *msgqueue.Message) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), msgqueue.PUB_TIMEOUT)
	defer cancel()

	var err error

	for attempt := 1; attempt <= fallbackAttempts && ctx.Err() == nil; attempt++ {
		if attempt > 1 {
			select {
			case <-ctx.Done():
				continue
			case <-time.After(time.Duration(attempt-1) * fallbackRetryBackoff):
			}
		}

		if err = w.mq.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg); err == nil {
			return nil
		}
	}

	w.l.Error().Ctx(ctx).Err(err).
		Str("message_id", msg.ID).
		Str("tenant_id", msg.TenantID.String()).
		Msg("could not send task message over the task queue")

	return err
}

func (w *Writer) tryAcquire() bool {
	if w.slots == nil {
		return true
	}

	select {
	case w.slots <- struct{}{}:
		return true
	default:
		return false
	}
}

func (w *Writer) release() {
	if w.slots != nil {
		<-w.slots
	}
}

func (w *Writer) record(msgId, result string, d time.Duration) {
	prometheus.DirectTaskWrites.WithLabelValues(msgId, result).Inc()

	if result != resultFallbackNoSlot {
		prometheus.DirectTaskWriteDuration.WithLabelValues(msgId, result).Observe(d.Seconds())
	}
}
