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
	ListActiveSDKsPerTenant() (map[TenantIdSDKTuple]int64, error)
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
	OperatingSystem string
	Language        string
	LanguageVersion string
	SdkVersion      string
}

type TenantIdSDKTuple struct {
	TenantId string
	SDK      SDK
}

func (w *workerRepository) ListActiveSDKsPerTenant() (map[TenantIdSDKTuple]int64, error) {
	sdks, err := w.queries.ListActiveSDKsPerTenant(context.Background(), w.pool)

	if err != nil {
		return nil, fmt.Errorf("could not list active sdks per tenant: %w", err)
	}

	tenantIdSDKTupleToCount := make(map[TenantIdSDKTuple]int64)

	for _, sdk := range sdks {
		tenantId := sdk.TenantId.String()
		tenantIdSdkTuple := TenantIdSDKTuple{
			TenantId: tenantId,
			SDK: SDK{
				OperatingSystem: sdk.Os,
				Language:        sdk.Language,
				LanguageVersion: sdk.LanguageVersion,
				SdkVersion:      sdk.SdkVersion,
			},
		}

		tenantIdSDKTupleToCount[tenantIdSdkTuple] = sdk.Count
	}

	return tenantIdSDKTupleToCount, nil
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
