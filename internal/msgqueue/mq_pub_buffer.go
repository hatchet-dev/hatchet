package msgqueue

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/syncx"
)

// nolint: staticcheck
var (
	PUB_FLUSH_INTERVAL  = 10 * time.Millisecond
	PUB_BUFFER_SIZE     = 10
	PUB_MAX_CONCURRENCY = 1
	PUB_TIMEOUT         = 10 * time.Second
)

type PubFunc func(m *Message) error

// MQPubBuffer buffers messages coming out of the task queue, groups them by tenantId and msgId, and then flushes them
// to the task handler as necessary.
type MQPubBuffer struct {
	mq MessageQueue

	l *zerolog.Logger

	// buffers is keyed on a composite (tenantId, msgId) and contains a buffer of messages for that tenantId and msgId.
	buffers syncx.Map[string, *msgIdPubBuffer]

	ctx    context.Context
	cancel context.CancelFunc
}

type mqPubBufferOpts struct {
	l *zerolog.Logger
}

type mqPubBufferOptFunc func(*mqPubBufferOpts)

func WithPubLogger(l *zerolog.Logger) mqPubBufferOptFunc {
	return func(opts *mqPubBufferOpts) { opts.l = l }
}

func NewMQPubBuffer(mq MessageQueue, fs ...mqPubBufferOptFunc) *MQPubBuffer {
	opts := &mqPubBufferOpts{l: &defaultLogger}
	for _, f := range fs {
		f(opts)
	}

	ctx, cancel := context.WithCancel(context.Background())
	m := &MQPubBuffer{mq: mq, l: opts.l, ctx: ctx, cancel: cancel}
	go m.runEvictor(ctx)
	return m
}

// runEvictor periodically removes buffers that have not seen a message for
// BUFFER_IDLE_TIMEOUT. Without eviction the buffers map grows with every
// (queue, tenantId, msgId) key seen over the lifetime of the process.
func (m *MQPubBuffer) runEvictor(ctx context.Context) {
	ticker := time.NewTicker(BUFFER_IDLE_TIMEOUT / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.buffers.Range(func(k string, b *msgIdPubBuffer) bool {
				if len(b.msgIdPubBufferCh) == 0 && b.tryEvict(BUFFER_IDLE_TIMEOUT) {
					m.buffers.Delete(k)
					b.stop()
				}

				return true
			})
		}
	}
}

func (m *MQPubBuffer) Stop() {
	m.cancel()
}

type msgWithErrCh struct {
	msg   *Message
	errCh chan error
}

func (m *MQPubBuffer) Pub(ctx context.Context, queue Queue, msg *Message, wait bool) error {
	if msg.TenantID == uuid.Nil {
		return nil
	}

	k := getPubKey(queue, msg.TenantID, msg.ID)

	var msgBuf *msgIdPubBuffer

	for {
		var ok bool
		msgBuf, ok = m.buffers.Load(k)

		if !ok {
			newBuf := newMsgIDPubBuffer(m.ctx, m.l, queue, msg.TenantID, msg.ID, func(msg *Message) error {
				msgCtx, cancel := context.WithTimeout(context.Background(), PUB_TIMEOUT)
				defer cancel()
				return m.mq.SendMessage(msgCtx, queue, msg)
			})

			var loaded bool
			msgBuf, loaded = m.buffers.LoadOrStore(k, newBuf)

			if loaded {
				// lost the store race; stop the discarded buffer's goroutines
				newBuf.stop()
			}
		}

		// the buffer may have been evicted between the load and the acquire, in
		// which case it is no longer in the map and we create a fresh one
		if msgBuf.tryAcquire() {
			break
		}
	}

	msgWithErr := &msgWithErrCh{msg: msg}
	if wait {
		msgWithErr.errCh = make(chan error)
	}

	// Signal early flush if the channel is already at capacity, since the send below may block.
	if len(msgBuf.msgIdPubBufferCh) >= msgBuf.bufferSize {
		select {
		case msgBuf.capacityRelease <- struct{}{}:
		default:
		}
	}

	// this places some backpressure on the consumer if buffers are full
	msgBuf.msgIdPubBufferCh <- msgWithErr
	msgBuf.notifier <- struct{}{}
	msgBuf.release()

	if wait {
		return <-msgWithErr.errCh
	}

	return nil
}

func getPubKey(q Queue, tenantId uuid.UUID, msgId string) string {
	return q.Name() + tenantId.String() + msgId
}

type msgIdPubBuffer struct {
	*bufferCore

	l *zerolog.Logger

	queue            Queue
	tenantId         uuid.UUID
	msgId            string
	msgIdPubBufferCh chan *msgWithErrCh
	pub              PubFunc
}

func newMsgIDPubBuffer(ctx context.Context, l *zerolog.Logger, queue Queue, tenantID uuid.UUID, msgID string, pub PubFunc) *msgIdPubBuffer {
	ctx, stop := context.WithCancel(ctx)

	b := &msgIdPubBuffer{
		bufferCore:       newBufferCore(PUB_FLUSH_INTERVAL, PUB_BUFFER_SIZE, PUB_MAX_CONCURRENCY, false, true),
		l:                l,
		queue:            queue,
		tenantId:         tenantID,
		msgId:            msgID,
		msgIdPubBufferCh: make(chan *msgWithErrCh, PUB_BUFFER_SIZE),
		pub:              pub,
	}
	b.stop = stop
	b.startFlusher(ctx, func() int { return len(b.msgIdPubBufferCh) }, b.flush)
	b.startSemaphoreReleaser(ctx, func() int { return len(b.msgIdPubBufferCh) }, b.flush)
	return b
}

func (m *msgIdPubBuffer) flush() {
	select {
	case m.semaphore <- struct{}{}:
	default:
		return
	}

	startedFlush := time.Now()
	defer func() {
		go func() {
			m.semaphoreRelease <- m.flushInterval - time.Since(startedFlush)
		}()
	}()

	drained := drainN(m.msgIdPubBufferCh, m.bufferSize)
	if len(drained) == 0 {
		return
	}

	payloadBytes := make([][]byte, 0, len(drained))
	var isPersistent *bool
	var immediatelyExpire *bool
	var retries *int
	numWaiters := 0

	for _, item := range drained {
		payloadBytes = append(payloadBytes, item.msg.Payloads...)
		if isPersistent == nil {
			isPersistent = &item.msg.Persistent
		}
		if immediatelyExpire == nil {
			immediatelyExpire = &item.msg.ImmediatelyExpire
		}
		if retries == nil {
			retries = &item.msg.Retries
		}
		if item.errCh != nil {
			numWaiters++
		}
	}

	msgToSend := &Message{
		TenantID: m.tenantId,
		ID:       m.msgId,
		Payloads: payloadBytes,
	}
	if isPersistent != nil {
		msgToSend.Persistent = *isPersistent
	}
	if immediatelyExpire != nil {
		msgToSend.ImmediatelyExpire = *immediatelyExpire
	}
	if retries != nil {
		msgToSend.Retries = *retries
	}

	err := m.pub(msgToSend)

	// the batch is not retried: a backend error does not prove the broker
	// rejected it (see the amqp091 publish contract and the chunked publish
	// path), and OLAP ingestion has no durable dedupe. callers publishing with
	// wait=false never see the error, so this log is the only record of the loss
	if err != nil {
		m.l.Error().
			Err(err).
			Str("queue", m.queue.Name()).
			Str("tenant_id", m.tenantId.String()).
			Str("message_id", m.msgId).
			Int("num_payloads", len(payloadBytes)).
			Int("num_waiters", numWaiters).
			Msg("dropping buffered message after publish failure")
	}

	for _, item := range drained {
		if item.errCh != nil {
			item.errCh <- err
		}
	}
}
