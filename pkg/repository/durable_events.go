package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type CreateEventLogFileOpts struct {
	TenantId                      uuid.UUID
	DurableTaskId                 int64
	DurableTaskInsertedAt         pgtype.Timestamptz
	LatestInsertedAt              pgtype.Timestamptz
	LatestNodeId                  int64
	LatestBranchId                int64
	LatestBranchFirstParentNodeId int64
}

type CreateEventLogEntryOpts struct {
	TenantId               uuid.UUID
	ExternalId             uuid.UUID
	DurableTaskId          int64
	DurableTaskInsertedAt  pgtype.Timestamptz
	InsertedAt             pgtype.Timestamptz
	Kind                   sqlcv1.V1DurableEventLogEntryKind
	NodeId                 int64
	ParentNodeId           int64
	BranchId               int64
	Data                   []byte
	TriggeredRunExternalId *uuid.UUID
}

type CreateEventLogCallbackOpts struct {
	TenantId              uuid.UUID
	DurableTaskId         int64
	DurableTaskInsertedAt pgtype.Timestamptz
	InsertedAt            pgtype.Timestamptz
	ExternalId            uuid.UUID
	Kind                  sqlcv1.V1DurableEventLogCallbackKind
	NodeId                int64
	IsSatisfied           bool
	DispatcherId          uuid.UUID
}

type EventLogCallbackWithPayload struct {
	Callback *sqlcv1.V1DurableEventLogCallback
	Result   []byte
}

type EventLogEntryWithPayload struct {
	Entry   *sqlcv1.V1DurableEventLogEntry
	Payload []byte
}

type TaskExternalIdNodeId struct {
	TaskExternalId string
	NodeId         int64
}

type SatisfiedCallbackWithPayload struct {
	TaskExternalId uuid.UUID
	NodeID         int64
	Result         []byte
}

type IngestDurableTaskEventOpts struct {
	TenantId          uuid.UUID
	Task              *sqlcv1.FlattenExternalIdsRow
	Kind              sqlcv1.V1DurableEventLogEntryKind
	Payload           []byte
	DispatcherId      uuid.UUID
	WaitForConditions *v1.DurableEventListenerConditions
}

type IngestDurableTaskEventResult struct {
	Callback      *EventLogCallbackWithPayload
	EventLogEntry *EventLogEntryWithPayload
	EventLogFile  *sqlcv1.V1DurableEventLogFile
}

type DurableEventsRepository interface {
	UpdateEventLogCallbackSatisfied(ctx context.Context, tenantId uuid.UUID, nodeId, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz, isSatisfied bool, result []byte) (*sqlcv1.V1DurableEventLogCallback, error)

	IngestDurableTaskEvent(ctx context.Context, opts IngestDurableTaskEventOpts) (*IngestDurableTaskEventResult, error)

	GetSatisfiedCallbacks(ctx context.Context, tenantId uuid.UUID, callbacks []TaskExternalIdNodeId) ([]*SatisfiedCallbackWithPayload, error)
}

type durableEventsRepository struct {
	*sharedRepository
}

func newDurableEventsRepository(shared *sharedRepository) DurableEventsRepository {
	return &durableEventsRepository{
		sharedRepository: shared,
	}
}

func (r *durableEventsRepository) getOrCreateEventLogEntry(
	ctx context.Context,
	tx sqlcv1.DBTX,
	tenantId uuid.UUID,
	params sqlcv1.GetOrCreateDurableEventLogEntryParams,
	payload []byte,
) (*EventLogEntryWithPayload, error) {
	row, err := r.queries.GetOrCreateDurableEventLogEntry(ctx, tx, params)
	if err != nil {
		return nil, err
	}

	entry := &sqlcv1.V1DurableEventLogEntry{
		TenantID:              row.TenantID,
		ExternalID:            row.ExternalID,
		InsertedAt:            row.InsertedAt,
		ID:                    row.ID,
		DurableTaskID:         row.DurableTaskID,
		DurableTaskInsertedAt: row.DurableTaskInsertedAt,
		Kind:                  row.Kind,
		NodeID:                row.NodeID,
		ParentNodeID:          row.ParentNodeID,
		BranchID:              row.BranchID,
		DataHash:              row.DataHash,
		DataHashAlg:           row.DataHashAlg,
	}

	if row.AlreadyExists {
		existingPayload, err := r.payloadStore.RetrieveSingle(ctx, tx, RetrievePayloadOpts{
			Id:         entry.ID,
			InsertedAt: entry.InsertedAt,
			Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYDATA,
			TenantId:   tenantId,
		})

		if err != nil {
			return nil, err
		}

		return &EventLogEntryWithPayload{Entry: entry, Payload: existingPayload}, nil
	}

	if len(payload) > 0 {
		err = r.payloadStore.Store(ctx, tx, StorePayloadOpts{
			Id:         entry.ID,
			InsertedAt: entry.InsertedAt,
			ExternalId: entry.ExternalID,
			Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYDATA,
			Payload:    payload,
			TenantId:   tenantId,
		})
		if err != nil {
			return nil, err
		}
	}

	return &EventLogEntryWithPayload{Entry: entry, Payload: payload}, nil
}

func (r *durableEventsRepository) getOrCreateEventLogCallback(
	ctx context.Context,
	tx sqlcv1.DBTX,
	tenantId uuid.UUID,
	params sqlcv1.GetOrCreateDurableEventLogCallbackParams,
) (*EventLogCallbackWithPayload, error) {
	row, err := r.queries.GetOrCreateDurableEventLogCallback(ctx, tx, params)

	if err != nil {
		return nil, err
	}

	callback := &sqlcv1.V1DurableEventLogCallback{
		TenantID:              row.TenantID,
		DurableTaskID:         row.DurableTaskID,
		DurableTaskInsertedAt: row.DurableTaskInsertedAt,
		InsertedAt:            row.InsertedAt,
		ID:                    row.ID,
		ExternalID:            row.ExternalID,
		Kind:                  row.Kind,
		NodeID:                row.NodeID,
		IsSatisfied:           row.IsSatisfied,
		DispatcherID:          row.DispatcherID,
	}

	var result []byte
	if row.AlreadyExists {
		result, err = r.payloadStore.RetrieveSingle(ctx, tx, RetrievePayloadOpts{
			Id:         callback.ID,
			InsertedAt: callback.InsertedAt,
			Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGCALLBACKRESULTDATA,
			TenantId:   tenantId,
		})

		if err != nil {
			result = nil
		}
	}

	return &EventLogCallbackWithPayload{Callback: callback, Result: result}, nil
}

func (r *durableEventsRepository) ListEventLogCallbacks(ctx context.Context, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz) ([]*sqlcv1.V1DurableEventLogCallback, error) {
	return r.queries.ListDurableEventLogCallbacks(ctx, r.pool, sqlcv1.ListDurableEventLogCallbacksParams{
		Durabletaskid:         durableTaskId,
		Durabletaskinsertedat: durableTaskInsertedAt,
	})
}

func (r *durableEventsRepository) UpdateEventLogCallbackSatisfied(ctx context.Context, tenantId uuid.UUID, nodeId, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz, isSatisfied bool, result []byte) (*sqlcv1.V1DurableEventLogCallback, error) {
	// note: might need to pass a tx in here instead
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)
	if err != nil {
		return nil, err
	}
	defer rollback()

	callback, err := r.queries.UpdateDurableEventLogCallbackSatisfied(ctx, tx, sqlcv1.UpdateDurableEventLogCallbackSatisfiedParams{
		Durabletaskid:         durableTaskId,
		Durabletaskinsertedat: durableTaskInsertedAt,
		Nodeid:                nodeId,
		Issatisfied:           isSatisfied,
	})

	if err != nil {
		return nil, err
	}

	if isSatisfied && len(result) > 0 {
		storePayloadOpts := StorePayloadOpts{
			Id:         callback.ID,
			InsertedAt: callback.InsertedAt,
			Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGCALLBACKRESULTDATA,
			Payload:    result,
			ExternalId: callback.ExternalID,
			TenantId:   tenantId,
		}

		err = r.payloadStore.Store(ctx, tx, storePayloadOpts)

		if err != nil {
			return nil, err
		}
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return callback, nil
}

func (r *durableEventsRepository) GetSatisfiedCallbacks(ctx context.Context, tenantId uuid.UUID, callbacks []TaskExternalIdNodeId) ([]*SatisfiedCallbackWithPayload, error) {
	if len(callbacks) == 0 {
		return nil, nil
	}

	taskExternalIds := make([]uuid.UUID, len(callbacks))
	nodeIds := make([]int64, len(callbacks))

	for i, cb := range callbacks {
		taskId, err := uuid.Parse(cb.TaskExternalId)
		if err != nil {
			return nil, fmt.Errorf("invalid task external id %s: %w", cb.TaskExternalId, err)
		}
		taskExternalIds[i] = taskId
		nodeIds[i] = cb.NodeId
	}

	rows, err := r.queries.GetSatisfiedCallbacks(ctx, r.pool, sqlcv1.GetSatisfiedCallbacksParams{
		Tenantid:        tenantId,
		Taskexternalids: taskExternalIds,
		Nodeids:         nodeIds,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get and claim satisfied callbacks: %w", err)
	}

	result := make([]*SatisfiedCallbackWithPayload, 0, len(rows))

	for _, row := range rows {
		payload, err := r.payloadStore.RetrieveSingle(ctx, r.pool, RetrievePayloadOpts{
			Id:         row.ID,
			InsertedAt: row.InsertedAt,
			Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGCALLBACKRESULTDATA,
			TenantId:   tenantId,
		})
		if err != nil {
			r.l.Warn().Err(err).Msgf("failed to retrieve payload for callback %d", row.NodeID)
			payload = nil
		}

		result = append(result, &SatisfiedCallbackWithPayload{
			TaskExternalId: row.TaskExternalID,
			NodeID:         row.NodeID,
			Result:         payload,
		})
	}

	return result, nil
}

func getDurableTaskSignalKey(taskExternalId uuid.UUID, nodeId int64) string {
	return fmt.Sprintf("durable:%s:%d", taskExternalId.String(), nodeId)
}

func (r *durableEventsRepository) IngestDurableTaskEvent(ctx context.Context, opts IngestDurableTaskEventOpts) (*IngestDurableTaskEventResult, error) {
	task := opts.Task

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare tx: %w", err)
	}
	defer rollback()

	logFile, err := r.queries.IncrementAndGetNextNodeId(ctx, tx, sqlcv1.IncrementAndGetNextNodeIdParams{
		Tenantid:              opts.TenantId,
		Durabletaskid:         task.ID,
		Durabletaskinsertedat: task.InsertedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to increment and get next node id: %w", err)
	}

	nodeId := logFile.LatestNodeID

	now := sqlchelpers.TimestamptzFromTime(time.Now().UTC())

	parentNodeId := pgtype.Int8{
		Int64: nodeId - 1,
		Valid: nodeId > 1,
	}

	entryResult, err := r.getOrCreateEventLogEntry(ctx, tx, opts.TenantId, sqlcv1.GetOrCreateDurableEventLogEntryParams{
		Tenantid:              opts.TenantId,
		Externalid:            uuid.New(),
		Durabletaskid:         task.ID,
		Durabletaskinsertedat: task.InsertedAt,
		Insertedat:            now,
		Kind:                  opts.Kind,
		Nodeid:                nodeId,
		ParentNodeId:          parentNodeId,
		Branchid:              logFile.LatestBranchID,
		Datahash:              nil,
		Datahashalg:           "",
	}, opts.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create event log entry: %w", err)
	}

	callbackResult, err := r.getOrCreateEventLogCallback(ctx, tx, opts.TenantId, sqlcv1.GetOrCreateDurableEventLogCallbackParams{
		Tenantid:              opts.TenantId,
		Durabletaskid:         task.ID,
		Durabletaskinsertedat: task.InsertedAt,
		Insertedat:            now,
		Kind:                  sqlcv1.V1DurableEventLogCallbackKindWAITFORCOMPLETED,
		Nodeid:                nodeId,
		Issatisfied:           false,
		Externalid:            uuid.New(),
		Dispatcherid:          opts.DispatcherId,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get or create callback entry: %w", err)
	}

	switch opts.Kind {
	case sqlcv1.V1DurableEventLogEntryKindWAITFORSTARTED:
		if opts.WaitForConditions != nil {
			externalId := opts.Task.ExternalID
			signalKey := getDurableTaskSignalKey(externalId, nodeId)

			createConditionOpts := make([]CreateExternalSignalConditionOpt, 0)

			for _, condition := range opts.WaitForConditions.SleepConditions {
				orGroupId, err := uuid.Parse(condition.Base.OrGroupId)
				if err != nil {
					return nil, fmt.Errorf("or group id is not a valid uuid: %w", err)
				}

				createConditionOpts = append(createConditionOpts, CreateExternalSignalConditionOpt{
					Kind:            CreateExternalSignalConditionKindSLEEP,
					ReadableDataKey: condition.Base.ReadableDataKey,
					OrGroupId:       orGroupId,
					SleepFor:        &condition.SleepFor,
				})
			}

			for _, condition := range opts.WaitForConditions.UserEventConditions {
				orGroupId, err := uuid.Parse(condition.Base.OrGroupId)
				if err != nil {
					return nil, fmt.Errorf("or group id is not a valid uuid: %w", err)
				}

				createConditionOpts = append(createConditionOpts, CreateExternalSignalConditionOpt{
					Kind:            CreateExternalSignalConditionKindUSEREVENT,
					ReadableDataKey: condition.Base.ReadableDataKey,
					OrGroupId:       orGroupId,
					UserEventKey:    &condition.UserEventKey,
					Expression:      condition.Base.Expression,
				})
			}

			if len(createConditionOpts) > 0 {
				taskExternalId := task.ExternalID
				createMatchOpts := []ExternalCreateSignalMatchOpts{{
					Conditions:                    createConditionOpts,
					SignalTaskId:                  task.ID,
					SignalTaskInsertedAt:          task.InsertedAt,
					SignalExternalId:              task.ExternalID,
					SignalKey:                     signalKey,
					DurableCallbackTaskId:         &task.ID,
					DurableCallbackTaskInsertedAt: task.InsertedAt,
					DurableCallbackNodeId:         &callbackResult.Callback.NodeID,
					DurableCallbackTaskExternalId: &taskExternalId,
				}}

				err = r.registerSignalMatchConditions(ctx, tx, opts.TenantId, createMatchOpts)
				if err != nil {
					return nil, fmt.Errorf("failed to register signal match conditions: %w", err)
				}
			}
		}
	case sqlcv1.V1DurableEventLogEntryKindRUNTRIGGERED:
		triggerOpt, err := r.NewTriggerOpt(ctx, opts.TenantId, nil, task)

		if err != nil {
			return nil, fmt.Errorf("failed to create trigger options: %w", err)
		}

		// todo: need to pub olap messages after this somewhere
		_, _, err = r.triggerFromWorkflowNames(ctx, tx, opts.TenantId, []*WorkflowNameTriggerOpts{triggerOpt})

		if err != nil {
			return nil, fmt.Errorf("failed to trigger workflows: %w", err)
		}

		// todo: pub to olap here
	case sqlcv1.V1DurableEventLogEntryKindMEMOSTARTED:
		// todo: memo here
	default:
		return nil, fmt.Errorf("unsupported durable event log entry kind: %s", opts.Kind)
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return &IngestDurableTaskEventResult{
		Callback:      callbackResult,
		EventLogFile:  logFile,
		EventLogEntry: entryResult,
	}, nil
}
