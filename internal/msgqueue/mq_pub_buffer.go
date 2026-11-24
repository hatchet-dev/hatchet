package msgqueue

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

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

	// buffers is keyed on a composite (tenantId, msgId) and contains a buffer of messages for that tenantId and msgId.
	buffers syncx.Map[string, *msgIdPubBuffer]

	ctx    context.Context
	cancel context.CancelFunc
}

func NewMQPubBuffer(mq MessageQueue) *MQPubBuffer {
	ctx, cancel := context.WithCancel(context.Background())
	m := &MQPubBuffer{mq: mq, ctx: ctx, cancel: cancel}
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

// Stop flushes every buffered message and returns once they are all published. A flusher's own
// shutdown flush isn't enough: it runs in the background, and it is skipped when the buffer flushed
// within the last flush interval, which is exactly when messages were just enqueued. Callers stop the
// buffer before disconnecting the queue, so anything still buffered after Stop would be lost. Buffers
// drain in parallel, so a slow broker costs Stop about one publish timeout rather than one per buffer.
func (m *MQPubBuffer) Stop() {
	m.cancel()

	var wg sync.WaitGroup

	m.buffers.Range(func(_ string, b *msgIdPubBuffer) bool {
		wg.Go(b.drainAll)
		return true
	})

	wg.Wait()
}

// pubNow publishes msg on its own, for messages that arrive once the buffer is stopped and nothing
// flushes it anymore.
func (m *MQPubBuffer) pubNow(queue Queue, msg *Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), PUB_TIMEOUT)
	defer cancel()

	return m.mq.SendMessage(ctx, queue, msg)
}

type msgWithErrCh struct {
	msg   *Message
	errCh chan error
}

func (m *MQPubBuffer) Pub(ctx context.Context, queue Queue, msg *Message, wait bool) error {
	if msg.TenantID == uuid.Nil {
		return nil
	}

	if m.ctx.Err() != nil {
		return m.pubNow(queue, msg)
	}

	k := getPubKey(queue, msg.TenantID, msg.ID)

	var msgBuf *msgIdPubBuffer

	for {
		var ok bool
		msgBuf, ok = m.buffers.Load(k)

		if !ok {
			newBuf := newMsgIDPubBuffer(m.ctx, msg.TenantID, msg.ID, func(msg *Message) error {
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
		// buffered, since after Stop this goroutine may flush its own message before it waits on it
		msgWithErr.errCh = make(chan error, 1)
	}

	// Signal early flush if the channel is already at capacity — the send below may block.
	if len(msgBuf.msgIdPubBufferCh) >= msgBuf.bufferSize {
		select {
		case msgBuf.capacityRelease <- struct{}{}:
		default:
		}
	}

	// this places some backpressure on the consumer if buffers are full. Once the buffer is stopped
	// nothing reads these channels, so a Pub racing with Stop publishes its message itself.
	select {
	case msgBuf.msgIdPubBufferCh <- msgWithErr:
	case <-m.ctx.Done():
		msgBuf.release()
		return m.pubNow(queue, msg)
	}

	select {
	case msgBuf.notifier <- struct{}{}:
	case <-m.ctx.Done():
		// Stop may have drained this buffer before the message landed in it.
		msgBuf.drainAll()
	}

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

	tenantId         uuid.UUID
	msgId            string
	msgIdPubBufferCh chan *msgWithErrCh
	pub              PubFunc

	// flushMu lets drainAll wait for flushes in progress, which have taken messages off the channel
	// but not published them yet; flushes share it so they still run concurrently with each other.
	flushMu sync.RWMutex
}

func newMsgIDPubBuffer(ctx context.Context, tenantID uuid.UUID, msgID string, pub PubFunc) *msgIdPubBuffer {
	ctx, stop := context.WithCancel(ctx)

	b := &msgIdPubBuffer{
		bufferCore:       newBufferCore(PUB_FLUSH_INTERVAL, PUB_BUFFER_SIZE, PUB_MAX_CONCURRENCY, false, true),
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

	m.flushMu.RLock()
	defer m.flushMu.RUnlock()

	drained := drainN(m.msgIdPubBufferCh, m.bufferSize)
	if len(drained) == 0 {
		return
	}

	m.publish(drained)
}

// drainAll publishes everything left in the buffer, ignoring the flush rate limit, once flushes in
// progress have finished.
func (m *msgIdPubBuffer) drainAll() {
	m.flushMu.Lock()
	defer m.flushMu.Unlock()

	for {
		drained := drainN(m.msgIdPubBufferCh, m.bufferSize)
		if len(drained) == 0 {
			return
		}

		m.publish(drained)
	}
}

func (m *msgIdPubBuffer) publish(drained []*msgWithErrCh) {
	payloadBytes := make([][]byte, 0, len(drained))
	var isPersistent *bool
	var immediatelyExpire *bool
	var retries *int

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

	for _, item := range drained {
		if item.errCh != nil {
			item.errCh <- err
		}
	}
}
