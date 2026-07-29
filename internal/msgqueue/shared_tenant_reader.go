package msgqueue

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"

	"github.com/hatchet-dev/hatchet/internal/syncx"
)

type sharedTenantSub struct {
	fs        *syncx.Map[int, MsgHandler]
	counter   int
	isRunning bool
	mu        sync.Mutex
	cleanup   func() error
}

type SharedTenantReader struct {
	tenants *syncx.Map[uuid.UUID, *sharedTenantSub]
	ps      PubSub
}

func NewSharedTenantReader(ps PubSub) *SharedTenantReader {
	return &SharedTenantReader{
		tenants: &syncx.Map[uuid.UUID, *sharedTenantSub]{},
		ps:      ps,
	}
}

func (s *SharedTenantReader) Subscribe(tenantId uuid.UUID, postAck MsgHandler) (func() error, error) {
	t, _ := s.tenants.LoadOrStore(tenantId, &sharedTenantSub{
		fs: &syncx.Map[int, MsgHandler]{},
	})

	t.mu.Lock()
	defer t.mu.Unlock()

	t.counter++

	subId := t.counter

	t.fs.Store(subId, postAck)

	if !t.isRunning {
		t.isRunning = true

		cleanupSingleSub, err := s.ps.Sub(TenantTopic(tenantId), func(task *Message) error {
			var innerErr error

			t.fs.Range(func(key int, f MsgHandler) bool {
				if err := f(task); err != nil {
					innerErr = multierror.Append(innerErr, err)
				}

				return true
			})

			return innerErr
		})

		if err != nil {
			return nil, err
		}

		t.cleanup = cleanupSingleSub
	}

	return func() error {
		t.mu.Lock()
		defer t.mu.Unlock()

		t.fs.Delete(subId)

		if t.fs.Len() == 0 {
			// shut down the subscription
			if t.cleanup != nil {
				if err := t.cleanup(); err != nil {
					return err
				}
			}

			t.isRunning = false
		}

		return nil
	}, nil
}

type sharedBufferedTenantSub struct {
	cleanup   func() error
	fs        *syncx.Map[int, DstFunc]
	counter   int
	mu        sync.Mutex
	isRunning bool
}

type SharedBufferedTenantReader struct {
	tenants *syncx.Map[uuid.UUID, *sharedBufferedTenantSub]
	ps      PubSub
}

func NewSharedBufferedTenantReader(ps PubSub) *SharedBufferedTenantReader {
	return &SharedBufferedTenantReader{
		tenants: &syncx.Map[uuid.UUID, *sharedBufferedTenantSub]{},
		ps:      ps,
	}
}

func (s *SharedBufferedTenantReader) Subscribe(tenantId uuid.UUID, f DstFunc) (func() error, error) {
	t, _ := s.tenants.LoadOrStore(tenantId, &sharedBufferedTenantSub{
		fs: &syncx.Map[int, DstFunc]{},
	})

	t.mu.Lock()
	defer t.mu.Unlock()

	t.counter++

	subId := t.counter

	t.fs.Store(subId, f)

	if !t.isRunning {
		t.isRunning = true

		// the buffer runs in PostAck mode, which only uses the post hook, so the
		// single-handler pubsub Sub maps cleanly onto the subscribe function
		subBuffer := NewSubBufferFromSubscribe(func(preAck MsgHandler, postAck MsgHandler) (func() error, error) {
			return s.ps.Sub(TenantTopic(tenantId), postAck)
		}, func(tenantId uuid.UUID, msgId string, payloads [][]byte) error {
			var innerErr error

			t.fs.Range(func(key int, f DstFunc) bool {
				if err := f(tenantId, msgId, payloads); err != nil {
					innerErr = multierror.Append(innerErr, err)
				}

				return true
			})

			return innerErr
		}, WithKind(PostAck), WithMaxConcurrency(1), WithFlushInterval(20*time.Millisecond), WithDisableImmediateFlush(true))

		cleanupSingleSub, err := subBuffer.Start()

		if err != nil {
			return nil, err
		}

		t.cleanup = cleanupSingleSub
	}

	return func() error {
		t.mu.Lock()
		defer t.mu.Unlock()

		t.fs.Delete(subId)

		if t.fs.Len() == 0 {
			// shut down the subscription
			if t.cleanup != nil {
				if err := t.cleanup(); err != nil {
					return err
				}
			}

			t.isRunning = false
		}

		return nil
	}, nil
}
