package v1

import (
	"sync"

	"github.com/hashicorp/go-multierror"
)

type sharedTenantSub struct {
	fs        *sync.Map
	counter   int
	isRunning bool
	mu        sync.Mutex
	cleanup   func() error
}

type SharedTenantReader struct {
	tenants *sync.Map
	mq      MessageQueue
}

func NewSharedTenantReader(mq MessageQueue) *SharedTenantReader {
	return &SharedTenantReader{
		tenants: &sync.Map{},
		mq:      mq,
	}
}

func (s *SharedTenantReader) Subscribe(tenantId string, postAck AckHook) (func() error, error) {
	tenant, _ := s.tenants.LoadOrStore(tenantId, &sharedTenantSub{
		fs: &sync.Map{},
	})

	t := tenant.(*sharedTenantSub)

	t.mu.Lock()
	defer t.mu.Unlock()

	t.counter++

	subId := t.counter

	t.fs.Store(subId, postAck)

	if !t.isRunning {
		t.isRunning = true

		q := TenantEventConsumerQueue(tenantId)

		cleanupSingleSub, err := s.mq.Subscribe(q, NoOpHook, func(task *Message) error {
			var innerErr error

			t.fs.Range(func(key, value interface{}) bool {
				f := value.(AckHook)

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

		if lenSyncMap(t.fs) == 0 {
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

func lenSyncMap(m *sync.Map) int {
	var i int
	m.Range(func(k, v interface{}) bool {
		i++
		return true
	})
	return i
}
