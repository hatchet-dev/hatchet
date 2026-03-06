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

type BaseIngestEventOpts struct {
	Kind            sqlcv1.V1DurableEventLogKind  `validate:"required"`
	TenantId        uuid.UUID                     `validate:"required"`
	Task            *sqlcv1.FlattenExternalIdsRow `validate:"required"`
	InvocationCount int32
}

type IngestMemoOpts struct {
	Payload []byte
	MemoKey []byte
}

type IngestTriggerRunsOpts struct {
	Payload     []byte
	TriggerOpts []*WorkflowNameTriggerOpts `validate:"required,min=1"`
}

type IngestWaitForOpts struct {
	WaitForConditions []CreateExternalSignalConditionOpt
}

type IngestDurableTaskEventOpts struct {
	*BaseIngestEventOpts
	Memo        *IngestMemoOpts
	TriggerRuns *IngestTriggerRunsOpts
	WaitFor     *IngestWaitForOpts
}

type IngestMemoResult struct {
	InvocationCount int32
	IsSatisfied     bool
	NodeId          int64
	BranchId        int64
	ResultPayload   []byte
	AlreadyExisted  bool
}

type IngestTriggerRunsEntry struct {
	NodeId         int64
	BranchId       int64
	IsSatisfied    bool
	AlreadyExisted bool
	ResultPayload  []byte
}

type IngestTriggerRunsResult struct {
	InvocationCount int32
	Entries         []*IngestTriggerRunsEntry
	CreatedTasks    []*V1TaskWithPayload
	CreatedDAGs     []*DAGWithData
}

type IngestWaitForResult struct {
	InvocationCount int32
	IsSatisfied     bool
	NodeId          int64
	BranchId        int64
	AlreadyExisted  bool
}

type IngestDurableTaskEventResult struct {
	Kind              sqlcv1.V1DurableEventLogKind
	MemoResult        *IngestMemoResult
	TriggerRunsResult *IngestTriggerRunsResult
	WaitForResult     *IngestWaitForResult
}

type HandleBranchResult struct {
	EventLogFile *sqlcv1.V1DurableEventLogFile
	NodeId       int64
	BranchId     int64
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

type NodeIdBranchIdTuple struct {
	NodeId   int64
	BranchId int64
}

type DurableEventsRepository interface {
	IngestDurableTaskEvent(ctx context.Context, opts IngestDurableTaskEventOpts) (*IngestDurableTaskEventResult, error)
	HandleBranch(ctx context.Context, tenantId uuid.UUID, nodeId, branchId int64, task *sqlcv1.FlattenExternalIdsRow) (*HandleBranchResult, error)

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
	DurableTaskInsertedAt pgtype.Timestamptz
	Kind                  sqlcv1.V1DurableEventLogKind
	IdempotencyKey        []byte
	InputPayload          []byte
	ResultPayload         []byte
	DurableTaskId         int64
	NodeId                int64
	BranchId              int64
	InvocationCount       int32
	TenantId              uuid.UUID
	DurableTaskExternalId uuid.UUID
	IsSatisfied           bool
}

type EventLogEntryWithPayloads struct {
	Entry          *sqlcv1.V1DurableEventLogEntry
	InputPayload   []byte
	ResultPayload  []byte
	AlreadyExisted bool
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
	return r.queries.GetAndLockLogFile(ctx, tx, sqlcv1.GetAndLockLogFileParams{
		Durabletaskid:         durableTaskId,
		Durabletaskinsertedat: durableTaskInsertedAt,
		Tenantid:              tenantId,
	})
}

func (r *durableEventsRepository) listEventLogBranchPoints(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz) (map[int64]*sqlcv1.V1DurableEventLogBranchPoint, error) {
	branchPoints, err := r.queries.ListDurableEventLogBranchPoints(ctx, tx, sqlcv1.ListDurableEventLogBranchPointsParams{
		Durabletaskid:         durableTaskId,
		Durabletaskinsertedat: durableTaskInsertedAt,
		Tenantid:              tenantId,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list durable event log branch points: %w", err)
	}

	nextBranchIdToBranchPoint := make(map[int64]*sqlcv1.V1DurableEventLogBranchPoint, len(branchPoints))

	for _, bp := range branchPoints {
		nextBranchIdToBranchPoint[bp.NextBranchID] = bp
	}

	return nextBranchIdToBranchPoint, nil
}

type BranchIdFromNodeIdTuple struct {
	BranchId   int64
	FromNodeId int64
}

func resolveBranchForNode(nodeId, currentBranchId int64, nextBranchIdToBranchPoint map[int64]*sqlcv1.V1DurableEventLogBranchPoint) int64 {
	tree := make([]BranchIdFromNodeIdTuple, 0)

	currBranchId := currentBranchId
	for {
		branchPoint, found := nextBranchIdToBranchPoint[currBranchId]

		if !found {
			tree = append(tree, BranchIdFromNodeIdTuple{currBranchId, 0})
			break
		}

		tree = append(tree, BranchIdFromNodeIdTuple{currBranchId, branchPoint.FirstNodeIDInNewBranch})
		currBranchId = branchPoint.ParentBranchID
	}

	sort.Slice(tree, func(i, j int) bool {
		if tree[i].FromNodeId != tree[j].FromNodeId {
			return tree[i].FromNodeId < tree[j].FromNodeId
		}
		return tree[i].BranchId < tree[j].BranchId
	})

	i := sort.Search(len(tree), func(i int) bool { return tree[i].FromNodeId > nodeId })
	return tree[i-1].BranchId
}

func (r *durableEventsRepository) getOrCreateEventLogEntries(
	ctx context.Context,
	tx sqlcv1.DBTX,
	opts []GetOrCreateLogEntryOpts,
) ([]*EventLogEntryWithPayloads, error) {
	if len(opts) == 0 {
		return nil, nil
	}

	n := len(opts)
	branchIds := make([]int64, n)
	nodeIds := make([]int64, n)

	for i, o := range opts {
		branchIds[i] = o.BranchId
		nodeIds[i] = o.NodeId
	}

	// Bulk-fetch any existing entries for these (task, branch, node) tuples.
	existingEntries, err := r.queries.BulkGetDurableEventLogEntries(ctx, tx, sqlcv1.BulkGetDurableEventLogEntriesParams{
		Durabletaskid:         opts[0].DurableTaskId,
		Durabletaskinsertedat: opts[0].DurableTaskInsertedAt,
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

	type newEntryInfo struct{ optIdx int }
	type staleEntryInfo struct {
		entry  *sqlcv1.V1DurableEventLogEntry
		optIdx int
	}

	var newEntries []newEntryInfo
	var staleEntries []staleEntryInfo
	existedEntries := make(map[int]*sqlcv1.V1DurableEventLogEntry)

	for i, o := range opts {
		key := branchNodeKey{o.BranchId, o.NodeId}
		existing, found := existingByKey[key]

		if !found {
			newEntries = append(newEntries, newEntryInfo{optIdx: i})
			continue
		}

		if !bytes.Equal(o.IdempotencyKey, existing.IdempotencyKey) {
			return nil, &NonDeterminismError{
				BranchId:               o.BranchId,
				NodeId:                 o.NodeId,
				TaskExternalId:         o.DurableTaskExternalId,
				ExpectedIdempotencyKey: existing.IdempotencyKey,
				ActualIdempotencyKey:   o.IdempotencyKey,
			}
		}

		if existing.InvocationCount != o.InvocationCount {
			staleEntries = append(staleEntries, staleEntryInfo{optIdx: i, entry: existing})
		} else {
			existedEntries[i] = existing
		}
	}

	// Bulk-create entries that don't exist yet.
	var createdByNodeId map[int64]*sqlcv1.V1DurableEventLogEntry

	if len(newEntries) > 0 {
		createParams := sqlcv1.BulkCreateDurableEventLogEntriesParams{
			Tenantids:              make([]uuid.UUID, len(newEntries)),
			Externalids:            make([]uuid.UUID, len(newEntries)),
			Durabletaskids:         make([]int64, len(newEntries)),
			Durabletaskinsertedats: make([]pgtype.Timestamptz, len(newEntries)),
			Kinds:                  make([]string, len(newEntries)),
			Nodeids:                make([]int64, len(newEntries)),
			Branchids:              make([]int64, len(newEntries)),
			Invocationcounts:       make([]int32, len(newEntries)),
			Idempotencykeys:        make([][]byte, len(newEntries)),
			Issatisfieds:           make([]bool, len(newEntries)),
		}

		for j, ne := range newEntries {
			o := opts[ne.optIdx]
			createParams.Tenantids[j] = o.TenantId
			createParams.Externalids[j] = uuid.New()
			createParams.Durabletaskids[j] = o.DurableTaskId
			createParams.Durabletaskinsertedats[j] = o.DurableTaskInsertedAt
			createParams.Kinds[j] = string(o.Kind)
			createParams.Nodeids[j] = o.NodeId
			createParams.Branchids[j] = o.BranchId
			createParams.Invocationcounts[j] = o.InvocationCount
			createParams.Idempotencykeys[j] = o.IdempotencyKey
			createParams.Issatisfieds[j] = o.IsSatisfied
		}

		createdRows, createErr := r.queries.BulkCreateDurableEventLogEntries(ctx, tx, createParams)
		if createErr != nil {
			return nil, fmt.Errorf("failed to bulk-create event log entries: %w", createErr)
		}

		createdByNodeId = make(map[int64]*sqlcv1.V1DurableEventLogEntry, len(createdRows))
		for _, row := range createdRows {
			createdByNodeId[row.NodeID] = row
		}

		// Store input and result payloads for newly created entries.
		storePayloadOpts := make([]StorePayloadOpts, 0, len(newEntries)*2)
		for _, ne := range newEntries {
			o := opts[ne.optIdx]
			created, ok := createdByNodeId[o.NodeId]
			if !ok {
				continue
			}
			if len(o.InputPayload) > 0 {
				storePayloadOpts = append(storePayloadOpts, StorePayloadOpts{
					Id:         created.ID,
					InsertedAt: created.InsertedAt,
					ExternalId: created.ExternalID,
					Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYDATA,
					Payload:    o.InputPayload,
					TenantId:   o.TenantId,
				})
			}
			if len(o.ResultPayload) > 0 {
				storePayloadOpts = append(storePayloadOpts, StorePayloadOpts{
					Id:         created.ID,
					InsertedAt: created.InsertedAt,
					ExternalId: created.ExternalID,
					Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYRESULTDATA,
					Payload:    o.ResultPayload,
					TenantId:   o.TenantId,
				})
			}
		}

		if len(storePayloadOpts) > 0 {
			if storeErr := r.payloadStore.Store(ctx, tx, storePayloadOpts...); storeErr != nil {
				return nil, fmt.Errorf("failed to store payloads for new entries: %w", storeErr)
			}
		}
	}

	// Stale entries (mismatched invocation count) are treated as already-existed.
	for _, se := range staleEntries {
		existedEntries[se.optIdx] = se.entry
	}

	// Retrieve result payloads for entries that already existed.
	var retrieveOpts []RetrievePayloadOpts
	for _, entry := range existedEntries {
		retrieveOpts = append(retrieveOpts, RetrievePayloadOpts{
			Id:         entry.ID,
			InsertedAt: entry.InsertedAt,
			Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYRESULTDATA,
			TenantId:   opts[0].TenantId,
		})
	}

	var existingPayloads map[RetrievePayloadOpts][]byte
	if len(retrieveOpts) > 0 {
		existingPayloads, err = r.payloadStore.Retrieve(ctx, tx, retrieveOpts...)
		if err != nil {
			existingPayloads = nil
		}
	}

	// Build results, one per input opt, preserving order.
	results := make([]*EventLogEntryWithPayloads, n)
	for i, o := range opts {
		if existingEntry, ok := existedEntries[i]; ok {
			var resultPayload []byte
			if existingPayloads != nil {
				resultPayload = existingPayloads[RetrievePayloadOpts{
					Id:         existingEntry.ID,
					InsertedAt: existingEntry.InsertedAt,
					Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYRESULTDATA,
					TenantId:   o.TenantId,
				}]
			}
			results[i] = &EventLogEntryWithPayloads{
				Entry:          existingEntry,
				InputPayload:   o.InputPayload,
				ResultPayload:  resultPayload,
				AlreadyExisted: true,
			}
		} else {
			created := createdByNodeId[o.NodeId]
			results[i] = &EventLogEntryWithPayloads{
				Entry:          created,
				InputPayload:   o.InputPayload,
				ResultPayload:  o.ResultPayload,
				AlreadyExisted: false,
			}
		}
	}

	return results, nil
}

func resolveIsSatisfied(le *EventLogEntryWithPayloads, o GetOrCreateLogEntryOpts) bool {
	if le.AlreadyExisted {
		return le.Entry.IsSatisfied
	}

	return o.IsSatisfied
}

func (r *durableEventsRepository) IngestDurableTaskEvent(ctx context.Context, opts IngestDurableTaskEventOpts) (*IngestDurableTaskEventResult, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, fmt.Errorf("invalid opts: %w", err)
	}

	if opts.Kind == sqlcv1.V1DurableEventLogKindRUN && len(opts.TriggerRuns.TriggerOpts) == 0 {
		return nil, fmt.Errorf("TriggerOptsList is required and must be non-empty for RUN kind")
	}

	tenantId := opts.TenantId
	task := opts.Task

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

	nextBranchIdToBranchPoint, err := r.listEventLogBranchPoints(ctx, tx, tenantId, task.ID, task.InsertedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to list log branch points: %w", err)
	}

	if logFile.LatestInvocationCount != opts.InvocationCount {
		// TODO-DURABLE: should evict this invocation if this happens
		return nil, fmt.Errorf("invocation count mismatch: expected %d, got %d, rejecting event write", logFile.LatestInvocationCount, opts.InvocationCount)
	}

	baseNodeId := logFile.LatestNodeID + 1

	var getOrCreateOpts []GetOrCreateLogEntryOpts

	nodeIdBranchIdToTriggerOpts := make(map[NodeIdBranchIdTuple]*WorkflowNameTriggerOpts)
	runExternalIdToNodeIdBranchId := make(map[uuid.UUID]NodeIdBranchIdTuple)

	switch opts.Kind {
	case sqlcv1.V1DurableEventLogKindRUN:
		getOrCreateOpts = make([]GetOrCreateLogEntryOpts, len(opts.TriggerRuns.TriggerOpts))

		for i, triggerOpts := range opts.TriggerRuns.TriggerOpts {
			nodeId := baseNodeId + int64(i)
			branchId := resolveBranchForNode(nodeId, logFile.LatestBranchID, nextBranchIdToBranchPoint)

			inputPayload, marshalErr := json.Marshal(triggerOpts)
			if marshalErr != nil {
				return nil, fmt.Errorf("failed to marshal trigger opts: %w", marshalErr)
			}

			idempotencyKey, keyErr := r.createIdempotencyKey(sqlcv1.V1DurableEventLogKindRUN, triggerOpts, nil)
			if keyErr != nil {
				return nil, fmt.Errorf("failed to create idempotency key: %w", keyErr)
			}

			getOrCreateOpts[i] = GetOrCreateLogEntryOpts{
				TenantId:              tenantId,
				DurableTaskExternalId: task.ExternalID,
				DurableTaskId:         task.ID,
				DurableTaskInsertedAt: task.InsertedAt,
				Kind:                  sqlcv1.V1DurableEventLogKindRUN,
				NodeId:                nodeId,
				BranchId:              branchId,
				InvocationCount:       opts.InvocationCount,
				IdempotencyKey:        idempotencyKey,
				InputPayload:          inputPayload,
			}

			nodeBranchKey := NodeIdBranchIdTuple{NodeId: nodeId, BranchId: branchId}
			nodeIdBranchIdToTriggerOpts[nodeBranchKey] = triggerOpts
			runExternalIdToNodeIdBranchId[triggerOpts.ExternalId] = nodeBranchKey
		}
	case sqlcv1.V1DurableEventLogKindWAITFOR:
		branchId := resolveBranchForNode(baseNodeId, logFile.LatestBranchID, nextBranchIdToBranchPoint)

		inputPayload, marshalErr := json.Marshal(opts.WaitFor.WaitForConditions)
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal wait for conditions: %w", marshalErr)
		}

		idempotencyKey, keyErr := r.createIdempotencyKey(sqlcv1.V1DurableEventLogKindWAITFOR, nil, opts.WaitFor.WaitForConditions)
		if keyErr != nil {
			return nil, fmt.Errorf("failed to create idempotency key: %w", keyErr)
		}

		getOrCreateOpts = []GetOrCreateLogEntryOpts{{
			TenantId:              tenantId,
			DurableTaskExternalId: task.ExternalID,
			DurableTaskId:         task.ID,
			DurableTaskInsertedAt: task.InsertedAt,
			Kind:                  sqlcv1.V1DurableEventLogKindWAITFOR,
			NodeId:                baseNodeId,
			BranchId:              branchId,
			InvocationCount:       opts.InvocationCount,
			IdempotencyKey:        idempotencyKey,
			InputPayload:          inputPayload,
		}}
	case sqlcv1.V1DurableEventLogKindMEMO:
		branchId := resolveBranchForNode(baseNodeId, logFile.LatestBranchID, nextBranchIdToBranchPoint)

		var resultPayload []byte
		isSatisfied := false
		if len(opts.Memo.Payload) > 0 {
			isSatisfied = true
			resultPayload = opts.Memo.Payload
		}

		getOrCreateOpts = []GetOrCreateLogEntryOpts{{
			TenantId:              tenantId,
			DurableTaskExternalId: task.ExternalID,
			DurableTaskId:         task.ID,
			DurableTaskInsertedAt: task.InsertedAt,
			Kind:                  sqlcv1.V1DurableEventLogKindMEMO,
			NodeId:                baseNodeId,
			BranchId:              branchId,
			InvocationCount:       opts.InvocationCount,
			IdempotencyKey:        opts.Memo.MemoKey,
			IsSatisfied:           isSatisfied,
			ResultPayload:         resultPayload,
		}}
	default:
		return nil, fmt.Errorf("unsupported durable event log entry kind: %s", opts.Kind)
	}

	// todo: probably need to return a map from node + branch to entry
	// or ensure these are sorted
	logEntries, err := r.getOrCreateEventLogEntries(ctx, tx, getOrCreateOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create event log entries: %w", err)
	}

	var memoResult *IngestMemoResult
	var waitForResult *IngestWaitForResult
	var triggerRunsResult *IngestTriggerRunsResult

	switch opts.Kind {
	case sqlcv1.V1DurableEventLogKindRUN:
		entries := make([]*IngestTriggerRunsEntry, len(getOrCreateOpts))

		for i := range getOrCreateOpts {
			le := logEntries[i]
			o := getOrCreateOpts[i]

			entries[i] = &IngestTriggerRunsEntry{
				NodeId:         o.NodeId,
				BranchId:       o.BranchId,
				IsSatisfied:    resolveIsSatisfied(le, o),
				AlreadyExisted: le.AlreadyExisted,
				ResultPayload:  le.ResultPayload,
			}
		}

		triggerRunsResult = &IngestTriggerRunsResult{
			InvocationCount: opts.InvocationCount,
			Entries:         entries,
		}

		var newTriggerOpts []*WorkflowNameTriggerOpts

		for _, le := range logEntries {
			if le.AlreadyExisted {
				continue
			}

			newTriggerOpts = append(newTriggerOpts, nodeIdBranchIdToTriggerOpts[NodeIdBranchIdTuple{
				NodeId:   le.Entry.NodeID,
				BranchId: le.Entry.BranchID,
			}])
		}

		if len(newTriggerOpts) > 0 {
			createdTasks, createdDags, triggerErr := r.triggerFromWorkflowNames(ctx, optTx, tenantId, newTriggerOpts)

			if triggerErr != nil {
				return nil, fmt.Errorf("failed to trigger workflows: %w", triggerErr)
			}

			triggerRunsResult.CreatedTasks = createdTasks
			triggerRunsResult.CreatedDAGs = createdDags

			childTaskExternalIds := make([]uuid.UUID, len(createdTasks))
			createMatchOpts := make([]CreateMatchOpts, 0, len(createdTasks))

			for _, task := range createdTasks {
				childTaskExternalIds = append(childTaskExternalIds, task.ExternalID)
			}

			for _, childTask := range createdTasks {
				childHint := childTask.ExternalID.String()
				orGroupId := uuid.New()

				conditions := []GroupMatchCondition{
					{
						GroupId:           orGroupId,
						EventType:         sqlcv1.V1EventTypeINTERNAL,
						EventKey:          string(sqlcv1.V1TaskEventTypeCOMPLETED),
						ReadableDataKey:   "output",
						EventResourceHint: &childHint,
						Expression:        "true",
						Action:            sqlcv1.V1MatchConditionActionCREATE,
					},
					{
						GroupId:           orGroupId,
						EventType:         sqlcv1.V1EventTypeINTERNAL,
						EventKey:          string(sqlcv1.V1TaskEventTypeFAILED),
						ReadableDataKey:   "output",
						EventResourceHint: &childHint,
						Expression:        "true",
						Action:            sqlcv1.V1MatchConditionActionCREATE,
					},
					{
						GroupId:           orGroupId,
						EventType:         sqlcv1.V1EventTypeINTERNAL,
						EventKey:          string(sqlcv1.V1TaskEventTypeCANCELLED),
						ReadableDataKey:   "output",
						EventResourceHint: &childHint,
						Expression:        "true",
						Action:            sqlcv1.V1MatchConditionActionCREATE,
					},
				}

				nodeIdBranchId := runExternalIdToNodeIdBranchId[childTask.ExternalID]

				nodeId := nodeIdBranchId.NodeId
				branchId := nodeIdBranchId.BranchId

				runEventLogEntrySignalKey := fmt.Sprintf("durable_run:%s:%d:%d", task.ExternalID.String(), branchId, nodeId)

				createMatchOpts = append(createMatchOpts, CreateMatchOpts{
					Kind:                         sqlcv1.V1MatchKindSIGNAL,
					Conditions:                   conditions,
					SignalTaskId:                 &childTask.ID,
					SignalTaskInsertedAt:         task.InsertedAt,
					SignalExternalId:             &childTask.ExternalID,
					SignalTaskExternalId:         &childTask.ExternalID,
					SignalKey:                    &runEventLogEntrySignalKey,
					DurableEventLogEntryNodeId:   &nodeId,
					DurableEventLogEntryBranchId: &branchId,
				})
			}

			if len(createMatchOpts) > 0 {
				if matchErr := r.createEventMatches(ctx, tx, tenantId, createMatchOpts); matchErr != nil {
					return nil, fmt.Errorf("failed to register run completion matches: %w", matchErr)
				}
			}
		}
	case sqlcv1.V1DurableEventLogKindWAITFOR:
		// TODO-DURABLE: Figure out what to do here if we read more than one row out of the db
		le := logEntries[0]
		o := getOrCreateOpts[0]

		if !le.AlreadyExisted {
			if err = r.handleWaitFor(ctx, tx, tenantId, o.BranchId, o.NodeId, opts.WaitFor.WaitForConditions, task); err != nil {
				return nil, fmt.Errorf("failed to handle wait for conditions: %w", err)
			}
		}

		waitForResult = &IngestWaitForResult{
			InvocationCount: opts.InvocationCount,
			IsSatisfied:     resolveIsSatisfied(le, o),
			NodeId:          o.NodeId,
			BranchId:        o.BranchId,
			AlreadyExisted:  le.AlreadyExisted,
		}
	case sqlcv1.V1DurableEventLogKindMEMO:
		// TODO-DURABLE: Figure out what to do here if we read more than one row out of the db
		le := logEntries[0]
		o := getOrCreateOpts[0]

		memoResult = &IngestMemoResult{
			InvocationCount: opts.InvocationCount,
			IsSatisfied:     resolveIsSatisfied(le, o),
			NodeId:          o.NodeId,
			BranchId:        o.BranchId,
			ResultPayload:   le.ResultPayload,
			AlreadyExisted:  le.AlreadyExisted,
		}
	}

	n := len(getOrCreateOpts)

	finalNodeId := baseNodeId + int64(n) - 1
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

	return &IngestDurableTaskEventResult{
		Kind:              opts.Kind,
		MemoResult:        memoResult,
		WaitForResult:     waitForResult,
		TriggerRunsResult: triggerRunsResult,
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

func (r *durableEventsRepository) HandleBranch(ctx context.Context, tenantId uuid.UUID, nodeId, branchId int64, task *sqlcv1.FlattenExternalIdsRow) (*HandleBranchResult, error) {
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
	zero := int64(0)

	logFile, err = r.queries.UpdateLogFile(ctx, tx, sqlcv1.UpdateLogFileParams{
		BranchId:              sqlchelpers.ToBigInt(&newBranchId),
		NodeId:                sqlchelpers.ToBigInt(&zero),
		Durabletaskid:         task.ID,
		Durabletaskinsertedat: task.InsertedAt,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to update log file for branch: %w", err)
	}

	err = r.queries.CreateDurableEventLogBranchPoint(ctx, tx, sqlcv1.CreateDurableEventLogBranchPointParams{
		Tenantid:               tenantId,
		Firstnodeidinnewbranch: nodeId,
		Parentbranchid:         branchId,
		Nextbranchid:           newBranchId,
		Durabletaskid:          task.ID,
		Durabletaskinsertedat:  task.InsertedAt,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create log branch point for fork: %w", err)
	}

	if err := optTx.Commit(ctx); err != nil {
		return nil, err
	}

	return &HandleBranchResult{
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
