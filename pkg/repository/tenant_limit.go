package repository

import (
	"context"
	"errors"
	"fmt"
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

	Meter(ctx context.Context, resource sqlcv1.LimitResource, tenantId uuid.UUID, numberOfResources int32) (precommit func() error, postcommit func())

	Stop()
}

type tenantLimitRepository struct {
	c cache.Cacheable
	*sharedRepository
	config        limits.LimitConfigFile
	enforceLimits bool
}

func newTenantLimitRepository(shared *sharedRepository, s limits.LimitConfigFile, enforceLimits bool, cacheDuration time.Duration) TenantLimitRepository {
	return &tenantLimitRepository{
		sharedRepository: shared,
		config:           s,
		enforceLimits:    enforceLimits,
		c:                cache.New(cacheDuration),
	}
}

func (t *tenantLimitRepository) ResolveAllTenantResourceLimits(ctx context.Context) error {
	_, err := t.queries.ResolveAllLimitsIfWindowPassed(ctx, t.pool)
	return err
}

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
			workerSlotCount, err := t.queries.CountTenantWorkerSlots(ctx, t.pool, tenantId)
			if err != nil {
				return nil, err
			}
			limit.Value = workerSlotCount
		}

	}

	return limits, nil
}

func (t *tenantLimitRepository) CanCreate(ctx context.Context, resource sqlcv1.LimitResource, tenantId uuid.UUID, numberOfResources int32) (bool, int, error) {
	if !t.enforceLimits {
		return true, 0, nil
	}

	limit, err := t.queries.GetTenantResourceLimit(ctx, t.pool, sqlcv1.GetTenantResourceLimitParams{
		Tenantid: tenantId,
		Resource: sqlcv1.NullLimitResource{
			LimitResource: resource,
			Valid:         true,
		},
	})

	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		t.l.Warn().Msgf("no %s tenant limit found, creating default limit", string(resource))

		err = t.UpdateLimits(ctx, tenantId, t.DefaultLimits())

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
		count, err := t.queries.CountTenantWorkers(ctx, t.pool, tenantId)
		value = int32(count) // nolint: gosec

		if err != nil {
			return false, 0, err
		}
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

func (t *tenantLimitRepository) saveMeter(ctx context.Context, resource sqlcv1.LimitResource, tenantId uuid.UUID, numberOfResources int32) (*sqlcv1.TenantResourceLimit, error) {
	if !t.enforceLimits {
		return nil, nil
	}

	r, err := t.queries.MeterTenantResource(ctx, t.pool, sqlcv1.MeterTenantResourceParams{
		Tenantid: tenantId,
		Resource: sqlcv1.NullLimitResource{
			LimitResource: resource,
			Valid:         true,
		},
		Numresources: numberOfResources,
	})

	if err != nil {
		return nil, err
	}

	return r, nil
}

func (t *tenantLimitRepository) cachedCanCreate(ctx context.Context, resource sqlcv1.LimitResource, tenantId uuid.UUID, numberOfResources int32) (bool, int, error) {
	var key = fmt.Sprintf("%s:%s", resource, tenantId)

	var canCreate *bool
	var percent int

	if hit, ok := t.c.Get(key); ok {
		c := hit.(bool)
		canCreate = &c
	}

	if canCreate == nil {
		c, p, err := t.CanCreate(ctx, resource, tenantId, numberOfResources)

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

func (t *tenantLimitRepository) Meter(ctx context.Context, resource sqlcv1.LimitResource, tenantId uuid.UUID, numberOfResources int32) (precommit func() error, postcommit func()) {
	return func() error {
			canCreate, _, err := t.cachedCanCreate(ctx, resource, tenantId, numberOfResources)

			if err != nil {
				return fmt.Errorf("could not check tenant limit: %w", err)
			}

			if !canCreate {
				return ErrResourceExhausted
			}

			return nil
		}, func() {
			_, percent, err := t.cachedCanCreate(ctx, resource, tenantId, numberOfResources)

			if err != nil {
				t.l.Error().Err(err).Msg("could not check tenant limit")
				return
			}

			limit, err := t.saveMeter(ctx, resource, tenantId, numberOfResources)

			if limit != nil && (percent <= 50 || percent >= 100) {
				var key = fmt.Sprintf("%s:%s", resource, tenantId)
				t.c.Set(key, limit.Value < limit.LimitValue)
			}

			if err != nil {
				t.l.Error().Err(err).Msg("could not meter resource")
			}
		}
}

func (t *tenantLimitRepository) UpdateLimits(ctx context.Context, tenantId uuid.UUID, limits []Limit) error {
	if len(limits) == 0 {
		return nil
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

	return t.queries.UpsertTenantResourceLimits(ctx, t.pool, sqlcv1.UpsertTenantResourceLimitsParams{
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
	t.c.Stop()
}
