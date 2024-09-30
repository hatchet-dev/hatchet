package prisma

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// This is a wrapper around the IngestBuf to manage multiple tenants
// An example would be T is eventOps and U is *dbsqlc.Event

type TenantBufferManager[T any, U any] struct {
	tenants     sync.Map
	l           *zerolog.Logger
	defaultOpts IngestBufOpts[T, U]
}

type TenantBufManagerOpts[T any, U any] struct {
	OutputFunc func(ctx context.Context, items []T) ([]U, error)
	SizeFunc   func(T) int
	l          *zerolog.Logger
}

// Create a new TenantBufferManager with generic types T for input and U for output
func NewTenantBufManager[T any, U any](opts TenantBufManagerOpts[T, U]) (*TenantBufferManager[T, U], error) {

	if opts.OutputFunc == nil {
		return nil, fmt.Errorf("output function is required")
	}

	if opts.SizeFunc == nil {
		return nil, fmt.Errorf("size function is required")
	}
	if opts.l == nil {
		return nil, fmt.Errorf("logger is required")
	}

	megabyte := 1024 * 1024

	defaultOpts := IngestBufOpts[T, U]{
		maxCapacity:        1000,
		flushPeriod:        50 * time.Millisecond,
		maxDataSizeInQueue: 4 * megabyte,
		outputFunc:         opts.OutputFunc,
		sizeFunc:           opts.SizeFunc,
		l:                  opts.l,
	}

	return &TenantBufferManager[T, U]{
		tenants:     sync.Map{},
		l:           opts.l,
		defaultOpts: defaultOpts,
	}, nil
}

// Retrieve a tenant's IngestBuf with the specified tenantID
func (t *TenantBufferManager[T, U]) GetTenantBuf(tenantID string) *IngestBuf[T, U] {
	if v, ok := t.tenants.Load(tenantID); ok {
		return v.(*IngestBuf[T, U])
	}
	return nil
}

// Create a new IngestBuf for the tenant and store it in the tenants map
func (t *TenantBufferManager[T, U]) CreateTenantBuf(
	tenantID string,
	opts IngestBufOpts[T, U],
) (*IngestBuf[T, U], error) {

	ingestBuf := NewIngestBuffer(opts)

	err := ingestBuf.validate()
	if err != nil {
		return nil, err
	}

	t.tenants.Store(tenantID, ingestBuf)
	_, err = t.StartTenantBuf(tenantID)
	if err != nil {
		t.l.Error().Err(err).Msg("error starting tenant buffer")

		return nil, err
	}
	return ingestBuf, nil
}

// Cleanup all tenant buffers
func (t *TenantBufferManager[T, U]) Cleanup() error {
	t.tenants.Range(func(key, value interface{}) bool {
		ingestBuf := value.(*IngestBuf[T, U])
		_ = ingestBuf.cleanup()
		return true
	})
	return nil
}

// Start the tenant's buffer
func (t *TenantBufferManager[T, U]) StartTenantBuf(tenantID string) (func() error, error) {
	if v, ok := t.tenants.Load(tenantID); ok {
		return v.(*IngestBuf[T, U]).Start()
	}
	return nil, fmt.Errorf("tenant buffer not found")
}

// Retrieve or create a tenant buffer
func (t *TenantBufferManager[T, U]) GetOrCreateTenantBuf(
	tenantBufKey string,
	opts IngestBufOpts[T, U],
) (*IngestBuf[T, U], error) {

	if v, ok := t.tenants.Load(tenantBufKey); ok {
		return v.(*IngestBuf[T, U]), nil
	}
	return t.CreateTenantBuf(tenantBufKey, opts)
}

func (t *TenantBufferManager[T, U]) BuffItem(tenantID string, eventOps T) (chan *flushResponse[U], error) {
	tenantBuf, err := t.GetOrCreateTenantBuf(tenantID, t.defaultOpts)
	if err != nil {
		return nil, err
	}
	return tenantBuf.buffEvent(eventOps)
}
