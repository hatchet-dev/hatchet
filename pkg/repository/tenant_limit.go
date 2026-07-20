package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/pkg/config/limits"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type TenantLimitConfig struct {
	EnforceLimits bool
}

type Limit struct {
	Resource sqlcv1.LimitResource
	Limit    int32
	Alarm    *int32
	Window   *time.Duration
}

type TenantLimitRepository interface {
	GetLimits(ctx context.Context, tenantId uuid.UUID) ([]*sqlcv1.TenantResourceLimit, error)

	// UpdateLimits updates the limits for a tenant
	UpdateLimits(ctx context.Context, tenantId uuid.UUID, limits []Limit) error

	// CanCreate checks if the tenant can create a resource
	CanCreate(ctx context.Context, resource sqlcv1.LimitResource, tenantId uuid.UUID, numberOfResources int32) (bool, int, error)

	// Resolve all tenant resource limits
	ResolveAllTenantResourceLimits(ctx context.Context) error

	DefaultLimits() []Limit

	Meter(ctx context.Context, dbtx sqlcv1.DBTX, resource sqlcv1.LimitResource, tenantId uuid.UUID, numberOfResources int32) (precommit func() error, postcommit func())

	Stop()
}

type meterKey struct {
	resource sqlcv1.LimitResource
	tenantId uuid.UUID
}

type meterSet map[meterKey]int32

type tenantLimitRepository struct {
	c cache.Cacheable
	*sharedRepository
	config        limits.LimitConfigFile
	enforceLimits bool

	unflushedMu sync.RWMutex
	unflushed   meterSet

	cleanup func()
}

func newTenantLimitRepository(shared *sharedRepository, s limits.LimitConfigFile, enforceLimits bool, cacheDuration time.Duration) TenantLimitRepository {
	ctx, cancel := context.WithCancel(context.Background())

	t := &tenantLimitRepository{
		sharedRepository: shared,
		config:           s,
		enforceLimits:    enforceLimits,
		c:                cache.New(cacheDuration),
		unflushed:        make(meterSet),
		cleanup:          cancel,
	}

	go t.loopFlush(ctx)

	return t
}

func (t *tenantLimitRepository) ResolveAllTenantResourceLimits(ctx context.Context) error {
	_, err := t.queries.ResolveAllLimitsIfWindowPassed(ctx, t.pool)
	return err
}

// FIXME(mnafees): WE NEED TO GET RID OF CUSTOM VALUE METERS
func hasCustomValueMeter(resource sqlcv1.LimitResource) bool {
	switch resource {
	case sqlcv1.LimitResourceWORKER, sqlcv1.LimitResourceWORKERSLOT, sqlcv1.LimitResourceINCOMINGWEBHOOK:
		return true
	}

	return false
}

func (t *tenantLimitRepository) DefaultLimits() []Limit {
	return []Limit{
		{
			Resource: sqlcv1.LimitResourceTASKRUN,
			Limit:    t.config.DefaultTaskRunLimit,                // nolint: gosec
			Alarm:    Int32Ptr(t.config.DefaultTaskRunAlarmLimit), // nolint: gosec
			Window:   &t.config.DefaultTaskRunWindow,
		},
		{
			Resource: sqlcv1.LimitResourceEVENT,
			Limit:    t.config.DefaultEventLimit,                // nolint: gosec
			Alarm:    Int32Ptr(t.config.DefaultEventAlarmLimit), // nolint: gosec
			Window:   &t.config.DefaultEventWindow,
		},
		{
			Resource: sqlcv1.LimitResourceWORKER,
			Limit:    t.config.DefaultWorkerLimit,                // nolint: gosec
			Alarm:    Int32Ptr(t.config.DefaultWorkerAlarmLimit), // nolint: gosec
		},
		{
			Resource: sqlcv1.LimitResourceWORKERSLOT,
			Limit:    t.config.DefaultWorkerSlotLimit,                // nolint: gosec
			Alarm:    Int32Ptr(t.config.DefaultWorkerSlotAlarmLimit), // nolint: gosec
		},
		{
			Resource: sqlcv1.LimitResourceINCOMINGWEBHOOK,
			Limit:    t.config.DefaultIncomingWebhookLimit,                // nolint: gosec
			Alarm:    Int32Ptr(t.config.DefaultIncomingWebhookAlarmLimit), // nolint: gosec
		},
	}
}

func (t *tenantLimitRepository) GetLimits(ctx context.Context, tenantId uuid.UUID) ([]*sqlcv1.TenantResourceLimit, error) {
	if !t.enforceLimits {
		return []*sqlcv1.TenantResourceLimit{}, nil
	}

	limits, err := t.queries.ListTenantResourceLimits(ctx, t.pool, tenantId)

	if err != nil {
		return nil, err
	}

	// patch custom worker limits
	for _, limit := range limits {

		if limit.Resource == sqlcv1.LimitResourceWORKER {
			workerCount, err := t.queries.CountTenantWorkers(ctx, t.pool, tenantId)
			if err != nil {
				return nil, err
			}
			limit.Value = int32(workerCount) // nolint: gosec
		}

		if limit.Resource == sqlcv1.LimitResourceWORKERSLOT {
			totalSlotsRows, err := t.queries.ListTotalActiveSlotsPerTenant(ctx, t.pool)
			if err != nil {
				return nil, err
			}
			var workerSlotCount int32
			for _, row := range totalSlotsRows {
				if row.TenantId == tenantId {
					workerSlotCount = int32(row.TotalActiveSlots) // nolint: gosec
					break
				}
			}
			limit.Value = workerSlotCount
		}

	}

	return limits, nil
}

func (t *tenantLimitRepository) CanCreate(ctx context.Context, resource sqlcv1.LimitResource, tenantId uuid.UUID, numberOfResources int32) (bool, int, error) {
	return t.canCreate(ctx, nil, resource, tenantId, numberOfResources)
}

func (t *tenantLimitRepository) canCreate(ctx context.Context, dbtx sqlcv1.DBTX, resource sqlcv1.LimitResource, tenantId uuid.UUID, numberOfResources int32) (bool, int, error) {
	if !t.enforceLimits {
		return true, 0, nil
	}

	if dbtx == nil {
		dbtx = t.pool
	}

	limit, err := t.queries.GetTenantResourceLimit(ctx, dbtx, sqlcv1.GetTenantResourceLimitParams{
		Tenantid: tenantId,
		Resource: sqlcv1.NullLimitResource{
			LimitResource: resource,
			Valid:         true,
		},
	})

	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		t.l.Warn().Ctx(ctx).Msgf("no %s tenant limit found, creating default limit", string(resource))

		err = t.updateLimits(ctx, dbtx, tenantId, t.DefaultLimits())

		if err != nil {
			return false, 0, err
		}

		return true, 0, nil
	} else if err != nil {
		return false, 0, err
	}

	var value = limit.Value

	// patch custom worker limits aggregate methods
	if resource == sqlcv1.LimitResourceWORKER {
		count, err := t.queries.CountTenantWorkers(ctx, dbtx, tenantId)
		value = int32(count) // nolint: gosec

		if err != nil {
			return false, 0, err
		}
	}

	// include meters accepted but not yet flushed (same idea as rateLimiter.subtractUnflushed)
	if !hasCustomValueMeter(resource) {
		value += t.unflushedAmount(tenantId, resource)
	}

	// subtract 1 for backwards compatibility

	if value+numberOfResources-1 >= limit.LimitValue {
		return false, 100, nil
	}

	return true, calcPercent(value+numberOfResources, limit.LimitValue), nil
}

func calcPercent(value int32, limit int32) int {
	return int((float64(value) / float64(limit)) * 100)
}

func (t *tenantLimitRepository) unflushedAmount(tenantId uuid.UUID, resource sqlcv1.LimitResource) int32 {
	t.unflushedMu.RLock()
	defer t.unflushedMu.RUnlock()

	return t.unflushed[meterKey{tenantId: tenantId, resource: resource}]
}

func (t *tenantLimitRepository) addToUnflushed(resource sqlcv1.LimitResource, tenantId uuid.UUID, numberOfResources int32) {
	if numberOfResources == 0 {
		return
	}

	t.unflushedMu.Lock()
	defer t.unflushedMu.Unlock()

	t.unflushed[meterKey{tenantId: tenantId, resource: resource}] += numberOfResources
}

func (t *tenantLimitRepository) loopFlush(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := t.flushToDatabase(ctx)

			if err != nil {
				t.l.Error().Ctx(ctx).Err(err).Msg("error flushing tenant resource meters")
			}
		}
	}
}

// flushToDatabase writes coalesced meter deltas in a single bulk update. The
// unflushed set is swapped out before the DB call so the hot paths (Meter
// postcommit, canCreate) never block on a slow flush since mutex is global.
func (t *tenantLimitRepository) flushToDatabase(ctx context.Context) error {
	if !t.enforceLimits {
		return nil
	}

	t.unflushedMu.Lock()

	if len(t.unflushed) == 0 {
		t.unflushedMu.Unlock()
		return nil
	}

	updates := t.unflushed
	t.unflushed = make(meterSet)
	t.unflushedMu.Unlock()

	tenantIds := make([]uuid.UUID, 0, len(updates))
	resources := make([]string, 0, len(updates))
	numResources := make([]int32, 0, len(updates))

	for key, n := range updates {
		tenantIds = append(tenantIds, key.tenantId)
		resources = append(resources, string(key.resource))
		numResources = append(numResources, n)
	}

	limits, err := t.queries.BulkMeterTenantResources(ctx, t.pool, sqlcv1.BulkMeterTenantResourcesParams{
		Tenantids:    tenantIds,
		Resources:    resources,
		Numresources: numResources,
	})

	if err != nil {
		// merge the deltas back so the next flush retries them
		t.unflushedMu.Lock()
		for key, n := range updates {
			t.unflushed[key] += n
		}
		t.unflushedMu.Unlock()

		return err
	}

	for _, limit := range limits {
		percent := calcPercent(limit.Value, limit.LimitValue)
		if percent <= 50 || percent >= 100 {
			key := fmt.Sprintf("%s:%s", limit.Resource, limit.TenantId)
			t.c.Set(key, limit.Value < limit.LimitValue)
		}
	}

	return nil
}

func (t *tenantLimitRepository) cachedCanCreate(ctx context.Context, dbtx sqlcv1.DBTX, resource sqlcv1.LimitResource, tenantId uuid.UUID, numberOfResources int32) (bool, int, error) {
	var key = fmt.Sprintf("%s:%s", resource, tenantId)

	var canCreate *bool
	var percent int

	if hit, ok := t.c.Get(key); ok {
		c := hit.(bool)
		canCreate = &c
	}

	if canCreate == nil {
		c, p, err := t.canCreate(ctx, dbtx, resource, tenantId, numberOfResources)

		if err != nil {
			return false, 0, fmt.Errorf("could not check tenant limit: %w", err)
		}

		canCreate = &c
		percent = p

		if percent <= 50 || percent >= 100 {
			t.c.Set(key, c)
		}
	}

	return *canCreate, percent, nil
}

func (t *tenantLimitRepository) Meter(ctx context.Context, dbtx sqlcv1.DBTX, resource sqlcv1.LimitResource, tenantId uuid.UUID, numberOfResources int32) (precommit func() error, postcommit func()) {
	return func() error {
			canCreate, _, err := t.cachedCanCreate(ctx, dbtx, resource, tenantId, numberOfResources)

			if err != nil {
				return fmt.Errorf("could not check tenant limit: %w", err)
			}

			if !canCreate {
				return ErrResourceExhausted
			}

			return nil
		}, func() {
			if !t.enforceLimits || numberOfResources == 0 {
				return
			}

			t.addToUnflushed(resource, tenantId, numberOfResources)
		}
}

func (t *tenantLimitRepository) UpdateLimits(ctx context.Context, tenantId uuid.UUID, limits []Limit) error {
	return t.updateLimits(ctx, nil, tenantId, limits)
}

func (t *tenantLimitRepository) updateLimits(ctx context.Context, dbtx sqlcv1.DBTX, tenantId uuid.UUID, limits []Limit) error {
	if len(limits) == 0 {
		return nil
	}

	if dbtx == nil {
		dbtx = t.pool
	}

	resources := make([]string, len(limits))
	limitValues := make([]int32, len(limits))
	alarmValues := make([]int32, len(limits))
	windows := make([]string, len(limits))
	customValueMeters := make([]bool, len(limits))

	for i, limit := range limits {
		resources[i] = string(limit.Resource)
		limitValues[i] = limit.Limit
		customValueMeters[i] = hasCustomValueMeter(limit.Resource)

		if limit.Alarm != nil {
			alarmValues[i] = *limit.Alarm
		} else {
			alarmValues[i] = int32(float64(limit.Limit) * 0.8) // nolint: gosec
		}

		if limit.Window != nil {
			windows[i] = limit.Window.String()
		}
	}

	return t.queries.UpsertTenantResourceLimits(ctx, dbtx, sqlcv1.UpsertTenantResourceLimitsParams{
		Tenantid:          tenantId,
		Resources:         resources,
		Limitvalues:       limitValues,
		Alarmvalues:       alarmValues,
		Windows:           windows,
		Customvaluemeters: customValueMeters,
	})
}

var ErrResourceExhausted = fmt.Errorf("resource exhausted")

func (t *tenantLimitRepository) Stop() {
	if t.cleanup != nil {
		t.cleanup()
	}

	// final flush (rateLimiter leaves this to the next process; metering should not drop counts)
	flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := t.flushToDatabase(flushCtx); err != nil {
		t.l.Error().Err(err).Msg("error flushing tenant resource meters on shutdown")
	}

	t.c.Stop()
}
