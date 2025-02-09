package msgqueue

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

const PUB_FLUSH_INTERVAL = 10 * time.Millisecond
const PUB_BUFFER_SIZE = 1000
const PUB_MAX_CONCURRENCY = 2

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

type msgWithErrCh struct {
	msg   *Message
	errCh chan error
}

func (m *MQPubBuffer) Pub(ctx context.Context, queue Queue, msg *Message, wait bool) error {
	if msg.TenantID == "" {
		return nil
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

	msgWithErr := &msgWithErrCh{
		msg: msg,
	}

	if wait {
		msgWithErr.errCh = make(chan error)
	}

	// this places some backpressure on the consumer if buffers are full
	msgBuf := buf.(*msgIdPubBuffer)
	msgBuf.msgIdPubBufferCh <- msgWithErr
	msgBuf.notifier <- struct{}{}

	if wait {
		return <-msgWithErr.errCh
	}

	return nil
}

func getPubKey(q Queue, tenantId, msgId string) string {
	return q.Name() + tenantId + msgId
}

type msgIdPubBuffer struct {
	tenantId string
	msgId    string

	msgIdPubBufferCh chan *msgWithErrCh
	notifier         chan struct{}

	pub PubFunc

	semaphore chan struct{}

	serialize func(t any) ([]byte, error)
}

func newMsgIdPubBuffer(tenantId, msgId string, pub PubFunc) *msgIdPubBuffer {
	b := &msgIdPubBuffer{
		tenantId:         tenantId,
		msgId:            msgId,
		msgIdPubBufferCh: make(chan *msgWithErrCh, PUB_BUFFER_SIZE),
		notifier:         make(chan struct{}),
		pub:              pub,
		serialize:        json.Marshal,
		semaphore:        make(chan struct{}, PUB_MAX_CONCURRENCY),
	}

	err := b.startFlusher()

	if err != nil {
		// TODO: remove panic
		panic(err)
	}

	return b
}

func (m *msgIdPubBuffer) startFlusher() error {
	ticker := time.NewTicker(PUB_FLUSH_INTERVAL)

	go func() {
		for {
			select {
			case <-ticker.C:
				go m.flush()
			case <-m.notifier:
				go m.flush()
			}
		}
	}()

	return nil
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
			<-time.After(PUB_FLUSH_INTERVAL - time.Since(startedFlush))
			<-m.semaphore
		}()
	}()

	msgsWithErrCh := make([]*msgWithErrCh, 0)
	payloadBytes := make([][]byte, 0)

	// read all messages currently in the buffer
	for i := 0; i < PUB_BUFFER_SIZE; i++ {
		select {
		case msg := <-m.msgIdPubBufferCh:
			msgsWithErrCh = append(msgsWithErrCh, msg)

			payloadBytes = append(payloadBytes, msg.msg.Payloads...)
		default:
			i = PUB_BUFFER_SIZE
		}
	}

	if len(payloadBytes) == 0 {
		return
	}

	err := m.pub(&Message{
		TenantID: m.tenantId,
		ID:       m.msgId,
		Payloads: payloadBytes,
	})

	for _, msgWithErrCh := range msgsWithErrCh {
		if msgWithErrCh.errCh != nil {
			msgWithErrCh.errCh <- err
		}
	}
}
