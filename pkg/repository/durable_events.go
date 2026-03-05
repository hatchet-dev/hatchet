package repository

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

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

	// used when kind = WAIT_FOR
	WaitForConditions []CreateExternalSignalConditionOpt

	// used when kind = RUN: list of triggers to spawn in one transaction
	TriggerOptsList []*WorkflowNameTriggerOpts

	// used when kind = MEMO
	MemoKey []byte
}

type IngestDurableTaskEventEntry struct {
	ResultPayload  []byte
	CreatedTasks   []*V1TaskWithPayload
	CreatedDAGs    []*DAGWithData
	NodeId         int64
	BranchId       int64
	IsSatisfied    bool
	AlreadyExisted bool
}

type IngestDurableTaskEventResult struct {
	// Flat fields populated from Entries[0] for single-event callers (WAIT_FOR, MEMO).
	BranchId        int64
	NodeId          int64
	InvocationCount int32
	IsSatisfied     bool
	ResultPayload   []byte
	AlreadyExisted  bool
	CreatedTasks    []*V1TaskWithPayload
	CreatedDAGs     []*DAGWithData

	// Populated for all kinds; bulk RUN callers should iterate this.
	Entries []IngestDurableTaskEventEntry
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

type DurableEventsRepository interface {
	IngestDurableTaskEvent(ctx context.Context, opts IngestDurableTaskEventOpts) (*IngestDurableTaskEventResult, error)
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

func (r *durableEventsRepository) createIdempotencyKey(kind sqlcv1.V1DurableEventLogKind, triggerOpts *WorkflowNameTriggerOpts, waitForConditions []CreateExternalSignalConditionOpt) ([]byte, error) {
	// TODO-DURABLE: be more intentional about how we construct this key (e.g. do we want to marshal all of the opts?)
	dataToHash := []byte(kind)

	if triggerOpts != nil {
		dataToHash = append(dataToHash, triggerOpts.Data...)
		dataToHash = append(dataToHash, []byte(triggerOpts.WorkflowName)...)
	}

	if waitForConditions != nil {
		sort.Slice(waitForConditions, func(i, j int) bool {
			condI := waitForConditions[i]
			condJ := waitForConditions[j]

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

		for _, cond := range waitForConditions {
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

type nodeAndBranch struct {
	parentNodeId   *int64
	parentBranchId *int64
	nodeId         int64
	branchId       int64
}

func computeNodeAndBranch(logFile *sqlcv1.V1DurableEventLogFile, baseNodeId int64, index int) nodeAndBranch {
	nodeId := baseNodeId + 1 + int64(index)

	var parentNodeId *int64
	if prevNode := baseNodeId + int64(index); prevNode > 0 {
		p := prevNode
		parentNodeId = &p
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

	return nodeAndBranch{
		nodeId:         nodeId,
		branchId:       branchId,
		parentNodeId:   parentNodeId,
		parentBranchId: parentBranchId,
	}
}

func (r *durableEventsRepository) IngestDurableTaskEvent(ctx context.Context, opts IngestDurableTaskEventOpts) (*IngestDurableTaskEventResult, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, fmt.Errorf("invalid opts: %w", err)
	}

	if opts.Kind == sqlcv1.V1DurableEventLogKindRUN && len(opts.TriggerOptsList) == 0 {
		return nil, fmt.Errorf("TriggerOptsList is required and must be non-empty for RUN kind")
	}

	tenantId := opts.TenantId
	task := opts.Task
	invocationCount := opts.InvocationCount

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
		// TODO-DURABLE: should evict this invocation if this happens
		return nil, fmt.Errorf("invocation count mismatch: expected %d, got %d, rejecting event write", logFile.LatestInvocationCount, invocationCount)
	}

	type entryMeta struct {
		nb             nodeAndBranch
		kind           sqlcv1.V1DurableEventLogKind
		idempotencyKey []byte
		inputPayload   []byte
		resultPayload  []byte
		isSatisfied    bool
		triggerOpts    *WorkflowNameTriggerOpts
		waitForConds   []CreateExternalSignalConditionOpt
	}

	baseNodeId := logFile.LatestNodeID
	var metas []entryMeta

	switch opts.Kind {
	case sqlcv1.V1DurableEventLogKindRUN:
		metas = make([]entryMeta, len(opts.TriggerOptsList))
		for i, triggerOpts := range opts.TriggerOptsList {
			nb := computeNodeAndBranch(logFile, baseNodeId, i)

			inputPayload, marshalErr := json.Marshal(triggerOpts)
			if marshalErr != nil {
				return nil, fmt.Errorf("failed to marshal trigger opts: %w", marshalErr)
			}

			idempotencyKey, keyErr := r.createIdempotencyKey(sqlcv1.V1DurableEventLogKindRUN, triggerOpts, nil)
			if keyErr != nil {
				return nil, fmt.Errorf("failed to create idempotency key: %w", keyErr)
			}

			metas[i] = entryMeta{
				nb:             nb,
				kind:           sqlcv1.V1DurableEventLogKindRUN,
				idempotencyKey: idempotencyKey,
				inputPayload:   inputPayload,
				triggerOpts:    triggerOpts,
			}
		}
	case sqlcv1.V1DurableEventLogKindWAITFOR:
		nb := computeNodeAndBranch(logFile, baseNodeId, 0)

		inputPayload, marshalErr := json.Marshal(opts.WaitForConditions)
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal wait for conditions: %w", marshalErr)
		}

		idempotencyKey, keyErr := r.createIdempotencyKey(sqlcv1.V1DurableEventLogKindWAITFOR, nil, opts.WaitForConditions)
		if keyErr != nil {
			return nil, fmt.Errorf("failed to create idempotency key: %w", keyErr)
		}

		metas = []entryMeta{{
			nb:             nb,
			kind:           sqlcv1.V1DurableEventLogKindWAITFOR,
			idempotencyKey: idempotencyKey,
			inputPayload:   inputPayload,
			waitForConds:   opts.WaitForConditions,
		}}
	case sqlcv1.V1DurableEventLogKindMEMO:
		nb := computeNodeAndBranch(logFile, baseNodeId, 0)

		var resultPayload []byte
		isSatisfied := false
		if len(opts.Payload) > 0 {
			isSatisfied = true
			resultPayload = opts.Payload
		}

		metas = []entryMeta{{
			nb:             nb,
			kind:           sqlcv1.V1DurableEventLogKindMEMO,
			idempotencyKey: opts.MemoKey,
			isSatisfied:    isSatisfied,
			resultPayload:  resultPayload,
		}}
	default:
		return nil, fmt.Errorf("unsupported durable event log entry kind: %s", opts.Kind)
	}

	n := len(metas)

	branchIds := make([]int64, n)
	nodeIds := make([]int64, n)
	for i, m := range metas {
		branchIds[i] = m.nb.branchId
		nodeIds[i] = m.nb.nodeId
	}

	existingEntries, err := r.queries.BulkGetDurableEventLogEntries(ctx, tx, sqlcv1.BulkGetDurableEventLogEntriesParams{
		Durabletaskid:         task.ID,
		Durabletaskinsertedat: task.InsertedAt,
		Branchids:             branchIds,
		Nodeids:               nodeIds,
	})
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

	type newEntryInfo struct{ metaIdx int }
	type staleEntryInfo struct {
		entry   *sqlcv1.V1DurableEventLogEntry
		metaIdx int
	}

	var newEntries []newEntryInfo
	var staleEntries []staleEntryInfo
	existedEntries := make(map[int]*sqlcv1.V1DurableEventLogEntry)

	for i, m := range metas {
		key := branchNodeKey{m.nb.branchId, m.nb.nodeId}
		existing, found := existingByKey[key]

		if !found {
			newEntries = append(newEntries, newEntryInfo{metaIdx: i})
			continue
		}

		if !bytes.Equal(m.idempotencyKey, existing.IdempotencyKey) {
			return nil, &NonDeterminismError{
				BranchId:               m.nb.branchId,
				NodeId:                 m.nb.nodeId,
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
			createParams.Kinds[j] = string(m.kind)
			createParams.Nodeids[j] = m.nb.nodeId
			createParams.Parentnodeids[j] = int64OrNull(m.nb.parentNodeId)
			createParams.Branchids[j] = m.nb.branchId
			createParams.Parentbranchids[j] = int64OrNull(m.nb.parentBranchId)
			createParams.Invocationcounts[j] = invocationCount
			createParams.Idempotencykeys[j] = m.idempotencyKey
			createParams.Issatisfieds[j] = m.isSatisfied
		}

		createdRows, createErr := r.queries.BulkCreateDurableEventLogEntries(ctx, tx, createParams)
		if createErr != nil {
			return nil, fmt.Errorf("failed to bulk-create event log entries: %w", createErr)
		}

		createdByNodeId = make(map[int64]*sqlcv1.V1DurableEventLogEntry, len(createdRows))
		for _, row := range createdRows {
			createdByNodeId[row.NodeID] = row
		}

		storePayloadOpts := make([]StorePayloadOpts, 0, len(newEntries)*2)
		for _, ne := range newEntries {
			m := metas[ne.metaIdx]
			created, ok := createdByNodeId[m.nb.nodeId]
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
			if len(m.resultPayload) > 0 {
				storePayloadOpts = append(storePayloadOpts, StorePayloadOpts{
					Id:         created.ID,
					InsertedAt: created.InsertedAt,
					ExternalId: created.ExternalID,
					Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYRESULTDATA,
					Payload:    m.resultPayload,
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
	// TODO-DURABLE: I don't think this should be required or at least should not be handled here...
	// NOTE: entry exists but belongs to a previous invocation (e.g. after eviction+restore
	// or cancel+replay). Idempotency key was already checked above; update invocation_count
	// so callbacks route correctly and reuse existing wait conditions.
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
			updateParams.Branchids[j] = m.nb.branchId
			updateParams.Nodeids[j] = m.nb.nodeId
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
				if row.NodeID == m.nb.nodeId && row.BranchID == m.nb.branchId {
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

	// Side effects for new RUN entries
	tasksByMetaIdx := make(map[int][]*V1TaskWithPayload)
	dagsByMetaIdx := make(map[int][]*DAGWithData)

	var newTriggerOpts []*WorkflowNameTriggerOpts
	var newTriggerMetaIdxs []int

	for _, ne := range newEntries {
		m := metas[ne.metaIdx]
		if m.kind != sqlcv1.V1DurableEventLogKindRUN {
			continue
		}
		if _, created := createdByNodeId[m.nb.nodeId]; !created {
			continue
		}
		newTriggerOpts = append(newTriggerOpts, m.triggerOpts)
		newTriggerMetaIdxs = append(newTriggerMetaIdxs, ne.metaIdx)
	}

	if len(newTriggerOpts) > 0 {
		triggerExternalIdToMetaIdx := make(map[uuid.UUID]int, len(newTriggerOpts))
		for j, tOpts := range newTriggerOpts {
			triggerExternalIdToMetaIdx[tOpts.ExternalId] = newTriggerMetaIdxs[j]
		}

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

			nodeId := m.nb.nodeId
			branchId := m.nb.branchId
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

	// Side effects for new WAIT_FOR entries
	for _, ne := range newEntries {
		m := metas[ne.metaIdx]
		if m.kind != sqlcv1.V1DurableEventLogKindWAITFOR {
			continue
		}
		if _, created := createdByNodeId[m.nb.nodeId]; !created {
			continue
		}
		if err = r.handleWaitFor(ctx, tx, tenantId, m.nb.branchId, m.nb.nodeId, m.waitForConds, task); err != nil {
			return nil, fmt.Errorf("failed to handle wait for conditions: %w", err)
		}
	}

	// NOTE: MEMO has no side effects — it just writes the cache entry and returns

	entries := make([]IngestDurableTaskEventEntry, n)

	for i, m := range metas {
		entry := IngestDurableTaskEventEntry{
			NodeId:   m.nb.nodeId,
			BranchId: m.nb.branchId,
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
			entry.IsSatisfied = m.isSatisfied
			entry.ResultPayload = m.resultPayload
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

	result := &IngestDurableTaskEventResult{
		Entries:         entries,
		InvocationCount: invocationCount,
	}

	if len(entries) > 0 {
		result.NodeId = entries[0].NodeId
		result.BranchId = entries[0].BranchId
		result.IsSatisfied = entries[0].IsSatisfied
		result.AlreadyExisted = entries[0].AlreadyExisted
		result.ResultPayload = entries[0].ResultPayload
		result.CreatedTasks = entries[0].CreatedTasks
		result.CreatedDAGs = entries[0].CreatedDAGs
	}

	return result, nil
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
