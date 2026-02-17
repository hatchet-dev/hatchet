package repository

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"

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

type NonDeterminismError struct {
	NodeId                 int64
	TaskExternalId         uuid.UUID
	ExpectedIdempotencyKey []byte
	ActualIdempotencyKey   []byte
}

func (m *NonDeterminismError) Error() string {
	return fmt.Sprintf("non-determinism detected for durable event log entry in task %s at node id %d", m.TaskExternalId.String(), m.NodeId)
}

type GetOrCreateLogEntryOpts struct {
	TenantId              uuid.UUID
	DurableTaskExternalId uuid.UUID
	DurableTaskId         int64
	DurableTaskInsertedAt pgtype.Timestamptz
	Kind                  sqlcv1.V1DurableEventLogKind
	NodeId                int64
	ParentNodeId          pgtype.Int8
	BranchId              int64
	IdempotencyKey        []byte
	IsSatisfied           bool
}

func (r *durableEventsRepository) getOrCreateEventLogEntry(
	ctx context.Context,
	tx sqlcv1.DBTX,
	tenantId uuid.UUID,
	params GetOrCreateLogEntryOpts,
	inputPayload []byte,
	resultPayload []byte,
) (*EventLogEntryWithPayloads, error) {
	entryExternalId := uuid.New()
	alreadyExisted := true
	entry, err := r.queries.GetDurableEventLogEntry(ctx, tx, sqlcv1.GetDurableEventLogEntryParams{
		Durabletaskid:         params.DurableTaskId,
		Durabletaskinsertedat: params.DurableTaskInsertedAt,
		Nodeid:                params.NodeId,
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	} else if errors.Is(err, pgx.ErrNoRows) {
		alreadyExisted = false
		entry, err := r.queries.CreateDurableEventLogEntry(ctx, tx, sqlcv1.CreateDurableEventLogEntryParams{
			Tenantid:              params.TenantId,
			Externalid:            entryExternalId,
			Durabletaskid:         params.DurableTaskId,
			Durabletaskinsertedat: params.DurableTaskInsertedAt,
			Kind:                  params.Kind,
			Nodeid:                params.NodeId,
			ParentNodeId:          params.ParentNodeId,
			Branchid:              params.BranchId,
			Idempotencykey:        params.IdempotencyKey,
			Issatisfied:           params.IsSatisfied,
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
	} else {
		incomingIdempotencyKey := params.IdempotencyKey
		existingIdempotencyKey := entry.IdempotencyKey

		if !bytes.Equal(incomingIdempotencyKey, existingIdempotencyKey) {
			return nil, &NonDeterminismError{
				NodeId:                 params.NodeId,
				TaskExternalId:         params.DurableTaskExternalId,
				ExpectedIdempotencyKey: existingIdempotencyKey,
				ActualIdempotencyKey:   incomingIdempotencyKey,
			}
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
		return nil, fmt.Errorf("failed to list satisfied entries: %w", err)
	}

	retrievePayloadOpts := make([]RetrievePayloadOpts, len(rows))

	for i, row := range rows {
		retrievePayloadOpts[i] = RetrievePayloadOpts{
			Id:         row.ID,
			InsertedAt: row.InsertedAt,
			Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYRESULTDATA,
			TenantId:   tenantId,
		}
	}

	payloads, err := r.payloadStore.Retrieve(ctx, r.pool, retrievePayloadOpts...)

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve payloads for satisfied callbacks: %w", err)
	}

	result := make([]*SatisfiedEventWithPayload, 0, len(rows))

	for _, row := range rows {
		retrieveOpt := RetrievePayloadOpts{
			Id:         row.ID,
			InsertedAt: row.InsertedAt,
			Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYRESULTDATA,
			TenantId:   tenantId,
		}

		payload := payloads[retrieveOpt]

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

func (r *durableEventsRepository) createIdempotencyKey(ctx context.Context, opts IngestDurableTaskEventOpts) ([]byte, error) {
	// todo: be more intentional about how we construct this key (e.g. do we want to marshal all of the opts?)
	dataToHash := []byte(opts.Kind)

	if opts.TriggerOpts != nil {
		dataToHash = append(dataToHash, opts.TriggerOpts.Data...)
		dataToHash = append(dataToHash, []byte(opts.TriggerOpts.WorkflowName)...)
	}

	if opts.WaitForConditions != nil {
		sort.Slice(opts.WaitForConditions, func(i, j int) bool {
			condI := opts.WaitForConditions[i]
			condJ := opts.WaitForConditions[j]

			if condI.Expression != condJ.Expression {
				return condI.Expression < condJ.Expression
			}

			if condI.ReadableDataKey != condJ.ReadableDataKey {
				return condI.ReadableDataKey < condJ.ReadableDataKey
			}

			if condI.Kind != condJ.Kind {
				return condI.Kind < condJ.Kind
			}

			if condI.SleepFor != nil && condJ.SleepFor != nil {
				if *condI.SleepFor != *condJ.SleepFor {
					return *condI.SleepFor < *condJ.SleepFor
				}
			}

			if condI.UserEventKey != nil && condJ.UserEventKey != nil {
				if *condI.UserEventKey != *condJ.UserEventKey {
					return *condI.UserEventKey < *condJ.UserEventKey
				}
			}

			return false
		})

		for _, cond := range opts.WaitForConditions {
			toHash := cond.Expression + cond.ReadableDataKey + string(cond.Kind)

			if cond.SleepFor != nil {
				toHash += *cond.SleepFor
			}

			if cond.UserEventKey != nil {
				toHash += *cond.UserEventKey
			}

			dataToHash = append(dataToHash, []byte(toHash)...)
		}
	}

	h := sha256.New()
	h.Write(dataToHash)
	hashBytes := h.Sum(nil)
	idempotencyKey := make([]byte, hex.EncodedLen(len(hashBytes)))
	hex.Encode(idempotencyKey, hashBytes)

	return idempotencyKey, nil
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
		// for memoization, we don't need to wait for anything before marking the entry as satisfied since it's just a cache entry
		isSatisfied = true
		resultPayload = opts.Payload
	default:
		return nil, fmt.Errorf("unsupported durable event log entry kind: %s", opts.Kind)
	}

	idempotencyKey, err := r.createIdempotencyKey(ctx, opts)

	if err != nil {
		return nil, fmt.Errorf("failed to create idempotency key: %w", err)
	}

	logEntry, err := r.getOrCreateEventLogEntry(
		ctx,
		tx,
		opts.TenantId,
		GetOrCreateLogEntryOpts{
			TenantId:              opts.TenantId,
			DurableTaskExternalId: task.ExternalID,
			DurableTaskId:         task.ID,
			DurableTaskInsertedAt: task.InsertedAt,
			Kind:                  opts.Kind,
			NodeId:                nodeId,
			ParentNodeId:          parentNodeId,
			BranchId:              branchId,
			IsSatisfied:           isSatisfied,
			IdempotencyKey:        idempotencyKey,
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
		Conditions:                 opts.WaitForConditions,
		SignalTaskId:               task.ID,
		SignalTaskInsertedAt:       task.InsertedAt,
		SignalTaskExternalId:       task.ExternalID,
		SignalExternalId:           taskExternalId,
		SignalKey:                  signalKey,
		DurableEventLogEntryNodeId: &nodeId,
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
		runEventLogEntrySignalKey := fmt.Sprintf("durable_run:%s:%d", task.ExternalID.String(), nodeId)

		err = r.createEventMatches(ctx, tx.tx, opts.TenantId, []CreateMatchOpts{{
			Kind:                       sqlcv1.V1MatchKindSIGNAL,
			Conditions:                 conditions,
			SignalTaskId:               &taskId,
			SignalTaskInsertedAt:       task.InsertedAt,
			SignalExternalId:           &taskExternalId,
			SignalTaskExternalId:       &taskExternalId,
			SignalKey:                  &runEventLogEntrySignalKey,
			DurableEventLogEntryNodeId: &nodeId,
		}})

		if err != nil {
			return nil, nil, fmt.Errorf("failed to register run completion match: %w", err)
		}
	}

	return createdDAGs, createdTasks, nil
}
