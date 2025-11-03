package v1

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

type WorkerRepository interface {
	ListWorkers(tenantId string, opts *repository.ListWorkersOpts) ([]*sqlcv1.ListWorkersWithSlotCountRow, error)
	GetWorkerById(workerId string) (*sqlcv1.GetWorkerByIdRow, error)
	ListWorkerState(tenantId, workerId string, maxRuns int) ([]*sqlcv1.ListSemaphoreSlotsWithStateForWorkerRow, []*dbsqlc.GetStepRunForEngineRow, error)
	CountActiveSlotsPerTenant() (map[string]int64, error)
	CountActiveWorkersPerTenant() (map[string]int64, error)
	ListActiveSDKsPerTenant() (map[string][]SDK, error)
}

type workerRepository struct {
	*sharedRepository
}

func newWorkerRepository(shared *sharedRepository) WorkerRepository {
	return &workerRepository{
		sharedRepository: shared,
	}
}

func (r *workerRepository) ListWorkers(tenantId string, opts *repository.ListWorkersOpts) ([]*sqlcv1.ListWorkersWithSlotCountRow, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	queryParams := sqlcv1.ListWorkersWithSlotCountParams{
		Tenantid: pgTenantId,
	}

	if opts.Action != nil {
		queryParams.ActionId = sqlchelpers.TextFromStr(*opts.Action)
	}

	if opts.LastHeartbeatAfter != nil {
		queryParams.LastHeartbeatAfter = sqlchelpers.TimestampFromTime(opts.LastHeartbeatAfter.UTC())
	}

	if opts.Assignable != nil {
		queryParams.Assignable = pgtype.Bool{
			Bool:  *opts.Assignable,
			Valid: true,
		}
	}

	workers, err := r.queries.ListWorkersWithSlotCount(context.Background(), r.pool, queryParams)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			workers = make([]*sqlcv1.ListWorkersWithSlotCountRow, 0)
		} else {
			return nil, fmt.Errorf("could not list workers: %w", err)
		}
	}

	return workers, nil
}

func (w *workerRepository) GetWorkerById(workerId string) (*sqlcv1.GetWorkerByIdRow, error) {
	return w.queries.GetWorkerById(context.Background(), w.pool, sqlchelpers.UUIDFromStr(workerId))
}

func (w *workerRepository) ListWorkerState(tenantId, workerId string, maxRuns int) ([]*sqlcv1.ListSemaphoreSlotsWithStateForWorkerRow, []*dbsqlc.GetStepRunForEngineRow, error) {
	slots, err := w.queries.ListSemaphoreSlotsWithStateForWorker(context.Background(), w.pool, sqlcv1.ListSemaphoreSlotsWithStateForWorkerParams{
		Workerid: sqlchelpers.UUIDFromStr(workerId),
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Limit: pgtype.Int4{
			Int32: int32(maxRuns), // nolint: gosec
			Valid: true,
		},
	})

	if err != nil {
		return nil, nil, fmt.Errorf("could not list worker slot state: %w", err)
	}

	return slots, []*dbsqlc.GetStepRunForEngineRow{}, nil
}

func (w *workerRepository) CountActiveSlotsPerTenant() (map[string]int64, error) {
	slots, err := w.queries.ListTotalActiveSlotsPerTenant(context.Background(), w.pool)

	if err != nil {
		return nil, fmt.Errorf("could not list active slots per tenant: %w", err)
	}

	tenantToSlots := make(map[string]int64)

	for _, slot := range slots {
		tenantToSlots[slot.TenantId.String()] = slot.TotalActiveSlots
	}

	return tenantToSlots, nil
}

type SDK struct {
	OperatingSystem *string
	Language        *string
	LanguageVersion *string
	SdkVersion      *string
}

func (w *workerRepository) ListActiveSDKsPerTenant() (map[string][]SDK, error) {
	sdks, err := w.queries.ListActiveSDKsPerTenant(context.Background(), w.pool)

	if err != nil {
		return nil, fmt.Errorf("could not list active sdks per tenant: %w", err)
	}

	tenantToSDKs := make(map[string][]SDK)

	for _, sdk := range sdks {
		tenantId := sdk.TenantId.String()

		language := ""
		languageVersion := ""
		os := ""
		sdkVersion := ""

		if sdk.Language.Valid {
			language = string(sdk.Language.WorkerSDKS)
		}

		if sdk.LanguageVersion.Valid {
			languageVersion = sdk.LanguageVersion.String
		}

		if sdk.Os.Valid {
			os = sdk.Os.String
		}

		if sdk.SdkVersion.Valid {
			sdkVersion = sdk.SdkVersion.String
		}

		tenantToSDKs[tenantId] = append(tenantToSDKs[tenantId], SDK{
			OperatingSystem: &os,
			Language:        &language,
			LanguageVersion: &languageVersion,
			SdkVersion:      &sdkVersion,
		})
	}

	return tenantToSDKs, nil
}

func (w *workerRepository) CountActiveWorkersPerTenant() (map[string]int64, error) {
	workers, err := w.queries.ListActiveWorkersPerTenant(context.Background(), w.pool)

	if err != nil {
		return nil, fmt.Errorf("could not list active workers per tenant: %w", err)
	}

	tenantToWorkers := make(map[string]int64)

	for _, worker := range workers {
		tenantToWorkers[worker.TenantId.String()] = worker.Count
	}

	return tenantToWorkers, nil
}