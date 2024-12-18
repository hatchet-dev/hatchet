package buffer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/validator"

	"github.com/rs/zerolog"
)

var (
	defaultFlushPeriod = 10 * time.Millisecond
	defaultMaxCapacity = 100
)

func SetDefaults(flushPeriodMilliseconds int, flushItemsThreshold int) {
	if flushPeriodMilliseconds != 0 {
		defaultFlushPeriod = time.Duration(flushPeriodMilliseconds) * time.Millisecond
	}

	if flushItemsThreshold != 0 {
		defaultMaxCapacity = flushItemsThreshold
	}
}

// This is a wrapper around the IngestBuf to manage multiple tenants
// An example would be T is eventOps and U is *dbsqlc.Event

type TenantBufferManager[T any, U any] struct {
	name        string // a human readable name for the buffer
	tenants     sync.Map
	tenantLock  sync.Map
	l           *zerolog.Logger
	defaultOpts IngestBufOpts[T, U]
	v           validator.Validator
}

type TenantBufManagerOpts[T any, U any] struct {
	Name       string                                             `validate:"required"`
	OutputFunc func(ctx context.Context, items []T) ([]*U, error) `validate:"required"`
	SizeFunc   func(T) int                                        `validate:"required"`
	L          *zerolog.Logger                                    `validate:"required"`
	V          validator.Validator                                `validate:"required"`
	Config     ConfigFileBuffer                                   `validate:"required"`
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
		MaxCapacity:        defaultMaxCapacity,
		FlushPeriod:        defaultFlushPeriod,
		MaxDataSizeInQueue: 4 * megabyte,
		OutputFunc:         opts.OutputFunc,
		SizeFunc:           opts.SizeFunc,
		L:                  opts.L,
		MaxConcurrent:      opts.Config.MaxConcurrent,
		WaitForFlush:       opts.Config.WaitForFlush,
		FlushStrategy:      opts.Config.FlushStrategy,
	}

	if opts.Config.FlushPeriodMilliseconds != 0 {
		defaultOpts.FlushPeriod = time.Duration(opts.Config.FlushPeriodMilliseconds) * time.Millisecond
	}

	if opts.Config.FlushItemsThreshold != 0 {
		defaultOpts.MaxCapacity = opts.Config.FlushItemsThreshold
	}

	if opts.Config.MaxConcurrent != 0 {
		defaultOpts.MaxConcurrent = opts.Config.MaxConcurrent
	}

	if opts.Config.WaitForFlush != 0 {
		defaultOpts.WaitForFlush = opts.Config.WaitForFlush
	}

	if defaultOpts.FlushStrategy == "" {
		defaultOpts.FlushStrategy = Dynamic
	}

	opts.L.Debug().Msgf("creating new tenant buffer manager %s with default flush period %s and max capacity %d", opts.Name, defaultOpts.FlushPeriod, defaultOpts.MaxCapacity)

	return &TenantBufferManager[T, U]{
		name:        opts.Name,
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
func (t *TenantBufferManager[T, U]) Cleanup() error {
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
	opts.Name = fmt.Sprintf("%s-%s", t.name, tenantBufKey)
	return t.createTenantBuf(tenantBufKey, opts)
}

func (t *TenantBufferManager[T, U]) FireForget(tenantKey string, item T) error {
	_, err := t.buffItem(tenantKey, item)
	return err
}

func (t *TenantBufferManager[T, U]) FireAndWait(ctx context.Context, tenantKey string, item T) (*U, error) {
	doneChan, err := t.buffItem(tenantKey, item)
	if err != nil {
		return nil, err
	}

	select {
	case resp, ok := <-doneChan:
		if !ok {
			return nil, fmt.Errorf("error flushing tenant buffer for tenant %s: channel is closed", tenantKey)
		}

		return resp.Result, resp.Err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (t *TenantBufferManager[T, U]) buffItem(tenantKey string, eventOps T) (chan *FlushResponse[U], error) {
	tenantBuf, err := t.getOrCreateTenantBuf(tenantKey, t.defaultOpts)
	if err != nil {
		return nil, err
	}
	return tenantBuf.buffItem(eventOps)
}
