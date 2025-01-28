package msgqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

const FLUSH_INTERVAL = 10 * time.Millisecond
const BUFFER_SIZE = 1000
const MAX_CONCURRENCY = 10

type DstFunc func(tenantId, msgId string, payloads [][]byte) error

func JSONConvert[T any](payloads [][]byte) []*T {
	ret := make([]*T, 0)

	for _, p := range payloads {
		var t T

		if err := json.Unmarshal(p, &t); err != nil {
			return nil
		}

		ret = append(ret, &t)
	}

	return ret
}

// MQSubBuffer buffers messages coming out of the task queue, groups them by tenantId and msgId, and then flushes them
// to the task handler as necessary.
type MQSubBuffer struct {
	queue Queue

	mq MessageQueue

	// buffers is keyed on (tenantId, msgId) and contains a buffer of messages for that tenantId and msgId.
	buffers sync.Map

	// the destination function to send the messages to
	dst DstFunc
}

func NewMQSubBuffer(queue Queue, mq MessageQueue, dst DstFunc) *MQSubBuffer {
	return &MQSubBuffer{
		queue: queue,
		mq:    mq,
		dst:   dst,
	}
}

func (m *MQSubBuffer) Start() (func() error, error) {
	ctx, cancel := context.WithCancel(context.Background())

	f := func(msg *Message) error {
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

func (m *MQSubBuffer) handleMsg(ctx context.Context, msg *Message) error {
	if msg.TenantID == "" {
		return nil
	}

	msgWithResult := &msgWithResultCh{
		msg:    msg,
		result: make(chan error),
	}

	k := getKey(msg.TenantID, msg.ID)

	buf, ok := m.buffers.Load(k)

	if !ok {
		buf = newMsgIdBuffer(msg.TenantID, msg.ID, m.dst)

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
	tenantId string
	msgId    string

	lastFlushedAt   time.Time
	lastFlushedAtMu sync.Mutex

	msgIdBufferCh chan *msgWithResultCh
	notifier      chan struct{}

	semaphore chan struct{}

	dst DstFunc
}

func newMsgIdBuffer(tenantId, msgId string, dst DstFunc) *msgIdBuffer {
	b := &msgIdBuffer{
		tenantId:      tenantId,
		msgId:         msgId,
		msgIdBufferCh: make(chan *msgWithResultCh, BUFFER_SIZE),
		notifier:      make(chan struct{}),
		dst:           dst,
		semaphore:     make(chan struct{}, MAX_CONCURRENCY),
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
				go m.flush()
			case <-m.notifier:
				go m.flush()
			}
		}
	}()

	return nil
}

func (m *msgIdBuffer) isBeforeWindow(t time.Time) bool {
	m.lastFlushedAtMu.Lock()
	defer m.lastFlushedAtMu.Unlock()

	return m.lastFlushedAt.Add(FLUSH_INTERVAL).After(t)
}

func (m *msgIdBuffer) setLastFlushedAt(t time.Time) {
	m.lastFlushedAtMu.Lock()
	defer m.lastFlushedAtMu.Unlock()

	m.lastFlushedAt = t
}

func (m *msgIdBuffer) flush() {
	select {
	case m.semaphore <- struct{}{}:
	default:
		return
	}

	defer func() { <-m.semaphore }()

	now := time.Now()

	if m.isBeforeWindow(now) {
		return
	}

	defer m.setLastFlushedAt(now)

	msgsWithResultCh := make([]*msgWithResultCh, 0)
	payloads := make([][]byte, 0)

	// read all messages currently in the buffer
	for i := 0; i < BUFFER_SIZE; i++ {
		select {
		case msg := <-m.msgIdBufferCh:
			msgsWithResultCh = append(msgsWithResultCh, msg)

			payloads = append(payloads, msg.msg.Payloads...)
		default:
			i = BUFFER_SIZE
		}
	}

	if len(payloads) == 0 {
		return
	}

	err := m.dst(m.tenantId, m.msgId, payloads)

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
