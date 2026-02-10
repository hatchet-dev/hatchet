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

type EventLogEntryWithData struct {
	Entry *sqlcv1.V1DurableEventLogEntry
	Data  []byte
}

type IngestDurableTaskEventOpts struct {
	TenantId          uuid.UUID
	Task              *sqlcv1.FlattenExternalIdsRow
	NodeId            int64
	Kind              sqlcv1.V1DurableEventLogEntryKind
	Payload           []byte
	DispatcherId      uuid.UUID
	WaitForConditions *v1.DurableEventListenerConditions
}

type IngestDurableTaskEventResult struct {
	Callback      *sqlcv1.GetOrCreateDurableEventLogCallbackRow
	EventLogEntry *sqlcv1.GetOrCreateDurableEventLogEntryRow
	EventLogFile  *sqlcv1.GetOrCreateDurableEventLogFileRow
}

type DurableEventsRepository interface {
	UpdateEventLogCallbackSatisfied(ctx context.Context, tenantId uuid.UUID, nodeId, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz, isSatisfied bool, result []byte) (*sqlcv1.V1DurableEventLogCallback, error)

	IngestDurableTaskEvent(ctx context.Context, opts IngestDurableTaskEventOpts) (*IngestDurableTaskEventResult, error)
}

type durableEventsRepository struct {
	*sharedRepository
}

func newDurableEventsRepository(shared *sharedRepository) DurableEventsRepository {
	return &durableEventsRepository{
		sharedRepository: shared,
	}
}

func (r *durableEventsRepository) getEventLogEntry(ctx context.Context, db sqlcv1.DBTX, tenantId uuid.UUID, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz, nodeId int64) (*EventLogEntryWithData, error) {
	entry, err := r.queries.GetDurableEventLogEntry(ctx, db, sqlcv1.GetDurableEventLogEntryParams{
		Durabletaskid:         durableTaskId,
		Durabletaskinsertedat: durableTaskInsertedAt,
		Nodeid:                nodeId,
	})
	if err != nil {
		return nil, err
	}

	data, err := r.payloadStore.RetrieveSingle(ctx, r.pool, RetrievePayloadOpts{
		Id:         entry.ID,
		InsertedAt: entry.InsertedAt,
		Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYDATA,
		TenantId:   tenantId,
	})
	if err != nil {
		return nil, err
	}

	return &EventLogEntryWithData{
		Entry: entry,
		Data:  data,
	}, nil
}

func (r *durableEventsRepository) GetEventLogEntry(ctx context.Context, tenantId uuid.UUID, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz, nodeId int64) (*EventLogEntryWithData, error) {
	return r.getEventLogEntry(ctx, r.pool, tenantId, durableTaskId, durableTaskInsertedAt, nodeId)
}

func (r *durableEventsRepository) ListEventLogEntries(ctx context.Context, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz) ([]*sqlcv1.V1DurableEventLogEntry, error) {
	return r.queries.ListDurableEventLogEntries(ctx, r.pool, sqlcv1.ListDurableEventLogEntriesParams{
		Durabletaskid:         durableTaskId,
		Durabletaskinsertedat: durableTaskInsertedAt,
	})
}

func (r *durableEventsRepository) GetEventLogCallback(ctx context.Context, tenantId uuid.UUID, nodeId, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz) (*EventLogCallbackWithPayload, error) {
	callback, err := r.queries.GetDurableEventLogCallback(ctx, r.pool, sqlcv1.GetDurableEventLogCallbackParams{
		Durabletaskid:         durableTaskId,
		Durabletaskinsertedat: durableTaskInsertedAt,
		Nodeid:                nodeId,
	})

	if err != nil {
		return nil, err
	}

	result, err := r.payloadStore.RetrieveSingle(ctx, r.pool, RetrievePayloadOpts{
		Id:         callback.ID,
		InsertedAt: callback.InsertedAt,
		Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGCALLBACKRESULTDATA,
		TenantId:   tenantId,
	})

	if err != nil {
		return nil, err
	}

	return &EventLogCallbackWithPayload{
		Callback: callback,
		Result:   result,
	}, nil
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

	logFile, err := r.queries.GetOrCreateDurableEventLogFile(ctx, tx, sqlcv1.GetOrCreateDurableEventLogFileParams{
		Tenantid:                      opts.TenantId,
		Durabletaskid:                 task.ID,
		Durabletaskinsertedat:         task.InsertedAt,
		Latestinsertedat:              task.InsertedAt,
		Latestnodeid:                  0,
		Latestbranchid:                1,
		Latestbranchfirstparentnodeid: 0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get or create event log file for task: %w", err)
	}

	now := sqlchelpers.TimestamptzFromTime(time.Now().UTC())
	entryExternalId := uuid.New()

	parentNodeId := pgtype.Int8{
		Int64: logFile.LatestNodeID,
		Valid: true,
	}

	entry, err := r.queries.GetOrCreateDurableEventLogEntry(ctx, tx, sqlcv1.GetOrCreateDurableEventLogEntryParams{
		Tenantid:              opts.TenantId,
		Externalid:            entryExternalId,
		Durabletaskid:         task.ID,
		Durabletaskinsertedat: task.InsertedAt,
		Insertedat:            now,
		Kind:                  opts.Kind,
		Nodeid:                opts.NodeId,
		ParentNodeId:          parentNodeId,
		Branchid:              logFile.LatestBranchID,
		Datahash:              nil, // todo: populate this
		Datahashalg:           "",
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create event log entry: %w", err)
	}

	if len(opts.Payload) > 0 {
		err = r.payloadStore.Store(ctx, tx, StorePayloadOpts{
			Id:         entry.ID,
			InsertedAt: entry.InsertedAt,
			ExternalId: entry.ExternalID,
			Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYDATA,
			Payload:    opts.Payload,
			TenantId:   opts.TenantId,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to store event log entry payload: %w", err)
		}
	}

	callbackExternalId := uuid.New()
	callback, err := r.queries.GetOrCreateDurableEventLogCallback(ctx, tx, sqlcv1.GetOrCreateDurableEventLogCallbackParams{
		Tenantid:              opts.TenantId,
		Durabletaskid:         task.ID,
		Durabletaskinsertedat: task.InsertedAt,
		Insertedat:            now,
		Kind:                  sqlcv1.V1DurableEventLogCallbackKindWAITFORCOMPLETED,
		Nodeid:                opts.NodeId,
		Issatisfied:           false,
		Externalid:            callbackExternalId,
		Dispatcherid:          opts.DispatcherId,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create callback entry: %w", err)
	}

	switch opts.Kind {
	case sqlcv1.V1DurableEventLogEntryKindWAITFORSTARTED:
		if opts.WaitForConditions != nil {
			externalId := opts.Task.ExternalID
			signalKey := getDurableTaskSignalKey(externalId, opts.NodeId)

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
				createMatchOpts := []ExternalCreateSignalMatchOpts{{
					Conditions:                    createConditionOpts,
					SignalTaskId:                  task.ID,
					SignalTaskInsertedAt:          task.InsertedAt,
					SignalExternalId:              task.ExternalID,
					SignalKey:                     signalKey,
					DurableCallbackTaskId:         &task.ID,
					DurableCallbackTaskInsertedAt: task.InsertedAt,
					DurableCallbackNodeId:         &callback.NodeID,
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

		tasks, dags, err := r.triggerFromWorkflowNames(ctx, tx, opts.TenantId, []*WorkflowNameTriggerOpts{triggerOpt})

		if err != nil {
			return nil, fmt.Errorf("failed to trigger workflows: %w", err)
		}

		fmt.Println("triggered workflows, got tasks: ", tasks, " and dags: ", dags)

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
		Callback:      callback,
		EventLogFile:  logFile,
		EventLogEntry: entry,
	}, nil
}
