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
	TenantId        uuid.UUID                     `validate:"required"`
	Task            *sqlcv1.FlattenExternalIdsRow `validate:"required"`
	Kind            sqlcv1.V1DurableEventLogKind  `validate:"required,oneof=RUN WAIT_FOR MEMO"`
	Payload         []byte
	InvocationCount int32

	// optional, used only when kind = WAIT_FOR
	WaitForConditions []CreateExternalSignalConditionOpt

	// optional, used only when kind = RUN
	TriggerOpts *WorkflowNameTriggerOpts

	// optional, used only when kind = MEMO
	MemoKey []byte
}

type IngestDurableTaskEventResult struct {
	BranchId        int64
	NodeId          int64
	InvocationCount int32

	IsSatisfied    bool
	ResultPayload  []byte
	AlreadyExisted bool

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

type CompleteMemoEntryOpts struct {
	TenantId        uuid.UUID
	TaskExternalId  uuid.UUID
	InvocationCount int32
	BranchId        int64
	NodeId          int64
	MemoKey         []byte
	Payload         []byte
}

type IngestBulkDurableTaskRunEntry struct {
	ResultPayload  []byte
	CreatedTasks   []*V1TaskWithPayload
	CreatedDAGs    []*DAGWithData
	NodeId         int64
	BranchId       int64
	IsSatisfied    bool
	AlreadyExisted bool
}

type IngestBulkDurableTaskRunResult struct {
	Entries         []IngestBulkDurableTaskRunEntry
	InvocationCount int32
}

type DurableEventsRepository interface {
	IngestDurableTaskEvent(ctx context.Context, opts IngestDurableTaskEventOpts) (*IngestDurableTaskEventResult, error)
	IngestBulkDurableTaskRunEvents(ctx context.Context, tenantId uuid.UUID, task *sqlcv1.FlattenExternalIdsRow, invocationCount int32, triggerOptsList []*WorkflowNameTriggerOpts) (*IngestBulkDurableTaskRunResult, error)
	HandleFork(ctx context.Context, tenantId uuid.UUID, nodeId int64, task *sqlcv1.FlattenExternalIdsRow) (*HandleForkResult, error)

	GetSatisfiedDurableEvents(ctx context.Context, tenantId uuid.UUID, events []TaskExternalIdNodeIdBranchId) ([]*SatisfiedEventWithPayload, error)
	GetDurableTaskInvocationCounts(ctx context.Context, tenantId uuid.UUID, tasks []IdInsertedAt) (map[IdInsertedAt]*int32, error)
	CompleteMemoEntry(ctx context.Context, opts CompleteMemoEntryOpts) error
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
	alreadyExisted := true
	entry, err := r.queries.GetDurableEventLogEntry(ctx, tx, sqlcv1.GetDurableEventLogEntryParams{
		Durabletaskid:         opts.DurableTaskId,
		Durabletaskinsertedat: opts.DurableTaskInsertedAt,
		Nodeid:                opts.NodeId,
		Branchid:              opts.BranchId,
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	isStaleEntry := err == nil && entry != nil && entry.InvocationCount != opts.InvocationCount

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		alreadyExisted = false
		entry, err = r.queries.CreateDurableEventLogEntry(ctx, tx, sqlcv1.CreateDurableEventLogEntryParams{
			Tenantid:              opts.TenantId,
			Externalid:            uuid.New(),
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
	case isStaleEntry:
		// TODO-DURABLE: I don't think this should be required or at least should not be handled here...
		// NOTE: entry exists but belongs to a previous invocation (e.g. after eviction+restore
		// or cancel+replay). Check idempotency key for non-determinism; if it matches, update
		// invocation_count so callbacks route correctly and reuse existing wait conditions.
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

		entry, err = r.queries.UpdateDurableEventLogEntryInvocationCount(ctx, tx, sqlcv1.UpdateDurableEventLogEntryInvocationCountParams{
			Invocationcount:       opts.InvocationCount,
			Idempotencykey:        opts.IdempotencyKey,
			Durabletaskid:         opts.DurableTaskId,
			Durabletaskinsertedat: opts.DurableTaskInsertedAt,
			Branchid:              opts.BranchId,
			Nodeid:                opts.NodeId,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to update invocation count on stale log entry: %w", err)
		}
	default:
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
		// TODO-DURABLE: should evict this invocation if this happens
		return nil, fmt.Errorf("invocation count mismatch: expected %d, got %d. rejecting event write.", logFile.LatestInvocationCount, opts.InvocationCount)
	}

	nodeId := logFile.LatestNodeID + 1

	var parentNodeId *int64
	if logFile.LatestNodeID > 0 {
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
	var idempotencyKey []byte
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
		// if we get a payload here, it means we should persist it and mark the memo event as having been satisfied,
		// since it's now replayable by retrieving that payload
		if len(opts.Payload) > 0 {
			isSatisfied = true
			resultPayload = opts.Payload
		}
		idempotencyKey = opts.MemoKey
	default:
		return nil, fmt.Errorf("unsupported durable event log entry kind: %s", opts.Kind)
	}

	if opts.Kind != sqlcv1.V1DurableEventLogKindMEMO {
		idempotencyKey, err = r.createIdempotencyKey(opts)

		if err != nil {
			return nil, fmt.Errorf("failed to create idempotency key: %w", err)
		}
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
			// do nothing - we don't need to do anything downstream since memo just writes the cache entry and returns
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
		AlreadyExisted:  logEntry.AlreadyExisted,
		ResultPayload:   logEntry.ResultPayload,
		CreatedTasks:    spawnedTasks,
		CreatedDAGs:     spawnedDAGs,
	}, nil
}

func createRunIdempotencyKey(triggerOpts *WorkflowNameTriggerOpts) ([]byte, error) {
	dataToHash := []byte(sqlcv1.V1DurableEventLogKindRUN)
	dataToHash = append(dataToHash, triggerOpts.Data...)
	dataToHash = append(dataToHash, []byte(triggerOpts.WorkflowName)...)

	h := sha256.New()
	h.Write(dataToHash)
	hashBytes := h.Sum(nil)
	idempotencyKey := make([]byte, hex.EncodedLen(len(hashBytes)))
	hex.Encode(idempotencyKey, hashBytes)

	return idempotencyKey, nil
}

func (r *durableEventsRepository) IngestBulkDurableTaskRunEvents(
	ctx context.Context,
	tenantId uuid.UUID,
	task *sqlcv1.FlattenExternalIdsRow,
	invocationCount int32,
	triggerOptsList []*WorkflowNameTriggerOpts,
) (*IngestBulkDurableTaskRunResult, error) {
	if len(triggerOptsList) == 0 {
		return &IngestBulkDurableTaskRunResult{InvocationCount: invocationCount}, nil
	}

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

	if logFile.LatestInvocationCount != invocationCount {
		return nil, fmt.Errorf("invocation count mismatch: expected %d, got %d, rejecting event write", logFile.LatestInvocationCount, invocationCount)
	}

	baseNodeId := logFile.LatestNodeID
	n := len(triggerOptsList)

	type entryMeta struct {
		parentNodeId   *int64
		parentBranchId *int64
		triggerOpts    *WorkflowNameTriggerOpts
		idempotencyKey []byte
		inputPayload   []byte
		nodeId         int64
		branchId       int64
	}

	metas := make([]entryMeta, n)

	for i, triggerOpts := range triggerOptsList {
		nodeId := baseNodeId + 1 + int64(i)

		var parentNodeId *int64
		if prevNode := baseNodeId + int64(i); prevNode > 0 {
			parentNodeId = &prevNode
		}

		branchId := logFile.LatestBranchID
		pb := logFile.LatestBranchID
		parentBranchId := &pb

		if logFile.LatestBranchFirstParentNodeID > 0 && nodeId <= logFile.LatestBranchFirstParentNodeID {
			parentBranch := logFile.LatestBranchID - 1
			branchId = parentBranch
			parentBranchId = &parentBranch
		}

		if logFile.LatestBranchFirstParentNodeID > 0 && nodeId == logFile.LatestBranchFirstParentNodeID+1 {
			pb2 := logFile.LatestBranchID - 1
			parentBranchId = &pb2
		}

		inputPayload, marshalErr := json.Marshal(triggerOpts)
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal trigger opts: %w", marshalErr)
		}

		idempotencyKey, keyErr := createRunIdempotencyKey(triggerOpts)
		if keyErr != nil {
			return nil, fmt.Errorf("failed to create idempotency key: %w", keyErr)
		}

		metas[i] = entryMeta{
			nodeId:         nodeId,
			branchId:       branchId,
			parentNodeId:   parentNodeId,
			parentBranchId: parentBranchId,
			idempotencyKey: idempotencyKey,
			inputPayload:   inputPayload,
			triggerOpts:    triggerOpts,
		}
	}

	// bulk-get existing entries
	getParams := sqlcv1.BulkGetDurableEventLogEntriesParams{
		Durabletaskids:         make([]int64, n),
		Durabletaskinsertedats: make([]pgtype.Timestamptz, n),
		Branchids:              make([]int64, n),
		Nodeids:                make([]int64, n),
	}

	for i, m := range metas {
		getParams.Durabletaskids[i] = task.ID
		getParams.Durabletaskinsertedats[i] = task.InsertedAt
		getParams.Branchids[i] = m.branchId
		getParams.Nodeids[i] = m.nodeId
	}

	existingEntries, err := r.queries.BulkGetDurableEventLogEntries(ctx, tx, getParams)
	if err != nil {
		return nil, fmt.Errorf("failed to bulk-get existing entries: %w", err)
	}

	type branchNodeKey struct {
		branchId int64
		nodeId   int64
	}

	existingByKey := make(map[branchNodeKey]*sqlcv1.V1DurableEventLogEntry, len(existingEntries))
	for _, e := range existingEntries {
		existingByKey[branchNodeKey{e.BranchID, e.NodeID}] = e
	}

	// classify and validate
	type newEntryInfo struct {
		metaIdx int
	}

	type staleEntryInfo struct {
		entry   *sqlcv1.V1DurableEventLogEntry
		metaIdx int
	}

	var newEntries []newEntryInfo
	var staleEntries []staleEntryInfo
	existedEntries := make(map[int]*sqlcv1.V1DurableEventLogEntry)

	for i, m := range metas {
		key := branchNodeKey{m.branchId, m.nodeId}
		existing, found := existingByKey[key]

		if !found {
			newEntries = append(newEntries, newEntryInfo{metaIdx: i})
			continue
		}

		if !bytes.Equal(m.idempotencyKey, existing.IdempotencyKey) {
			return nil, &NonDeterminismError{
				BranchId:               m.branchId,
				NodeId:                 m.nodeId,
				TaskExternalId:         task.ExternalID,
				ExpectedIdempotencyKey: existing.IdempotencyKey,
				ActualIdempotencyKey:   m.idempotencyKey,
			}
		}

		if existing.InvocationCount != invocationCount {
			staleEntries = append(staleEntries, staleEntryInfo{metaIdx: i, entry: existing})
		} else {
			existedEntries[i] = existing
		}
	}

	// bulk-create new entries
	var createdByNodeId map[int64]*sqlcv1.V1DurableEventLogEntry

	if len(newEntries) > 0 {
		createParams := sqlcv1.BulkCreateDurableEventLogEntriesParams{
			Tenantids:              make([]uuid.UUID, len(newEntries)),
			Externalids:            make([]uuid.UUID, len(newEntries)),
			Durabletaskids:         make([]int64, len(newEntries)),
			Durabletaskinsertedats: make([]pgtype.Timestamptz, len(newEntries)),
			Kinds:                  make([]string, len(newEntries)),
			Nodeids:                make([]int64, len(newEntries)),
			Parentnodeids:          make([]int64, len(newEntries)),
			Branchids:              make([]int64, len(newEntries)),
			Parentbranchids:        make([]int64, len(newEntries)),
			Invocationcounts:       make([]int32, len(newEntries)),
			Idempotencykeys:        make([][]byte, len(newEntries)),
			Issatisfieds:           make([]bool, len(newEntries)),
		}

		for j, ne := range newEntries {
			m := metas[ne.metaIdx]
			createParams.Tenantids[j] = tenantId
			createParams.Externalids[j] = uuid.New()
			createParams.Durabletaskids[j] = task.ID
			createParams.Durabletaskinsertedats[j] = task.InsertedAt
			createParams.Kinds[j] = string(sqlcv1.V1DurableEventLogKindRUN)
			createParams.Nodeids[j] = m.nodeId
			createParams.Parentnodeids[j] = int64OrNull(m.parentNodeId)
			createParams.Branchids[j] = m.branchId
			createParams.Parentbranchids[j] = int64OrNull(m.parentBranchId)
			createParams.Invocationcounts[j] = invocationCount
			createParams.Idempotencykeys[j] = m.idempotencyKey
			createParams.Issatisfieds[j] = false
		}

		createdRows, createErr := r.queries.BulkCreateDurableEventLogEntries(ctx, tx, createParams)
		if createErr != nil {
			return nil, fmt.Errorf("failed to bulk-create event log entries: %w", createErr)
		}

		createdByNodeId = make(map[int64]*sqlcv1.V1DurableEventLogEntry, len(createdRows))
		for _, row := range createdRows {
			createdByNodeId[row.NodeID] = row
		}

		storePayloadOpts := make([]StorePayloadOpts, 0, len(newEntries))
		for _, ne := range newEntries {
			m := metas[ne.metaIdx]
			created, ok := createdByNodeId[m.nodeId]
			if !ok {
				continue
			}
			if len(m.inputPayload) > 0 {
				storePayloadOpts = append(storePayloadOpts, StorePayloadOpts{
					Id:         created.ID,
					InsertedAt: created.InsertedAt,
					ExternalId: created.ExternalID,
					Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYDATA,
					Payload:    m.inputPayload,
					TenantId:   tenantId,
				})
			}
		}

		if len(storePayloadOpts) > 0 {
			if storeErr := r.payloadStore.Store(ctx, tx, storePayloadOpts...); storeErr != nil {
				return nil, fmt.Errorf("failed to store payloads for new entries: %w", storeErr)
			}
		}
	}

	// bulk-update stale entries
	if len(staleEntries) > 0 {
		updateParams := sqlcv1.BulkUpdateDurableEventLogEntryInvocationCountsParams{
			Durabletaskids:         make([]int64, len(staleEntries)),
			Durabletaskinsertedats: make([]pgtype.Timestamptz, len(staleEntries)),
			Branchids:              make([]int64, len(staleEntries)),
			Nodeids:                make([]int64, len(staleEntries)),
			Invocationcounts:       make([]int32, len(staleEntries)),
			Idempotencykeys:        make([][]byte, len(staleEntries)),
		}

		for j, se := range staleEntries {
			m := metas[se.metaIdx]
			updateParams.Durabletaskids[j] = task.ID
			updateParams.Durabletaskinsertedats[j] = task.InsertedAt
			updateParams.Branchids[j] = m.branchId
			updateParams.Nodeids[j] = m.nodeId
			updateParams.Invocationcounts[j] = invocationCount
			updateParams.Idempotencykeys[j] = m.idempotencyKey
		}

		updatedRows, updateErr := r.queries.BulkUpdateDurableEventLogEntryInvocationCounts(ctx, tx, updateParams)
		if updateErr != nil {
			return nil, fmt.Errorf("failed to bulk-update stale entry invocation counts: %w", updateErr)
		}

		for _, row := range updatedRows {
			for _, se := range staleEntries {
				m := metas[se.metaIdx]
				if row.NodeID == m.nodeId && row.BranchID == m.branchId {
					existedEntries[se.metaIdx] = row
					break
				}
			}
		}
	}

	// retrieve result payloads for all existing entries
	var retrieveOpts []RetrievePayloadOpts

	for _, entry := range existedEntries {
		retrieveOpts = append(retrieveOpts, RetrievePayloadOpts{
			Id:         entry.ID,
			InsertedAt: entry.InsertedAt,
			Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYRESULTDATA,
			TenantId:   tenantId,
		})
	}

	var existingPayloads map[RetrievePayloadOpts][]byte
	if len(retrieveOpts) > 0 {
		existingPayloads, err = r.payloadStore.Retrieve(ctx, tx, retrieveOpts...)
		if err != nil {
			existingPayloads = nil
		}
	}

	// batch trigger runs for new entries
	var newTriggerOpts []*WorkflowNameTriggerOpts
	var newTriggerMetaIdxs []int

	for _, ne := range newEntries {
		m := metas[ne.metaIdx]
		if _, created := createdByNodeId[m.nodeId]; created {
			newTriggerOpts = append(newTriggerOpts, m.triggerOpts)
			newTriggerMetaIdxs = append(newTriggerMetaIdxs, ne.metaIdx)
		}
	}

	// externalId -> metaIdx for mapping created tasks back to entries
	triggerExternalIdToMetaIdx := make(map[uuid.UUID]int, len(newTriggerOpts))
	for j, opts := range newTriggerOpts {
		triggerExternalIdToMetaIdx[opts.ExternalId] = newTriggerMetaIdxs[j]
	}

	// tasks/DAGs grouped by metaIdx
	tasksByMetaIdx := make(map[int][]*V1TaskWithPayload)
	dagsByMetaIdx := make(map[int][]*DAGWithData)

	if len(newTriggerOpts) > 0 {
		allCreatedTasks, allCreatedDAGs, triggerErr := r.triggerFromWorkflowNames(ctx, optTx, tenantId, newTriggerOpts)
		if triggerErr != nil {
			return nil, fmt.Errorf("failed to trigger workflows: %w", triggerErr)
		}

		for _, ct := range allCreatedTasks {
			if metaIdx, ok := triggerExternalIdToMetaIdx[ct.WorkflowRunID]; ok {
				tasksByMetaIdx[metaIdx] = append(tasksByMetaIdx[metaIdx], ct)
			}
		}

		for _, cd := range allCreatedDAGs {
			if metaIdx, ok := triggerExternalIdToMetaIdx[cd.ExternalID]; ok {
				dagsByMetaIdx[metaIdx] = append(dagsByMetaIdx[metaIdx], cd)
			}
		}

		// build all signal match conditions in one batch
		var allMatchOpts []CreateMatchOpts
		taskId := task.ID
		taskExternalId := task.ExternalID

		for _, metaIdx := range newTriggerMetaIdxs {
			m := metas[metaIdx]
			childTasks := tasksByMetaIdx[metaIdx]
			if len(childTasks) == 0 {
				continue
			}

			childHints := make([]string, 0, len(childTasks))
			for _, childTask := range childTasks {
				childHints = append(childHints, childTask.ExternalID.String())
			}
			conditions := ChildTerminalMatchConditions(childHints, "output")

			nodeId := m.nodeId
			branchId := m.branchId
			runEventLogEntrySignalKey := fmt.Sprintf("durable_run:%s:%d", task.ExternalID.String(), nodeId)

			allMatchOpts = append(allMatchOpts, CreateMatchOpts{
				Kind:                         sqlcv1.V1MatchKindSIGNAL,
				Conditions:                   conditions,
				SignalTaskId:                 &taskId,
				SignalTaskInsertedAt:         task.InsertedAt,
				SignalExternalId:             &taskExternalId,
				SignalTaskExternalId:         &taskExternalId,
				SignalKey:                    &runEventLogEntrySignalKey,
				DurableEventLogEntryNodeId:   &nodeId,
				DurableEventLogEntryBranchId: &branchId,
			})
		}

		if len(allMatchOpts) > 0 {
			if matchErr := r.createEventMatches(ctx, tx, tenantId, allMatchOpts); matchErr != nil {
				return nil, fmt.Errorf("failed to register run completion matches: %w", matchErr)
			}
		}
	}

	// build result entries
	entries := make([]IngestBulkDurableTaskRunEntry, n)

	for i, m := range metas {
		entry := IngestBulkDurableTaskRunEntry{
			NodeId:   m.nodeId,
			BranchId: m.branchId,
		}

		if existingEntry, ok := existedEntries[i]; ok {
			entry.AlreadyExisted = true
			entry.IsSatisfied = existingEntry.IsSatisfied
			if existingPayloads != nil {
				entry.ResultPayload = existingPayloads[RetrievePayloadOpts{
					Id:         existingEntry.ID,
					InsertedAt: existingEntry.InsertedAt,
					Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYRESULTDATA,
					TenantId:   tenantId,
				}]
			}
		} else {
			entry.AlreadyExisted = false
			entry.CreatedTasks = tasksByMetaIdx[i]
			entry.CreatedDAGs = dagsByMetaIdx[i]
		}

		entries[i] = entry
	}

	// advance log file node cursor and commit
	finalNodeId := baseNodeId + int64(n)
	_, err = r.queries.UpdateLogFile(ctx, tx, sqlcv1.UpdateLogFileParams{
		NodeId:                sqlchelpers.ToBigInt(&finalNodeId),
		Durabletaskid:         task.ID,
		Durabletaskinsertedat: task.InsertedAt,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to update latest node id: %w", err)
	}

	if err := optTx.Commit(ctx); err != nil {
		return nil, err
	}

	return &IngestBulkDurableTaskRunResult{
		Entries:         entries,
		InvocationCount: invocationCount,
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

	childHints := make([]string, 0, len(createdTasks))
	for _, childTask := range createdTasks {
		childHints = append(childHints, childTask.ExternalID.String())
	}
	conditions := ChildTerminalMatchConditions(childHints, "output")

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

func (r *durableEventsRepository) CompleteMemoEntry(ctx context.Context, opts CompleteMemoEntryOpts) error {
	task, err := r.GetTaskByExternalId(ctx, opts.TenantId, opts.TaskExternalId, false)
	if err != nil {
		return fmt.Errorf("failed to get task by external id: %w", err)
	}

	entry, err := r.queries.GetDurableEventLogEntry(ctx, r.pool, sqlcv1.GetDurableEventLogEntryParams{
		Durabletaskid:         task.ID,
		Durabletaskinsertedat: task.InsertedAt,
		Nodeid:                opts.NodeId,
		Branchid:              opts.BranchId,
	})
	if err != nil {
		return fmt.Errorf("failed to get durable event log entry at branch %d node %d: %w", opts.BranchId, opts.NodeId, err)
	}

	if entry.IsSatisfied {
		return nil
	}

	_, err = r.queries.MarkDurableEventLogEntrySatisfied(ctx, r.pool, sqlcv1.MarkDurableEventLogEntrySatisfiedParams{
		Durabletaskid:         task.ID,
		Durabletaskinsertedat: task.InsertedAt,
		Nodeid:                opts.NodeId,
		Branchid:              opts.BranchId,
	})

	if err != nil {
		return fmt.Errorf("failed to mark memo entry as satisfied: %w", err)
	}

	if len(opts.Payload) > 0 {
		err = r.payloadStore.Store(ctx, r.pool, StorePayloadOpts{
			Id:         entry.ID,
			InsertedAt: entry.InsertedAt,
			ExternalId: entry.ExternalID,
			Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYRESULTDATA,
			Payload:    opts.Payload,
			TenantId:   opts.TenantId,
		})

		if err != nil {
			return fmt.Errorf("failed to store memo result payload: %w", err)
		}
	}

	return nil
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

// HACK: sqlc wont correctly typecast to Int8 neatly here so we need to use NULLIF
func int64OrNull(v *int64) int64 {
	if v == nil {
		return -1
	}
	return *v
}
