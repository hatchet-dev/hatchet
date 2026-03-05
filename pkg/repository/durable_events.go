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
	InvocationCount int32

	// Populated for all kinds; callers should iterate entries.
	Entries []IngestDurableTaskEventEntry
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
			Branchid:              opts.BranchId,
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

	nextBranchIdToBranchPoint, err := r.listEventLogBranchPoints(ctx, tx, opts.TenantId, task.ID, task.InsertedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to list log branch points: %w", err)
	}

	if logFile.LatestInvocationCount != opts.InvocationCount {
		// TODO-DURABLE: should evict this invocation if this happens
		return nil, fmt.Errorf("invocation count mismatch: expected %d, got %d, rejecting event write", logFile.LatestInvocationCount, invocationCount)
	}

	type entryMeta struct {
		nodeId         int64
		branchId       int64
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
			nodeId := baseNodeId + 1 + int64(i)
			branchId := resolveBranchForNode(nodeId, logFile.LatestBranchID, nextBranchIdToBranchPoint)

			inputPayload, marshalErr := json.Marshal(triggerOpts)
			if marshalErr != nil {
				return nil, fmt.Errorf("failed to marshal trigger opts: %w", marshalErr)
			}

			idempotencyKey, keyErr := r.createIdempotencyKey(sqlcv1.V1DurableEventLogKindRUN, triggerOpts, nil)
			if keyErr != nil {
				return nil, fmt.Errorf("failed to create idempotency key: %w", keyErr)
			}

			metas[i] = entryMeta{
				nodeId:         nodeId,
				branchId:       branchId,
				kind:           sqlcv1.V1DurableEventLogKindRUN,
				idempotencyKey: idempotencyKey,
				inputPayload:   inputPayload,
				triggerOpts:    triggerOpts,
			}
		}
	case sqlcv1.V1DurableEventLogKindWAITFOR:
		nodeId := baseNodeId + 1
		branchId := resolveBranchForNode(nodeId, logFile.LatestBranchID, nextBranchIdToBranchPoint)

		inputPayload, marshalErr := json.Marshal(opts.WaitForConditions)
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal wait for conditions: %w", marshalErr)
		}

		idempotencyKey, keyErr := r.createIdempotencyKey(sqlcv1.V1DurableEventLogKindWAITFOR, nil, opts.WaitForConditions)
		if keyErr != nil {
			return nil, fmt.Errorf("failed to create idempotency key: %w", keyErr)
		}

		metas = []entryMeta{{
			nodeId:         nodeId,
			branchId:       branchId,
			kind:           sqlcv1.V1DurableEventLogKindWAITFOR,
			idempotencyKey: idempotencyKey,
			inputPayload:   inputPayload,
			waitForConds:   opts.WaitForConditions,
		}}
	case sqlcv1.V1DurableEventLogKindMEMO:
		nodeId := baseNodeId + 1
		branchId := resolveBranchForNode(nodeId, logFile.LatestBranchID, nextBranchIdToBranchPoint)

		var resultPayload []byte
		isSatisfied := false
		if len(opts.Payload) > 0 {
			isSatisfied = true
			resultPayload = opts.Payload
		}

		metas = []entryMeta{{
			nodeId:         nodeId,
			branchId:       branchId,
			kind:           sqlcv1.V1DurableEventLogKindMEMO,
			idempotencyKey: opts.MemoKey,
			isSatisfied:    isSatisfied,
			resultPayload:  resultPayload,
		}}
	default:
		return nil, fmt.Errorf("unsupported durable event log entry kind: %s", opts.Kind)
	}

	n := len(metas)
	entries := make([]IngestDurableTaskEventEntry, n)
	var newTriggerOpts []*WorkflowNameTriggerOpts
	var newTriggerMetaIdxs []int

	for i, m := range metas {
		logEntry, getOrCreateErr := r.getOrCreateEventLogEntry(
			ctx,
			tx,
			GetOrCreateLogEntryOpts{
				TenantId:              opts.TenantId,
				DurableTaskExternalId: task.ExternalID,
				DurableTaskId:         task.ID,
				DurableTaskInsertedAt: task.InsertedAt,
				Kind:                  m.kind,
				NodeId:                m.nodeId,
				BranchId:              m.branchId,
				InvocationCount:       opts.InvocationCount,
				IsSatisfied:           m.isSatisfied,
				IdempotencyKey:        m.idempotencyKey,
				InputPayload:          m.inputPayload,
				ResultPayload:         m.resultPayload,
			},
		)
		if getOrCreateErr != nil {
			return nil, fmt.Errorf("failed to get or create event log entry: %w", getOrCreateErr)
		}

		entries[i] = IngestDurableTaskEventEntry{
			NodeId:         m.nodeId,
			BranchId:       m.branchId,
			IsSatisfied:    logEntry.Entry.IsSatisfied,
			AlreadyExisted: logEntry.AlreadyExisted,
			ResultPayload:  logEntry.ResultPayload,
		}

		if !logEntry.AlreadyExisted && m.kind == sqlcv1.V1DurableEventLogKindRUN {
			newTriggerOpts = append(newTriggerOpts, m.triggerOpts)
			newTriggerMetaIdxs = append(newTriggerMetaIdxs, i)
		}

		if !logEntry.AlreadyExisted && m.kind == sqlcv1.V1DurableEventLogKindWAITFOR {
			if err = r.handleWaitFor(ctx, tx, tenantId, m.branchId, m.nodeId, m.waitForConds, task); err != nil {
				return nil, fmt.Errorf("failed to handle wait for conditions: %w", err)
			}
		}
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
				entries[metaIdx].CreatedTasks = append(entries[metaIdx].CreatedTasks, ct)
			}
		}

		for _, cd := range allCreatedDAGs {
			if metaIdx, ok := triggerExternalIdToMetaIdx[cd.ExternalID]; ok {
				entries[metaIdx].CreatedDAGs = append(entries[metaIdx].CreatedDAGs, cd)
			}
		}

		var allMatchOpts []CreateMatchOpts
		taskId := task.ID
		taskExternalId := task.ExternalID

		for _, metaIdx := range newTriggerMetaIdxs {
			m := metas[metaIdx]
			childTasks := entries[metaIdx].CreatedTasks
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

	return &IngestDurableTaskEventResult{
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
