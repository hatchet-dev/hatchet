package prisma

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/validator"

	"github.com/rs/zerolog"
)

// This is a wrapper around the IngestBuf to manage multiple tenants
// An example would be T is eventOps and U is *dbsqlc.Event

type TenantBufferManager[T any, U any] struct {
	tenants     sync.Map
	tenantLock  sync.Map
	l           *zerolog.Logger
	defaultOpts IngestBufOpts[T, U]
	v           validator.Validator
}

type TenantBufManagerOpts[T any, U any] struct {
	OutputFunc func(ctx context.Context, items []T) ([]U, error) `validate:"required"`
	SizeFunc   func(T) int                                       `validate:"required"`
	L          *zerolog.Logger                                   `validate:"required"`
	V          validator.Validator                               `validate:"required"`
}

// Create a new TenantBufferManager with generic types T for input and U for output
func NewTenantBufManager[T any, U any](opts TenantBufManagerOpts[T, U]) (*TenantBufferManager[T, U], error) {

	v := opts.V
	err := v.Validate(opts)

	if err != nil {
		return nil, err
	}

	megabyte := 1024 * 1024

	defaultOpts := IngestBufOpts[T, U]{
		// something we can tune if we see this DB transaction is too slow
		MaxCapacity:        10000,
		FlushPeriod:        50 * time.Millisecond,
		MaxDataSizeInQueue: 4 * megabyte,
		OutputFunc:         opts.OutputFunc,
		SizeFunc:           opts.SizeFunc,
		L:                  opts.L,
	}

	return &TenantBufferManager[T, U]{
		tenants:     sync.Map{},
		l:           opts.L,
		defaultOpts: defaultOpts,
		v:           v,
	}, nil
}

// Create a new IngestBuf for the tenant and store it in the tenants map
// If we want to have a buffer for each tenant, we can create a new buffer for each tenant
// But if we would like tenants to share a buffer (maybe for lots of smaller tenants), we can use the same key for them.

func (t *TenantBufferManager[T, U]) createTenantBuf(
	tenantKey string,
	opts IngestBufOpts[T, U],
) (*IngestBuf[T, U], error) {

	err := t.v.Validate(opts)
	if err != nil {
		return nil, err
	}

	ingestBuf := NewIngestBuffer(opts)

	// we already have a lock for the tenant but just being paranoid
	if _, ok := t.tenants.Load(tenantKey); ok {
		return nil, fmt.Errorf("tenant buffer already exists for tenant %s", tenantKey)
	}

	t.tenants.Store(tenantKey, ingestBuf)

	_, err = t.startTenantBuf(tenantKey)
	if err != nil {
		t.l.Error().Err(err).Msg("error starting tenant buffer")

		return nil, err
	}
	return ingestBuf, nil
}

// cleanup all tenant buffers
func (t *TenantBufferManager[T, U]) cleanup() error {
	t.tenants.Range(func(key, value interface{}) bool {
		ingestBuf := value.(*IngestBuf[T, U])
		_ = ingestBuf.cleanup()
		return true
	})
	return nil
}

// Start the tenant's buffer
func (t *TenantBufferManager[T, U]) startTenantBuf(tenantKey string) (func() error, error) {
	if v, ok := t.tenants.Load(tenantKey); ok {
		return v.(*IngestBuf[T, U]).Start()
	}
	return nil, fmt.Errorf("tenant buffer not found for tenant %s", tenantKey)
}

// Retrieve or create a tenant buffer
func (t *TenantBufferManager[T, U]) getOrCreateTenantBuf(
	tenantBufKey string,
	opts IngestBufOpts[T, U],
) (*IngestBuf[T, U], error) {

	tlock, _ := t.tenantLock.LoadOrStore(tenantBufKey, &sync.Mutex{})
	tlock.(*sync.Mutex).Lock()
	defer tlock.(*sync.Mutex).Unlock()

	if v, ok := t.tenants.Load(tenantBufKey); ok {
		return v.(*IngestBuf[T, U]), nil
	}
	t.l.Debug().Msgf("creating new tenant buffer for tenant %s", tenantBufKey)
	return t.createTenantBuf(tenantBufKey, opts)
}

func (t *TenantBufferManager[T, U]) BuffItem(tenantKey string, eventOps T) (chan *flushResponse[U], error) {
	tenantBuf, err := t.getOrCreateTenantBuf(tenantKey, t.defaultOpts)
	if err != nil {
		return nil, err
	}
	return tenantBuf.BuffItem(eventOps)
}
