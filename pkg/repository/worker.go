package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type RuntimeInfo struct {
	SdkVersion      *string         `validate:"omitempty"`
	Language        *contracts.SDKS `validate:"omitempty"`
	LanguageVersion *string         `validate:"omitempty"`
	Os              *string         `validate:"omitempty"`
	Extra           *string         `validate:"omitempty"`
}

type CreateWorkerOpts struct {
	// The id of the dispatcher
	DispatcherId uuid.UUID `validate:"required,uuid"`

	// The maximum number of runs this worker can run at a time
	MaxRuns *int `validate:"omitempty,gte=1"`

	// The name of the worker
	Name string `validate:"required,hatchetName"`

	// The name of the service
	Services []string `validate:"dive,hatchetName"`

	// A list of actions this worker can run
	Actions []string `validate:"dive,actionId"`

	// (optional) Runtime info for the worker
	RuntimeInfo *RuntimeInfo `validate:"omitempty"`
}

type UpdateWorkerOpts struct {
	// The id of the dispatcher
	DispatcherId *string `validate:"omitempty,uuid"`

	// When the last worker heartbeat was
	LastHeartbeatAt *time.Time

	// If the worker is active and accepting new runs
	IsActive *bool

	// A list of actions this worker can run
	Actions []string `validate:"dive,actionId"`

	// If the worker is paused
	IsPaused *bool
}

type ListWorkersOpts struct {
	Action *string `validate:"omitempty,actionId"`

	LastHeartbeatAfter *time.Time

	Assignable *bool
}

type UpsertWorkerLabelOpts struct {
	Key      string
	IntValue *int32
	StrValue *string
}

type WorkerRepository interface {
	ListWorkers(tenantId uuid.UUID, opts *ListWorkersOpts) ([]*sqlcv1.ListWorkersWithSlotCountRow, error)
	GetWorkerById(workerId string) (*sqlcv1.GetWorkerByIdRow, error)
	ListWorkerState(tenantId uuid.UUID, workerId string, maxRuns int) ([]*sqlcv1.ListSemaphoreSlotsWithStateForWorkerRow, error)
	CountActiveSlotsPerTenant() (map[uuid.UUID]int64, error)
	CountActiveWorkersPerTenant() (map[uuid.UUID]int64, error)
	ListActiveSDKsPerTenant() (map[TenantIdSDKTuple]int64, error)

	// GetWorkerActionsByWorkerId returns a list of actions for a worker
	GetWorkerActionsByWorkerId(tenantId uuid.UUID, workerId []string) (map[string][]string, error)

	// GetWorkerWorkflowsByWorkerId returns a list of workflows for a worker
	GetWorkerWorkflowsByWorkerId(tenantId uuid.UUID, workerId string) ([]*sqlcv1.Workflow, error)

	// ListWorkerLabels returns a list of labels config for a worker
	ListWorkerLabels(tenantId uuid.UUID, workerId string) ([]*sqlcv1.ListWorkerLabelsRow, error)

	// CreateNewWorker creates a new worker for a given tenant.
	CreateNewWorker(ctx context.Context, tenantId uuid.UUID, opts *CreateWorkerOpts) (*sqlcv1.Worker, error)

	// UpdateWorker updates a worker for a given tenant.
	UpdateWorker(ctx context.Context, tenantId uuid.UUID, workerId string, opts *UpdateWorkerOpts) (*sqlcv1.Worker, error)

	// UpdateWorker updates a worker in the
	// It will only update the worker if there is no lock on the worker, else it will skip.
	UpdateWorkerHeartbeat(ctx context.Context, tenantId uuid.UUID, workerId string, lastHeartbeatAt time.Time) error

	// DeleteWorker removes the worker from the database
	DeleteWorker(ctx context.Context, tenantId uuid.UUID, workerId string) error

	GetWorkerForEngine(ctx context.Context, tenantId uuid.UUID, workerId string) (*sqlcv1.GetWorkerForEngineRow, error)

	UpdateWorkerActiveStatus(ctx context.Context, tenantId uuid.UUID, workerId string, isActive bool, timestamp time.Time) (*sqlcv1.Worker, error)

	UpsertWorkerLabels(ctx context.Context, workerId uuid.UUID, opts []UpsertWorkerLabelOpts) ([]*sqlcv1.WorkerLabel, error)

	DeleteOldWorkers(ctx context.Context, tenantId uuid.UUID, lastHeartbeatBefore time.Time) (bool, error)

	GetDispatcherIdsForWorkers(ctx context.Context, tenantId uuid.UUID, workerIds []string) (map[string][]string, error)
}

type workerRepository struct {
	*sharedRepository
}

func newWorkerRepository(shared *sharedRepository) WorkerRepository {
	return &workerRepository{
		sharedRepository: shared,
	}
}

func (w *workerRepository) ListWorkers(tenantId uuid.UUID, opts *ListWorkersOpts) ([]*sqlcv1.ListWorkersWithSlotCountRow, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	queryParams := sqlcv1.ListWorkersWithSlotCountParams{
		Tenantid: tenantId,
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

	workers, err := w.queries.ListWorkersWithSlotCount(context.Background(), w.pool, queryParams)

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
	return w.queries.GetWorkerById(context.Background(), w.pool, uuid.MustParse(workerId))
}

func (w *workerRepository) ListWorkerState(tenantId uuid.UUID, workerId string, maxRuns int) ([]*sqlcv1.ListSemaphoreSlotsWithStateForWorkerRow, error) {
	slots, err := w.queries.ListSemaphoreSlotsWithStateForWorker(context.Background(), w.pool, sqlcv1.ListSemaphoreSlotsWithStateForWorkerParams{
		Workerid: uuid.MustParse(workerId),
		Tenantid: tenantId,
		Limit: pgtype.Int4{
			Int32: int32(maxRuns), // nolint: gosec
			Valid: true,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("could not list worker slot state: %w", err)
	}

	return slots, nil
}

func (w *workerRepository) CountActiveSlotsPerTenant() (map[uuid.UUID]int64, error) {
	slots, err := w.queries.ListTotalActiveSlotsPerTenant(context.Background(), w.pool)

	if err != nil {
		return nil, fmt.Errorf("could not list active slots per tenant: %w", err)
	}

	tenantToSlots := make(map[uuid.UUID]int64)

	for _, slot := range slots {
		tenantToSlots[slot.TenantId] = slot.TotalActiveSlots
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
	TenantId uuid.UUID
	SDK      SDK
}

func (w *workerRepository) ListActiveSDKsPerTenant() (map[TenantIdSDKTuple]int64, error) {
	sdks, err := w.queries.ListActiveSDKsPerTenant(context.Background(), w.pool)

	if err != nil {
		return nil, fmt.Errorf("could not list active sdks per tenant: %w", err)
	}

	tenantIdSDKTupleToCount := make(map[TenantIdSDKTuple]int64)

	for _, sdk := range sdks {
		tenantId := sdk.TenantId
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

func (w *workerRepository) CountActiveWorkersPerTenant() (map[uuid.UUID]int64, error) {
	workers, err := w.queries.ListActiveWorkersPerTenant(context.Background(), w.pool)

	if err != nil {
		return nil, fmt.Errorf("could not list active workers per tenant: %w", err)
	}

	tenantToWorkers := make(map[uuid.UUID]int64)

	for _, worker := range workers {
		tenantToWorkers[worker.TenantId] = worker.Count
	}

	return tenantToWorkers, nil
}

func (w *workerRepository) GetWorkerActionsByWorkerId(tenantId uuid.UUID, workerIds []string) (map[string][]string, error) {
	uuidWorkerIds := make([]uuid.UUID, len(workerIds))
	for i, workerId := range workerIds {
		uuidWorkerIds[i] = uuid.MustParse(workerId)
	}

	records, err := w.queries.GetWorkerActionsByWorkerId(context.Background(), w.pool, sqlcv1.GetWorkerActionsByWorkerIdParams{
		Workerids: uuidWorkerIds,
		Tenantid:  tenantId,
	})

	if err != nil {
		return nil, err
	}

	workerIdToActionIds := make(map[string][]string)

	for _, record := range records {
		workerId := record.WorkerId.String()
		actionId := record.Actionid.String

		if _, ok := workerIdToActionIds[workerId]; !ok {
			workerIdToActionIds[workerId] = make([]string, 0)
		}

		workerIdToActionIds[workerId] = append(workerIdToActionIds[workerId], actionId)
	}

	return workerIdToActionIds, nil
}

func (w *workerRepository) GetWorkerWorkflowsByWorkerId(tenantId uuid.UUID, workerId string) ([]*sqlcv1.Workflow, error) {
	return w.queries.GetWorkerWorkflowsByWorkerId(context.Background(), w.pool, sqlcv1.GetWorkerWorkflowsByWorkerIdParams{
		Workerid: uuid.MustParse(workerId),
		Tenantid: tenantId,
	})
}

func (w *workerRepository) ListWorkerLabels(tenantId uuid.UUID, workerId string) ([]*sqlcv1.ListWorkerLabelsRow, error) {
	return w.queries.ListWorkerLabels(context.Background(), w.pool, uuid.MustParse(workerId))
}

func (w *workerRepository) GetWorkerForEngine(ctx context.Context, tenantId uuid.UUID, workerId string) (*sqlcv1.GetWorkerForEngineRow, error) {
	return w.queries.GetWorkerForEngine(ctx, w.pool, sqlcv1.GetWorkerForEngineParams{
		ID:       uuid.MustParse(workerId),
		Tenantid: tenantId,
	})
}

func (w *workerRepository) CreateNewWorker(ctx context.Context, tenantId uuid.UUID, opts *CreateWorkerOpts) (*sqlcv1.Worker, error) {
	preWorker, postWorker := w.m.Meter(ctx, sqlcv1.LimitResourceWORKER, tenantId, 1)

	if err := preWorker(); err != nil {
		return nil, err
	}

	maxRuns := int32(100)

	if opts.MaxRuns != nil {
		maxRuns = int32(*opts.MaxRuns) // nolint: gosec
	}

	preWorkerSlot, postWorkerSlot := w.m.Meter(ctx, sqlcv1.LimitResourceWORKERSLOT, tenantId, maxRuns)

	if err := preWorkerSlot(); err != nil {
		return nil, err
	}

	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	tx, err := w.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer sqlchelpers.DeferRollback(ctx, w.l, tx.Rollback)

	createParams := sqlcv1.CreateWorkerParams{
		Tenantid:     tenantId,
		Dispatcherid: uuid.MustParse(opts.DispatcherId),
		Name:         opts.Name,
	}

	// Default to self hosted
	createParams.Type = sqlcv1.NullWorkerType{
		WorkerType: sqlcv1.WorkerTypeSELFHOSTED,
		Valid:      true,
	}

	if opts.MaxRuns != nil {
		createParams.MaxRuns = pgtype.Int4{
			Int32: int32(*opts.MaxRuns), // nolint: gosec
			Valid: true,
		}
	} else {
		createParams.MaxRuns = pgtype.Int4{
			Int32: 100,
			Valid: true,
		}
	}

	var worker *sqlcv1.Worker

	if opts.RuntimeInfo != nil {
		if opts.RuntimeInfo.SdkVersion != nil {
			createParams.SdkVersion = sqlchelpers.TextFromStr(*opts.RuntimeInfo.SdkVersion)
		}
		if opts.RuntimeInfo.Language != nil {
			switch *opts.RuntimeInfo.Language {
			case contracts.SDKS_GO:
				createParams.Language = sqlcv1.NullWorkerSDKS{
					WorkerSDKS: sqlcv1.WorkerSDKSGO,
					Valid:      true,
				}
			case contracts.SDKS_PYTHON:
				createParams.Language = sqlcv1.NullWorkerSDKS{
					WorkerSDKS: sqlcv1.WorkerSDKSPYTHON,
					Valid:      true,
				}
			case contracts.SDKS_TYPESCRIPT:
				createParams.Language = sqlcv1.NullWorkerSDKS{
					WorkerSDKS: sqlcv1.WorkerSDKSTYPESCRIPT,
					Valid:      true,
				}
			default:
				return nil, fmt.Errorf("invalid sdk: %s", *opts.RuntimeInfo.Language)
			}
		}
		if opts.RuntimeInfo.LanguageVersion != nil {
			createParams.LanguageVersion = sqlchelpers.TextFromStr(*opts.RuntimeInfo.LanguageVersion)
		}
		if opts.RuntimeInfo.Os != nil {
			createParams.Os = sqlchelpers.TextFromStr(*opts.RuntimeInfo.Os)
		}
		if opts.RuntimeInfo.Extra != nil {
			createParams.RuntimeExtra = sqlchelpers.TextFromStr(*opts.RuntimeInfo.Extra)
		}
	}

	if worker == nil {
		worker, err = w.queries.CreateWorker(ctx, tx, createParams)

		if err != nil {
			return nil, fmt.Errorf("could not create worker: %w", err)
		}
	}

	svcUUIDs := make([]uuid.UUID, len(opts.Services))

	for i, svc := range opts.Services {
		dbSvc, err := w.queries.UpsertService(ctx, tx, sqlcv1.UpsertServiceParams{
			Name:     svc,
			Tenantid: tenantId,
		})

		if err != nil {
			return nil, fmt.Errorf("could not upsert service: %w", err)
		}

		svcUUIDs[i] = dbSvc.ID
	}

	err = w.queries.LinkServicesToWorker(ctx, tx, sqlcv1.LinkServicesToWorkerParams{
		Services: svcUUIDs,
		Workerid: worker.ID,
	})

	if err != nil {
		return nil, fmt.Errorf("could not link services to worker: %w", err)
	}

	actionUUIDs := make([]uuid.UUID, len(opts.Actions))

	for i, action := range opts.Actions {
		dbAction, err := w.queries.UpsertAction(ctx, tx, sqlcv1.UpsertActionParams{
			Action:   action,
			Tenantid: tenantId,
		})

		if err != nil {
			return nil, fmt.Errorf("could not upsert action: %w", err)
		}

		actionUUIDs[i] = dbAction.ID
	}

	err = w.queries.LinkActionsToWorker(ctx, tx, sqlcv1.LinkActionsToWorkerParams{
		Actionids: actionUUIDs,
		Workerid:  worker.ID,
	})

	if err != nil {
		return nil, fmt.Errorf("could not link actions to worker: %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	postWorker()
	postWorkerSlot()

	return worker, nil
}

// UpdateWorker updates a worker.
// It will only update the worker if there is no lock on the worker, else it will skip.
func (w *workerRepository) UpdateWorker(ctx context.Context, tenantId uuid.UUID, workerId string, opts *UpdateWorkerOpts) (*sqlcv1.Worker, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	tx, err := w.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer sqlchelpers.DeferRollback(ctx, w.l, tx.Rollback)

	updateParams := sqlcv1.UpdateWorkerParams{
		ID: uuid.MustParse(workerId),
	}

	if opts.LastHeartbeatAt != nil {
		updateParams.LastHeartbeatAt = sqlchelpers.TimestampFromTime(*opts.LastHeartbeatAt)
	}

	if opts.DispatcherId != nil {
		parsed := uuid.MustParse(*opts.DispatcherId)
		updateParams.DispatcherId = &parsed
	}

	if opts.IsActive != nil {
		updateParams.IsActive = pgtype.Bool{
			Bool:  *opts.IsActive,
			Valid: true,
		}
	}

	if opts.IsPaused != nil {
		updateParams.IsPaused = pgtype.Bool{
			Bool:  *opts.IsPaused,
			Valid: true,
		}
	}

	worker, err := w.queries.UpdateWorker(ctx, tx, updateParams)

	if err != nil {
		return nil, fmt.Errorf("could not update worker: %w", err)
	}

	if len(opts.Actions) > 0 {
		actionUUIDs := make([]uuid.UUID, len(opts.Actions))

		for i, action := range opts.Actions {
			dbAction, err := w.queries.UpsertAction(ctx, tx, sqlcv1.UpsertActionParams{
				Action:   action,
				Tenantid: tenantId,
			})

			if err != nil {
				return nil, fmt.Errorf("could not upsert action: %w", err)
			}

			actionUUIDs[i] = dbAction.ID
		}

		err = w.queries.LinkActionsToWorker(ctx, tx, sqlcv1.LinkActionsToWorkerParams{
			Actionids: actionUUIDs,
			Workerid:  uuid.MustParse(workerId),
		})

		if err != nil {
			return nil, fmt.Errorf("could not link actions to worker: %w", err)
		}
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	return worker, nil
}

func (w *workerRepository) UpdateWorkerHeartbeat(ctx context.Context, tenantId uuid.UUID, workerId string, lastHeartbeat time.Time) error {
	_, err := w.queries.UpdateWorkerHeartbeat(ctx, w.pool, sqlcv1.UpdateWorkerHeartbeatParams{
		ID:              uuid.MustParse(workerId),
		LastHeartbeatAt: sqlchelpers.TimestampFromTime(lastHeartbeat),
	})

	if err != nil {
		return fmt.Errorf("could not update worker heartbeat: %w", err)
	}

	return nil
}

func (w *workerRepository) DeleteWorker(ctx context.Context, tenantId uuid.UUID, workerId string) error {
	_, err := w.queries.DeleteWorker(ctx, w.pool, uuid.MustParse(workerId))

	return err
}

func (w *workerRepository) UpdateWorkerActiveStatus(ctx context.Context, tenantId uuid.UUID, workerId string, isActive bool, timestamp time.Time) (*sqlcv1.Worker, error) {
	worker, err := w.queries.UpdateWorkerActiveStatus(ctx, w.pool, sqlcv1.UpdateWorkerActiveStatusParams{
		ID:                      uuid.MustParse(workerId),
		Isactive:                isActive,
		LastListenerEstablished: sqlchelpers.TimestampFromTime(timestamp),
	})

	if err != nil {
		return nil, fmt.Errorf("could not update worker active status: %w", err)
	}

	return worker, nil
}

func (w *workerRepository) UpsertWorkerLabels(ctx context.Context, workerId uuid.UUID, opts []UpsertWorkerLabelOpts) ([]*sqlcv1.WorkerLabel, error) {
	if len(opts) == 0 {
		return nil, nil
	}

	affinities := make([]*sqlcv1.WorkerLabel, 0, len(opts))

	for _, opt := range opts {

		intValue := pgtype.Int4{Valid: false}
		if opt.IntValue != nil {
			intValue = pgtype.Int4{
				Int32: *opt.IntValue,
				Valid: true,
			}
		}

		strValue := pgtype.Text{Valid: false}
		if opt.StrValue != nil {
			strValue = pgtype.Text{
				String: *opt.StrValue,
				Valid:  true,
			}
		}

		dbsqlcOpts := sqlcv1.UpsertWorkerLabelParams{
			Workerid: workerId,
			Key:      opt.Key,
			IntValue: intValue,
			StrValue: strValue,
		}

		affinity, err := w.queries.UpsertWorkerLabel(ctx, w.pool, dbsqlcOpts)
		if err != nil {
			return nil, fmt.Errorf("could not update worker affinity state: %w", err)
		}

		affinities = append(affinities, affinity)
	}

	return affinities, nil
}

func (w *workerRepository) DeleteOldWorkers(ctx context.Context, tenantId uuid.UUID, lastHeartbeatBefore time.Time) (bool, error) {
	hasMore, err := w.queries.DeleteOldWorkers(ctx, w.pool, sqlcv1.DeleteOldWorkersParams{
		Tenantid:            tenantId,
		Lastheartbeatbefore: sqlchelpers.TimestampFromTime(lastHeartbeatBefore),
		Limit:               20,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, err
	}

	return hasMore, nil
}

func (w *workerRepository) GetDispatcherIdsForWorkers(ctx context.Context, tenantId uuid.UUID, workerIds []string) (map[string][]string, error) {
	pgWorkerIds := make([]uuid.UUID, len(workerIds))

	for i, workerId := range workerIds {
		pgWorkerIds[i] = uuid.MustParse(workerId)
	}

	rows, err := w.queries.ListDispatcherIdsForWorkers(ctx, w.pool, sqlcv1.ListDispatcherIdsForWorkersParams{
		Tenantid:  tenantId,
		Workerids: sqlchelpers.UniqueSet(pgWorkerIds),
	})

	if err != nil {
		return nil, fmt.Errorf("could not get dispatcher ids for workers: %w", err)
	}

	dispatcherIdsToWorkers := make(map[string][]string)

	for _, row := range rows {
		dispatcherId := row.DispatcherId.String()
		workerId := row.WorkerId.String()

		if _, ok := dispatcherIdsToWorkers[dispatcherId]; !ok {
			dispatcherIdsToWorkers[dispatcherId] = make([]string, 0)
		}

		dispatcherIdsToWorkers[dispatcherId] = append(dispatcherIdsToWorkers[dispatcherId], workerId)
	}

	return dispatcherIdsToWorkers, nil
}
