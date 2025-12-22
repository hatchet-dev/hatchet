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
