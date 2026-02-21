package repository

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

type TaskExternalIdNodeIdBranchId struct {
	TaskExternalId uuid.UUID `validate:"required"`
	NodeId         int64     `validate:"required"`
	BranchId       int64     `validate:"required"`
}

type SatisfiedEventWithPayload struct {
	TaskExternalId  uuid.UUID
	InvocationCount int32
	BranchID        int64
	NodeID          int64
	Result          []byte
}

type IngestDurableTaskEventOpts struct {
	TenantId          uuid.UUID                     `validate:"required"`
	Task              *sqlcv1.FlattenExternalIdsRow `validate:"required"`
	Kind              sqlcv1.V1DurableEventLogKind  `validate:"required,oneof=RUN WAIT_FOR MEMO"`
	Payload           []byte
	WaitForConditions []CreateExternalSignalConditionOpt
	InvocationCount   int32
	TriggerOpts       *WorkflowNameTriggerOpts
}

type IngestDurableTaskEventResult struct {
	BranchId        int64
	NodeId          int64
	InvocationCount int32

	IsSatisfied   bool
	ResultPayload []byte

	// Populated for RUNTRIGGERED: the tasks/DAGs created by the child spawn.
	CreatedTasks []*V1TaskWithPayload
	CreatedDAGs  []*DAGWithData
}

type HandleForkResult struct {
	NodeId       int64
	BranchId     int64
	EventLogFile *sqlcv1.V1DurableEventLogFile
}

type IncrementDurableTaskInvocationCountsOpts struct {
	TenantId       uuid.UUID
	TaskId         int64
	TaskInsertedAt pgtype.Timestamptz
}

type DurableEventsRepository interface {
	IngestDurableTaskEvent(ctx context.Context, opts IngestDurableTaskEventOpts) (*IngestDurableTaskEventResult, error)
	HandleFork(ctx context.Context, tenantId uuid.UUID, nodeId int64, task *sqlcv1.FlattenExternalIdsRow) (*HandleForkResult, error)

	GetSatisfiedDurableEvents(ctx context.Context, tenantId uuid.UUID, events []TaskExternalIdNodeIdBranchId) ([]*SatisfiedEventWithPayload, error)
	GetDurableTaskInvocationCounts(ctx context.Context, tenantId uuid.UUID, tasks []IdInsertedAt) (map[IdInsertedAt]*int32, error)
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
	BranchId               int64
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
	ParentNodeId          *int64
	BranchId              int64
	ParentBranchId        *int64
	InvocationCount       int32
	IdempotencyKey        []byte
	IsSatisfied           bool
	InputPayload          []byte
	ResultPayload         []byte
}

func (r *durableEventsRepository) getOrCreateEventLogEntry(
	ctx context.Context,
	tx sqlcv1.DBTX,
	opts GetOrCreateLogEntryOpts,
) (*EventLogEntryWithPayloads, error) {
	entryExternalId := uuid.New()
	alreadyExisted := true
	entry, err := r.queries.GetDurableEventLogEntry(ctx, tx, sqlcv1.GetDurableEventLogEntryParams{
		Durabletaskid:         opts.DurableTaskId,
		Durabletaskinsertedat: opts.DurableTaskInsertedAt,
		Nodeid:                opts.NodeId,
		Branchid:              opts.BranchId,
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	} else if errors.Is(err, pgx.ErrNoRows) {
		alreadyExisted = false
		entry, err = r.queries.CreateDurableEventLogEntry(ctx, tx, sqlcv1.CreateDurableEventLogEntryParams{
			Tenantid:              opts.TenantId,
			Externalid:            entryExternalId,
			Durabletaskid:         opts.DurableTaskId,
			Durabletaskinsertedat: opts.DurableTaskInsertedAt,
			Kind:                  opts.Kind,
			Nodeid:                opts.NodeId,
			ParentNodeId:          sqlchelpers.ToBigInt(opts.ParentNodeId),
			Branchid:              opts.BranchId,
			ParentBranchId:        sqlchelpers.ToBigInt(opts.ParentBranchId),
			Idempotencykey:        opts.IdempotencyKey,
			Issatisfied:           opts.IsSatisfied,
			Invocationcount:       opts.InvocationCount,
		})

		if err != nil {
			return nil, err
		}

		storePayloadOpts := make([]StorePayloadOpts, 0)

		if len(opts.InputPayload) > 0 {
			storePayloadOpts = append(storePayloadOpts, StorePayloadOpts{
				Id:         entry.ID,
				InsertedAt: entry.InsertedAt,
				ExternalId: entry.ExternalID,
				Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYDATA,
				Payload:    opts.InputPayload,
				TenantId:   opts.TenantId,
			})
		}

		if len(opts.ResultPayload) > 0 {
			storePayloadOpts = append(storePayloadOpts, StorePayloadOpts{
				Id:         entry.ID,
				InsertedAt: entry.InsertedAt,
				ExternalId: entry.ExternalID,
				Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYRESULTDATA,
				Payload:    opts.ResultPayload,
				TenantId:   opts.TenantId,
			})
		}

		err = r.payloadStore.Store(ctx, tx, storePayloadOpts...)
		if err != nil {
			return nil, err
		}
	} else {
		incomingIdempotencyKey := opts.IdempotencyKey
		existingIdempotencyKey := entry.IdempotencyKey

		if !bytes.Equal(incomingIdempotencyKey, existingIdempotencyKey) {
			return nil, &NonDeterminismError{
				BranchId:               opts.BranchId,
				NodeId:                 opts.NodeId,
				TaskExternalId:         opts.DurableTaskExternalId,
				ExpectedIdempotencyKey: existingIdempotencyKey,
				ActualIdempotencyKey:   incomingIdempotencyKey,
			}
		}
	}

	var resultPayload []byte

	if alreadyExisted {
		resultPayload, err = r.payloadStore.RetrieveSingle(ctx, tx, RetrievePayloadOpts{
			Id:         entry.ID,
			InsertedAt: entry.InsertedAt,
			Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYRESULTDATA,
			TenantId:   opts.TenantId,
		})

		if err != nil {
			resultPayload = nil
		}
	}

	return &EventLogEntryWithPayloads{
		Entry:          entry,
		InputPayload:   opts.InputPayload,
		ResultPayload:  resultPayload,
		AlreadyExisted: alreadyExisted,
	}, nil
}

func (r *durableEventsRepository) GetSatisfiedDurableEvents(ctx context.Context, tenantId uuid.UUID, events []TaskExternalIdNodeIdBranchId) ([]*SatisfiedEventWithPayload, error) {
	if len(events) == 0 {
		return nil, nil
	}

	taskExternalIds := make([]uuid.UUID, len(events))
	nodeIds := make([]int64, len(events))
	branchIds := make([]int64, len(events))
	isSatisfieds := make([]bool, len(events))

	for i, e := range events {
		if err := r.v.Validate(e); err != nil {
			return nil, fmt.Errorf("invalid event at index %d: %w", i, err)
		}

		taskExternalIds[i] = e.TaskExternalId
		nodeIds[i] = e.NodeId
		branchIds[i] = e.BranchId
		isSatisfieds[i] = true
	}

	rows, err := r.queries.ListSatisfiedEntries(ctx, r.pool, sqlcv1.ListSatisfiedEntriesParams{
		Taskexternalids: taskExternalIds,
		Nodeids:         nodeIds,
		Branchids:       branchIds,
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
			TaskExternalId:  row.TaskExternalID,
			NodeID:          row.NodeID,
			BranchID:        row.BranchID,
			InvocationCount: row.InvocationCount,
			Result:          payload,
		})
	}

	return result, nil
}

func getDurableTaskSignalKey(taskExternalId uuid.UUID, nodeId int64) string {
	return fmt.Sprintf("durable:%s:%d", taskExternalId.String(), nodeId)
}

func (r *durableEventsRepository) createIdempotencyKey(opts IngestDurableTaskEventOpts) ([]byte, error) {
	// TODO-DURABLE: be more intentional about how we construct this key (e.g. do we want to marshal all of the opts?)
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

func (r *sharedRepository) incrementDurableTaskInvocationCounts(ctx context.Context, tx sqlcv1.DBTX, opts []IncrementDurableTaskInvocationCountsOpts) (map[IncrementDurableTaskInvocationCountsOpts]*int32, error) {
	taskIds := make([]int64, len(opts))
	taskInsertedAts := make([]pgtype.Timestamptz, len(opts))
	tenantIds := make([]uuid.UUID, len(opts))

	for i, opt := range opts {
		taskIds[i] = opt.TaskId
		taskInsertedAts[i] = opt.TaskInsertedAt
		tenantIds[i] = opt.TenantId
	}

	logFiles, err := r.queries.IncrementLogFileInvocationCounts(ctx, tx, sqlcv1.IncrementLogFileInvocationCountsParams{
		Durabletaskids:         taskIds,
		Durabletaskinsertedats: taskInsertedAts,
		Tenantids:              tenantIds,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to increment invocation counts: %w", err)
	}

	result := make(map[IncrementDurableTaskInvocationCountsOpts]*int32, len(opts))

	for _, logFile := range logFiles {
		opt := IncrementDurableTaskInvocationCountsOpts{
			TenantId:       logFile.TenantID,
			TaskId:         logFile.DurableTaskID,
			TaskInsertedAt: logFile.DurableTaskInsertedAt,
		}

		result[opt] = &logFile.LatestInvocationCount
	}

	return result, nil
}

func (r *durableEventsRepository) getAndLockLogFile(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz) (*sqlcv1.V1DurableEventLogFile, error) {
	logFile, err := r.queries.GetAndLockLogFile(ctx, tx, sqlcv1.GetAndLockLogFileParams{
		Durabletaskid:         durableTaskId,
		Durabletaskinsertedat: durableTaskInsertedAt,
		Tenantid:              tenantId,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to lock log file: %w", err)
	}

	return logFile, nil
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

	logFile, err := r.getAndLockLogFile(ctx, tx, opts.TenantId, task.ID, task.InsertedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to lock log file: %w", err)
	}

	if logFile.LatestInvocationCount != opts.InvocationCount {
		// todo: should evict this invocation if this happens
		return nil, fmt.Errorf("invocation count mismatch: expected %d, got %d. rejecting event write.", logFile.LatestInvocationCount, opts.InvocationCount)
	}

	// TODO-DURABLE probably need to grab the previous entry here, if it exists, to determine how to increment?
	// basically need some way to figure out that we've reached a branching point
	// maybe we need to use `latest_branch_first_parent_node_id` for this?
	isNewInvocation := logFile.LatestInvocationCount < opts.InvocationCount

	var nodeId int64
	if isNewInvocation {
		one := int64(1)
		newNode, err := r.queries.UpdateLogFile(ctx, tx, sqlcv1.UpdateLogFileParams{
			NodeId:                sqlchelpers.ToBigInt(&one),
			InvocationCount:       sqlchelpers.ToInt(&opts.InvocationCount),
			Durabletaskid:         task.ID,
			Durabletaskinsertedat: task.InsertedAt,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to fork latest node id for new invocation: %w", err)
		}

		nodeId = newNode.LatestNodeID
	} else {
		// if it's not a new invocation, we need to increment the latest node id (of the current invocation)
		nodeId = logFile.LatestNodeID + 1
	}

	var parentNodeId *int64
	if !isNewInvocation && logFile.LatestNodeID > 0 {
		p := logFile.LatestNodeID
		parentNodeId = &p
	}

	branchId := logFile.LatestBranchID
	parentBranchId := logFile.LatestBranchID

	if logFile.LatestBranchFirstParentNodeID > 0 && nodeId <= logFile.LatestBranchFirstParentNodeID {
		parentBranch := logFile.LatestBranchID - 1
		branchId = parentBranch
		parentBranchId = parentBranch
	}

	if logFile.LatestBranchFirstParentNodeID > 0 && nodeId == logFile.LatestBranchFirstParentNodeID+1 {
		parentBranchId = logFile.LatestBranchID - 1
	}

	var inputPayload []byte
	var resultPayload []byte
	isSatisfied := false

	switch opts.Kind {
	case sqlcv1.V1DurableEventLogKindWAITFOR:
		inputPayload, err = json.Marshal(opts.WaitForConditions)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal wait for conditions: %w", err)
		}
	case sqlcv1.V1DurableEventLogKindRUN:
		inputPayload, err = json.Marshal(opts.TriggerOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal trigger opts: %w", err)
		}
	case sqlcv1.V1DurableEventLogKindMEMO:
		// for memoization, we don't need to wait for anything before marking the entry as satisfied since it's just a cache entry
		isSatisfied = true
		resultPayload = opts.Payload
	default:
		return nil, fmt.Errorf("unsupported durable event log entry kind: %s", opts.Kind)
	}

	idempotencyKey, err := r.createIdempotencyKey(opts)

	if err != nil {
		return nil, fmt.Errorf("failed to create idempotency key: %w", err)
	}

	logEntry, err := r.getOrCreateEventLogEntry(
		ctx,
		tx,
		GetOrCreateLogEntryOpts{
			TenantId:              opts.TenantId,
			DurableTaskExternalId: task.ExternalID,
			DurableTaskId:         task.ID,
			DurableTaskInsertedAt: task.InsertedAt,
			Kind:                  opts.Kind,
			NodeId:                nodeId,
			ParentNodeId:          parentNodeId,
			ParentBranchId:        &parentBranchId,
			BranchId:              branchId,
			InvocationCount:       opts.InvocationCount,
			IsSatisfied:           isSatisfied,
			IdempotencyKey:        idempotencyKey,
			InputPayload:          inputPayload,
			ResultPayload:         resultPayload,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get or create event log entry: %w", err)
	}

	var spawnedTasks []*V1TaskWithPayload
	var spawnedDAGs []*DAGWithData

	if !logEntry.AlreadyExisted {
		switch opts.Kind {
		case sqlcv1.V1DurableEventLogKindWAITFOR:
			err := r.handleWaitFor(ctx, tx, opts.TenantId, branchId, nodeId, opts.WaitForConditions, task)

			if err != nil {
				return nil, fmt.Errorf("failed to handle wait for conditions: %w", err)
			}
		case sqlcv1.V1DurableEventLogKindRUN:
			spawnedDAGs, spawnedTasks, err = r.handleTriggerRuns(ctx, optTx, opts.TenantId, branchId, nodeId, opts.TriggerOpts, task)

			if err != nil {
				return nil, fmt.Errorf("failed to handle trigger runs: %w", err)
			}
		case sqlcv1.V1DurableEventLogKindMEMO:
			// TODO-DURABLE: memo here
		default:
			return nil, fmt.Errorf("unsupported durable event log entry kind: %s", opts.Kind)
		}
	}

	logFile, err = r.queries.UpdateLogFile(ctx, tx, sqlcv1.UpdateLogFileParams{
		NodeId:                sqlchelpers.ToBigInt(&nodeId),
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
		NodeId:          nodeId,
		BranchId:        branchId,
		InvocationCount: opts.InvocationCount,
		IsSatisfied:     logEntry.Entry.IsSatisfied,
		ResultPayload:   logEntry.ResultPayload,
		CreatedTasks:    spawnedTasks,
		CreatedDAGs:     spawnedDAGs,
	}, nil
}

func (r *durableEventsRepository) handleWaitFor(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, branchId, nodeId int64, waitForConditions []CreateExternalSignalConditionOpt, task *sqlcv1.FlattenExternalIdsRow) error {
	if waitForConditions == nil {
		return nil
	}

	if len(waitForConditions) == 0 {
		return nil
	}

	taskExternalId := task.ExternalID
	signalKey := getDurableTaskSignalKey(taskExternalId, nodeId)

	createMatchOpts := []ExternalCreateSignalMatchOpts{{
		Conditions:                   waitForConditions,
		SignalTaskId:                 task.ID,
		SignalTaskInsertedAt:         task.InsertedAt,
		SignalTaskExternalId:         task.ExternalID,
		SignalExternalId:             taskExternalId,
		SignalKey:                    signalKey,
		DurableEventLogEntryNodeId:   &nodeId,
		DurableEventLogEntryBranchId: &branchId,
	}}

	return r.registerSignalMatchConditions(ctx, tx, tenantId, createMatchOpts)
}

func (r *durableEventsRepository) handleTriggerRuns(ctx context.Context, tx *OptimisticTx, tenantId uuid.UUID, branchId, nodeId int64, triggerOpts *WorkflowNameTriggerOpts, task *sqlcv1.FlattenExternalIdsRow) ([]*DAGWithData, []*V1TaskWithPayload, error) {
	if triggerOpts == nil {
		return nil, nil, fmt.Errorf("trigger options cannot be nil for RUN kind durable event log entry")
	}

	createdTasks, createdDAGs, err := r.triggerFromWorkflowNames(ctx, tx, tenantId, []*WorkflowNameTriggerOpts{triggerOpts})

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

		err = r.createEventMatches(ctx, tx.tx, tenantId, []CreateMatchOpts{{
			Kind:                         sqlcv1.V1MatchKindSIGNAL,
			Conditions:                   conditions,
			SignalTaskId:                 &taskId,
			SignalTaskInsertedAt:         task.InsertedAt,
			SignalExternalId:             &taskExternalId,
			SignalTaskExternalId:         &taskExternalId,
			SignalKey:                    &runEventLogEntrySignalKey,
			DurableEventLogEntryNodeId:   &nodeId,
			DurableEventLogEntryBranchId: &branchId,
		}})

		if err != nil {
			return nil, nil, fmt.Errorf("failed to register run completion match: %w", err)
		}
	}

	return createdDAGs, createdTasks, nil
}

func (r *durableEventsRepository) HandleFork(ctx context.Context, tenantId uuid.UUID, nodeId int64, task *sqlcv1.FlattenExternalIdsRow) (*HandleForkResult, error) {
	optTx, err := r.PrepareOptimisticTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare tx: %w", err)
	}
	defer optTx.Rollback()

	tx := optTx.tx

	logFile, err := r.getAndLockLogFile(ctx, tx, tenantId, task.ID, task.InsertedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to lock log file: %w", err)
	}

	newBranchId := logFile.LatestBranchID + 1
	lastFastForwardedNode := nodeId - 1
	zero := int64(0)

	logFile, err = r.queries.UpdateLogFile(ctx, tx, sqlcv1.UpdateLogFileParams{
		BranchId:                sqlchelpers.ToBigInt(&newBranchId),
		NodeId:                  sqlchelpers.ToBigInt(&zero),
		BranchFirstParentNodeId: sqlchelpers.ToBigInt(&lastFastForwardedNode),
		Durabletaskid:           task.ID,
		Durabletaskinsertedat:   task.InsertedAt,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to update log file for fork: %w", err)
	}

	if err := optTx.Commit(ctx); err != nil {
		return nil, err
	}

	return &HandleForkResult{
		NodeId:       nodeId,
		BranchId:     newBranchId,
		EventLogFile: logFile,
	}, nil
}

func (r *durableEventsRepository) GetDurableTaskInvocationCounts(ctx context.Context, tenantId uuid.UUID, tasks []IdInsertedAt) (map[IdInsertedAt]*int32, error) {
	if len(tasks) == 0 {
		return nil, nil
	}

	taskIds := make([]int64, len(tasks))
	taskInsertedAts := make([]pgtype.Timestamptz, len(tasks))
	tenantIds := make([]uuid.UUID, len(tasks))

	for i, t := range tasks {
		taskIds[i] = t.ID
		taskInsertedAts[i] = t.InsertedAt
		tenantIds[i] = tenantId
	}

	logFiles, err := r.queries.GetDurableTaskLogFiles(ctx, r.pool, sqlcv1.GetDurableTaskLogFilesParams{
		Durabletaskids:         taskIds,
		Durabletaskinsertedats: taskInsertedAts,
		Tenantids:              tenantIds,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get log files: %w", err)
	}

	result := make(map[IdInsertedAt]*int32, len(tasks))

	for _, logFile := range logFiles {
		key := IdInsertedAt{
			ID:         logFile.DurableTaskID,
			InsertedAt: logFile.DurableTaskInsertedAt,
		}

		result[key] = &logFile.LatestInvocationCount
	}

	return result, nil
}
