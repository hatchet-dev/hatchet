package msgqueue

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const FLUSH_INTERVAL = 10 * time.Millisecond
const BUFFER_SIZE = 1000

type DstFunc func(tenantId, msgId string, msgs []*Message) error

// MQBuffer buffers messages coming out of the task queue, groups them by tenantId and msgId, and then flushes them
// to the task handler as necessary.
type MQBuffer struct {
	queue Queue

	mq MessageQueue

	// buffers is keyed on (tenantId, msgId) and contains a buffer of messages for that tenantId and msgId.
	buffers sync.Map

	// the destination function to send the messages to
	dst DstFunc
}

func NewMQBuffer(queue Queue, mq MessageQueue, dst DstFunc) *MQBuffer {
	return &MQBuffer{
		queue: queue,
		mq:    mq,
		dst:   dst,
	}
}

func (m *MQBuffer) Start() (func() error, error) {
	ctx, cancel := context.WithCancel(context.Background())

	f := func(msg *Message) error {
		// TODO: MOVE WGS OUT TO TOP LEVEL
		// wg.Add(1)
		// defer wg.Done()

		return m.handleMsg(ctx, msg)
	}

	// TODO: argument for queue type
	cleanupQueue, err := m.mq.Subscribe(m.queue, f, NoOpHook)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not subscribe in mq buffer: %w", err)
	}

	cleanup := func() error {
		defer cancel()

		if err := cleanupQueue(); err != nil {
			return fmt.Errorf("could not cleanup message queue listener: %w", err)
		}

		return nil
	}

	return cleanup, nil
}

type msgWithResultCh struct {
	msg    *Message
	result chan error
}

func (m *MQBuffer) handleMsg(ctx context.Context, msg *Message) error {
	if msg.TenantID() == "" || msg.ID == "" {
		return nil
	}

	msgWithResult := &msgWithResultCh{
		msg:    msg,
		result: make(chan error),
	}

	k := getKey(msg.TenantID(), msg.ID)

	buf, ok := m.buffers.Load(k)

	if !ok {
		buf = newMsgIdBuffer(msg.TenantID(), msg.ID, m.dst)

		m.buffers.Store(k, buf)
	}

	// this places some backpressure on the consumer if buffers are full
	msgBuf := buf.(*msgIdBuffer)
	msgBuf.msgIdBufferCh <- msgWithResult
	msgBuf.notifier <- struct{}{}

	// wait for the message to be processed
	err, ok := <-msgWithResult.result

	// if the channel is closed, then the buffer has been flushed without error
	if !ok {
		return nil
	}

	return err
}

func getKey(tenantId, msgId string) string {
	return tenantId + msgId
}

type msgIdBuffer struct {
	tenantId      string
	msgId         string
	lastFlushedAt time.Time

	msgIdBufferCh chan *msgWithResultCh
	notifier      chan struct{}

	dst DstFunc
}

func newMsgIdBuffer(tenantId, msgId string, dst DstFunc) *msgIdBuffer {
	b := &msgIdBuffer{
		tenantId:      tenantId,
		msgId:         msgId,
		msgIdBufferCh: make(chan *msgWithResultCh, BUFFER_SIZE),
		notifier:      make(chan struct{}),
		dst:           dst,
	}

	err := b.startFlusher()

	if err != nil {
		// TODO: remove panic
		panic(err)
	}

	return b
}

func (m *msgIdBuffer) startFlusher() error {
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

func (m *msgIdBuffer) flush() {
	// TODO: PROTECT THIS WITH A MUTEX

	if m.lastFlushedAt.Add(FLUSH_INTERVAL).After(time.Now()) {
		return
	}

	defer func() {
		m.lastFlushedAt = time.Now()
	}()

	msgsWithResultCh := make([]*msgWithResultCh, 0)
	msgs := make([]*Message, 0)

	// read all messages currently in the buffer
	for i := 0; i < BUFFER_SIZE; i++ {
		select {
		case msg := <-m.msgIdBufferCh:
			msgsWithResultCh = append(msgsWithResultCh, msg)
			msgs = append(msgs, msg.msg)
		default:
			i = BUFFER_SIZE
		}
	}

	if len(msgs) == 0 {
		return
	}

	err := m.dst(m.tenantId, m.msgId, msgs)

	if err != nil {
		// write err to all the message channels
		for _, msg := range msgsWithResultCh {
			msg.result <- err
		}
	}

	for _, msg := range msgsWithResultCh {
		close(msg.result)
	}
}
