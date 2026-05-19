package repository

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

type TaskExternalIdNodeIdBranchId struct {
	TaskExternalId uuid.UUID `validate:"required"`
	NodeId         int64     `validate:"required"`
	BranchId       int64     `validate:"required"`
}

type SatisfiedEventWithPayload struct {
	Result          []byte
	BranchID        int64
	NodeID          int64
	InvocationCount int32
	TaskExternalId  uuid.UUID
}

type BaseIngestEventOpts struct {
	Task            *sqlcv1.FlattenExternalIdsRow `validate:"required"`
	Kind            sqlcv1.V1DurableEventLogKind  `validate:"required"`
	InvocationCount int32
	TenantId        uuid.UUID `validate:"required"`
}

type IngestMemoOpts struct {
	Payload []byte
	MemoKey []byte
}

type IngestTriggerRunsOpts struct {
	TriggerOpts []*WorkflowNameTriggerOpts `validate:"required,min=1"`
}

type IngestWaitForOpts struct {
	WaitForConditions []CreateExternalSignalConditionOpt
	// Label is an optional human-readable message to display for this wait (e.g. "Waiting for payment confirmation").
	Label *string
}

type IngestDurableTaskEventOpts struct {
	*BaseIngestEventOpts
	Memo        *IngestMemoOpts
	TriggerRuns *IngestTriggerRunsOpts
	WaitFor     *IngestWaitForOpts
}

type IngestMemoResult struct {
	ResultPayload   []byte
	NodeId          int64
	BranchId        int64
	InvocationCount int32
	IsSatisfied     bool
	AlreadyExisted  bool
}

type IngestTriggerRunsEntry struct {
	ResultPayload         []byte
	NodeId                int64
	BranchId              int64
	WorkflowRunExternalId uuid.UUID
	IsSatisfied           bool
	AlreadyExisted        bool
}

type IngestTriggerRunsResult struct {
	Entries         []*IngestTriggerRunsEntry
	CreatedTasks    []*V1TaskWithPayload
	CreatedDAGs     []*DAGWithData
	InvocationCount int32
}

type IngestWaitForResult struct {
	ResultPayload   []byte
	NodeId          int64
	BranchId        int64
	InvocationCount int32
	IsSatisfied     bool
	AlreadyExisted  bool
}

type IngestDurableTaskEventResult struct {
	MemoResult        *IngestMemoResult
	TriggerRunsResult *IngestTriggerRunsResult
	WaitForResult     *IngestWaitForResult
	Kind              sqlcv1.V1DurableEventLogKind
}

type HandleBranchResult struct {
	EventLogFile *sqlcv1.V1DurableEventLogFile
	NodeId       int64
	BranchId     int64
}

type IncrementDurableTaskInvocationCountsOpts struct {
	TaskInsertedAt pgtype.Timestamptz
	TaskId         int64
	TenantId       uuid.UUID
}

type CompleteMemoEntryOpts struct {
	MemoKey         []byte
	Payload         []byte
	BranchId        int64
	NodeId          int64
	InvocationCount int32
	TenantId        uuid.UUID
	TaskExternalId  uuid.UUID
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
	ListDurableEventLog(ctx context.Context, tenantId uuid.UUID, taskInsertedAt pgtype.Timestamptz, taskId, limit, offset int64) ([]*sqlcv1.ListDurableEventLogForTaskRow, error)
}

type durableEventsRepository struct {
	*sharedRepository
}

func newDurableEventsRepository(shared *sharedRepository) DurableEventsRepository {
	return &durableEventsRepository{
		sharedRepository: shared,
	}
}

type NonDeterminismDetail struct {
	Expected string
	Received string
}

type NonDeterminismError struct {
	Detail                  *NonDeterminismDetail
	ExistingEntryInsertedAt pgtype.Timestamptz
	ExpectedKind            sqlcv1.V1DurableEventLogKind
	ActualKind              sqlcv1.V1DurableEventLogKind
	ExpectedIdempotencyKey  []byte
	ActualIdempotencyKey    []byte
	NodeId                  int64
	BranchId                int64
	ExistingEntryId         int64
	TaskExternalId          uuid.UUID
	ExistingEntryTenantId   uuid.UUID
}

func (m *NonDeterminismError) Error() string {
	msg := fmt.Sprintf("non-determinism error in task %s at node %d:%d", m.TaskExternalId, m.NodeId, m.BranchId)

	if m.Detail != nil {
		msg += "\n  expected: " + m.Detail.Expected + "\n  received: " + m.Detail.Received
	}

	return msg
}

type StaleInvocationError struct {
	TaskExternalId          uuid.UUID
	ExpectedInvocationCount int32
	ActualInvocationCount   int32
}

func (e *StaleInvocationError) Error() string {
	return fmt.Sprintf("invocation count mismatch for task %s: server has %d, worker sent %d", e.TaskExternalId.String(), e.ExpectedInvocationCount, e.ActualInvocationCount)
}

type DurableWaitConditionKind string

const (
	DurableWaitConditionKindSleep         DurableWaitConditionKind = "SLEEP"
	DurableWaitConditionKindUserEvent     DurableWaitConditionKind = "USER_EVENT"
	DurableWaitConditionKindChildWorkflow DurableWaitConditionKind = "CHILD_WORKFLOW"
)

type DurableWaitCondition struct {
	Kind            DurableWaitConditionKind `json:"kind"`
	SleepDurationMs *int64                   `json:"sleepDurationMs,omitempty"`
	EventKey        *string                  `json:"eventKey,omitempty"`
	WorkflowName    *string                  `json:"workflowName,omitempty"`
}

type DurableWaitOrGroup struct {
	Conditions []DurableWaitCondition `json:"conditions"`
}

type WaitData struct {
	Conditions []DurableWaitCondition `json:"conditions,omitempty"`
	OrGroups   []DurableWaitOrGroup   `json:"orGroups,omitempty"`
}

func parseDurationMs(s *string) *int64 {
	if s == nil {
		return nil
	}

	d, err := time.ParseDuration(*s)
	if err != nil {
		return nil
	}

	ms := d.Milliseconds()
	return &ms
}

func describeCondition(c DurableWaitCondition) string {
	switch c.Kind {
	case DurableWaitConditionKindSleep:
		if c.SleepDurationMs != nil {
			sleepDurationMs := time.Duration(*c.SleepDurationMs) * time.Millisecond
			return "sleep(" + sleepDurationMs.String() + ")"
		}
		return "sleep"
	case DurableWaitConditionKindUserEvent:
		if c.EventKey != nil {
			return "event(" + *c.EventKey + ")"
		}
		return "event"
	case DurableWaitConditionKindChildWorkflow:
		if c.WorkflowName != nil {
			return "run(" + *c.WorkflowName + ")"
		}
		return "run"
	default:
		return string(c.Kind)
	}
}

func describeOrGroup(group DurableWaitOrGroup) string {
	if len(group.Conditions) == 0 {
		return "waiting"
	}

	parts := make([]string, 0, len(group.Conditions))
	for _, c := range group.Conditions {
		parts = append(parts, describeCondition(c))
	}

	if len(parts) == 1 {
		return parts[0]
	}

	return "any of: " + strings.Join(parts, ", ")
}

func (w *WaitData) ToReadableMessage() string {
	if w == nil || (len(w.Conditions) == 0 && len(w.OrGroups) == 0) {
		return "waiting"
	}

	parts := make([]string, 0, len(w.Conditions)+len(w.OrGroups))

	for _, c := range w.Conditions {
		parts = append(parts, describeCondition(c))
	}

	for _, group := range w.OrGroups {
		parts = append(parts, describeOrGroup(group))
	}

	if len(parts) == 1 {
		return parts[0]
	}

	return strings.Join(parts, " and ")
}

func waitDataFromWaitForConditions(conditions []CreateExternalSignalConditionOpt) *WaitData {
	if len(conditions) == 0 {
		return nil
	}

	groupOrder := make([]uuid.UUID, 0, len(conditions))
	seen := make(map[uuid.UUID]struct{}, len(conditions))
	groups := make(map[uuid.UUID][]DurableWaitCondition, len(conditions))

	for _, c := range conditions {
		if _, exists := seen[c.OrGroupId]; !exists {
			groupOrder = append(groupOrder, c.OrGroupId)
			seen[c.OrGroupId] = struct{}{}
		}

		var cond DurableWaitCondition
		switch c.Kind {
		case CreateExternalSignalConditionKindSLEEP:
			cond = DurableWaitCondition{Kind: DurableWaitConditionKindSleep, SleepDurationMs: parseDurationMs(c.SleepFor)}
		case CreateExternalSignalConditionKindUSEREVENT:
			cond = DurableWaitCondition{Kind: DurableWaitConditionKindUserEvent, EventKey: c.UserEventKey}
		default:
			continue
		}

		groups[c.OrGroupId] = append(groups[c.OrGroupId], cond)
	}

	var standalone []DurableWaitCondition
	var orGroups []DurableWaitOrGroup

	for _, id := range groupOrder {
		g := groups[id]
		if len(g) == 1 {
			standalone = append(standalone, g[0])
		} else if len(g) > 1 {
			orGroups = append(orGroups, DurableWaitOrGroup{Conditions: g})
		}
	}

	return &WaitData{Conditions: standalone, OrGroups: orGroups}
}

func waitDataFromTriggerOpt(triggerOpt *WorkflowNameTriggerOpts) *WaitData {
	if triggerOpt == nil {
		return nil
	}

	name := triggerOpt.WorkflowName
	return &WaitData{
		Conditions: []DurableWaitCondition{{Kind: DurableWaitConditionKindChildWorkflow, WorkflowName: &name}},
	}
}

func marshalWaitData(wd *WaitData) string {
	if wd == nil {
		return ""
	}

	b, err := json.Marshal(wd)
	if err != nil {
		return ""
	}

	return string(b)
}

func (opts IngestDurableTaskEventOpts) formatCall() string {
	switch opts.Kind {
	case sqlcv1.V1DurableEventLogKindRUN:
		if opts.TriggerRuns != nil {
			names := make([]string, 0, len(opts.TriggerRuns.TriggerOpts))
			for _, t := range opts.TriggerRuns.TriggerOpts {
				names = append(names, t.WorkflowName)
			}
			return "run(" + strings.Join(names, ", ") + ")"
		}
	case sqlcv1.V1DurableEventLogKindWAITFOR:
		if opts.WaitFor != nil {
			wd := waitDataFromWaitForConditions(opts.WaitFor.WaitForConditions)
			if wd != nil {
				return wd.ToReadableMessage()
			}
		}
	case sqlcv1.V1DurableEventLogKindMEMO:
		return "memo"
	}

	return string(opts.Kind)
}

func formatStoredPayload(kind sqlcv1.V1DurableEventLogKind, payload []byte) string {
	if len(payload) == 0 {
		return string(kind)
	}

	switch kind {
	case sqlcv1.V1DurableEventLogKindRUN:
		var triggerOpts WorkflowNameTriggerOpts

		if err := json.Unmarshal(payload, &triggerOpts); err != nil {
			return string(kind)
		}

		if triggerOpts.WorkflowName != "" {
			return "run(" + triggerOpts.WorkflowName + ")"
		}
	case sqlcv1.V1DurableEventLogKindWAITFOR:
		var conditions []CreateExternalSignalConditionOpt

		if err := json.Unmarshal(payload, &conditions); err != nil {
			return string(kind)
		}

		if len(conditions) > 0 {
			wd := waitDataFromWaitForConditions(conditions)
			if wd != nil {
				return wd.ToReadableMessage()
			}
		}
	case sqlcv1.V1DurableEventLogKindMEMO:
		return "memo"
	}

	return string(kind)
}

func nonDeterminismDetail(opts IngestDurableTaskEventOpts, expectedKind sqlcv1.V1DurableEventLogKind, existingPayload []byte) *NonDeterminismDetail {
	return &NonDeterminismDetail{
		Expected: formatStoredPayload(expectedKind, existingPayload),
		Received: opts.formatCall(),
	}
}

type GetOrCreateLogEntryOpt struct {
	Kind            sqlcv1.V1DurableEventLogKind
	IdempotencyKey  []byte
	InputPayload    []byte
	ResultPayload   []byte
	NodeId          int64
	BranchId        int64
	InvocationCount int32
	IsSatisfied     bool
	UserMessage     *string
	WaitData        string // JSON-encoded WaitData, empty string means no wait data
	SatisfiedAt     *time.Time
}

type GetOrCreateLogEntryOpts struct {
	DurableTaskInsertedAt pgtype.Timestamptz
	Entries               []GetOrCreateLogEntryOpt
	DurableTaskId         int64
	TenantId              uuid.UUID
	DurableTaskExternalId uuid.UUID
}

type EventLogEntryWithPayloads struct {
	Entry          *sqlcv1.BulkGetDurableEventLogEntriesRow
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
	// note: can't use additional metadata here because it's not stable, since we store trace information in it w/ the otel instrumentors
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
	opts GetOrCreateLogEntryOpts,
) ([]*EventLogEntryWithPayloads, error) {
	if len(opts.Entries) == 0 {
		return nil, nil
	}

	n := len(opts.Entries)
	branchIds := make([]int64, n)
	nodeIds := make([]int64, n)

	for i, o := range opts.Entries {
		branchIds[i] = o.BranchId
		nodeIds[i] = o.NodeId
	}

	existingEntries, err := r.queries.BulkGetDurableEventLogEntries(ctx, tx, sqlcv1.BulkGetDurableEventLogEntriesParams{
		Durabletaskid:         opts.DurableTaskId,
		Durabletaskinsertedat: opts.DurableTaskInsertedAt,
		Branchids:             branchIds,
		Nodeids:               nodeIds,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to bulk-get existing entries: %w", err)
	}

	nodeIdBranchIdToExistingEntry := make(map[NodeIdBranchIdTuple]*sqlcv1.BulkGetDurableEventLogEntriesRow, len(existingEntries))
	for _, e := range existingEntries {
		nodeIdBranchIdToExistingEntry[NodeIdBranchIdTuple{e.NodeID, e.BranchID}] = e
	}

	existedEntries := make(map[NodeIdBranchIdTuple]*sqlcv1.BulkGetDurableEventLogEntriesRow)
	nodeIdBranchIdToNewEntry := make(map[NodeIdBranchIdTuple]GetOrCreateLogEntryOpt, 0)

	for _, o := range opts.Entries {
		key := NodeIdBranchIdTuple{o.NodeId, o.BranchId}
		existingEntry, found := nodeIdBranchIdToExistingEntry[key]

		if !found {
			nodeIdBranchIdToNewEntry[key] = o
			continue
		}

		if !bytes.Equal(o.IdempotencyKey, existingEntry.IdempotencyKey) {
			return nil, &NonDeterminismError{
				BranchId:                o.BranchId,
				NodeId:                  o.NodeId,
				TaskExternalId:          opts.DurableTaskExternalId,
				ExpectedIdempotencyKey:  existingEntry.IdempotencyKey,
				ActualIdempotencyKey:    o.IdempotencyKey,
				ExpectedKind:            existingEntry.Kind,
				ActualKind:              o.Kind,
				ExistingEntryId:         existingEntry.ID,
				ExistingEntryInsertedAt: existingEntry.InsertedAt,
				ExistingEntryTenantId:   existingEntry.TenantID,
			}
		}

		existedEntries[key] = existingEntry
	}

	nodeIdBranchIdToCreatedEntry := make(map[NodeIdBranchIdTuple]*sqlcv1.BulkCreateDurableEventLogEntriesRow)

	if len(nodeIdBranchIdToNewEntry) > 0 {
		createParams := sqlcv1.BulkCreateDurableEventLogEntriesParams{
			Tenantids:              make([]uuid.UUID, 0),
			Externalids:            make([]uuid.UUID, 0),
			Durabletaskids:         make([]int64, 0),
			Durabletaskinsertedats: make([]pgtype.Timestamptz, 0),
			Kinds:                  make([]string, 0),
			Nodeids:                make([]int64, 0),
			Branchids:              make([]int64, 0),
			Idempotencykeys:        make([][]byte, 0),
			Issatisfieds:           make([]bool, 0),
			Usermessages:           make([]string, 0),
			Waitdatas:              make([]string, 0),
		}

		for _, entry := range nodeIdBranchIdToNewEntry {
			createParams.Tenantids = append(createParams.Tenantids, opts.TenantId)
			createParams.Externalids = append(createParams.Externalids, uuid.New())
			createParams.Durabletaskids = append(createParams.Durabletaskids, opts.DurableTaskId)
			createParams.Durabletaskinsertedats = append(createParams.Durabletaskinsertedats, opts.DurableTaskInsertedAt)
			createParams.Kinds = append(createParams.Kinds, string(entry.Kind))
			createParams.Nodeids = append(createParams.Nodeids, entry.NodeId)
			createParams.Branchids = append(createParams.Branchids, entry.BranchId)
			createParams.Idempotencykeys = append(createParams.Idempotencykeys, entry.IdempotencyKey)
			createParams.Issatisfieds = append(createParams.Issatisfieds, entry.IsSatisfied)

			if entry.UserMessage != nil {
				createParams.Usermessages = append(createParams.Usermessages, *entry.UserMessage)
			} else {
				createParams.Usermessages = append(createParams.Usermessages, "")
			}

			createParams.Waitdatas = append(createParams.Waitdatas, entry.WaitData)
		}

		createdRows, createErr := r.queries.BulkCreateDurableEventLogEntries(ctx, tx, createParams)
		if createErr != nil {
			return nil, fmt.Errorf("failed to bulk-create event log entries: %w", createErr)
		}

		for _, createdRow := range createdRows {
			nodeIdBranchIdToCreatedEntry[NodeIdBranchIdTuple{createdRow.NodeID, createdRow.BranchID}] = createdRow
		}

		storePayloadOpts := make([]StorePayloadOpts, 0, len(nodeIdBranchIdToNewEntry)*2)
		for _, createdRow := range createdRows {
			opt, ok := nodeIdBranchIdToNewEntry[NodeIdBranchIdTuple{createdRow.NodeID, createdRow.BranchID}]

			if !ok {
				continue
			}

			if len(opt.InputPayload) > 0 {
				storePayloadOpts = append(storePayloadOpts, StorePayloadOpts{
					Id:         createdRow.ID,
					InsertedAt: createdRow.InsertedAt,
					ExternalId: createdRow.ExternalID,
					Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYDATA,
					Payload:    opt.InputPayload,
					TenantId:   opts.TenantId,
				})
			}

			if len(opt.ResultPayload) > 0 {
				storePayloadOpts = append(storePayloadOpts, StorePayloadOpts{
					Id:         createdRow.ID,
					InsertedAt: createdRow.InsertedAt,
					ExternalId: createdRow.ResultPayloadExternalID,
					Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYRESULTDATA,
					Payload:    opt.ResultPayload,
					TenantId:   opts.TenantId,
				})
			}
		}

		if len(storePayloadOpts) > 0 {
			if storeErr := r.payloadStore.Store(ctx, tx, storePayloadOpts...); storeErr != nil {
				return nil, fmt.Errorf("failed to store payloads for new entries: %w", storeErr)
			}
		}
	}

	var retrieveOpts []RetrievePayloadOpts
	for _, entry := range existedEntries {
		retrieveOpts = append(retrieveOpts, RetrievePayloadOpts{
			Id:         entry.ID,
			InsertedAt: entry.InsertedAt,
			Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYRESULTDATA,
			TenantId:   opts.TenantId,
		})
	}

	var existingPayloads map[RetrievePayloadOpts][]byte
	if len(retrieveOpts) > 0 {
		existingPayloads, err = r.payloadStore.Retrieve(ctx, tx, retrieveOpts...)
		if err != nil {
			existingPayloads = nil
		}
	}

	results := make([]*EventLogEntryWithPayloads, n)
	for i, o := range opts.Entries {
		key := NodeIdBranchIdTuple{o.NodeId, o.BranchId}
		if existingEntry, ok := existedEntries[key]; ok {
			var resultPayload []byte
			if existingPayloads != nil {
				resultPayload = existingPayloads[RetrievePayloadOpts{
					Id:         existingEntry.ID,
					InsertedAt: existingEntry.InsertedAt,
					Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYRESULTDATA,
					TenantId:   opts.TenantId,
				}]
			}
			results[i] = &EventLogEntryWithPayloads{
				Entry:          existingEntry,
				InputPayload:   o.InputPayload,
				ResultPayload:  resultPayload,
				AlreadyExisted: true,
			}
		} else {
			created := nodeIdBranchIdToCreatedEntry[key]
			results[i] = &EventLogEntryWithPayloads{
				Entry: &sqlcv1.BulkGetDurableEventLogEntriesRow{
					TenantID:              created.TenantID,
					ExternalID:            created.ExternalID,
					ID:                    created.ID,
					DurableTaskID:         created.DurableTaskID,
					DurableTaskInsertedAt: created.DurableTaskInsertedAt,
					Kind:                  created.Kind,
					NodeID:                created.NodeID,
					BranchID:              created.BranchID,
					IdempotencyKey:        created.IdempotencyKey,
					IsSatisfied:           created.IsSatisfied,
					InvocationCount:       created.InvocationCount,
				},
				InputPayload:   o.InputPayload,
				ResultPayload:  o.ResultPayload,
				AlreadyExisted: false,
			}
		}
	}

	slices.SortFunc(results, func(i, j *EventLogEntryWithPayloads) int {
		if i.Entry.NodeID != j.Entry.NodeID {
			return int(i.Entry.NodeID - j.Entry.NodeID)
		}

		return int(i.Entry.BranchID - j.Entry.BranchID)
	})

	return results, nil
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
		return nil, &StaleInvocationError{
			TaskExternalId:          opts.Task.ExternalID,
			ExpectedInvocationCount: logFile.LatestInvocationCount,
			ActualInvocationCount:   opts.InvocationCount,
		}
	}

	baseNodeId := logFile.LatestNodeID + 1

	var getOrCreateOpts GetOrCreateLogEntryOpts

	nodeIdBranchIdToTriggerOpts := make(map[NodeIdBranchIdTuple]*WorkflowNameTriggerOpts)
	runExternalIdToNodeIdBranchId := make(map[uuid.UUID]NodeIdBranchIdTuple)

	switch opts.Kind {
	case sqlcv1.V1DurableEventLogKindRUN:
		innerOpts := make([]GetOrCreateLogEntryOpt, len(opts.TriggerRuns.TriggerOpts))

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

			innerOpts[i] = GetOrCreateLogEntryOpt{
				Kind:            sqlcv1.V1DurableEventLogKindRUN,
				NodeId:          nodeId,
				BranchId:        branchId,
				InvocationCount: opts.InvocationCount,
				IdempotencyKey:  idempotencyKey,
				InputPayload:    inputPayload,
				WaitData:        marshalWaitData(waitDataFromTriggerOpt(triggerOpts)),
			}

			nodeBranchKey := NodeIdBranchIdTuple{NodeId: nodeId, BranchId: branchId}
			nodeIdBranchIdToTriggerOpts[nodeBranchKey] = triggerOpts
			runExternalIdToNodeIdBranchId[triggerOpts.ExternalId] = nodeBranchKey
		}

		getOrCreateOpts = GetOrCreateLogEntryOpts{
			TenantId:              tenantId,
			DurableTaskId:         task.ID,
			DurableTaskInsertedAt: task.InsertedAt,
			DurableTaskExternalId: task.ExternalID,
			Entries:               innerOpts,
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

		getOrCreateOpts = GetOrCreateLogEntryOpts{
			TenantId:              tenantId,
			DurableTaskExternalId: task.ExternalID,
			DurableTaskId:         task.ID,
			DurableTaskInsertedAt: task.InsertedAt,
			Entries: []GetOrCreateLogEntryOpt{{
				Kind:            sqlcv1.V1DurableEventLogKindWAITFOR,
				NodeId:          baseNodeId,
				BranchId:        branchId,
				InvocationCount: opts.InvocationCount,
				IdempotencyKey:  idempotencyKey,
				InputPayload:    inputPayload,
				UserMessage:     opts.WaitFor.Label,
				WaitData:        marshalWaitData(waitDataFromWaitForConditions(opts.WaitFor.WaitForConditions)),
			}},
		}
	case sqlcv1.V1DurableEventLogKindMEMO:
		branchId := resolveBranchForNode(baseNodeId, logFile.LatestBranchID, nextBranchIdToBranchPoint)

		var resultPayload []byte
		isSatisfied := false
		if len(opts.Memo.Payload) > 0 {
			isSatisfied = true
			resultPayload = opts.Memo.Payload
		}

		getOrCreateOpts = GetOrCreateLogEntryOpts{
			TenantId:              tenantId,
			DurableTaskExternalId: task.ExternalID,
			DurableTaskId:         task.ID,
			DurableTaskInsertedAt: task.InsertedAt,
			Entries: []GetOrCreateLogEntryOpt{{
				Kind:            sqlcv1.V1DurableEventLogKindMEMO,
				NodeId:          baseNodeId,
				BranchId:        branchId,
				InvocationCount: opts.InvocationCount,
				IdempotencyKey:  opts.Memo.MemoKey,
				IsSatisfied:     isSatisfied,
				ResultPayload:   resultPayload,
			}},
		}
	default:
		return nil, fmt.Errorf("unsupported durable event log entry kind: %s", opts.Kind)
	}

	logEntries, err := r.getOrCreateEventLogEntries(ctx, tx, getOrCreateOpts)
	if err != nil {
		var nde *NonDeterminismError
		if errors.As(err, &nde) {
			var existingPayload []byte
			payloads, retrieveErr := r.payloadStore.Retrieve(ctx, tx, RetrievePayloadOpts{
				Id:         nde.ExistingEntryId,
				InsertedAt: nde.ExistingEntryInsertedAt,
				Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYDATA,
				TenantId:   nde.ExistingEntryTenantId,
			})
			if retrieveErr == nil {
				existingPayload = payloads[RetrievePayloadOpts{
					Id:         nde.ExistingEntryId,
					InsertedAt: nde.ExistingEntryInsertedAt,
					Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYDATA,
					TenantId:   nde.ExistingEntryTenantId,
				}]
			}
			nde.Detail = nonDeterminismDetail(opts, nde.ExpectedKind, existingPayload)
		}

		return nil, fmt.Errorf("failed to get or create event log entries: %w", err)
	}

	var memoResult *IngestMemoResult
	var waitForResult *IngestWaitForResult
	var triggerRunsResult *IngestTriggerRunsResult

	switch opts.Kind {
	case sqlcv1.V1DurableEventLogKindRUN:
		entries := make([]*IngestTriggerRunsEntry, len(getOrCreateOpts.Entries))

		for i, entry := range logEntries {
			triggerOpts, ok := nodeIdBranchIdToTriggerOpts[NodeIdBranchIdTuple{
				NodeId:   entry.Entry.NodeID,
				BranchId: entry.Entry.BranchID,
			}]

			if !ok {
				return nil, fmt.Errorf("missing trigger opts for nodeId %d and branchId %d", entry.Entry.NodeID, entry.Entry.BranchID)
			}

			entries[i] = &IngestTriggerRunsEntry{
				NodeId:                entry.Entry.NodeID,
				BranchId:              entry.Entry.BranchID,
				IsSatisfied:           entry.Entry.IsSatisfied,
				AlreadyExisted:        entry.AlreadyExisted,
				ResultPayload:         entry.ResultPayload,
				WorkflowRunExternalId: triggerOpts.ExternalId,
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

			createMatchOpts := make([]CreateMatchOpts, 0, len(createdTasks)+len(createdDags))

			dagExternalIds := make(map[uuid.UUID]struct{}, len(createdDags))

			for _, dag := range createdDags {
				dagExternalIds[dag.ExternalID] = struct{}{}
			}

			for _, ct := range createdTasks {
				if _, isDagTask := dagExternalIds[ct.WorkflowRunID]; isDagTask {
					continue
				}

				childHint := ct.ExternalID.String()
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

				nodeIdBranchId := runExternalIdToNodeIdBranchId[ct.ExternalID]

				nodeId := nodeIdBranchId.NodeId
				branchId := nodeIdBranchId.BranchId

				runEventLogEntrySignalKey := fmt.Sprintf("durable_run:%s:%d:%d", task.ExternalID.String(), branchId, nodeId)

				taskId := task.ID

				createMatchOpts = append(createMatchOpts, CreateMatchOpts{
					Kind:                         sqlcv1.V1MatchKindSIGNAL,
					Conditions:                   conditions,
					SignalTaskId:                 &taskId,
					SignalTaskInsertedAt:         task.InsertedAt,
					SignalExternalId:             &ct.ExternalID,
					SignalTaskExternalId:         &task.ExternalID,
					SignalKey:                    &runEventLogEntrySignalKey,
					DurableEventLogEntryNodeId:   &nodeId,
					DurableEventLogEntryBranchId: &branchId,
				})
			}

			for _, dag := range createdDags {
				conditions := make([]GroupMatchCondition, 0, len(dag.TaskExternalIDs)*3)

				for i, taskExtId := range dag.TaskExternalIDs {
					childHint := taskExtId.String()
					orGroupId := uuid.New()

					readableDataKey := "output"
					if i < len(dag.TaskStepReadableIDs) {
						readableDataKey = dag.TaskStepReadableIDs[i]
					}

					conditions = append(conditions,
						GroupMatchCondition{
							GroupId:           orGroupId,
							EventType:         sqlcv1.V1EventTypeINTERNAL,
							EventKey:          string(sqlcv1.V1TaskEventTypeCOMPLETED),
							ReadableDataKey:   readableDataKey,
							EventResourceHint: &childHint,
							Expression:        "true",
							Action:            sqlcv1.V1MatchConditionActionCREATE,
						},
						GroupMatchCondition{
							GroupId:           orGroupId,
							EventType:         sqlcv1.V1EventTypeINTERNAL,
							EventKey:          string(sqlcv1.V1TaskEventTypeFAILED),
							ReadableDataKey:   readableDataKey,
							EventResourceHint: &childHint,
							Expression:        "true",
							Action:            sqlcv1.V1MatchConditionActionCREATE,
						},
						GroupMatchCondition{
							GroupId:           orGroupId,
							EventType:         sqlcv1.V1EventTypeINTERNAL,
							EventKey:          string(sqlcv1.V1TaskEventTypeCANCELLED),
							ReadableDataKey:   readableDataKey,
							EventResourceHint: &childHint,
							Expression:        "true",
							Action:            sqlcv1.V1MatchConditionActionCREATE,
						},
					)
				}

				nodeIdBranchId := runExternalIdToNodeIdBranchId[dag.ExternalID]

				nodeId := nodeIdBranchId.NodeId
				branchId := nodeIdBranchId.BranchId

				runEventLogEntrySignalKey := fmt.Sprintf("durable_run:%s:%d:%d", task.ExternalID.String(), branchId, nodeId)

				taskId := task.ID
				dagExternalId := dag.ExternalID

				createMatchOpts = append(createMatchOpts, CreateMatchOpts{
					Kind:                         sqlcv1.V1MatchKindSIGNAL,
					Conditions:                   conditions,
					SignalTaskId:                 &taskId,
					SignalTaskInsertedAt:         task.InsertedAt,
					SignalExternalId:             &dagExternalId,
					SignalTaskExternalId:         &task.ExternalID,
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
		if len(logEntries) != 1 {
			// note: we implicitly assume that there will only be one log entry for wait for conditions
			// if we get more than one, it's an indication something is wrong
			return nil, fmt.Errorf("expected to get exactly one log entry for wait for condition, but got %d", len(logEntries))
		}
		le := logEntries[0]

		if !le.AlreadyExisted {
			if err = r.handleWaitFor(ctx, tx, tenantId, le.Entry.BranchID, le.Entry.NodeID, opts.WaitFor.WaitForConditions, task); err != nil {
				return nil, fmt.Errorf("failed to handle wait for conditions: %w", err)
			}
		}

		waitForResult = &IngestWaitForResult{
			InvocationCount: opts.InvocationCount,
			IsSatisfied:     le.Entry.IsSatisfied,
			NodeId:          le.Entry.NodeID,
			BranchId:        le.Entry.BranchID,
			AlreadyExisted:  le.AlreadyExisted,
			ResultPayload:   le.ResultPayload,
		}
	case sqlcv1.V1DurableEventLogKindMEMO:
		if len(logEntries) != 1 {
			// note: we implicitly assume that there will only be one log entry for memo
			// if we get more than one, it's an indication something is wrong
			return nil, fmt.Errorf("expected to get exactly one log entry for memo, but got %d", len(logEntries))
		}

		le := logEntries[0]

		memoResult = &IngestMemoResult{
			InvocationCount: opts.InvocationCount,
			IsSatisfied:     le.Entry.IsSatisfied,
			NodeId:          le.Entry.NodeID,
			BranchId:        le.Entry.BranchID,
			ResultPayload:   le.ResultPayload,
			AlreadyExisted:  le.AlreadyExisted,
		}
	}

	n := len(getOrCreateOpts.Entries)

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

	if opts.Kind == sqlcv1.V1DurableEventLogKindWAITFOR {
		waitForResult, err = r.handleEventLookback(ctx, tenantId, waitForResult, opts.WaitFor.WaitForConditions)
	}

	return &IngestDurableTaskEventResult{
		Kind:              opts.Kind,
		MemoResult:        memoResult,
		WaitForResult:     waitForResult,
		TriggerRunsResult: triggerRunsResult,
	}, nil
}

func (r *durableEventsRepository) handleEventLookback(ctx context.Context, tenantId uuid.UUID, initialWaitForResult *IngestWaitForResult, waitForConditions []CreateExternalSignalConditionOpt) (*IngestWaitForResult, error) {
	lookbackOptTx, err := r.PrepareOptimisticTx(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to prepare tx for retroactive matching: %w", err)
	}

	defer lookbackOptTx.Rollback()

	lookbackTx := lookbackOptTx.tx

	lookbackParams := sqlcv1.GetPreviousMatchingEventsByKeysWithScopeHintParams{
		Tenantid: tenantId,
	}

	for _, c := range waitForConditions {
		if c.UserEventScope != nil && c.UserEventConsiderEventsSince != nil && c.UserEventKey != nil {
			lookbackParams.Keys = append(lookbackParams.Keys, *c.UserEventKey)
			lookbackParams.Seensinces = append(lookbackParams.Seensinces, sqlchelpers.TimestamptzFromTime(*c.UserEventConsiderEventsSince))
			lookbackParams.Scopes = append(lookbackParams.Scopes, *c.UserEventScope)
		}
	}

	previousEventsFound, err := r.queries.GetPreviousMatchingEventsByKeysWithScopeHint(ctx, lookbackTx, lookbackParams)

	if err != nil {
		return nil, fmt.Errorf("failed to look up recent user events for retroactive matching: %w", err)
	}

	if len(previousEventsFound) == 0 {
		return initialWaitForResult, nil
	}

	retrievePayloadOpts := make([]RetrievePayloadOpts, 0, len(previousEventsFound))

	for _, row := range previousEventsFound {
		retrievePayloadOpts = append(retrievePayloadOpts, RetrievePayloadOpts{
			Id:         row.ID,
			InsertedAt: row.SeenAt,
			Type:       sqlcv1.V1PayloadTypeUSEREVENTINPUT,
			TenantId:   tenantId,
		})
	}

	retrieveOptsToPayload, err := r.payloadStore.Retrieve(ctx, lookbackTx, retrievePayloadOpts...)

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve payloads for recent user events for retroactive matching: %w", err)
	}

	retroCandidates := make([]CandidateEventMatch, 0, len(previousEventsFound))

	for _, row := range previousEventsFound {
		retrieveOpts := RetrievePayloadOpts{
			Id:         row.ID,
			InsertedAt: row.SeenAt,
			Type:       sqlcv1.V1PayloadTypeUSEREVENTINPUT,
			TenantId:   tenantId,
		}

		payload, ok := retrieveOptsToPayload[retrieveOpts]

		if !ok {
			r.l.Warn().Ctx(ctx).Msgf("payload not found for recent user event with id %d and seen_at %s", row.ID, row.SeenAt.Time)
			payload = nil
		}

		retroCandidates = append(retroCandidates, CandidateEventMatch{
			ID:             row.ExternalID,
			EventTimestamp: row.SeenAt.Time,
			Key:            row.Key,
			Data:           payload,
		})
	}

	if len(retroCandidates) > 0 {
		retroMatchResults, err := r.processEventMatches(ctx, lookbackTx, tenantId, retroCandidates, sqlcv1.V1EventTypeUSER)

		if err != nil {
			return nil, fmt.Errorf("failed to process retroactive event matches: %w", err)
		}

		if retroMatchResults != nil && len(retroMatchResults.SatisfiedDurableEventLogEntries) > 0 {
			// note: this might be buggy but I _think_ it's okay to grab the first match here
			// the main assumption is that we only ever get one entry back
			entry := retroMatchResults.SatisfiedDurableEventLogEntries[0]

			if err := lookbackOptTx.Commit(ctx); err != nil {
				return nil, fmt.Errorf("failed to commit lookback transaction: %w", err)
			}

			return &IngestWaitForResult{
				IsSatisfied:     true,
				ResultPayload:   entry.Data,
				InvocationCount: entry.InvocationCount,
				NodeId:          initialWaitForResult.NodeId,
				BranchId:        initialWaitForResult.BranchId,
				AlreadyExisted:  initialWaitForResult.AlreadyExisted,
			}, nil
		}
	}

	if err := lookbackOptTx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit lookback transaction: %w", err)
	}

	return initialWaitForResult, nil
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
			ExternalId: entry.ResultPayloadExternalID,
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

func (r *durableEventsRepository) ListDurableEventLog(ctx context.Context, tenantId uuid.UUID, taskInsertedAt pgtype.Timestamptz, taskId, limit, offset int64) ([]*sqlcv1.ListDurableEventLogForTaskRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-durable-event-log-olap")
	defer span.End()

	return r.queries.ListDurableEventLogForTask(ctx, r.pool, sqlcv1.ListDurableEventLogForTaskParams{
		Durabletaskid:         taskId,
		Durabletaskinsertedat: taskInsertedAt,
		Tenantid:              tenantId,
		Eventlogoffset:        offset,
		Eventloglimit:         limit,
	})
}
