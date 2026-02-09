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
	DispatcherId uuid.UUID `validate:"required"`

	// Slot config for this worker (slot_type -> max units)
	SlotConfig map[string]int32 `validate:"omitempty"`

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
	DispatcherId *uuid.UUID `validate:"omitempty"`

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
	ListWorkers(ctx context.Context, tenantId uuid.UUID, opts *ListWorkersOpts) ([]*sqlcv1.ListWorkersRow, error)
	GetWorkerById(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID) (*sqlcv1.GetWorkerByIdRow, error)
	ListTotalActiveSlotsPerTenant(ctx context.Context) (map[uuid.UUID]int64, error)
	ListActiveSlotsPerTenantAndSlotType(ctx context.Context) (map[TenantIdSlotTypeTuple]int64, error)
	CountActiveWorkersPerTenant(ctx context.Context) (map[uuid.UUID]int64, error)
	ListActiveSDKsPerTenant(ctx context.Context) (map[TenantIdSDKTuple]int64, error)

	// GetWorkerActionsByWorkerId returns a list of actions for a worker
	GetWorkerActionsByWorkerId(ctx context.Context, tenantId uuid.UUID, workerId []uuid.UUID) (map[string][]string, error)

	// GetWorkerWorkflowsByWorkerId returns a list of workflows for a worker
	GetWorkerWorkflowsByWorkerId(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID) ([]*sqlcv1.Workflow, error)

	// ListWorkerLabels returns a list of labels config for a worker
	ListWorkerLabels(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID) ([]*sqlcv1.ListWorkerLabelsRow, error)

	// ListWorkerSlotConfigs returns slot config for workers.
	ListWorkerSlotConfigs(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) (map[uuid.UUID]map[string]int32, error)

	// ListAvailableSlotsForWorkers returns available slot units by worker for a slot type.
	ListAvailableSlotsForWorkers(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID, slotType string) (map[uuid.UUID]int32, error)

	// ListAvailableSlotsForWorkersAndTypes returns available slot units by worker for a set of slot types.
	ListAvailableSlotsForWorkersAndTypes(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID, slotTypes []string) (map[uuid.UUID]map[string]int32, error)

	// CreateNewWorker creates a new worker for a given tenant.
	CreateNewWorker(ctx context.Context, tenantId uuid.UUID, opts *CreateWorkerOpts) (*sqlcv1.Worker, error)

	// UpdateWorker updates a worker for a given tenant.
	UpdateWorker(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, opts *UpdateWorkerOpts) (*sqlcv1.Worker, error)

	// UpdateWorker updates a worker in the
	// It will only update the worker if there is no lock on the worker, else it will skip.
	UpdateWorkerHeartbeat(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, lastHeartbeatAt time.Time) error

	// DeleteWorker removes the worker from the database
	DeleteWorker(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID) error

	GetWorkerForEngine(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID) (*sqlcv1.GetWorkerForEngineRow, error)

	UpdateWorkerActiveStatus(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, isActive bool, timestamp time.Time) (*sqlcv1.Worker, error)

	UpsertWorkerLabels(ctx context.Context, workerId uuid.UUID, opts []UpsertWorkerLabelOpts) ([]*sqlcv1.WorkerLabel, error)

	DeleteOldWorkers(ctx context.Context, tenantId uuid.UUID, lastHeartbeatBefore time.Time) (bool, error)

	GetDispatcherIdsForWorkers(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) (map[uuid.UUID]uuid.UUID, map[uuid.UUID]struct{}, error)
}

type workerRepository struct {
	*sharedRepository
}

func newWorkerRepository(shared *sharedRepository) WorkerRepository {
	return &workerRepository{
		sharedRepository: shared,
	}
}

func (w *workerRepository) ListWorkers(ctx context.Context, tenantId uuid.UUID, opts *ListWorkersOpts) ([]*sqlcv1.ListWorkersRow, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	queryParams := sqlcv1.ListWorkersParams{
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

	workers, err := w.queries.ListWorkers(ctx, w.pool, queryParams)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			workers = make([]*sqlcv1.ListWorkersRow, 0)
		} else {
			return nil, fmt.Errorf("could not list workers: %w", err)
		}
	}

	return workers, nil
}

func (w *workerRepository) GetWorkerById(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID) (*sqlcv1.GetWorkerByIdRow, error) {
	return w.queries.GetWorkerById(ctx, w.pool, sqlcv1.GetWorkerByIdParams{
		Tenantid: tenantId,
		ID:       workerId,
	})
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

type TenantIdSlotTypeTuple struct {
	TenantId uuid.UUID
	SlotType string
}

func (w *workerRepository) ListActiveSDKsPerTenant(ctx context.Context) (map[TenantIdSDKTuple]int64, error) {
	sdks, err := w.queries.ListActiveSDKsPerTenant(ctx, w.pool)

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

func (w *workerRepository) ListTotalActiveSlotsPerTenant(ctx context.Context) (map[uuid.UUID]int64, error) {
	rows, err := w.queries.ListTotalActiveSlotsPerTenant(ctx, w.pool)
	if err != nil {
		return nil, fmt.Errorf("could not list total active slots per tenant: %w", err)
	}

	tenantToSlots := make(map[uuid.UUID]int64, len(rows))
	for _, row := range rows {
		tenantToSlots[row.TenantId] = row.TotalActiveSlots
	}

	return tenantToSlots, nil
}

func (w *workerRepository) ListActiveSlotsPerTenantAndSlotType(ctx context.Context) (map[TenantIdSlotTypeTuple]int64, error) {
	rows, err := w.queries.ListActiveSlotsPerTenantAndSlotType(ctx, w.pool)
	if err != nil {
		return nil, fmt.Errorf("could not list active slots per tenant and slot type: %w", err)
	}

	res := make(map[TenantIdSlotTypeTuple]int64, len(rows))
	for _, row := range rows {
		res[TenantIdSlotTypeTuple{
			TenantId: row.TenantId,
			SlotType: row.SlotType,
		}] = row.ActiveSlots
	}

	return res, nil
}

func (w *workerRepository) CountActiveWorkersPerTenant(ctx context.Context) (map[uuid.UUID]int64, error) {
	workers, err := w.queries.ListActiveWorkersPerTenant(ctx, w.pool)

	if err != nil {
		return nil, fmt.Errorf("could not list active workers per tenant: %w", err)
	}

	tenantToWorkers := make(map[uuid.UUID]int64)

	for _, worker := range workers {
		tenantToWorkers[worker.TenantId] = worker.Count
	}

	return tenantToWorkers, nil
}

func (w *workerRepository) GetWorkerActionsByWorkerId(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) (map[string][]string, error) {
	records, err := w.queries.GetWorkerActionsByWorkerId(ctx, w.pool, sqlcv1.GetWorkerActionsByWorkerIdParams{
		Workerids: workerIds,
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

func (w *workerRepository) GetWorkerWorkflowsByWorkerId(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID) ([]*sqlcv1.Workflow, error) {
	return w.queries.GetWorkerWorkflowsByWorkerId(ctx, w.pool, sqlcv1.GetWorkerWorkflowsByWorkerIdParams{
		Workerid: workerId,
		Tenantid: tenantId,
	})
}

func (w *workerRepository) ListWorkerLabels(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID) ([]*sqlcv1.ListWorkerLabelsRow, error) {
	return w.queries.ListWorkerLabels(ctx, w.pool, workerId)
}

func (w *workerRepository) ListWorkerSlotConfigs(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) (map[uuid.UUID]map[string]int32, error) {
	rows, err := w.queries.ListWorkerSlotConfigs(ctx, w.pool, sqlcv1.ListWorkerSlotConfigsParams{
		Tenantid:  tenantId,
		Workerids: workerIds,
	})

	if err != nil {
		return nil, err
	}

	res := make(map[uuid.UUID]map[string]int32)
	for _, row := range rows {
		if _, ok := res[row.WorkerID]; !ok {
			res[row.WorkerID] = make(map[string]int32)
		}
		res[row.WorkerID][row.SlotType] = row.MaxUnits
	}

	return res, nil
}

func (w *workerRepository) ListAvailableSlotsForWorkers(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID, slotType string) (map[uuid.UUID]int32, error) {
	rows, err := w.queries.ListAvailableSlotsForWorkers(ctx, w.pool, sqlcv1.ListAvailableSlotsForWorkersParams{
		Tenantid:  tenantId,
		Workerids: workerIds,
		Slottype:  slotType,
	})

	if err != nil {
		return nil, fmt.Errorf("could not list available slots for workers: %w", err)
	}

	res := make(map[uuid.UUID]int32, len(rows))
	for _, row := range rows {
		res[row.ID] = row.AvailableSlots
	}

	return res, nil
}

func (w *workerRepository) ListAvailableSlotsForWorkersAndTypes(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID, slotTypes []string) (map[uuid.UUID]map[string]int32, error) {
	rows, err := w.queries.ListAvailableSlotsForWorkersAndTypes(ctx, w.pool, sqlcv1.ListAvailableSlotsForWorkersAndTypesParams{
		Tenantid:  tenantId,
		Workerids: workerIds,
		Slottypes: slotTypes,
	})

	if err != nil {
		return nil, fmt.Errorf("could not list available slots for workers and types: %w", err)
	}

	res := make(map[uuid.UUID]map[string]int32)
	for _, row := range rows {
		if _, ok := res[row.ID]; !ok {
			res[row.ID] = make(map[string]int32)
		}
		res[row.ID][row.SlotType] = row.AvailableSlots
	}

	return res, nil
}

func (w *workerRepository) GetWorkerForEngine(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID) (*sqlcv1.GetWorkerForEngineRow, error) {
	return w.queries.GetWorkerForEngine(ctx, w.pool, sqlcv1.GetWorkerForEngineParams{
		ID:       workerId,
		Tenantid: tenantId,
	})
}

func (w *workerRepository) CreateNewWorker(ctx context.Context, tenantId uuid.UUID, opts *CreateWorkerOpts) (*sqlcv1.Worker, error) {
	preWorker, postWorker := w.m.Meter(ctx, sqlcv1.LimitResourceWORKER, tenantId, 1)

	if err := preWorker(); err != nil {
		return nil, err
	}

	slotConfig := opts.SlotConfig
	slots := int32(0)

	for _, units := range slotConfig {
		slots += units
	}

	preWorkerSlot, postWorkerSlot := w.m.Meter(ctx, sqlcv1.LimitResourceWORKERSLOT, tenantId, slots)

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
		Dispatcherid: opts.DispatcherId,
		Name:         opts.Name,
	}

	// Default to self hosted
	createParams.Type = sqlcv1.NullWorkerType{
		WorkerType: sqlcv1.WorkerTypeSELFHOSTED,
		Valid:      true,
	}

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

	worker, err := w.queries.CreateWorker(ctx, tx, createParams)

	if err != nil {
		return nil, fmt.Errorf("could not create worker: %w", err)
	}

	slotTypes := make([]string, 0)
	maxUnits := make([]int32, 0)

	for slotType, units := range slotConfig {
		slotTypes = append(slotTypes, slotType)
		maxUnits = append(maxUnits, units)
	}

	if len(slotTypes) > 0 {
		err = w.queries.CreateWorkerSlotConfigs(ctx, tx, sqlcv1.CreateWorkerSlotConfigsParams{
			Tenantid:  tenantId,
			Workerid:  worker.ID,
			Slottypes: slotTypes,
			Maxunits:  maxUnits,
		})
		if err != nil {
			return nil, fmt.Errorf("could not create worker slot config: %w", err)
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
func (w *workerRepository) UpdateWorker(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, opts *UpdateWorkerOpts) (*sqlcv1.Worker, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, err
	}

	tx, err := w.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer sqlchelpers.DeferRollback(ctx, w.l, tx.Rollback)

	updateParams := sqlcv1.UpdateWorkerParams{
		ID: workerId,
	}

	if opts.LastHeartbeatAt != nil {
		updateParams.LastHeartbeatAt = sqlchelpers.TimestampFromTime(*opts.LastHeartbeatAt)
	}

	if opts.DispatcherId != nil {
		parsed := *opts.DispatcherId
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
			Workerid:  workerId,
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

func (w *workerRepository) UpdateWorkerHeartbeat(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, lastHeartbeat time.Time) error {
	_, err := w.queries.UpdateWorkerHeartbeat(ctx, w.pool, sqlcv1.UpdateWorkerHeartbeatParams{
		ID:              workerId,
		LastHeartbeatAt: sqlchelpers.TimestampFromTime(lastHeartbeat),
	})

	if err != nil {
		return fmt.Errorf("could not update worker heartbeat: %w", err)
	}

	return nil
}

func (w *workerRepository) DeleteWorker(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID) error {
	_, err := w.queries.DeleteWorker(ctx, w.pool, workerId)

	return err
}

func (w *workerRepository) UpdateWorkerActiveStatus(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, isActive bool, timestamp time.Time) (*sqlcv1.Worker, error) {
	worker, err := w.queries.UpdateWorkerActiveStatus(ctx, w.pool, sqlcv1.UpdateWorkerActiveStatusParams{
		ID:                      workerId,
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

func (w *workerRepository) GetDispatcherIdsForWorkers(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) (map[uuid.UUID]uuid.UUID, map[uuid.UUID]struct{}, error) {
	rows, err := w.queries.ListDispatcherIdsForWorkers(ctx, w.pool, sqlcv1.ListDispatcherIdsForWorkersParams{
		Tenantid:  tenantId,
		Workerids: sqlchelpers.UniqueSet(workerIds),
	})

	if err != nil {
		return nil, nil, fmt.Errorf("could not get dispatcher ids for workers: %w", err)
	}

	workerIdToDispatcherId := make(map[uuid.UUID]uuid.UUID)
	workerIdToHasDispatcher := make(map[uuid.UUID]bool)

	for _, row := range rows {
		if row.DispatcherId == nil || (row.DispatcherId != nil && *row.DispatcherId == uuid.Nil) {
			continue
		}

		dispatcherId := *row.DispatcherId
		workerId := row.WorkerId

		workerIdToDispatcherId[workerId] = dispatcherId
		workerIdToHasDispatcher[workerId] = true
	}

	workerIdsWithoutDispatchers := make(map[uuid.UUID]struct{})

	for workerId, hasDispatcher := range workerIdToHasDispatcher {
		if !hasDispatcher {
			workerIdsWithoutDispatchers[workerId] = struct{}{}
		}
	}

	return workerIdToDispatcherId, workerIdsWithoutDispatchers, nil
}
