package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	Callback       *sqlcv1.V1DurableEventLogCallback
	Result         []byte
	AlreadyExisted bool
}

type EventLogEntryWithPayload struct {
	Entry          *sqlcv1.V1DurableEventLogEntry
	Payload        []byte
	AlreadyExisted bool
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
	InvocationCount   int64
}

type IngestDurableTaskEventResult struct {
	NodeId        int64
	Callback      *EventLogCallbackWithPayload
	EventLogEntry *EventLogEntryWithPayload
	EventLogFile  *sqlcv1.V1DurableEventLogFile
}

type DurableEventsRepository interface {
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
	params sqlcv1.CreateDurableEventLogEntryParams,
	payload []byte,
) (*EventLogEntryWithPayload, error) {
	alreadyExisted := true
	entry, err := r.queries.GetDurableEventLogEntry(ctx, tx, sqlcv1.GetDurableEventLogEntryParams{
		Durabletaskid:         params.Durabletaskid,
		Durabletaskinsertedat: params.Durabletaskinsertedat,
		Nodeid:                params.Nodeid,
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	} else if errors.Is(err, pgx.ErrNoRows) {
		alreadyExisted = false
		newEntry, err := r.queries.CreateDurableEventLogEntry(ctx, tx, sqlcv1.CreateDurableEventLogEntryParams{
			Tenantid:              params.Tenantid,
			Externalid:            params.Externalid,
			Durabletaskid:         params.Durabletaskid,
			Durabletaskinsertedat: params.Durabletaskinsertedat,
			Kind:                  params.Kind,
			Nodeid:                params.Nodeid,
			ParentNodeId:          params.ParentNodeId,
			Branchid:              params.Branchid,
			Datahash:              params.Datahash,
			Datahashalg:           params.Datahashalg,
		})

		if err != nil {
			return nil, err
		}

		entry = &sqlcv1.V1DurableEventLogEntry{
			TenantID:              newEntry.TenantID,
			ExternalID:            newEntry.ExternalID,
			InsertedAt:            newEntry.InsertedAt,
			ID:                    newEntry.ID,
			DurableTaskID:         newEntry.DurableTaskID,
			DurableTaskInsertedAt: newEntry.DurableTaskInsertedAt,
			Kind:                  newEntry.Kind,
			NodeID:                newEntry.NodeID,
			ParentNodeID:          newEntry.ParentNodeID,
			BranchID:              newEntry.BranchID,
			DataHash:              newEntry.DataHash,
			DataHashAlg:           newEntry.DataHashAlg,
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
	}

	return &EventLogEntryWithPayload{Entry: entry, Payload: payload, AlreadyExisted: alreadyExisted}, nil
}

func (r *durableEventsRepository) getOrCreateEventLogCallback(
	ctx context.Context,
	tx sqlcv1.DBTX,
	tenantId uuid.UUID,
	params sqlcv1.CreateDurableEventLogCallbackParams,
) (*EventLogCallbackWithPayload, error) {
	alreadyExists := true
	callback, err := r.queries.GetDurableEventLogCallback(ctx, tx, sqlcv1.GetDurableEventLogCallbackParams{
		Durabletaskid:         params.Durabletaskid,
		Durabletaskinsertedat: params.Durabletaskinsertedat,
		Nodeid:                params.Nodeid,
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	} else if errors.Is(err, pgx.ErrNoRows) {
		alreadyExists = false
		newCallback, err := r.queries.CreateDurableEventLogCallback(ctx, tx, sqlcv1.CreateDurableEventLogCallbackParams{
			Tenantid:              params.Tenantid,
			Durabletaskid:         params.Durabletaskid,
			Durabletaskinsertedat: params.Durabletaskinsertedat,
			Insertedat:            params.Insertedat,
			Externalid:            params.Externalid,
			Kind:                  params.Kind,
			Nodeid:                params.Nodeid,
			Issatisfied:           params.Issatisfied,
			Dispatcherid:          params.Dispatcherid,
		})

		if err != nil {
			return nil, err
		}

		callback = &sqlcv1.V1DurableEventLogCallback{
			TenantID:              newCallback.TenantID,
			DurableTaskID:         newCallback.DurableTaskID,
			DurableTaskInsertedAt: newCallback.DurableTaskInsertedAt,
			InsertedAt:            newCallback.InsertedAt,
			ID:                    newCallback.ID,
			ExternalID:            newCallback.ExternalID,
			Kind:                  newCallback.Kind,
			NodeID:                newCallback.NodeID,
			IsSatisfied:           newCallback.IsSatisfied,
			DispatcherID:          newCallback.DispatcherID,
		}
	}

	var result []byte
	if alreadyExists {
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

	return &EventLogCallbackWithPayload{Callback: callback, Result: result, AlreadyExisted: alreadyExists}, nil
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

	// todo: maybe acquire an exclusive lock on the row before incrementing the node id here
	logFile, err := r.queries.GetOrCreateEventLogFile(ctx, tx, sqlcv1.GetOrCreateEventLogFileParams{
		Tenantid:              opts.TenantId,
		Durabletaskid:         task.ID,
		Durabletaskinsertedat: task.InsertedAt,
		Invocationcount:       opts.InvocationCount,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get or create event log file: %w", err)
	}

	var nodeId int64
	if logFile.IsNewInvocation {
		newNode, err := r.queries.ResetLatestNodeId(ctx, tx, sqlcv1.ResetLatestNodeIdParams{
			Nodeid:                1,
			Durabletaskid:         task.ID,
			Durabletaskinsertedat: task.InsertedAt,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to reset latest node id for new invocation: %w", err)
		}

		nodeId = newNode.LatestNodeID
	} else {
		// if it's not a new invocation, we need to increment the latest node id (of the current invocation)
		nodeId = logFile.LatestNodeID + 1
	}

	now := sqlchelpers.TimestamptzFromTime(time.Now().UTC())

	// todo: real logic here for figuring out the parent
	parentNodeId := pgtype.Int8{
		Int64: 0,
		Valid: false,
	}

	// todo: real branching logic here
	branchId := logFile.LatestBranchID

	logEntry, err := r.getOrCreateEventLogEntry(ctx, tx, opts.TenantId, sqlcv1.CreateDurableEventLogEntryParams{
		Tenantid:              opts.TenantId,
		Externalid:            uuid.New(),
		Durabletaskid:         task.ID,
		Durabletaskinsertedat: task.InsertedAt,
		Kind:                  opts.Kind,
		Nodeid:                nodeId,
		ParentNodeId:          parentNodeId,
		Branchid:              branchId,
		Datahash:              nil, // todo: implement this for nondeterminism check
		Datahashalg:           "",
	}, opts.Payload)

	if err != nil {
		return nil, fmt.Errorf("failed to get or create event log entry: %w", err)
	}

	var kind sqlcv1.V1DurableEventLogCallbackKind

	switch opts.Kind {
	case sqlcv1.V1DurableEventLogEntryKindWAITFORSTARTED:
		kind = sqlcv1.V1DurableEventLogCallbackKindWAITFORCOMPLETED
	case sqlcv1.V1DurableEventLogEntryKindRUNTRIGGERED:
		kind = sqlcv1.V1DurableEventLogCallbackKindRUNCOMPLETED
	case sqlcv1.V1DurableEventLogEntryKindMEMOSTARTED:
		kind = sqlcv1.V1DurableEventLogCallbackKindMEMOCOMPLETED
	default:
		return nil, fmt.Errorf("unsupported durable event log entry kind: %s", opts.Kind)
	}

	callbackResult, err := r.getOrCreateEventLogCallback(ctx, tx, opts.TenantId, sqlcv1.CreateDurableEventLogCallbackParams{
		Tenantid:              opts.TenantId,
		Durabletaskid:         task.ID,
		Durabletaskinsertedat: task.InsertedAt,
		Insertedat:            now,
		Kind:                  kind,
		Nodeid:                nodeId,
		Issatisfied:           false,
		Externalid:            uuid.New(),
		Dispatcherid:          opts.DispatcherId,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get or create callback entry: %w", err)
	}

	if !logEntry.AlreadyExisted {
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

			_, _, err = r.triggerFromWorkflowNames(ctx, tx, opts.TenantId, []*WorkflowNameTriggerOpts{triggerOpt})

			if err != nil {
				return nil, fmt.Errorf("failed to trigger workflows: %w", err)
			}

		case sqlcv1.V1DurableEventLogEntryKindMEMOSTARTED:
			// todo: memo here
		default:
			return nil, fmt.Errorf("unsupported durable event log entry kind: %s", opts.Kind)
		}
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return &IngestDurableTaskEventResult{
		NodeId:   nodeId,
		Callback: callbackResult,
		EventLogFile: &sqlcv1.V1DurableEventLogFile{
			TenantID:                      logFile.TenantID,
			DurableTaskID:                 logFile.DurableTaskID,
			DurableTaskInsertedAt:         logFile.DurableTaskInsertedAt,
			LatestInvocationCount:         logFile.LatestInvocationCount,
			LatestInsertedAt:              logFile.LatestInsertedAt,
			LatestNodeID:                  logFile.LatestNodeID,
			LatestBranchID:                logFile.LatestBranchID,
			LatestBranchFirstParentNodeID: logFile.LatestBranchFirstParentNodeID,
		},
		EventLogEntry: logEntry,
	}, nil
}
