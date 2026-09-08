package repository

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.opentelemetry.io/otel/attribute"

	"github.com/hatchet-dev/hatchet/internal/listutils"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
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

	// (optional) The operator this worker backs. Set for workers created by operator
	// connections (for example an OperatorService Listen stream), nil for SDK workers.
	OperatorId *uuid.UUID `validate:"omitempty"`
}

type UpdateWorkerOpts struct {
	// The id of the dispatcher
	DispatcherId *uuid.UUID `validate:"omitempty"`

	// When the last worker heartbeat was
	LastHeartbeatAt *time.Time

	// A list of actions this worker can run
	Actions []string `validate:"dive,actionId"`

	// If the worker is paused
	IsPaused *bool
}

type ListWorkersOpts struct {
	Action *string `validate:"omitempty,actionId"`

	LastHeartbeatAfter *time.Time

	Assignable *bool

	Limit *int

	Offset *int

	Statuses []string

	// LabelKeys and LabelValues are positionally paired label filters. A worker
	// must have a label matching every key/value pair to be included.
	LabelKeys   []string
	LabelValues []string

	// IncludeOperators includes engine-managed operator workers, which are hidden by default.
	IncludeOperators *bool
}

type UpsertWorkerLabelOpts struct {
	Key      string
	IntValue *int32
	StrValue *string
}

type DurableTaskDispatcherLookup struct {
	DispatcherId *uuid.UUID
	IsEvicted    bool
}

type WorkerRepository interface {
	ListWorkers(ctx context.Context, tenantId uuid.UUID, opts *ListWorkersOpts) ([]*sqlcv1.ListWorkersRow, int64, error)
	GetWorkerById(ctx context.Context, workerId uuid.UUID) (*sqlcv1.GetWorkerByIdRow, error)
	ListTotalActiveSlotsPerTenant(ctx context.Context) (map[uuid.UUID]int64, error)
	ListActiveSlotsPerTenantAndSlotType(ctx context.Context) (map[TenantIdSlotTypeTuple]int64, error)
	CountActiveWorkersPerTenant(ctx context.Context) (map[uuid.UUID]int64, error)
	ListActiveSDKsPerTenant(ctx context.Context) (map[TenantIdSDKTuple]int64, error)

	// GetWorkerActionsForWorkers returns actions keyed by worker action hash.
	GetWorkerActionsForWorkers(ctx context.Context, tenantId uuid.UUID, workers []sqlcv1.Worker) (map[string][]string, error)

	// GetWorkerWorkflowsByWorkerId returns a list of workflows for a worker
	GetWorkerWorkflowsByWorkerId(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID) ([]*sqlcv1.Workflow, error)

	// ListWorkerLabels returns a list of labels config for a worker
	ListWorkerLabels(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) (map[uuid.UUID][]*sqlcv1.ListWorkerLabelsRow, error)

	// ListWorkerSlotConfigs returns slot config for workers.
	ListWorkerSlotConfigs(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) (map[uuid.UUID]map[string]int32, error)

	// ListAvailableSlotsForWorkers returns available slot units by worker for a slot type.
	ListAvailableSlotsForWorkers(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID, slotType string) (map[uuid.UUID]int32, error)

	// ListAvailableSlotsForWorkersAndTypes returns available slot units by worker for a set of slot types.
	ListAvailableSlotsForWorkersAndTypes(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID, slotTypes []string) (map[uuid.UUID]map[string]int32, error)

	// CreateNewWorker creates a new worker for a given tenant.
	CreateNewWorker(ctx context.Context, tenantId uuid.UUID, opts *CreateWorkerOpts) (*sqlcv1.Worker, error)

	// AddWorkerActions links actionIds to the worker and recomputes its action hash from the
	// resulting set. Actions the worker already has are skipped. It returns the number of
	// actions actually linked. The worker must belong to tenantId; otherwise nothing is
	// mutated and an error wrapping pgx.ErrNoRows is returned.
	AddWorkerActions(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, actionIds []string) (added int, err error)

	// RemoveWorkerActions unlinks actionIds from the worker and recomputes its action hash
	// from the resulting set. Actions the worker does not have are skipped. It returns the
	// number of actions actually unlinked. The tenant check is the same as AddWorkerActions.
	RemoveWorkerActions(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, actionIds []string) (removed int, err error)

	// UpdateWorker updates a worker for a given tenant.
	UpdateWorker(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, opts *UpdateWorkerOpts) (*sqlcv1.Worker, error)

	// UpdateWorker updates a worker in the
	// It will only update the worker if there is no lock on the worker, else it will skip.
	UpdateWorkerHeartbeat(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, lastHeartbeatAt time.Time) error

	// UpdateWorkerHeartbeats updates the heartbeat timestamp for many workers in a single statement.
	UpdateWorkerHeartbeats(ctx context.Context, workerIds []uuid.UUID, lastHeartbeatAt time.Time) error

	// PauseWorkers pauses many workers in a single statement.
	PauseWorkers(ctx context.Context, workerIds []uuid.UUID) error

	// DeleteWorker removes the worker from the database
	DeleteWorker(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID) error

	GetWorkerForEngine(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID) (*sqlcv1.GetWorkerForEngineRow, error)

	// ActivateWorkerListener marks the worker active on behalf of the listener session
	// identified by sessionId and records that id on the worker, so only this session can
	// later deactivate it.
	ActivateWorkerListener(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, sessionId uuid.UUID) (*sqlcv1.Worker, error)

	// DeactivateWorkerListener marks the worker inactive if sessionId is still the session
	// recorded by ActivateWorkerListener. It returns pgx.ErrNoRows when a newer session has
	// superseded this one, in which case the worker is left untouched.
	DeactivateWorkerListener(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, sessionId uuid.UUID) (*sqlcv1.Worker, error)

	UpsertWorkerLabels(ctx context.Context, workerId uuid.UUID, opts []UpsertWorkerLabelOpts) ([]*sqlcv1.WorkerLabel, error)

	CleanupOldWorkers(ctx context.Context, tenantId uuid.UUID, lastHeartbeatBefore time.Time) (bool, error)

	GetDispatcherIdsForWorkers(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) (map[uuid.UUID]uuid.UUID, map[uuid.UUID]struct{}, error)

	UpdateWorkerDurableTaskDispatcherId(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, dispatcherId uuid.UUID) error

	GetDurableDispatcherIdsForTasks(ctx context.Context, tenantId uuid.UUID, idInsertedAtTuples []IdInsertedAt) (map[IdInsertedAt]DurableTaskDispatcherLookup, error)
}

type workerRepository struct {
	*sharedRepository
}

func newWorkerRepository(shared *sharedRepository) WorkerRepository {
	return &workerRepository{
		sharedRepository: shared,
	}
}

func (w *workerRepository) ListWorkers(ctx context.Context, tenantId uuid.UUID, opts *ListWorkersOpts) ([]*sqlcv1.ListWorkersRow, int64, error) {
	if err := w.v.Validate(opts); err != nil {
		return nil, 0, err
	}

	queryParams := sqlcv1.ListWorkersParams{
		Tenantid: tenantId,
	}

	countParams := sqlcv1.CountWorkersParams{
		Tenantid: tenantId,
	}

	if opts.Action != nil {
		queryParams.ActionId = sqlchelpers.TextFromStr(*opts.Action)
		countParams.ActionId = queryParams.ActionId
	}

	if opts.LastHeartbeatAfter != nil {
		queryParams.LastHeartbeatAfter = sqlchelpers.TimestampFromTime(opts.LastHeartbeatAfter.UTC())
		countParams.LastHeartbeatAfter = queryParams.LastHeartbeatAfter
	}

	if opts.Assignable != nil {
		queryParams.Assignable = pgtype.Bool{
			Bool:  *opts.Assignable,
			Valid: true,
		}
		countParams.Assignable = queryParams.Assignable
	}

	if opts.Statuses != nil {
		queryParams.Statuses = opts.Statuses
		countParams.Statuses = opts.Statuses
	}

	if opts.IncludeOperators != nil {
		queryParams.IncludeOperators = pgtype.Bool{
			Bool:  *opts.IncludeOperators,
			Valid: true,
		}
		countParams.IncludeOperators = queryParams.IncludeOperators
	}

	if len(opts.LabelKeys) > 0 || len(opts.LabelValues) > 0 {
		if len(opts.LabelKeys) != len(opts.LabelValues) {
			return nil, 0, fmt.Errorf("label filter keys/values must be paired: got %d keys and %d values", len(opts.LabelKeys), len(opts.LabelValues))
		}

		queryParams.LabelKeys = opts.LabelKeys
		queryParams.LabelValues = opts.LabelValues
		countParams.LabelKeys = opts.LabelKeys
		countParams.LabelValues = opts.LabelValues
	}

	if opts.Limit != nil {
		queryParams.Limit = pgtype.Int4{
			Int32: int32(*opts.Limit), // nolint: gosec
			Valid: true,
		}
	}

	if opts.Offset != nil {
		queryParams.Offset = pgtype.Int4{
			Int32: int32(*opts.Offset), // nolint: gosec
			Valid: true,
		}
	}

	count, err := w.queries.CountWorkers(ctx, w.pool, countParams)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, fmt.Errorf("could not count workers: %w", err)
	}

	workers, err := w.queries.ListWorkers(ctx, w.pool, queryParams)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			workers = make([]*sqlcv1.ListWorkersRow, 0)
		} else {
			return nil, 0, fmt.Errorf("could not list workers: %w", err)
		}
	}

	return workers, count, nil
}

func (w *workerRepository) GetWorkerById(ctx context.Context, workerId uuid.UUID) (*sqlcv1.GetWorkerByIdRow, error) {
	return w.queries.GetWorkerById(ctx, w.pool, workerId)
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

func (w *workerRepository) GetWorkerActionsForWorkers(ctx context.Context, tenantId uuid.UUID, workers []sqlcv1.Worker) (map[string][]string, error) {
	ctx, span := telemetry.NewSpan(ctx, "WorkerRepository.GetWorkerActionsForWorkers")
	defer span.End()

	actionHashSet := make(map[string]struct{})
	workerIds := make([]uuid.UUID, 0)
	actionHashToWorkerIds := make(map[string][]uuid.UUID)

	for _, worker := range workers {
		if len(worker.ActionHash) == 0 {
			// if the worker has no action hash, we have no choice but to look
			// it up by its id
			workerIds = append(workerIds, worker.ID)
			continue
		}

		actionHashToWorkerIds[string(worker.ActionHash)] = append(actionHashToWorkerIds[string(worker.ActionHash)], worker.ID)

		if _, ok := actionHashSet[string(worker.ActionHash)]; !ok {
			actionHashSet[string(worker.ActionHash)] = struct{}{}
		}
	}

	actionHashes := make([][]byte, 0, len(actionHashSet))

	for actionHash := range actionHashSet {
		actionHashes = append(actionHashes, []byte(actionHash))
	}

	recordsFromActionHashes, err := w.queries.GetWorkerActionsByWorkerActionHash(ctx, w.pool, sqlcv1.GetWorkerActionsByWorkerActionHashParams{
		Actionhashes: actionHashes,
		Tenantid:     tenantId,
	})

	if err != nil {
		return nil, err
	}

	workerIdToActionIds := make(map[string][]string)

	if len(recordsFromActionHashes) > 0 {
		for _, record := range recordsFromActionHashes {
			actionWorkerIds, ok := actionHashToWorkerIds[string(record.ActionHash)]

			if !ok {
				continue
			}

			for _, workerIdUuid := range actionWorkerIds {
				workerId := workerIdUuid.String()
				if _, ok := workerIdToActionIds[workerId]; !ok {
					workerIdToActionIds[workerId] = make([]string, 0)
				}

				workerIdToActionIds[workerId] = append(workerIdToActionIds[workerId], record.ActionID)
			}
		}
	}

	span.SetAttributes(
		attribute.Int("num_worker_ids", len(workerIds)),
		attribute.Int("num_worker_action_hashes", len(actionHashes)),
	)

	if len(workerIds) > 0 {
		recordsFromWorkerIds, err := w.queries.GetWorkerActionsByWorkerId(ctx, w.pool, sqlcv1.GetWorkerActionsByWorkerIdParams{
			Workerids: workerIds,
			Tenantid:  tenantId,
		})

		if err != nil {
			return nil, err
		}
		for _, record := range recordsFromWorkerIds {
			workerId := record.WorkerId.String()

			if _, ok := workerIdToActionIds[workerId]; !ok {
				workerIdToActionIds[workerId] = make([]string, 0)
			}

			workerIdToActionIds[workerId] = append(workerIdToActionIds[workerId], record.Actionid)
		}
	}

	return workerIdToActionIds, nil
}

func (w *workerRepository) GetWorkerWorkflowsByWorkerId(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID) ([]*sqlcv1.Workflow, error) {
	return w.queries.GetWorkerWorkflowsByWorkerId(ctx, w.pool, sqlcv1.GetWorkerWorkflowsByWorkerIdParams{
		Workerid: workerId,
		Tenantid: tenantId,
	})
}

func (w *workerRepository) ListWorkerLabels(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) (map[uuid.UUID][]*sqlcv1.ListWorkerLabelsRow, error) {
	labels, err := w.queries.ListWorkerLabels(ctx, w.pool, workerIds)

	if err != nil {
		return nil, fmt.Errorf("could not list worker labels: %w", err)
	}

	workerIdToLabels := make(map[uuid.UUID][]*sqlcv1.ListWorkerLabelsRow)

	for _, label := range labels {
		workerIdToLabels[label.WorkerId] = append(workerIdToLabels[label.WorkerId], label)
	}

	return workerIdToLabels, nil
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

// hashActions is the canonical digest of an action set: sha256 over the ids sorted by byte
// order, each followed by ";" so ["ab", "c"] and ["a", "bc"] differ, after the same
// lower-casing and deduplication the "Action" table applies. It is a function of the final set
// alone, so a worker created with an initial set and a worker built by deltas hash equal for
// the same set, and it is not a combination of per-action digests that could be solved for a
// chosen value. ComputeWorkerActionHash in workers.sql computes the same digest from the
// linked rows; the two must stay in step because GetWorkerActionsByWorkerActionHash treats
// equal hashes as equal sets.
func hashActions(actions []string) []byte {
	ids := dedupeActionIds(actions)

	sort.Strings(ids)

	h := sha256.New()

	for _, action := range ids {
		h.Write([]byte(action))
		h.Write([]byte(";"))
	}

	return h.Sum(nil)
}

// workerSDKFromContract maps the SDK reported by a worker at registration to the "Worker"."language"
// column value.
func workerSDKFromContract(sdk contracts.SDKS) (sqlcv1.NullWorkerSDKS, error) {
	var language sqlcv1.WorkerSDKS

	switch sdk {
	case contracts.SDKS_GO:
		language = sqlcv1.WorkerSDKSGO
	case contracts.SDKS_PYTHON:
		language = sqlcv1.WorkerSDKSPYTHON
	case contracts.SDKS_TYPESCRIPT:
		language = sqlcv1.WorkerSDKSTYPESCRIPT
	case contracts.SDKS_RUBY:
		language = sqlcv1.WorkerSDKSRUBY
	default:
		return sqlcv1.NullWorkerSDKS{}, fmt.Errorf("invalid sdk: %s", sdk)
	}

	return sqlcv1.NullWorkerSDKS{
		WorkerSDKS: language,
		Valid:      true,
	}, nil
}

func (w *workerRepository) CreateNewWorker(ctx context.Context, tenantId uuid.UUID, opts *CreateWorkerOpts) (*sqlcv1.Worker, error) {
	slotConfig := opts.SlotConfig
	slots := int32(0)

	for _, units := range slotConfig {
		slots += units
	}

	// Operator workers are excluded from the WORKER and WORKER_SLOT limit counts by the
	// "operatorId" IS NULL filters in workers.sql and tenant_limits.sql, so metering them here
	// would charge for workers the limit queries never see.
	postWorker := func() {}
	postWorkerSlot := func() {}

	if opts.OperatorId == nil {
		var preWorker, preWorkerSlot func() error

		preWorker, postWorker = w.m.Meter(ctx, nil, sqlcv1.LimitResourceWORKER, tenantId, 1)

		if err := preWorker(); err != nil {
			return nil, err
		}

		preWorkerSlot, postWorkerSlot = w.m.Meter(ctx, nil, sqlcv1.LimitResourceWORKERSLOT, tenantId, slots)

		if err := preWorkerSlot(); err != nil {
			return nil, err
		}
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
		Actionhash:   hashActions(opts.Actions),
		OperatorId:   opts.OperatorId,
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
			language, err := workerSDKFromContract(*opts.RuntimeInfo.Language)

			if err != nil {
				return nil, err
			}

			createParams.Language = language
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

func (w *workerRepository) AddWorkerActions(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, actionIds []string) (int, error) {
	actionIds = dedupeActionIds(actionIds)

	if len(actionIds) == 0 {
		return 0, nil
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, w.pool, w.l)

	if err != nil {
		return 0, err
	}

	defer rollback()

	// the row lock is held until commit, so concurrent deltas for the same worker apply one
	// after the other and each recomputes the hash from the links it leaves behind
	if err := w.lockWorkerActions(ctx, tx, tenantId, workerId); err != nil {
		return 0, err
	}

	actionUUIDs, err := w.resolveActionIds(ctx, tx, tenantId, actionIds)

	if err != nil {
		return 0, err
	}

	linked, err := w.queries.LinkActionsToWorkerReturning(ctx, tx, sqlcv1.LinkActionsToWorkerReturningParams{
		Workerid:  workerId,
		Tenantid:  tenantId,
		Actionids: actionUUIDs,
	})

	if err != nil {
		return 0, fmt.Errorf("could not link actions to worker: %w", err)
	}

	if len(linked) == 0 {
		return 0, nil
	}

	if err := w.refreshWorkerActionHash(ctx, tx, workerId); err != nil {
		return 0, err
	}

	if err := commit(ctx); err != nil {
		return 0, err
	}

	return len(linked), nil
}

func (w *workerRepository) RemoveWorkerActions(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, actionIds []string) (int, error) {
	actionIds = dedupeActionIds(actionIds)

	if len(actionIds) == 0 {
		return 0, nil
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, w.pool, w.l)

	if err != nil {
		return 0, err
	}

	defer rollback()

	if err := w.lockWorkerActions(ctx, tx, tenantId, workerId); err != nil {
		return 0, err
	}

	actions, err := w.queries.ListActionsByActionIds(ctx, tx, sqlcv1.ListActionsByActionIdsParams{
		Tenantid:  tenantId,
		Actionids: actionIds,
	})

	if err != nil {
		return 0, fmt.Errorf("could not list actions: %w", err)
	}

	if len(actions) == 0 {
		return 0, nil
	}

	actionUUIDs := make([]uuid.UUID, 0, len(actions))

	for _, action := range actions {
		actionUUIDs = append(actionUUIDs, action.ID)
	}

	unlinked, err := w.queries.UnlinkActionsFromWorkerReturning(ctx, tx, sqlcv1.UnlinkActionsFromWorkerReturningParams{
		Workerid:  workerId,
		Tenantid:  tenantId,
		Actionids: actionUUIDs,
	})

	if err != nil {
		return 0, fmt.Errorf("could not unlink actions from worker: %w", err)
	}

	if len(unlinked) == 0 {
		return 0, nil
	}

	if err := w.refreshWorkerActionHash(ctx, tx, workerId); err != nil {
		return 0, err
	}

	if err := commit(ctx); err != nil {
		return 0, err
	}

	return len(unlinked), nil
}

// lockWorkerActions takes the worker's row lock for the rest of tx. A worker that does not
// belong to tenantId is reported as an error wrapping pgx.ErrNoRows before anything is
// mutated.
func (w *workerRepository) lockWorkerActions(ctx context.Context, tx pgx.Tx, tenantId, workerId uuid.UUID) error {
	if _, err := w.queries.LockWorkerActionHash(ctx, tx, sqlcv1.LockWorkerActionHashParams{
		Workerid: workerId,
		Tenantid: tenantId,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("worker %s does not belong to tenant %s: %w", workerId, tenantId, err)
		}

		return fmt.Errorf("could not lock worker %s: %w", workerId, err)
	}

	return nil
}

// resolveActionIds returns the "Action" row ids for actionIds, creating the rows that are
// missing. Existing rows are read, not upserted, so a delta that repeats actions the tenant
// already has takes no lock on them and writes nothing; the missing ones are inserted in
// sorted order so concurrent transactions creating overlapping sets lock in one order.
// actionIds must already be deduplicated and lower-cased.
func (w *workerRepository) resolveActionIds(ctx context.Context, tx pgx.Tx, tenantId uuid.UUID, actionIds []string) ([]uuid.UUID, error) {
	existing, err := w.queries.ListActionsByActionIds(ctx, tx, sqlcv1.ListActionsByActionIdsParams{
		Tenantid:  tenantId,
		Actionids: actionIds,
	})

	if err != nil {
		return nil, fmt.Errorf("could not list actions: %w", err)
	}

	uuidByActionId := make(map[string]uuid.UUID, len(actionIds))

	for _, action := range existing {
		uuidByActionId[action.ActionId] = action.ID
	}

	missing := make([]string, 0, len(actionIds)-len(existing))

	for _, actionId := range actionIds {
		if _, ok := uuidByActionId[actionId]; !ok {
			missing = append(missing, actionId)
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)

		inserted, err := w.queries.InsertMissingActions(ctx, tx, sqlcv1.InsertMissingActionsParams{
			Tenantid: tenantId,
			Actions:  missing,
		})

		if err != nil {
			return nil, fmt.Errorf("could not insert actions: %w", err)
		}

		for _, action := range inserted {
			uuidByActionId[action.ActionId] = action.ID
		}

		// an action a concurrent transaction created after the read above is skipped by the
		// insert and resolved here, once that transaction has committed
		if len(inserted) < len(missing) {
			raced := make([]string, 0, len(missing)-len(inserted))

			for _, actionId := range missing {
				if _, ok := uuidByActionId[actionId]; !ok {
					raced = append(raced, actionId)
				}
			}

			concurrent, err := w.queries.ListActionsByActionIds(ctx, tx, sqlcv1.ListActionsByActionIdsParams{
				Tenantid:  tenantId,
				Actionids: raced,
			})

			if err != nil {
				return nil, fmt.Errorf("could not list actions: %w", err)
			}

			for _, action := range concurrent {
				uuidByActionId[action.ActionId] = action.ID
			}
		}
	}

	actionUUIDs := make([]uuid.UUID, 0, len(actionIds))

	for _, actionId := range actionIds {
		id, ok := uuidByActionId[actionId]

		if !ok {
			return nil, fmt.Errorf("could not resolve action %s for tenant %s", actionId, tenantId)
		}

		actionUUIDs = append(actionUUIDs, id)
	}

	sort.Slice(actionUUIDs, func(i, j int) bool {
		return bytes.Compare(actionUUIDs[i][:], actionUUIDs[j][:]) < 0
	})

	return actionUUIDs, nil
}

// refreshWorkerActionHash recomputes the worker's action hash from its linked rows inside tx.
// The caller holds the worker's row lock, so the digest written here is the digest of the
// links this transaction leaves behind.
func (w *workerRepository) refreshWorkerActionHash(ctx context.Context, tx pgx.Tx, workerId uuid.UUID) error {
	hash, err := w.queries.ComputeWorkerActionHash(ctx, tx, workerId)

	if err != nil {
		return fmt.Errorf("could not compute worker actions hash: %w", err)
	}

	if err := w.queries.UpdateWorkerActionsHash(ctx, tx, sqlcv1.UpdateWorkerActionsHashParams{
		Workerid:   workerId,
		Actionhash: hash,
	}); err != nil {
		return fmt.Errorf("could not update worker actions hash: %w", err)
	}

	return nil
}

// dedupeActionIds lower-cases action ids the way the "Action" table stores them and drops
// duplicates and empty entries, so a bulk upsert never touches the same row twice.
func dedupeActionIds(actionIds []string) []string {
	seen := make(map[string]struct{}, len(actionIds))
	out := make([]string, 0, len(actionIds))

	for _, actionId := range actionIds {
		actionId = strings.ToLower(actionId)

		if actionId == "" {
			continue
		}

		if _, ok := seen[actionId]; ok {
			continue
		}

		seen[actionId] = struct{}{}
		out = append(out, actionId)
	}

	return out
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

		// links are only ever added here, so the stored hash is recomputed from the rows
		// rather than from opts.Actions alone
		if err := w.refreshWorkerActionHash(ctx, tx, workerId); err != nil {
			return nil, err
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

func (w *workerRepository) UpdateWorkerHeartbeats(ctx context.Context, workerIds []uuid.UUID, lastHeartbeat time.Time) error {
	if len(workerIds) == 0 {
		return nil
	}

	err := w.queries.UpdateWorkerHeartbeats(ctx, w.pool, sqlcv1.UpdateWorkerHeartbeatsParams{
		Ids:             workerIds,
		Lastheartbeatat: sqlchelpers.TimestampFromTime(lastHeartbeat),
	})

	if err != nil {
		return fmt.Errorf("could not update worker heartbeats: %w", err)
	}

	return nil
}

func (w *workerRepository) PauseWorkers(ctx context.Context, workerIds []uuid.UUID) error {
	if len(workerIds) == 0 {
		return nil
	}

	err := w.queries.PauseWorkers(ctx, w.pool, workerIds)

	if err != nil {
		return fmt.Errorf("could not pause workers: %w", err)
	}

	return nil
}

func (w *workerRepository) DeleteWorker(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID) error {
	_, err := w.queries.DeleteWorker(ctx, w.pool, workerId)

	return err
}

func (w *workerRepository) ActivateWorkerListener(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, sessionId uuid.UUID) (*sqlcv1.Worker, error) {
	worker, err := w.queries.ActivateWorkerListener(ctx, w.pool, sqlcv1.ActivateWorkerListenerParams{
		ID:        workerId,
		Tenantid:  tenantId,
		Sessionid: sessionId,
	})

	if err != nil {
		return nil, fmt.Errorf("could not activate worker listener: %w", err)
	}

	return worker, nil
}

func (w *workerRepository) DeactivateWorkerListener(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, sessionId uuid.UUID) (*sqlcv1.Worker, error) {
	worker, err := w.queries.DeactivateWorkerListener(ctx, w.pool, sqlcv1.DeactivateWorkerListenerParams{
		ID:        workerId,
		Tenantid:  tenantId,
		Sessionid: sessionId,
	})

	if err != nil {
		return nil, fmt.Errorf("could not deactivate worker listener: %w", err)
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

func (w *workerRepository) CleanupOldWorkers(ctx context.Context, tenantId uuid.UUID, lastHeartbeatBefore time.Time) (bool, error) {
	const timeout = 1000 * 60 * 3 // 3 minutes
	const batchSize int32 = 10000

	tx, commit, rollback, err := sqlchelpers.PrepareTxWithStatementTimeout(ctx, w.pool, w.l, timeout)
	if err != nil {
		return false, fmt.Errorf("error beginning transaction: %w", err)
	}
	defer rollback()

	result, err := w.queries.CleanupOldWorkers(ctx, tx, sqlcv1.CleanupOldWorkersParams{
		Tenantid:            tenantId,
		Lastheartbeatbefore: sqlchelpers.TimestampFromTime(lastHeartbeatBefore),
		Batchsize:           batchSize,
	})
	if err != nil {
		return false, fmt.Errorf("error cleaning up old workers: %w", err)
	}

	if err := commit(ctx); err != nil {
		return false, fmt.Errorf("error committing transaction: %w", err)
	}

	return result.RowsAffected() == int64(batchSize), nil
}

func (w *workerRepository) GetDispatcherIdsForWorkers(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) (map[uuid.UUID]uuid.UUID, map[uuid.UUID]struct{}, error) {
	rows, err := w.queries.ListDispatcherIdsForWorkers(ctx, w.pool, sqlcv1.ListDispatcherIdsForWorkersParams{
		Tenantid:  tenantId,
		Workerids: listutils.Uniq(workerIds),
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

func (w *workerRepository) UpdateWorkerDurableTaskDispatcherId(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, dispatcherId uuid.UUID) error {
	return w.queries.UpdateWorkerDurableTaskDispatcherId(ctx, w.pool, sqlcv1.UpdateWorkerDurableTaskDispatcherIdParams{
		Workerid:     workerId,
		Dispatcherid: dispatcherId,
		Tenantid:     tenantId,
	})
}

func (w *workerRepository) GetDurableDispatcherIdsForTasks(ctx context.Context, tenantId uuid.UUID, idInsertedAtTuples []IdInsertedAt) (map[IdInsertedAt]DurableTaskDispatcherLookup, error) {
	taskIds := make([]int64, len(idInsertedAtTuples))
	taskInsertedAts := make([]pgtype.Timestamptz, len(idInsertedAtTuples))

	for i, tuple := range idInsertedAtTuples {
		taskIds[i] = tuple.ID
		taskInsertedAts[i] = sqlchelpers.TimestamptzFromUnixMicros(tuple.InsertedAtUnixMicros)
	}

	rows, err := w.queries.ListDurableTaskDispatcherIdsForTasks(ctx, w.pool, sqlcv1.ListDurableTaskDispatcherIdsForTasksParams{
		Tenantid:        tenantId,
		Taskids:         taskIds,
		Taskinsertedats: taskInsertedAts,
	})

	if err != nil {
		return nil, fmt.Errorf("could not get durable dispatcher ids for tasks: %w", err)
	}

	taskIdToDispatcherInfo := make(map[IdInsertedAt]DurableTaskDispatcherLookup)

	for _, row := range rows {
		taskIdToDispatcherInfo[IdInsertedAt{
			ID:                   row.TaskID,
			InsertedAtUnixMicros: row.TaskInsertedAt.Time.UnixMicro(),
		}] = DurableTaskDispatcherLookup{
			DispatcherId: row.DurableTaskDispatcherId,
			IsEvicted:    row.EvictedAt.Valid,
		}
	}

	return taskIdToDispatcherInfo, nil
}
