package msgqueue

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"

	"github.com/hatchet-dev/hatchet/internal/syncx"
)

type sharedTenantSub struct {
	fs        *syncx.Map[int, AckHook]
	cleanup   func() error
	counter   int
	mu        sync.Mutex
	isRunning bool
}

type SharedTenantReader struct {
	tenants *syncx.Map[uuid.UUID, *sharedTenantSub]
	mq      MessageQueue
}

func NewSharedTenantReader(mq MessageQueue) *SharedTenantReader {
	return &SharedTenantReader{
		tenants: &syncx.Map[uuid.UUID, *sharedTenantSub]{},
		mq:      mq,
	}
}

func (s *SharedTenantReader) Subscribe(tenantId uuid.UUID, postAck AckHook) (func() error, error) {
	t, _ := s.tenants.LoadOrStore(tenantId, &sharedTenantSub{
		fs: &syncx.Map[int, AckHook]{},
	})

	t.mu.Lock()
	defer t.mu.Unlock()

	t.counter++

	subId := t.counter

	t.fs.Store(subId, postAck)

	if !t.isRunning {
		t.isRunning = true

		q := TenantEventConsumerQueue(tenantId)

		err := s.mq.RegisterTenant(context.Background(), tenantId)

		if err != nil {
			return nil, err
		}

		cleanupSingleSub, err := s.mq.Subscribe(q, NoOpHook, func(task *Message) error {
			var innerErr error

			t.fs.Range(func(key int, f AckHook) bool {
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
	mq      MessageQueue
}

func NewSharedBufferedTenantReader(mq MessageQueue) *SharedBufferedTenantReader {
	return &SharedBufferedTenantReader{
		tenants: &syncx.Map[uuid.UUID, *sharedBufferedTenantSub]{},
		mq:      mq,
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

		q := TenantEventConsumerQueue(tenantId)

		err := s.mq.RegisterTenant(context.Background(), tenantId)

		if err != nil {
			return nil, err
		}

		subBuffer := NewMQSubBuffer(q, s.mq, func(tenantId uuid.UUID, msgId string, payloads [][]byte) error {
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
