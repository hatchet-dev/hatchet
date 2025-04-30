package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// default values for the buffer
const SUB_FLUSH_INTERVAL = 10 * time.Millisecond
const SUB_BUFFER_SIZE = 1000
const SUB_MAX_CONCURRENCY = 10

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

type SubBufferKind string

const (
	PostAck SubBufferKind = "postAck"
	PreAck  SubBufferKind = "preAck"
)

// MQSubBuffer buffers messages coming out of the task queue, groups them by tenantId and msgId, and then flushes them
// to the task handler as necessary.
type MQSubBuffer struct {
	queue Queue

	mq MessageQueue

	// buffers is keyed on (tenantId, msgId) and contains a buffer of messages for that tenantId and msgId.
	buffers sync.Map

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
	return func(opts *mqSubBufferOpts) {
		opts.kind = kind
	}
}

func WithFlushInterval(flushInterval time.Duration) mqSubBufferOptFunc {
	return func(opts *mqSubBufferOpts) {
		opts.flushInterval = flushInterval
	}
}

func WithBufferSize(bufferSize int) mqSubBufferOptFunc {
	return func(opts *mqSubBufferOpts) {
		opts.bufferSize = bufferSize
	}
}

func WithMaxConcurrency(maxConcurrency int) mqSubBufferOptFunc {
	return func(opts *mqSubBufferOpts) {
		opts.maxConcurrency = maxConcurrency
	}
}

// "Immediate flush" means that if we haven't flushed yet, we can flush immediately without
// waiting on the flush interval timer.
func WithDisableImmediateFlush(disableImmediateFlush bool) mqSubBufferOptFunc {
	return func(opts *mqSubBufferOpts) {
		opts.disableImmediateFlush = disableImmediateFlush
	}
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
		buf, _ = m.buffers.LoadOrStore(k, newMsgIDBuffer(ctx, msg.TenantID, msg.ID, m.dst, m.flushInterval, m.bufferSize, m.maxConcurrency, m.disableImmediateFlush))
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

	msgIdBufferCh chan *msgWithResultCh
	notifier      chan struct{}

	// "Immediate flush" means that if we haven't flushed yet, we can flush immediately without
	// waiting on the timer.
	disableImmediateFlush bool

	semaphore chan struct{}

	dst DstFunc

	flushInterval time.Duration
}

func newMsgIDBuffer(ctx context.Context, tenantID, msgID string, dst DstFunc, flushInterval time.Duration, bufferSize, maxConcurrency int, disableImmediateFlush bool) *msgIdBuffer {
	b := &msgIdBuffer{
		tenantId:              tenantID,
		msgId:                 msgID,
		msgIdBufferCh:         make(chan *msgWithResultCh, bufferSize),
		notifier:              make(chan struct{}),
		dst:                   dst,
		disableImmediateFlush: disableImmediateFlush,
		semaphore:             make(chan struct{}, maxConcurrency),
		flushInterval:         flushInterval,
	}

	b.startFlusher(ctx)

	return b
}

func (m *msgIdBuffer) startFlusher(ctx context.Context) {
	ticker := time.NewTicker(m.flushInterval)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				go m.flush()
			case <-m.notifier:
				if !m.disableImmediateFlush {
					go m.flush()
				}
			}
		}
	}()
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
			<-time.After(m.flushInterval - time.Since(startedFlush))
			<-m.semaphore
		}()
	}()

	msgsWithResultCh := make([]*msgWithResultCh, 0)
	payloads := make([][]byte, 0)

	// read all messages currently in the buffer
	for i := 0; i < SUB_BUFFER_SIZE; i++ {
		select {
		case msg := <-m.msgIdBufferCh:
			msgsWithResultCh = append(msgsWithResultCh, msg)

			payloads = append(payloads, msg.msg.Payloads...)
		default:
			i = SUB_BUFFER_SIZE
		}
	}

	if len(payloads) == 0 {
		for _, msg := range msgsWithResultCh {
			close(msg.result)
		}

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
