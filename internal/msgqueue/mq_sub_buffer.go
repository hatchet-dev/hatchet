package msgqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/internal/syncx"
)

// nolint: staticcheck
var (
	SUB_FLUSH_INTERVAL  = 10 * time.Millisecond
	SUB_BUFFER_SIZE     = 10
	SUB_MAX_CONCURRENCY = 10
)

type DstFunc func(tenantId uuid.UUID, msgId string, payloads [][]byte) error

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

type SubBufferKind string

const (
	PostAck SubBufferKind = "postAck"
	PreAck  SubBufferKind = "preAck"
)

// MQSubBuffer buffers messages coming out of the task queue, groups them by tenantId and msgId, and then flushes them
// to the task handler as necessary.
type MQSubBuffer struct {
	queue Queue
	mq    MessageQueue

	// buffers is keyed on a composite (tenantId, msgId) and contains a buffer of messages for that tenantId and msgId.
	buffers syncx.Map[string, *msgIdBuffer]

	// the destination function to send the messages to
	dst DstFunc

	// the kind of sub buffer
	kind SubBufferKind

	flushInterval         time.Duration
	bufferSize            int
	maxConcurrency        int
	disableImmediateFlush bool
}

type mqSubBufferOpts struct {
	kind                  SubBufferKind
	flushInterval         time.Duration
	bufferSize            int
	maxConcurrency        int
	disableImmediateFlush bool
}

type mqSubBufferOptFunc func(*mqSubBufferOpts)

func WithKind(kind SubBufferKind) mqSubBufferOptFunc {
	return func(opts *mqSubBufferOpts) { opts.kind = kind }
}

func WithFlushInterval(flushInterval time.Duration) mqSubBufferOptFunc {
	return func(opts *mqSubBufferOpts) { opts.flushInterval = flushInterval }
}

func WithBufferSize(bufferSize int) mqSubBufferOptFunc {
	return func(opts *mqSubBufferOpts) { opts.bufferSize = bufferSize }
}

func WithMaxConcurrency(maxConcurrency int) mqSubBufferOptFunc {
	return func(opts *mqSubBufferOpts) { opts.maxConcurrency = maxConcurrency }
}

func WithDisableImmediateFlush(disableImmediateFlush bool) mqSubBufferOptFunc {
	return func(opts *mqSubBufferOpts) { opts.disableImmediateFlush = disableImmediateFlush }
}

func defaultMQSubBufferOpts() *mqSubBufferOpts {
	return &mqSubBufferOpts{
		kind:           PreAck,
		flushInterval:  SUB_FLUSH_INTERVAL,
		bufferSize:     SUB_BUFFER_SIZE,
		maxConcurrency: SUB_MAX_CONCURRENCY,
	}
}

func NewMQSubBuffer(queue Queue, mq MessageQueue, dst DstFunc, fs ...mqSubBufferOptFunc) *MQSubBuffer {
	opts := defaultMQSubBufferOpts()
	for _, f := range fs {
		f(opts)
	}
	return &MQSubBuffer{
		queue:                 queue,
		mq:                    mq,
		dst:                   dst,
		kind:                  opts.kind,
		flushInterval:         opts.flushInterval,
		bufferSize:            opts.bufferSize,
		maxConcurrency:        opts.maxConcurrency,
		disableImmediateFlush: opts.disableImmediateFlush,
	}
}

func (m *MQSubBuffer) Start() (func() error, error) {
	ctx, cancel := context.WithCancel(context.Background())

	f := func(msg *Message) error {
		return m.handleMsg(ctx, msg)
	}

	var cleanupQueue func() error
	var err error

	switch m.kind {
	case PreAck:
		cleanupQueue, err = m.mq.Subscribe(m.queue, f, NoOpHook)
	case PostAck:
		cleanupQueue, err = m.mq.Subscribe(m.queue, NoOpHook, f)
	}

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
	if msg.TenantID == uuid.Nil {
		return nil
	}

	msgWithResult := &msgWithResultCh{
		msg:    msg,
		result: make(chan error),
	}

	k := getKey(msg.TenantID, msg.ID)

	msgBuf, ok := m.buffers.Load(k)

	if !ok {
		msgBuf, _ = m.buffers.LoadOrStore(k, newMsgIDBuffer(ctx, msg.TenantID, msg.ID, m.dst, m.flushInterval, m.bufferSize, m.maxConcurrency, m.disableImmediateFlush))
	}

	// Signal early flush if the send would block due to capacity.
	select {
	case msgBuf.msgIdBufferCh <- msgWithResult:
		// sent without blocking
	default:
		select {
		case msgBuf.capacityRelease <- struct{}{}:
		default:
		}
		// this places some backpressure on the consumer if buffers are full
		msgBuf.msgIdBufferCh <- msgWithResult
	}
	msgBuf.notifier <- struct{}{}

	// wait for the message to be processed
	err, ok := <-msgWithResult.result

	// if the channel is closed, then the buffer has been flushed without error
	if !ok {
		return nil
	}

	return err
}

func getKey(tenantId uuid.UUID, msgId string) string {
	return tenantId.String() + msgId
}

type msgIdBuffer struct {
	bufferCore

	tenantId      uuid.UUID
	msgId         string
	msgIdBufferCh chan *msgWithResultCh
	dst           DstFunc
}

func newMsgIDBuffer(ctx context.Context, tenantID uuid.UUID, msgID string, dst DstFunc, flushInterval time.Duration, bufferSize, maxConcurrency int, disableImmediateFlush bool) *msgIdBuffer {
	b := &msgIdBuffer{
		bufferCore:    newBufferCore(flushInterval, bufferSize, maxConcurrency, disableImmediateFlush, false),
		tenantId:      tenantID,
		msgId:         msgID,
		msgIdBufferCh: make(chan *msgWithResultCh, bufferSize),
		dst:           dst,
	}
	b.startFlusher(ctx, b.flush)
	b.startSemaphoreReleaser(ctx, func() int { return len(b.msgIdBufferCh) }, b.flush)
	return b
}

func (m *msgIdBuffer) flush() {
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

	// drainN uses the instance bufferSize, fixing the previous bug where the global
	// SUB_BUFFER_SIZE was used regardless of how the buffer was configured.
	drained := drainN(m.msgIdBufferCh, m.bufferSize)

	payloads := make([][]byte, 0, len(drained))
	for _, item := range drained {
		payloads = append(payloads, item.msg.Payloads...)
	}

	if len(payloads) == 0 {
		for _, item := range drained {
			close(item.result)
		}
		return
	}

	err := m.dst(m.tenantId, m.msgId, payloads)

	if err != nil {
		for _, item := range drained {
			item.result <- err
		}
	}

	for _, item := range drained {
		close(item.result)
	}
}
