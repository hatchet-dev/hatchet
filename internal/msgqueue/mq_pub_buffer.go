package msgqueue

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

type PubFunc func(m *Message) error

// MQPubBuffer buffers messages coming out of the task queue, groups them by tenantId and msgId, and then flushes them
// to the task handler as necessary.
type MQPubBuffer struct {
	mq MessageQueue

	// buffers is keyed on (tenantId, msgId) and contains a buffer of messages for that tenantId and msgId.
	buffers sync.Map
}

func NewMQPubBuffer(mq MessageQueue) *MQPubBuffer {
	return &MQPubBuffer{
		mq: mq,
	}
}

func (m *MQPubBuffer) Pub(ctx context.Context, queue Queue, msg *Message) {
	if msg.TenantID == "" {
		return
	}

	k := getPubKey(queue, msg.TenantID, msg.ID)

	buf, ok := m.buffers.Load(k)

	if !ok {
		buf = newMsgIdPubBuffer(msg.TenantID, msg.ID, func(msg *Message) error {
			// TODO: DON'T USE BACKGROUND CONTEXT
			return m.mq.SendMessage(context.Background(), queue, msg)
		})

		m.buffers.Store(k, buf)
	}

	// this places some backpressure on the consumer if buffers are full
	msgBuf := buf.(*msgIdPubBuffer)
	msgBuf.msgIdPubBufferCh <- msg
	msgBuf.notifier <- struct{}{}

	return
}

func getPubKey(q Queue, tenantId, msgId string) string {
	return q.Name() + tenantId + msgId
}

type msgIdPubBuffer struct {
	tenantId      string
	msgId         string
	lastFlushedAt time.Time

	msgIdPubBufferCh chan *Message
	notifier         chan struct{}

	pub PubFunc

	serialize func(t any) ([]byte, error)
}

func newMsgIdPubBuffer(tenantId, msgId string, pub PubFunc) *msgIdPubBuffer {
	b := &msgIdPubBuffer{
		tenantId:         tenantId,
		msgId:            msgId,
		msgIdPubBufferCh: make(chan *Message, BUFFER_SIZE),
		notifier:         make(chan struct{}),
		pub:              pub,
		serialize:        json.Marshal,
	}

	err := b.startFlusher()

	if err != nil {
		// TODO: remove panic
		panic(err)
	}

	return b
}

func (m *msgIdPubBuffer) startFlusher() error {
	ticker := time.NewTicker(FLUSH_INTERVAL)

	go func() {
		for {
			select {
			case <-ticker.C:
				m.flush()
			case <-m.notifier:
				m.flush()
			}
		}
	}()

	return nil
}

func (m *msgIdPubBuffer) flush() {
	// TODO: PROTECT THIS WITH A MUTEX

	if m.lastFlushedAt.Add(FLUSH_INTERVAL).After(time.Now()) {
		return
	}

	defer func() {
		m.lastFlushedAt = time.Now()
	}()

	msgsWithResultCh := make([]*Message, 0)
	payloadBytes := make([][]byte, 0)

	// read all messages currently in the buffer
	for i := 0; i < BUFFER_SIZE; i++ {
		select {
		case msg := <-m.msgIdPubBufferCh:
			msgsWithResultCh = append(msgsWithResultCh, msg)

			payloadBytes = append(payloadBytes, msg.Payloads...)
		default:
			i = BUFFER_SIZE
		}
	}

	if len(payloadBytes) == 0 {
		return
	}

	m.pub(&Message{
		TenantID: m.tenantId,
		ID:       m.msgId,
		Payloads: payloadBytes,
	})
}
