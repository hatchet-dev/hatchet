package directwrite

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
)

const (
	resultDirect         = "direct"
	resultFallbackNoSlot = "fallback_no_slot"
	resultFallbackError  = "fallback_error"
	resultError          = "error"
)

// Writer applies task-processing messages in the calling process with the task controller's handler,
// instead of sending them over the durable task queue for the controller to consume. Messages are
// batched by tenant and message id with the same buffer the controller consumes the queue through, so
// the database sees the same batch sizes either way.
type Writer struct {
	mq       msgqueue.MessageQueue
	l        *zerolog.Logger
	handle   msgqueue.MsgHandler
	slots    chan struct{}
	fallback bool
}

// Opts bounds the database load a Writer adds to its process and sets what a failed write does.
type Opts struct {
	// Slots caps how many messages are applied in-process at once; messages beyond it go over the
	// durable task queue. Zero means no cap.
	Slots int

	// Fallback sends a message over the durable task queue when the handler fails, instead of
	// returning the error to the caller. WriteAsync always falls back, since it has no caller to
	// return the error to.
	Fallback bool
}

// New returns a Writer that applies messages with dst, normally the task controller's handler from
// task.NewMessageHandler, and sends them over mq when they can't be applied in-process. The Writer has
// no cleanup: its batching buffer is never stopped, since stopping it would strand writes still waiting
// on a flush, and the gRPC server keeps accepting worker events while the engine's services shut down.
func New(dst msgqueue.DstFunc, mq msgqueue.MessageQueue, l *zerolog.Logger, opts Opts) (*Writer, error) {
	w := &Writer{
		mq:       mq,
		l:        l,
		fallback: opts.Fallback,
	}

	if opts.Slots > 0 {
		w.slots = make(chan struct{}, opts.Slots)
	}

	// The buffer hands each subscriber message to the handler it registers here, which blocks until the
	// message's batch is flushed and returns the batch's error. Registering it as the subscription turns
	// the consumer-side buffer into an in-process batcher with the controller's flush settings.
	buf := msgqueue.NewSubBufferFromSubscribe(func(preAck, _ msgqueue.MsgHandler) (func() error, error) {
		w.handle = preAck
		return func() error { return nil }, nil
	}, dst)

	if _, err := buf.Start(); err != nil {
		return nil, err
	}

	return w, nil
}

// Write applies msg in-process and returns once its batch has been handled. It sends msg over the
// durable task queue instead when every slot is taken, or when the handler fails and fallback is
// enabled. Sending a message the handler partly applied is safe: the controller already handles
// messages the queue redelivers after a failed attempt.
func (w *Writer) Write(ctx context.Context, msg *msgqueue.Message) error {
	return w.write(ctx, msg, w.fallback)
}

// WriteAsync is Write for senders that don't wait for the outcome. It outlives the caller's context,
// since the caller typically returns before the batch flushes. A failed write always falls back to the
// durable task queue, even with fallback disabled: nothing is left to retry the message, so returning
// the error would drop it.
func (w *Writer) WriteAsync(ctx context.Context, msg *msgqueue.Message) {
	ctx = context.WithoutCancel(ctx)

	go func() {
		if err := w.write(ctx, msg, true); err != nil {
			w.l.Error().Ctx(ctx).Err(err).
				Str("message_id", msg.ID).
				Str("tenant_id", msg.TenantID.String()).
				Msg("could not write task message")
		}
	}()
}

func (w *Writer) write(ctx context.Context, msg *msgqueue.Message, fallback bool) error {
	if !w.tryAcquire() {
		w.record(msg.ID, resultFallbackNoSlot, 0)
		return w.mq.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg)
	}

	start := time.Now()
	err := w.handle(msg)
	w.release()

	if err == nil {
		w.record(msg.ID, resultDirect, time.Since(start))
		return nil
	}

	if !fallback {
		w.record(msg.ID, resultError, time.Since(start))
		return err
	}

	w.record(msg.ID, resultFallbackError, time.Since(start))

	w.l.Warn().Ctx(ctx).Err(err).
		Str("message_id", msg.ID).
		Str("tenant_id", msg.TenantID.String()).
		Msg("direct task write failed, sending the message over the task queue")

	return w.mq.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg)
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
