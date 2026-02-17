package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type EventLogEntryWithPayloads struct {
	Entry          *sqlcv1.V1DurableEventLogEntry
	InputPayload   []byte
	ResultPayload  []byte
	AlreadyExisted bool
}

type TaskExternalIdNodeId struct {
	TaskExternalId uuid.UUID `validate:"required"`
	NodeId         int64     `validate:"required"`
}

type SatisfiedEventWithPayload struct {
	TaskExternalId uuid.UUID
	NodeID         int64
	Result         []byte
}

type IngestDurableTaskEventOpts struct {
	TenantId          uuid.UUID                     `validate:"required"`
	Task              *sqlcv1.FlattenExternalIdsRow `validate:"required"`
	Kind              sqlcv1.V1DurableEventLogKind  `validate:"required,oneof=RUN WAIT_FOR MEMO"`
	Payload           []byte
	WaitForConditions []CreateExternalSignalConditionOpt
	InvocationCount   int64
	TriggerOpts       *WorkflowNameTriggerOpts
}

type IngestDurableTaskEventResult struct {
	NodeId        int64
	EventLogEntry *EventLogEntryWithPayloads
	EventLogFile  *sqlcv1.V1DurableEventLogFile

	// Populated for RUNTRIGGERED: the tasks/DAGs created by the child spawn.
	CreatedTasks []*V1TaskWithPayload
	CreatedDAGs  []*DAGWithData
}

type DurableEventsRepository interface {
	IngestDurableTaskEvent(ctx context.Context, opts IngestDurableTaskEventOpts) (*IngestDurableTaskEventResult, error)

	GetSatisfiedDurableEvents(ctx context.Context, tenantId uuid.UUID, events []TaskExternalIdNodeId) ([]*SatisfiedEventWithPayload, error)
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
	inputPayload []byte,
	resultPayload []byte,
) (*EventLogEntryWithPayloads, error) {
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
		entry, err := r.queries.CreateDurableEventLogEntry(ctx, tx, sqlcv1.CreateDurableEventLogEntryParams{
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

		storePayloadOpts := make([]StorePayloadOpts, 0)

		if len(inputPayload) > 0 {
			storePayloadOpts = append(storePayloadOpts, StorePayloadOpts{
				Id:         entry.ID,
				InsertedAt: entry.InsertedAt,
				ExternalId: entry.ExternalID,
				Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYDATA,
				Payload:    inputPayload,
				TenantId:   tenantId,
			})
		}

		if len(resultPayload) > 0 {
			storePayloadOpts = append(storePayloadOpts, StorePayloadOpts{
				Id:         entry.ID,
				InsertedAt: entry.InsertedAt,
				ExternalId: entry.ExternalID,
				Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYRESULTDATA,
				Payload:    resultPayload,
				TenantId:   tenantId,
			})
		}

		err = r.payloadStore.Store(ctx, tx, storePayloadOpts...)
		if err != nil {
			return nil, err
		}

	}

	if alreadyExisted {
		resultPayload, err = r.payloadStore.RetrieveSingle(ctx, tx, RetrievePayloadOpts{
			Id:         entry.ID,
			InsertedAt: entry.InsertedAt,
			Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYRESULTDATA,
			TenantId:   tenantId,
		})

		if err != nil {
			resultPayload = nil
		}
	}

	return &EventLogEntryWithPayloads{
		Entry:          entry,
		InputPayload:   inputPayload,
		ResultPayload:  resultPayload,
		AlreadyExisted: alreadyExisted,
	}, nil
}

func (r *durableEventsRepository) GetSatisfiedDurableEvents(ctx context.Context, tenantId uuid.UUID, events []TaskExternalIdNodeId) ([]*SatisfiedEventWithPayload, error) {
	if len(events) == 0 {
		return nil, nil
	}

	taskExternalIds := make([]uuid.UUID, len(events))
	nodeIds := make([]int64, len(events))
	isSatisfieds := make([]bool, len(events))

	for i, e := range events {
		if err := r.v.Validate(e); err != nil {
			return nil, fmt.Errorf("invalid event at index %d: %w", i, err)
		}

		taskExternalIds[i] = e.TaskExternalId
		nodeIds[i] = e.NodeId
		isSatisfieds[i] = true
	}

	rows, err := r.queries.ListSatisfiedEntries(ctx, r.pool, sqlcv1.ListSatisfiedEntriesParams{
		Taskexternalids: taskExternalIds,
		Nodeids:         nodeIds,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list satisfied callbacks: %w", err)
	}

	result := make([]*SatisfiedEventWithPayload, 0, len(rows))

	for _, row := range rows {
		payload, err := r.payloadStore.RetrieveSingle(ctx, r.pool, RetrievePayloadOpts{
			Id:         row.ID,
			InsertedAt: row.InsertedAt,
			Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYRESULTDATA,
			TenantId:   tenantId,
		})
		if err != nil {
			r.l.Warn().Err(err).Msgf("failed to retrieve payload for callback %d", row.NodeID)
			payload = nil
		}

		result = append(result, &SatisfiedEventWithPayload{
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
	if err := r.v.Validate(opts); err != nil {
		return nil, fmt.Errorf("invalid opts: %w", err)
	}

	task := opts.Task

	optTx, err := r.PrepareOptimisticTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare tx: %w", err)
	}
	defer optTx.Rollback()

	tx := optTx.tx

	// take a lock of the log file so nothing else can concurrently write to it and e.g. increment the node id or branch
	// id while this tx is running
	logFile, err := r.queries.GetAndLockLogFile(ctx, tx, sqlcv1.GetAndLockLogFileParams{
		Durabletaskid:         task.ID,
		Durabletaskinsertedat: task.InsertedAt,
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to lock log file: %w", err)
	}

	if errors.Is(err, pgx.ErrNoRows) {
		logFile, err = r.queries.CreateEventLogFile(ctx, tx, sqlcv1.CreateEventLogFileParams{
			Tenantid:              opts.TenantId,
			Durabletaskid:         task.ID,
			Durabletaskinsertedat: task.InsertedAt,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to get or create event log file: %w", err)
		}
	}

	isNewInvocation := false
	if logFile.LatestInvocationCount < opts.InvocationCount {
		isNewInvocation = true
	}

	var nodeId int64
	if isNewInvocation {
		newNode, err := r.queries.UpdateLogFileNodeIdInvocationCount(ctx, tx, sqlcv1.UpdateLogFileNodeIdInvocationCountParams{
			NodeId:                sqlchelpers.ToBigInt(1),
			InvocationCount:       sqlchelpers.ToBigInt(opts.InvocationCount),
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

	// todo: real logic here for figuring out the parent
	parentNodeId := pgtype.Int8{
		Int64: 0,
		Valid: false,
	}

	// todo: real branching logic here
	branchId := logFile.LatestBranchID

	var resultPayload []byte
	isSatisfied := false

	switch opts.Kind {
	case sqlcv1.V1DurableEventLogKindWAITFOR:
	case sqlcv1.V1DurableEventLogKindRUN:
		// do nothing
	case sqlcv1.V1DurableEventLogKindMEMO:
		// for memoization, we don't need to wait for anything before marking the callback as satisfied since it's just a cache entry
		isSatisfied = true
		resultPayload = opts.Payload
	default:
		return nil, fmt.Errorf("unsupported durable event log entry kind: %s", opts.Kind)
	}

	logEntry, err := r.getOrCreateEventLogEntry(
		ctx,
		tx,
		opts.TenantId,
		sqlcv1.CreateDurableEventLogEntryParams{
			Tenantid:              opts.TenantId,
			Externalid:            uuid.New(),
			Durabletaskid:         task.ID,
			Durabletaskinsertedat: task.InsertedAt,
			Kind:                  opts.Kind,
			Nodeid:                nodeId,
			ParentNodeId:          parentNodeId,
			Branchid:              branchId,
			Issatisfied:           isSatisfied,
			Datahash:              nil, // todo: implement this for nondeterminism check
			Datahashalg:           "",
			// todo: send payload here to write result
		},
		opts.Payload,
		resultPayload,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get or create event log entry: %w", err)
	}

	var spawnedTasks []*V1TaskWithPayload
	var spawnedDAGs []*DAGWithData

	if !logEntry.AlreadyExisted {
		switch opts.Kind {
		case sqlcv1.V1DurableEventLogKindWAITFOR:
			err := r.handleWaitFor(ctx, tx, nodeId, opts, task)

			if err != nil {
				return nil, fmt.Errorf("failed to handle wait for conditions: %w", err)
			}
		case sqlcv1.V1DurableEventLogKindRUN:
			spawnedDAGs, spawnedTasks, err = r.handleTriggerRuns(ctx, optTx, nodeId, opts, task)

			if err != nil {
				return nil, fmt.Errorf("failed to handle trigger runs: %w", err)
			}
		case sqlcv1.V1DurableEventLogKindMEMO:
			// todo: memo here
		default:
			return nil, fmt.Errorf("unsupported durable event log entry kind: %s", opts.Kind)
		}
	}

	logFile, err = r.queries.UpdateLogFileNodeIdInvocationCount(ctx, tx, sqlcv1.UpdateLogFileNodeIdInvocationCountParams{
		NodeId:                sqlchelpers.ToBigInt(nodeId),
		Durabletaskid:         task.ID,
		Durabletaskinsertedat: task.InsertedAt,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to update latest node id: %w", err)
	}

	if err := optTx.Commit(ctx); err != nil {
		return nil, err
	}

	return &IngestDurableTaskEventResult{
		NodeId:        nodeId,
		EventLogFile:  logFile,
		EventLogEntry: logEntry,
		CreatedTasks:  spawnedTasks,
		CreatedDAGs:   spawnedDAGs,
	}, nil
}

func (r *durableEventsRepository) handleWaitFor(ctx context.Context, tx sqlcv1.DBTX, nodeId int64, opts IngestDurableTaskEventOpts, task *sqlcv1.FlattenExternalIdsRow) error {
	if opts.WaitForConditions == nil {
		return nil
	}

	if len(opts.WaitForConditions) == 0 {
		return nil
	}

	taskExternalId := opts.Task.ExternalID
	signalKey := getDurableTaskSignalKey(taskExternalId, nodeId)

	createMatchOpts := []ExternalCreateSignalMatchOpts{{
		Conditions:                    opts.WaitForConditions,
		SignalTaskId:                  task.ID,
		SignalTaskInsertedAt:          task.InsertedAt,
		SignalTaskExternalId:          task.ExternalID,
		SignalExternalId:              task.ExternalID,
		SignalKey:                     signalKey,
		DurableCallbackTaskId:         &task.ID,
		DurableCallbackTaskInsertedAt: task.InsertedAt,
		DurableCallbackNodeId:         &nodeId,
		DurableCallbackTaskExternalId: &taskExternalId,
	}}

	return r.registerSignalMatchConditions(ctx, tx, opts.TenantId, createMatchOpts)
}

func (r *durableEventsRepository) handleTriggerRuns(ctx context.Context, tx *OptimisticTx, nodeId int64, opts IngestDurableTaskEventOpts, task *sqlcv1.FlattenExternalIdsRow) ([]*DAGWithData, []*V1TaskWithPayload, error) {
	createdTasks, createdDAGs, err := r.triggerFromWorkflowNames(ctx, tx, opts.TenantId, []*WorkflowNameTriggerOpts{opts.TriggerOpts})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to trigger workflows: %w", err)
	}

	taskId := task.ID
	taskExternalId := task.ExternalID

	var conditions []GroupMatchCondition

	for _, childTask := range createdTasks {
		childHint := childTask.ExternalID.String()
		orGroupId := uuid.New()

		conditions = append(conditions,
			GroupMatchCondition{
				GroupId:           orGroupId,
				EventType:         sqlcv1.V1EventTypeINTERNAL,
				EventKey:          string(sqlcv1.V1TaskEventTypeCOMPLETED),
				ReadableDataKey:   "output",
				EventResourceHint: &childHint,
				Expression:        "true",
				Action:            sqlcv1.V1MatchConditionActionCREATE,
			},
			GroupMatchCondition{
				GroupId:           orGroupId,
				EventType:         sqlcv1.V1EventTypeINTERNAL,
				EventKey:          string(sqlcv1.V1TaskEventTypeFAILED),
				ReadableDataKey:   "output",
				EventResourceHint: &childHint,
				Expression:        "true",
				Action:            sqlcv1.V1MatchConditionActionCREATE,
			},
			GroupMatchCondition{
				GroupId:           orGroupId,
				EventType:         sqlcv1.V1EventTypeINTERNAL,
				EventKey:          string(sqlcv1.V1TaskEventTypeCANCELLED),
				ReadableDataKey:   "output",
				EventResourceHint: &childHint,
				Expression:        "true",
				Action:            sqlcv1.V1MatchConditionActionCREATE,
			},
		)
	}

	if len(conditions) > 0 {
		runCallbackSignalKey := fmt.Sprintf("durable_run:%s:%d", task.ExternalID.String(), nodeId)

		err = r.createEventMatches(ctx, tx.tx, opts.TenantId, []CreateMatchOpts{{
			Kind:                          sqlcv1.V1MatchKindSIGNAL,
			Conditions:                    conditions,
			SignalTaskId:                  &taskId,
			SignalTaskInsertedAt:          task.InsertedAt,
			SignalExternalId:              &taskExternalId,
			SignalTaskExternalId:          &taskExternalId,
			SignalKey:                     &runCallbackSignalKey,
			DurableCallbackTaskId:         &taskId,
			DurableCallbackTaskInsertedAt: task.InsertedAt,
			DurableCallbackNodeId:         &nodeId,
			DurableCallbackTaskExternalId: &taskExternalId,
		}})

		if err != nil {
			return nil, nil, fmt.Errorf("failed to register run completion match: %w", err)
		}
	}

	return createdDAGs, createdTasks, nil
}
