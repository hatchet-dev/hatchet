package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/config/limits"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type TenantLimitConfig struct {
	EnforceLimits bool
}

type Limit struct {
	Resource         sqlcv1.LimitResource
	Limit            int32
	Alarm            int32
	Window           *time.Duration
	CustomValueMeter bool
}

type PlanLimitMap map[string][]Limit

type TenantLimitRepository interface {
	GetLimits(ctx context.Context, tenantId uuid.UUID) ([]*sqlcv1.TenantResourceLimit, error)

	// CanCreate checks if the tenant can create a resource
	CanCreate(ctx context.Context, resource sqlcv1.LimitResource, tenantId uuid.UUID, numberOfResources int32) (bool, int, error)

	// Create new Tenant Resource Limits for a tenant
	SelectOrInsertTenantLimits(ctx context.Context, tenantId uuid.UUID, plan *string) error

	// UpsertTenantLimits updates or inserts new tenant limits
	UpsertTenantLimits(ctx context.Context, tenantId uuid.UUID, plan *string) error

	// Resolve all tenant resource limits
	ResolveAllTenantResourceLimits(ctx context.Context) error

	// SetPlanLimitMap sets the plan limit map
	SetPlanLimitMap(planLimitMap PlanLimitMap) error

	DefaultLimits() []Limit

	Stop()

	Meter(ctx context.Context, resource sqlcv1.LimitResource, tenantId uuid.UUID, numberOfResources int32) (precommit func() error, postcommit func())

	SetOnSuccessMeterCallback(cb func(resource sqlcv1.LimitResource, tenantId uuid.UUID, currentUsage int64))
}

type tenantLimitRepository struct {
	c cache.Cacheable
	*sharedRepository
	plans             *PlanLimitMap
	enforceLimitsFunc func(ctx context.Context, tenantId string) (bool, error)
	onSuccessMeterCb  func(resource sqlcv1.LimitResource, tenantId uuid.UUID, currentUsage int64)
	config            limits.LimitConfigFile
	enforceLimits     bool
}

func newTenantLimitRepository(shared *sharedRepository, s limits.LimitConfigFile, enforceLimits bool, enforceLimitsFunc func(ctx context.Context, tenantId string) (bool, error), cacheDuration time.Duration) TenantLimitRepository {
	return &tenantLimitRepository{
		sharedRepository:  shared,
		config:            s,
		enforceLimits:     enforceLimits,
		plans:             nil,
		c:                 cache.New(cacheDuration),
		enforceLimitsFunc: enforceLimitsFunc,
	}
}

func (t *tenantLimitRepository) ResolveAllTenantResourceLimits(ctx context.Context) error {
	_, err := t.queries.ResolveAllLimitsIfWindowPassed(ctx, t.pool)
	return err
}

func (t *tenantLimitRepository) SetPlanLimitMap(planLimitMap PlanLimitMap) error {
	t.plans = &planLimitMap
	return nil
}

func (t *tenantLimitRepository) DefaultLimits() []Limit {
	return []Limit{
		{
			Resource:         sqlcv1.LimitResourceTASKRUN,
			Limit:            int32(t.config.DefaultTaskRunLimit),      // nolint: gosec
			Alarm:            int32(t.config.DefaultTaskRunAlarmLimit), // nolint: gosec
			Window:           &t.config.DefaultTaskRunWindow,
			CustomValueMeter: false,
		},
		{
			Resource:         sqlcv1.LimitResourceEVENT,
			Limit:            int32(t.config.DefaultEventLimit),      // nolint: gosec
			Alarm:            int32(t.config.DefaultEventAlarmLimit), // nolint: gosec
			Window:           &t.config.DefaultEventWindow,
			CustomValueMeter: false,
		},
		{
			Resource:         sqlcv1.LimitResourceWORKER,
			Limit:            int32(t.config.DefaultWorkerLimit),      // nolint: gosec
			Alarm:            int32(t.config.DefaultWorkerAlarmLimit), // nolint: gosec
			Window:           nil,
			CustomValueMeter: true,
		},
		{
			Resource:         sqlcv1.LimitResourceWORKERSLOT,
			Limit:            int32(t.config.DefaultWorkerSlotLimit),      // nolint: gosec
			Alarm:            int32(t.config.DefaultWorkerSlotAlarmLimit), // nolint: gosec
			Window:           nil,
			CustomValueMeter: true,
		},
		{
			Resource:         sqlcv1.LimitResourceINCOMINGWEBHOOK,
			Limit:            int32(t.config.DefaultIncomingWebhookLimit),      // nolint: gosec
			Alarm:            int32(t.config.DefaultIncomingWebhookAlarmLimit), // nolint: gosec
			Window:           nil,
			CustomValueMeter: true,
		},
	}
}

func (t *tenantLimitRepository) planLimitMap(plan *string) []Limit {

	if t.plans == nil || plan == nil {
		return t.DefaultLimits()
	}

	if _, ok := (*t.plans)[*plan]; !ok {
		t.l.Warn().Msgf("plan %s not found, using default limits", *plan)
		return t.DefaultLimits()
	}

	return (*t.plans)[*plan]
}

func (t *tenantLimitRepository) SelectOrInsertTenantLimits(ctx context.Context, tenantId uuid.UUID, plan *string) error {

	planLimits := t.planLimitMap(plan)

	for _, limits := range planLimits {
		err := t.patchTenantResourceLimit(ctx, tenantId, limits, false)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *tenantLimitRepository) UpsertTenantLimits(ctx context.Context, tenantId uuid.UUID, plan *string) error {
	planLimits := t.planLimitMap(plan)

	for _, limits := range planLimits {
		err := t.patchTenantResourceLimit(ctx, tenantId, limits, true)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *tenantLimitRepository) patchTenantResourceLimit(ctx context.Context, tenantId uuid.UUID, limits Limit, upsert bool) error {

	limit := pgtype.Int4{}

	if limits.Limit >= 0 {
		limit.Int32 = limits.Limit
		limit.Valid = true
	}

	alarm := pgtype.Int4{}

	if limits.Alarm >= 0 {
		alarm.Int32 = limits.Alarm
		alarm.Valid = true
	}

	window := pgtype.Text{}

	if limits.Window != nil {
		window.String = limits.Window.String()
		window.Valid = true
	}

	cvm := pgtype.Bool{Bool: false, Valid: true}

	if limits.CustomValueMeter {
		cvm.Bool = true
	}

	if upsert {
		_, err := t.queries.UpsertTenantResourceLimit(ctx, t.pool, sqlcv1.UpsertTenantResourceLimitParams{
			Tenantid: tenantId,
			Resource: sqlcv1.NullLimitResource{
				LimitResource: limits.Resource,
				Valid:         true,
			},
			LimitValue:       limit,
			AlarmValue:       alarm,
			Window:           window,
			CustomValueMeter: cvm,
		})

		return err
	}

	_, err := t.queries.SelectOrInsertTenantResourceLimit(ctx, t.pool, sqlcv1.SelectOrInsertTenantResourceLimitParams{
		Tenantid: tenantId,
		Resource: sqlcv1.NullLimitResource{
			LimitResource: limits.Resource,
			Valid:         true,
		},
		LimitValue:       limit,
		AlarmValue:       alarm,
		Window:           window,
		CustomValueMeter: cvm,
	})

	return err
}

func (t *tenantLimitRepository) GetLimits(ctx context.Context, tenantId uuid.UUID) ([]*sqlcv1.TenantResourceLimit, error) {
	if t.enforceLimitsFunc != nil {
		enforce, err := t.enforceLimitsFunc(ctx, tenantId.String())
		if err != nil {
			return nil, err
		}

		if !enforce {
			return []*sqlcv1.TenantResourceLimit{}, nil
		}
	} else if !t.enforceLimits {
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
	if t.enforceLimitsFunc != nil {
		enforce, err := t.enforceLimitsFunc(ctx, tenantId.String())
		if err != nil {
			return false, 0, err
		}

		if !enforce {
			return true, 0, nil
		}
	} else if !t.enforceLimits {
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

		err = t.SelectOrInsertTenantLimits(ctx, tenantId, nil)

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

func (t *tenantLimitRepository) SetOnSuccessMeterCallback(cb func(resource sqlcv1.LimitResource, tenantId uuid.UUID, currentUsage int64)) {
	t.onSuccessMeterCb = cb
}

func calcPercent(value int32, limit int32) int {
	return int((float64(value) / float64(limit)) * 100)
}

func (t *tenantLimitRepository) saveMeter(ctx context.Context, resource sqlcv1.LimitResource, tenantId uuid.UUID, numberOfResources int32) (*sqlcv1.TenantResourceLimit, error) {
	if t.enforceLimitsFunc != nil {
		enforce, err := t.enforceLimitsFunc(ctx, tenantId.String())
		if err != nil {
			return nil, err
		}

		if !enforce {
			return nil, nil
		}
	} else if !t.enforceLimits {
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

	if t.onSuccessMeterCb != nil {
		go func() { // non-blocking callback
			t.onSuccessMeterCb(resource, tenantId, int64(r.Value))
		}()
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

var ErrResourceExhausted = fmt.Errorf("resource exhausted")

func (t *tenantLimitRepository) Stop() {
	t.c.Stop()
}
