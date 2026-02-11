package msgqueue

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/internal/syncx"
)

// nolint: staticcheck
var (
	PUB_FLUSH_INTERVAL  = 10 * time.Millisecond
	PUB_BUFFER_SIZE     = 1000
	PUB_MAX_CONCURRENCY = 1
	PUB_TIMEOUT         = 10 * time.Second
)

func init() {
	if os.Getenv("SERVER_DEFAULT_BUFFER_FLUSH_INTERVAL") != "" {
		if v, err := time.ParseDuration(os.Getenv("SERVER_DEFAULT_BUFFER_FLUSH_INTERVAL")); err == nil {
			PUB_FLUSH_INTERVAL = v
		}
	}

	if os.Getenv("SERVER_DEFAULT_BUFFER_SIZE") != "" {
		v := os.Getenv("SERVER_DEFAULT_BUFFER_SIZE")

		maxSize, err := strconv.Atoi(v)

		if err == nil {
			PUB_BUFFER_SIZE = maxSize
		}
	}

	if os.Getenv("SERVER_DEFAULT_BUFFER_CONCURRENCY") != "" {
		v := os.Getenv("SERVER_DEFAULT_BUFFER_CONCURRENCY")

		maxConcurrency, err := strconv.Atoi(v)

		if err == nil {
			PUB_MAX_CONCURRENCY = maxConcurrency
		}
	}
}

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

	return &MQPubBuffer{
		mq:     mq,
		ctx:    ctx,
		cancel: cancel,
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

	msgBuf, ok := m.buffers.Load(k)

	if !ok {
		msgBuf, _ = m.buffers.LoadOrStore(k, newMsgIDPubBuffer(m.ctx, msg.TenantID, msg.ID, func(msg *Message) error {
			msgCtx, cancel := context.WithTimeout(context.Background(), PUB_TIMEOUT)
			defer cancel()

			return m.mq.SendMessage(msgCtx, queue, msg)
		}))
	}

	msgWithErr := &msgWithErrCh{
		msg: msg,
	}

	if wait {
		msgWithErr.errCh = make(chan error)
	}

	// this places some backpressure on the consumer if buffers are full
	msgBuf.msgIdPubBufferCh <- msgWithErr
	msgBuf.notifier <- struct{}{}

	if wait {
		return <-msgWithErr.errCh
	}

	return nil
}

func getPubKey(q Queue, tenantId uuid.UUID, msgId string) string {
	return q.Name() + tenantId.String() + msgId
}

type msgIdPubBuffer struct {
	tenantId uuid.UUID
	msgId    string

	msgIdPubBufferCh chan *msgWithErrCh
	notifier         chan struct{}

	pub PubFunc

	semaphore        chan struct{}
	semaphoreRelease chan time.Duration

	serialize func(t any) ([]byte, error)
}

func newMsgIDPubBuffer(ctx context.Context, tenantID uuid.UUID, msgID string, pub PubFunc) *msgIdPubBuffer {
	b := &msgIdPubBuffer{
		tenantId:         tenantID,
		msgId:            msgID,
		msgIdPubBufferCh: make(chan *msgWithErrCh, PUB_BUFFER_SIZE),
		notifier:         make(chan struct{}),
		pub:              pub,
		serialize:        json.Marshal,
		semaphore:        make(chan struct{}, PUB_MAX_CONCURRENCY),
		semaphoreRelease: make(chan time.Duration, PUB_MAX_CONCURRENCY),
	}

	b.startFlusher(ctx)
	b.startSemaphoreReleaser(ctx)

	return b
}

func (m *msgIdPubBuffer) startFlusher(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(PUB_FLUSH_INTERVAL)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				m.flush()
				return
			case <-ticker.C:
				go m.flush()
			case <-m.notifier:
				go m.flush()
			}
		}
	}()
}

func (m *msgIdPubBuffer) startSemaphoreReleaser(ctx context.Context) {
	go func() {
		timer := time.NewTimer(0)
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case delay := <-m.semaphoreRelease:
				if delay > 0 {
					timer.Reset(delay)
					<-timer.C
				}
				<-m.semaphore
			}
		}
	}()
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
			delay := PUB_FLUSH_INTERVAL - time.Since(startedFlush)
			m.semaphoreRelease <- delay
		}()
	}()

	msgsWithErrCh := make([]*msgWithErrCh, 0)
	payloadBytes := make([][]byte, 0)
	var isPersistent *bool
	var immediatelyExpire *bool
	var retries *int

	// read all messages currently in the buffer
	for i := 0; i < PUB_BUFFER_SIZE; i++ {
		select {
		case msg := <-m.msgIdPubBufferCh:
			msgsWithErrCh = append(msgsWithErrCh, msg)

			payloadBytes = append(payloadBytes, msg.msg.Payloads...)

			if isPersistent == nil {
				isPersistent = &msg.msg.Persistent
			}

			if immediatelyExpire == nil {
				immediatelyExpire = &msg.msg.ImmediatelyExpire
			}

			if retries == nil {
				retries = &msg.msg.Retries
			}
		default:
			i = PUB_BUFFER_SIZE
		}
	}

	if len(payloadBytes) == 0 {
		return
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

	for _, msgWithErrCh := range msgsWithErrCh {
		if msgWithErrCh.errCh != nil {
			msgWithErrCh.errCh <- err
		}
	}
}
