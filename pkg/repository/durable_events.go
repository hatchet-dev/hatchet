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
	"github.com/jackc/pgx/v5"
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
	Result                []byte
	SatisfiedOrder        *int64
	ChildTaskIsFailure    bool
	ChildTaskErrorMessage *string
	BranchID              int64
	NodeID                int64
	InvocationCount       int32
	TaskExternalId        uuid.UUID
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
	SatisfiedOrder        *int64
	ChildTaskIsFailure    bool
	ChildTaskErrorMessage *string
	NodeId                int64
	BranchId              int64
	WorkflowRunExternalId uuid.UUID
	IsSatisfied           bool
	AlreadyExisted        bool

	ChildNeedsReplay bool

	// ReExecuted is true when the entry was newly created this invocation, i.e. the child
	// actually runs rather than being satisfied from a cached log entry.
	ReExecuted bool
}

type IngestTriggerRunsResult struct {
	Entries         []*IngestTriggerRunsEntry
	PendingTriggers []PendingDurableRunTrigger
	InvocationCount int32
}

type IngestWaitForResult struct {
	ResultPayload   []byte
	SatisfiedOrder  *int64
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

type TriggerPendingRunEntriesOpt struct {
	Task        *sqlcv1.FlattenExternalIdsRow
	PendingRuns []PendingDurableRunTrigger
}

type DurableEventsRepository interface {
	IngestDurableTaskEvent(ctx context.Context, opts IngestDurableTaskEventOpts) (*IngestDurableTaskEventResult, error)
	HandleBranch(ctx context.Context, tenantId uuid.UUID, nodeId, branchId int64, task *sqlcv1.FlattenExternalIdsRow) (*HandleBranchResult, error)
	HandleBranchForDAGReplay(ctx context.Context, tenantId uuid.UUID, task *sqlcv1.FlattenExternalIdsRow, forcedChildExternalIds []uuid.UUID) (*HandleBranchResult, error)
	TriggerPendingRunEntries(ctx context.Context, tenantId uuid.UUID, tasks []TriggerPendingRunEntriesOpt) ([]*V1TaskWithPayload, []*DAGWithData, []CELEvaluationFailure, error)

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
	Detail                               *NonDeterminismDetail
	ExistingEntryInsertedAt              pgtype.Timestamptz
	ExpectedKind                         sqlcv1.V1DurableEventLogKind
	ActualKind                           sqlcv1.V1DurableEventLogKind
	ExpectedIdempotencyKey               []byte
	ActualIdempotencyKey                 []byte
	NodeId                               int64
	BranchId                             int64
	ExistingEntryId                      int64
	TaskExternalId                       uuid.UUID
	ExistingEntryTenantId                uuid.UUID
	ExistingEntryExternalId              uuid.UUID
	ExistingEntryResultPayloadExternalId uuid.UUID
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
			sleepDuration := time.Duration(*c.SleepDurationMs) * time.Millisecond
			return "sleep(" + sleepDuration.String() + ")"
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
	Kind                sqlcv1.V1DurableEventLogKind
	IdempotencyKey      []byte
	InputPayload        []byte
	ResultPayload       []byte
	NodeId              int64
	BranchId            int64
	InvocationCount     int32
	IsSatisfied         bool
	UserMessage         *string
	WaitData            string // JSON-encoded WaitData, empty string means no wait data
	SatisfiedAt         *time.Time
	ChildTaskExternalId uuid.UUID
	ShouldSkip          bool
}

type GetOrCreateLogEntryOpts struct {
	DurableTaskInsertedAt pgtype.Timestamptz
	Entries               []GetOrCreateLogEntryOpt
	DurableTaskId         int64
	TenantId              uuid.UUID
	DurableTaskExternalId uuid.UUID
}

type EventLogEntryWithPayloads struct {
	Entry           *sqlcv1.BulkGetDurableEventLogEntriesRow
	InputPayload    []byte
	ResultPayload   []byte
	AlreadyExisted  bool
	IsSkipReference bool
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
			ExternalId: row.ResultPayloadExternalID,
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
			ExternalId: row.ResultPayloadExternalID,
		}

		payload := payloads[retrieveOpt]

		var childTaskErrorMessage *string
		if row.ChildTaskErrorMessage.Valid {
			childTaskErrorMessage = &row.ChildTaskErrorMessage.String
		}

		result = append(result, &SatisfiedEventWithPayload{
			TaskExternalId:        row.TaskExternalID,
			NodeID:                row.NodeID,
			BranchID:              row.BranchID,
			InvocationCount:       row.InvocationCount,
			Result:                payload,
			SatisfiedOrder:        satisfiedOrderPtr(row.SatisfiedOrder),
			ChildTaskIsFailure:    row.ChildTaskIsFailure,
			ChildTaskErrorMessage: childTaskErrorMessage,
		})
	}

	return result, nil
}

func getDurableTaskSignalKey(taskExternalId uuid.UUID, nodeId int64) string {
	return fmt.Sprintf("durable:%s:%d", taskExternalId.String(), nodeId)
}

func satisfiedOrderPtr(v pgtype.Int8) *int64 {
	if !v.Valid {
		return nil
	}
	order := v.Int64
	return &order
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

func (r *durableEventsRepository) getAndLockLogFile(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz) (*sqlcv1.V1DurableEventLogFile, map[int64]*sqlcv1.V1DurableEventLogBranchPoint, error) {
	ctx, span := telemetry.NewSpan(ctx, "get-and-lock-durable-event-log-file")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant_id", Value: tenantId},
		telemetry.AttributeKV{Key: "durable_task_id", Value: durableTaskId},
	)

	rows, err := r.queries.GetAndLockLogFileWithBranchPoints(ctx, tx, sqlcv1.GetAndLockLogFileWithBranchPointsParams{
		Durabletaskid:         durableTaskId,
		Durabletaskinsertedat: durableTaskInsertedAt,
		Tenantid:              tenantId,
	})

	if err != nil {
		return nil, nil, err
	}

	if len(rows) == 0 {
		return nil, nil, pgx.ErrNoRows
	}

	logFile := rows[0].V1DurableEventLogFile

	nextBranchIdToBranchPoint := make(map[int64]*sqlcv1.V1DurableEventLogBranchPoint, len(rows))

	for _, row := range rows {
		logFile := row.V1DurableEventLogFile

		if !row.NextBranchID.Valid {
			continue
		}

		nextBranchIdToBranchPoint[row.NextBranchID.Int64] = &sqlcv1.V1DurableEventLogBranchPoint{
			TenantID:               logFile.TenantID,
			ID:                     row.ID.Int64,
			InsertedAt:             row.InsertedAt,
			DurableTaskID:          logFile.DurableTaskID,
			DurableTaskInsertedAt:  logFile.DurableTaskInsertedAt,
			FirstNodeIDInNewBranch: row.FirstNodeIDInNewBranch.Int64,
			ParentBranchID:         row.ParentBranchID.Int64,
			NextBranchID:           row.NextBranchID.Int64,
		}
	}

	return &logFile, nextBranchIdToBranchPoint, nil
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
) ([]*EventLogEntryWithPayloads, []StorePayloadOpts, error) {
	ctx, span := telemetry.NewSpan(ctx, "get-or-create-durable-event-log-entries")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "entry_count", Value: len(opts.Entries)})

	if len(opts.Entries) == 0 {
		return nil, nil, nil
	}

	var pendingStorePayloadOpts []StorePayloadOpts

	var skipOpts, nonSkipOpts []GetOrCreateLogEntryOpt
	for _, o := range opts.Entries {
		if o.ShouldSkip {
			skipOpts = append(skipOpts, o)
		} else {
			nonSkipOpts = append(nonSkipOpts, o)
		}
	}

	existedEntries := make(map[NodeIdBranchIdTuple]*sqlcv1.BulkGetDurableEventLogEntriesRow)
	nodeIdBranchIdToNewEntry := make(map[NodeIdBranchIdTuple]GetOrCreateLogEntryOpt)
	nodeIdBranchIdToCreatedEntry := make(map[NodeIdBranchIdTuple]*sqlcv1.BulkCreateDurableEventLogEntriesRow)

	if len(nonSkipOpts) > 0 {
		branchIds := make([]int64, len(nonSkipOpts))
		nodeIds := make([]int64, len(nonSkipOpts))
		for i, o := range nonSkipOpts {
			branchIds[i] = o.BranchId
			nodeIds[i] = o.NodeId
		}

		existing, err := r.queries.BulkGetDurableEventLogEntries(ctx, tx, sqlcv1.BulkGetDurableEventLogEntriesParams{
			Durabletaskid:         opts.DurableTaskId,
			Durabletaskinsertedat: opts.DurableTaskInsertedAt,
			Branchids:             branchIds,
			Nodeids:               nodeIds,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to bulk-get existing entries: %w", err)
		}

		existingByKey := make(map[NodeIdBranchIdTuple]*sqlcv1.BulkGetDurableEventLogEntriesRow, len(existing))
		for _, e := range existing {
			existingByKey[NodeIdBranchIdTuple{e.NodeID, e.BranchID}] = e
		}

		for _, o := range nonSkipOpts {
			key := NodeIdBranchIdTuple{o.NodeId, o.BranchId}
			e, found := existingByKey[key]
			if !found {
				nodeIdBranchIdToNewEntry[key] = o
				continue
			}
			if !bytes.Equal(o.IdempotencyKey, e.IdempotencyKey) {
				return nil, nil, &NonDeterminismError{
					BranchId:                o.BranchId,
					NodeId:                  o.NodeId,
					TaskExternalId:          opts.DurableTaskExternalId,
					ExpectedIdempotencyKey:  e.IdempotencyKey,
					ActualIdempotencyKey:    o.IdempotencyKey,
					ExpectedKind:            e.Kind,
					ActualKind:              o.Kind,
					ExistingEntryId:         e.ID,
					ExistingEntryInsertedAt: e.InsertedAt,
					ExistingEntryTenantId:   e.TenantID,
				}
			}
			existedEntries[key] = e
		}
	}

	if len(nodeIdBranchIdToNewEntry) > 0 {
		createParams := sqlcv1.BulkCreateDurableEventLogEntriesParams{
			Tenantids:              make([]uuid.UUID, 0),
			Externalids:            make([]uuid.UUID, 0),
			Childtaskexternalids:   make([]uuid.UUID, 0),
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
			createParams.Childtaskexternalids = append(createParams.Childtaskexternalids, entry.ChildTaskExternalId)
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
			return nil, nil, fmt.Errorf("failed to bulk-create event log entries: %w", createErr)
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

		pendingStorePayloadOpts = storePayloadOpts
	}

	childTaskExternalIdToSkipEntry := make(map[uuid.UUID]*sqlcv1.BulkGetDurableEventLogEntriesRow)

	if len(skipOpts) > 0 {
		childTaskExternalIds := make([]uuid.UUID, 0, len(skipOpts))
		seenChildTaskExternalIds := make(map[uuid.UUID]struct{}, len(skipOpts))
		for _, o := range skipOpts {
			if o.ChildTaskExternalId == uuid.Nil {
				return nil, nil, fmt.Errorf("skipped child entries must include a non-nil child task external id")
			}

			if _, ok := seenChildTaskExternalIds[o.ChildTaskExternalId]; ok {
				continue
			}

			seenChildTaskExternalIds[o.ChildTaskExternalId] = struct{}{}
			childTaskExternalIds = append(childTaskExternalIds, o.ChildTaskExternalId)
		}

		skipRows, err := r.queries.GetDurableEventLogEntriesByChildTaskExternalIds(ctx, tx, sqlcv1.GetDurableEventLogEntriesByChildTaskExternalIdsParams{
			Durabletaskid:         opts.DurableTaskId,
			Durabletaskinsertedat: opts.DurableTaskInsertedAt,
			Childtaskexternalids:  childTaskExternalIds,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get log entries by child task external ids: %w", err)
		}

		for _, row := range skipRows {
			if row.ChildTaskExternalID == nil {
				continue
			}
			// node ids restart every re-invocation, so a child's newest entry is on the highest branch
			if existing, ok := childTaskExternalIdToSkipEntry[*row.ChildTaskExternalID]; ok && existing.BranchID >= row.BranchID {
				continue
			}

			r := sqlcv1.BulkGetDurableEventLogEntriesRow(*row)
			childTaskExternalIdToSkipEntry[*row.ChildTaskExternalID] = &r
		}

		for _, o := range skipOpts {
			e, ok := childTaskExternalIdToSkipEntry[o.ChildTaskExternalId]

			if !ok {
				return nil, nil, fmt.Errorf("expected to find log entry for skipped child task external id %s", o.ChildTaskExternalId)
			}

			if len(o.IdempotencyKey) > 0 && !bytes.Equal(o.IdempotencyKey, e.IdempotencyKey) {
				return nil, nil, &NonDeterminismError{
					BranchId:                e.BranchID,
					NodeId:                  e.NodeID,
					TaskExternalId:          opts.DurableTaskExternalId,
					ExpectedIdempotencyKey:  e.IdempotencyKey,
					ActualIdempotencyKey:    o.IdempotencyKey,
					ExpectedKind:            e.Kind,
					ActualKind:              o.Kind,
					ExistingEntryId:         e.ID,
					ExistingEntryInsertedAt: e.InsertedAt,
					ExistingEntryTenantId:   e.TenantID,
					ExistingEntryExternalId: e.ExternalID,
				}
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
			ExternalId: entry.ResultPayloadExternalID,
		})
	}

	for _, entry := range childTaskExternalIdToSkipEntry {
		retrieveOpts = append(retrieveOpts, RetrievePayloadOpts{
			Id:         entry.ID,
			InsertedAt: entry.InsertedAt,
			Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYRESULTDATA,
			TenantId:   opts.TenantId,
			ExternalId: entry.ResultPayloadExternalID,
		})
	}

	var existingPayloads map[RetrievePayloadOpts][]byte
	if len(retrieveOpts) > 0 {
		var err error
		existingPayloads, err = r.payloadStore.Retrieve(ctx, tx, retrieveOpts...)
		if err != nil {
			existingPayloads = nil
		}
	}

	resultPayload := func(e *sqlcv1.BulkGetDurableEventLogEntriesRow) []byte {
		if existingPayloads == nil {
			return nil
		}

		return existingPayloads[RetrievePayloadOpts{
			Id:         e.ID,
			InsertedAt: e.InsertedAt,
			Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYRESULTDATA,
			TenantId:   opts.TenantId,
			ExternalId: e.ResultPayloadExternalID,
		}]
	}

	var results []*EventLogEntryWithPayloads

	for _, o := range nonSkipOpts {
		key := NodeIdBranchIdTuple{o.NodeId, o.BranchId}
		if e, ok := existedEntries[key]; ok {
			results = append(results, &EventLogEntryWithPayloads{
				Entry:          e,
				InputPayload:   o.InputPayload,
				ResultPayload:  resultPayload(e),
				AlreadyExisted: true,
			})
		} else {
			created := nodeIdBranchIdToCreatedEntry[key]
			results = append(results, &EventLogEntryWithPayloads{
				Entry: &sqlcv1.BulkGetDurableEventLogEntriesRow{
					TenantID:              created.TenantID,
					ExternalID:            created.ExternalID,
					ChildTaskExternalID:   created.ChildTaskExternalID,
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
			})
		}
	}

	for _, o := range skipOpts {
		e := childTaskExternalIdToSkipEntry[o.ChildTaskExternalId]
		results = append(results, &EventLogEntryWithPayloads{
			Entry:           e,
			AlreadyExisted:  true,
			ResultPayload:   resultPayload(e),
			IsSkipReference: true,
		})
	}

	slices.SortFunc(results, func(i, j *EventLogEntryWithPayloads) int {
		if i.Entry.NodeID != j.Entry.NodeID {
			return int(i.Entry.NodeID - j.Entry.NodeID)
		}
		return int(i.Entry.BranchID - j.Entry.BranchID)
	})

	return results, pendingStorePayloadOpts, nil
}

func (r *durableEventsRepository) resolveOrphanedChildDedupes(
	ctx context.Context,
	tx sqlcv1.DBTX,
	logFile *sqlcv1.V1DurableEventLogFile,
	nextBranchIdToBranchPoint map[int64]*sqlcv1.V1DurableEventLogBranchPoint,
	triggerOpts []*WorkflowNameTriggerOpts,
) (map[uuid.UUID]bool, error) {
	childrenToReplay := make(map[uuid.UUID]bool)
	skipChildIds := make([]uuid.UUID, 0)

	for _, to := range triggerOpts {
		if to.ShouldSkip {
			skipChildIds = append(skipChildIds, to.ExternalId)
		}
	}

	if len(skipChildIds) == 0 {
		return childrenToReplay, nil
	}

	rows, err := r.queries.GetDurableEventLogEntriesByChildTaskExternalIds(ctx, tx, sqlcv1.GetDurableEventLogEntriesByChildTaskExternalIdsParams{
		Durabletaskid:         logFile.DurableTaskID,
		Durabletaskinsertedat: logFile.DurableTaskInsertedAt,
		Childtaskexternalids:  skipChildIds,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get log entries for deduped child runs: %w", err)
	}

	// A NULL forced set (pre-column branch points, or worker-initiated HandleBranch) means every
	// orphaned child is replayed.
	var forcedReplayChildren map[uuid.UUID]bool

	if bp, ok := nextBranchIdToBranchPoint[logFile.LatestBranchID]; ok && bp.ReplayChildExternalIds != nil {
		forcedReplayChildren = make(map[uuid.UUID]bool, len(bp.ReplayChildExternalIds))
		for _, id := range bp.ReplayChildExternalIds {
			forcedReplayChildren[id] = true
		}
	}

	childOnActiveBranch := make(map[uuid.UUID]bool)
	latestEntryByChild := make(map[uuid.UUID]*sqlcv1.GetDurableEventLogEntriesByChildTaskExternalIdsRow)

	for _, row := range rows {
		if row.ChildTaskExternalID == nil {
			continue
		}

		if resolveBranchForNode(row.NodeID, logFile.LatestBranchID, nextBranchIdToBranchPoint) == row.BranchID {
			childOnActiveBranch[*row.ChildTaskExternalID] = true
		}

		// node ids restart every re-invocation, so a child's newest entry is on the highest branch
		if latest, ok := latestEntryByChild[*row.ChildTaskExternalID]; !ok || row.BranchID > latest.BranchID {
			latestEntryByChild[*row.ChildTaskExternalID] = row
		}
	}

	for _, to := range triggerOpts {
		if !to.ShouldSkip || childOnActiveBranch[to.ExternalId] {
			continue
		}

		if latestEntryByChild[to.ExternalId] == nil {
			to.ShouldSkip = false
			to.ExternalId = uuid.New()
			continue
		}

		if to.ReplayOrphanedChildren {
			forced := forcedReplayChildren[to.ExternalId] || to.ParentReExecuted
			latestSatisfied := latestEntryByChild[to.ExternalId].IsSatisfied

			if forcedReplayChildren != nil && !forced && latestSatisfied {
				// untouched subtree: the skip path returns the cached result without re-running
				continue
			}

			to.ShouldSkip = false
			childrenToReplay[to.ExternalId] = true
		} else {
			to.ShouldSkip = false
			to.ExternalId = uuid.New()
		}
	}

	return childrenToReplay, nil
}

type PendingDurableRunTrigger struct {
	NodeId      int64
	BranchId    int64
	TriggerOpts *WorkflowNameTriggerOpts
}

func (r *durableEventsRepository) IngestDurableTaskEvent(ctx context.Context, opts IngestDurableTaskEventOpts) (*IngestDurableTaskEventResult, error) {
	ctx, span := telemetry.NewSpan(ctx, "ingest-durable-task-event")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant_id", Value: opts.TenantId},
		telemetry.AttributeKV{Key: "kind", Value: string(opts.Kind)},
		telemetry.AttributeKV{Key: "invocation_count", Value: opts.InvocationCount},
	)

	if opts.Task != nil {
		telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "task_external_id", Value: opts.Task.ExternalID})
	}

	if err := r.v.Validate(opts); err != nil {
		return nil, fmt.Errorf("invalid opts: %w", err)
	}

	if opts.Kind == sqlcv1.V1DurableEventLogKindRUN && len(opts.TriggerRuns.TriggerOpts) == 0 {
		return nil, fmt.Errorf("TriggerOptsList is required and must be non-empty for RUN kind")
	}

	tenantId := opts.TenantId
	task := opts.Task

	logEntries, nodeIdBranchIdToTriggerOpts, childrenToReplay, err := r.appendDurableEventLog(ctx, opts)

	if err != nil {
		return nil, err
	}

	var memoResult *IngestMemoResult
	var waitForResult *IngestWaitForResult
	var triggerRunsResult *IngestTriggerRunsResult

	switch opts.Kind {
	case sqlcv1.V1DurableEventLogKindRUN:
		entries := make([]*IngestTriggerRunsEntry, len(logEntries))

		for i, entry := range logEntries {
			// the run's external id lives in child_task_external_id; entries written before that
			// column existed left it null and stored the id in external_id instead, so fall back to
			// external_id for those older entries.
			workflowRunExternalId := entry.Entry.ExternalID
			if entry.Entry.ChildTaskExternalID != nil {
				workflowRunExternalId = *entry.Entry.ChildTaskExternalID
			}

			var childTaskErrorMessage *string

			if entry.Entry.ChildTaskErrorMessage.Valid {
				childTaskErrorMessage = &entry.Entry.ChildTaskErrorMessage.String
			}

			childNeedsReplay := !entry.AlreadyExisted &&
				!entry.Entry.IsSatisfied &&
				entry.Entry.ChildTaskExternalID != nil &&
				childrenToReplay[*entry.Entry.ChildTaskExternalID]

			entries[i] = &IngestTriggerRunsEntry{
				NodeId:                entry.Entry.NodeID,
				BranchId:              entry.Entry.BranchID,
				IsSatisfied:           entry.Entry.IsSatisfied,
				AlreadyExisted:        entry.AlreadyExisted,
				ResultPayload:         entry.ResultPayload,
				WorkflowRunExternalId: workflowRunExternalId,
				SatisfiedOrder:        satisfiedOrderPtr(entry.Entry.SatisfiedOrder),
				ChildTaskIsFailure:    entry.Entry.ChildTaskIsFailure,
				ChildTaskErrorMessage: childTaskErrorMessage,
				ChildNeedsReplay:      childNeedsReplay,
				ReExecuted:            !entry.AlreadyExisted,
			}
		}

		triggerRunsResult = &IngestTriggerRunsResult{
			InvocationCount: opts.InvocationCount,
			Entries:         entries,
		}

		var pending []PendingDurableRunTrigger

		for _, le := range logEntries {
			if le.Entry.TriggeredAt.Valid {
				continue
			}

			if le.IsSkipReference {
				continue
			}

			if le.Entry.ChildTaskExternalID == nil {
				return nil, fmt.Errorf("untriggered RUN log entry at nodeId %d branchId %d is missing child_task_external_id", le.Entry.NodeID, le.Entry.BranchID)
			}

			// Re-executed children already have task rows; they are replayed by the caller
			// rather than triggered as new runs.
			if childrenToReplay[*le.Entry.ChildTaskExternalID] {
				continue
			}

			key := NodeIdBranchIdTuple{NodeId: le.Entry.NodeID, BranchId: le.Entry.BranchID}
			triggerOpts, ok := nodeIdBranchIdToTriggerOpts[key]
			if !ok {
				return nil, fmt.Errorf("untriggered RUN log entry at nodeId %d branchId %d has no matching trigger in the request", le.Entry.NodeID, le.Entry.BranchID)
			}

			// trigger using the child external id already committed on the entry so re-triggering
			// reuses the same child instead of spawning a new one
			triggerOptsCopy := *triggerOpts
			triggerOptsCopy.ExternalId = *le.Entry.ChildTaskExternalID

			pending = append(pending, PendingDurableRunTrigger{
				NodeId:      le.Entry.NodeID,
				BranchId:    le.Entry.BranchID,
				TriggerOpts: &triggerOptsCopy,
			})
		}

		triggerRunsResult.PendingTriggers = pending
	case sqlcv1.V1DurableEventLogKindWAITFOR:
		if len(logEntries) != 1 {
			// note: we implicitly assume that there will only be one log entry for wait for conditions
			// if we get more than one, it's an indication something is wrong
			return nil, fmt.Errorf("expected to get exactly one log entry for wait for condition, but got %d", len(logEntries))
		}
		le := logEntries[0]

		if !le.Entry.TriggeredAt.Valid {
			if err := r.triggerPendingWaitFor(ctx, tenantId, task, le.Entry.BranchID, le.Entry.NodeID, opts.WaitFor.WaitForConditions); err != nil {
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
			SatisfiedOrder:  satisfiedOrderPtr(le.Entry.SatisfiedOrder),
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

	if opts.Kind == sqlcv1.V1DurableEventLogKindWAITFOR {
		waitForResult, err = r.handleEventLookback(ctx, tenantId, task, waitForResult, opts.WaitFor.WaitForConditions)

		if err != nil {
			return nil, err
		}
	}

	return &IngestDurableTaskEventResult{
		Kind:              opts.Kind,
		MemoResult:        memoResult,
		WaitForResult:     waitForResult,
		TriggerRunsResult: triggerRunsResult,
	}, nil
}

func (r *durableEventsRepository) resolveChildExternalIds(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, task *sqlcv1.FlattenExternalIdsRow, opts []*WorkflowNameTriggerOpts) error {
	ctx, span := telemetry.NewSpan(ctx, "resolve-child-external-ids")
	defer span.End()

	candidateIdByKey := make(map[string]uuid.UUID, len(opts))
	eventKeys := make([]string, 0, len(opts))
	childExternalIds := make([]uuid.UUID, 0, len(opts))

	for _, opt := range opts {
		spawnKey := opt.childSpawnKey()

		if spawnKey == "" {
			opt.ExternalId = uuid.New()
			continue
		}

		if _, seen := candidateIdByKey[spawnKey]; seen {
			continue
		}

		childExternalId := uuid.New()
		candidateIdByKey[spawnKey] = childExternalId

		eventKeys = append(eventKeys, spawnKey)
		childExternalIds = append(childExternalIds, childExternalId)
	}

	if len(eventKeys) == 0 {
		return nil
	}

	rows, err := r.queries.UpsertDurableChildSignalCreatedEvents(ctx, tx, sqlcv1.UpsertDurableChildSignalCreatedEventsParams{
		Tenantid:              tenantId,
		Durabletaskid:         task.ID,
		Durabletaskinsertedat: task.InsertedAt,
		Eventkeys:             eventKeys,
		Childexternalids:      childExternalIds,
	})

	if err != nil {
		return fmt.Errorf("failed to upsert child signal created events: %w", err)
	}

	resolvedIdByKey := make(map[string]uuid.UUID, len(rows))

	for _, row := range rows {
		resolvedIdByKey[row.EventKey.String] = *row.ChildExternalID
	}

	claimed := make(map[string]bool, len(rows))

	for _, opt := range opts {
		spawnKey := opt.childSpawnKey()

		if spawnKey == "" {
			continue
		}

		resolvedId, ok := resolvedIdByKey[spawnKey]

		if !ok {
			return fmt.Errorf("no child external id resolved for spawn key %s", spawnKey)
		}

		opt.ExternalId = resolvedId
		opt.ShouldSkip = resolvedId != candidateIdByKey[spawnKey] || claimed[spawnKey]
		claimed[spawnKey] = true
	}

	return nil
}

func (r *durableEventsRepository) appendDurableEventLog(ctx context.Context, opts IngestDurableTaskEventOpts) ([]*EventLogEntryWithPayloads, map[NodeIdBranchIdTuple]*WorkflowNameTriggerOpts, map[uuid.UUID]bool, error) {
	tenantId := opts.TenantId
	task := opts.Task

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to prepare tx: %w", err)
	}

	defer rollback()

	logFile, nextBranchIdToBranchPoint, err := r.getAndLockLogFile(ctx, tx, tenantId, task.ID, task.InsertedAt)

	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to lock log file: %w", err)
	}

	if logFile.LatestInvocationCount != opts.InvocationCount {
		return nil, nil, nil, &StaleInvocationError{
			TaskExternalId:          opts.Task.ExternalID,
			ExpectedInvocationCount: logFile.LatestInvocationCount,
			ActualInvocationCount:   opts.InvocationCount,
		}
	}

	baseNodeId := logFile.LatestNodeID + 1

	var getOrCreateOpts GetOrCreateLogEntryOpts

	nodeIdBranchIdToTriggerOpts := make(map[NodeIdBranchIdTuple]*WorkflowNameTriggerOpts)
	childrenToReplay := make(map[uuid.UUID]bool)

	switch opts.Kind {
	case sqlcv1.V1DurableEventLogKindRUN:
		if resolveErr := r.resolveChildExternalIds(ctx, tx, tenantId, task, opts.TriggerRuns.TriggerOpts); resolveErr != nil {
			return nil, nil, nil, fmt.Errorf("failed to resolve child external ids: %w", resolveErr)
		}

		childrenToReplay, err = r.resolveOrphanedChildDedupes(ctx, tx, logFile, nextBranchIdToBranchPoint, opts.TriggerRuns.TriggerOpts)
		if err != nil {
			return nil, nil, nil, err
		}

		innerOpts := make([]GetOrCreateLogEntryOpt, len(opts.TriggerRuns.TriggerOpts))

		nonSkipOffset := int64(0)
		for i, triggerOpts := range opts.TriggerRuns.TriggerOpts {
			if triggerOpts.ShouldSkip {
				// only index-based dedupe is validated against the existing entry's
				// idempotency key: an explicit child_key intentionally reuses the
				// cached child even when the inputs differ
				var idempotencyKey []byte
				if triggerOpts.ChildKey == nil {
					key, keyErr := r.createIdempotencyKey(sqlcv1.V1DurableEventLogKindRUN, triggerOpts, nil)
					if keyErr != nil {
						return nil, nil, nil, fmt.Errorf("failed to create idempotency key: %w", keyErr)
					}
					idempotencyKey = key
				}

				innerOpts[i] = GetOrCreateLogEntryOpt{
					Kind:                sqlcv1.V1DurableEventLogKindRUN,
					ChildTaskExternalId: triggerOpts.ExternalId,
					IdempotencyKey:      idempotencyKey,
					ShouldSkip:          true,
				}
				continue
			}

			nodeId := baseNodeId + nonSkipOffset
			nonSkipOffset++
			branchId := resolveBranchForNode(nodeId, logFile.LatestBranchID, nextBranchIdToBranchPoint)

			nodeIdBranchIdToTriggerOpts[NodeIdBranchIdTuple{NodeId: nodeId, BranchId: branchId}] = triggerOpts

			inputPayload, marshalErr := json.Marshal(triggerOpts)
			if marshalErr != nil {
				return nil, nil, nil, fmt.Errorf("failed to marshal trigger opts: %w", marshalErr)
			}

			idempotencyKey, keyErr := r.createIdempotencyKey(sqlcv1.V1DurableEventLogKindRUN, triggerOpts, nil)
			if keyErr != nil {
				return nil, nil, nil, fmt.Errorf("failed to create idempotency key: %w", keyErr)
			}

			// A child that is being cancelled or skipped never runs, so no completion event
			// will arrive for it (trigger.go creates it directly in a terminal state,
			// bypassing the match pipeline): its entry is terminal at creation regardless of
			// whether this is a first run or a replay.
			isSatisfied := triggerOpts.IsCancelled || triggerOpts.IsSkipped

			innerOpts[i] = GetOrCreateLogEntryOpt{
				Kind:                sqlcv1.V1DurableEventLogKindRUN,
				NodeId:              nodeId,
				BranchId:            branchId,
				ChildTaskExternalId: triggerOpts.ExternalId,
				InvocationCount:     opts.InvocationCount,
				IdempotencyKey:      idempotencyKey,
				InputPayload:        inputPayload,
				WaitData:            marshalWaitData(waitDataFromTriggerOpt(triggerOpts)),
				UserMessage:         triggerOpts.UserMessage,
				IsSatisfied:         isSatisfied,
			}
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
			return nil, nil, nil, fmt.Errorf("failed to marshal wait for conditions: %w", marshalErr)
		}

		idempotencyKey, keyErr := r.createIdempotencyKey(sqlcv1.V1DurableEventLogKindWAITFOR, nil, opts.WaitFor.WaitForConditions)
		if keyErr != nil {
			return nil, nil, nil, fmt.Errorf("failed to create idempotency key: %w", keyErr)
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
		return nil, nil, nil, fmt.Errorf("unsupported durable event log entry kind: %s", opts.Kind)
	}

	logEntries, entryStorePayloadOpts, err := r.getOrCreateEventLogEntries(ctx, tx, getOrCreateOpts)
	if err != nil {
		var nde *NonDeterminismError
		if errors.As(err, &nde) {
			var existingPayload []byte
			payloads, retrieveErr := r.payloadStore.Retrieve(ctx, tx, RetrievePayloadOpts{
				Id:         nde.ExistingEntryId,
				InsertedAt: nde.ExistingEntryInsertedAt,
				Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYDATA,
				TenantId:   nde.ExistingEntryTenantId,
				ExternalId: nde.ExistingEntryExternalId,
			})

			if retrieveErr == nil {
				existingPayload = payloads[RetrievePayloadOpts{
					Id:         nde.ExistingEntryId,
					InsertedAt: nde.ExistingEntryInsertedAt,
					Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYDATA,
					TenantId:   nde.ExistingEntryTenantId,
					ExternalId: nde.ExistingEntryExternalId,
				}]
			}

			nde.Detail = nonDeterminismDetail(opts, nde.ExpectedKind, existingPayload)
		}

		return nil, nil, nil, fmt.Errorf("failed to get or create event log entries: %w", err)
	}

	if len(entryStorePayloadOpts) > 0 {
		if storeErr := r.payloadStore.Store(ctx, tx, entryStorePayloadOpts...); storeErr != nil {
			return nil, nil, nil, fmt.Errorf("failed to store payloads for new entries: %w", storeErr)
		}
	}

	// Re-executed children keep their task rows, so no new run will be triggered for them — but
	// the previous completion match was consumed when the old entry was satisfied, so a fresh
	// match tied to the new entry must be registered before the caller replays the task.
	if len(childrenToReplay) > 0 {
		replayMatchOpts := make([]CreateMatchOpts, 0)

		for _, le := range logEntries {
			if le.AlreadyExisted || le.Entry.IsSatisfied || le.Entry.ChildTaskExternalID == nil {
				continue
			}

			childExternalId := *le.Entry.ChildTaskExternalID

			if !childrenToReplay[childExternalId] {
				continue
			}

			childHint := childExternalId.String()
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

			nodeId := le.Entry.NodeID
			branchId := le.Entry.BranchID
			runEventLogEntrySignalKey := fmt.Sprintf("durable_run:%s:%d:%d", task.ExternalID.String(), branchId, nodeId)
			taskId := task.ID

			replayMatchOpts = append(replayMatchOpts, CreateMatchOpts{
				Kind:                         sqlcv1.V1MatchKindSIGNAL,
				Conditions:                   conditions,
				SignalTaskId:                 &taskId,
				SignalTaskInsertedAt:         task.InsertedAt,
				SignalExternalId:             &childExternalId,
				SignalTaskExternalId:         &task.ExternalID,
				SignalKey:                    &runEventLogEntrySignalKey,
				DurableEventLogEntryNodeId:   &nodeId,
				DurableEventLogEntryBranchId: &branchId,
			})
		}

		if len(replayMatchOpts) > 0 {
			if matchErr := r.createEventMatches(ctx, tx, tenantId, replayMatchOpts); matchErr != nil {
				return nil, nil, nil, fmt.Errorf("failed to register replayed run completion matches: %w", matchErr)
			}
		}
	}

	maxNodeId := int64(0)
	for _, le := range logEntries {
		if le.Entry.NodeID > maxNodeId {
			maxNodeId = le.Entry.NodeID
		}
	}

	if maxNodeId > 0 {
		_, err = r.queries.UpdateLogFile(ctx, tx, sqlcv1.UpdateLogFileParams{
			NodeId:                sqlchelpers.ToBigInt(&maxNodeId),
			Durabletaskid:         task.ID,
			Durabletaskinsertedat: task.InsertedAt,
		})

		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to update latest node id: %w", err)
		}
	}

	if err := commit(ctx); err != nil {
		return nil, nil, nil, err
	}

	return logEntries, nodeIdBranchIdToTriggerOpts, childrenToReplay, nil
}

func (r *durableEventsRepository) TriggerPendingRunEntries(ctx context.Context, tenantId uuid.UUID, tasks []TriggerPendingRunEntriesOpt) ([]*V1TaskWithPayload, []*DAGWithData, []CELEvaluationFailure, error) {
	ctx, span := telemetry.NewSpan(ctx, "trigger-pending-durable-run-entries")
	defer span.End()

	optTx, err := r.PrepareOptimisticTx(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to prepare tx: %w", err)
	}

	defer optTx.Rollback()

	tx := optTx.tx

	nodesToClaim := make([]DurableTaskEventLogEntryKey, 0)

	for _, t := range tasks {
		for _, p := range t.PendingRuns {
			nodesToClaim = append(nodesToClaim, DurableTaskEventLogEntryKey{
				NodeID:                p.NodeId,
				BranchID:              p.BranchId,
				DurableTaskID:         t.Task.ID,
				DurableTaskInsertedAt: t.Task.InsertedAt,
				DurableTaskExternalId: t.Task.ExternalID,
			})
		}
	}

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "nodes_to_claim", Value: len(nodesToClaim)})

	claimedSet, err := r.claimDurableEventLogEntriesForTrigger(ctx, tx, nodesToClaim)

	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to claim durable run entries for trigger: %w", err)
	}

	if len(claimedSet) == 0 {
		return nil, nil, nil, nil
	}

	triggerOpts := make([]*WorkflowNameTriggerOpts, 0, len(claimedSet))
	externalIdToEventLogEntryPK := make(map[uuid.UUID]DurableTaskEventLogEntryKey, len(claimedSet))

	for _, t := range tasks {
		for _, p := range t.PendingRuns {
			k := DurableTaskEventLogEntryKey{
				NodeID:                p.NodeId,
				BranchID:              p.BranchId,
				DurableTaskID:         t.Task.ID,
				DurableTaskInsertedAt: t.Task.InsertedAt,
				DurableTaskExternalId: t.Task.ExternalID,
			}

			if _, ok := claimedSet[k.claim()]; !ok {
				continue
			}

			triggerOpts = append(triggerOpts, p.TriggerOpts)
			externalIdToEventLogEntryPK[p.TriggerOpts.ExternalId] = k
		}
	}

	createdTasks, createdDags, _, celFailures, triggerStorePayloadOpts, err := r.triggerFromWorkflowNames(ctx, optTx, tenantId, triggerOpts)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to trigger workflows: %w", err)
	}

	if len(triggerStorePayloadOpts) > 0 {
		if err := r.payloadStore.Store(ctx, tx, triggerStorePayloadOpts...); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to store payloads for triggered runs: %w", err)
		}
	}

	createMatchOpts := make([]CreateMatchOpts, 0, len(createdTasks)+len(createdDags))

	dagExternalIds := make(map[uuid.UUID]struct{}, len(createdDags))

	for _, dag := range createdDags {
		// operator runs are signaled by their orchestrator task's completion, not
		// by per-step conditions, so they're handled in the created tasks loop below
		if dag.IsOperatorRun {
			continue
		}

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

		eventLogEntryKey := externalIdToEventLogEntryPK[ct.ExternalID]

		nodeId := eventLogEntryKey.NodeID
		branchId := eventLogEntryKey.BranchID
		taskExternalId := eventLogEntryKey.DurableTaskExternalId
		taskId := eventLogEntryKey.DurableTaskID

		runEventLogEntrySignalKey := fmt.Sprintf("durable_run:%s:%d:%d", taskExternalId.String(), branchId, nodeId)

		createMatchOpts = append(createMatchOpts, CreateMatchOpts{
			Kind:                         sqlcv1.V1MatchKindSIGNAL,
			Conditions:                   conditions,
			SignalTaskId:                 &taskId,
			SignalTaskInsertedAt:         eventLogEntryKey.DurableTaskInsertedAt,
			SignalExternalId:             &ct.ExternalID,
			SignalTaskExternalId:         &taskExternalId,
			SignalKey:                    &runEventLogEntrySignalKey,
			DurableEventLogEntryNodeId:   &nodeId,
			DurableEventLogEntryBranchId: &branchId,
		})
	}

	for _, dag := range createdDags {
		if dag.IsOperatorRun {
			continue
		}

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

		eventLogEntryKey := externalIdToEventLogEntryPK[dag.ExternalID]

		nodeId := eventLogEntryKey.NodeID
		branchId := eventLogEntryKey.BranchID
		taskExternalId := eventLogEntryKey.DurableTaskExternalId

		runEventLogEntrySignalKey := fmt.Sprintf("durable_run:%s:%d:%d", taskExternalId.String(), branchId, nodeId)

		taskId := eventLogEntryKey.DurableTaskID
		dagExternalId := dag.ExternalID

		createMatchOpts = append(createMatchOpts, CreateMatchOpts{
			Kind:                         sqlcv1.V1MatchKindSIGNAL,
			Conditions:                   conditions,
			SignalTaskId:                 &taskId,
			SignalTaskInsertedAt:         eventLogEntryKey.DurableTaskInsertedAt,
			SignalExternalId:             &dagExternalId,
			SignalTaskExternalId:         &taskExternalId,
			SignalKey:                    &runEventLogEntrySignalKey,
			DurableEventLogEntryNodeId:   &nodeId,
			DurableEventLogEntryBranchId: &branchId,
		})
	}

	if len(createMatchOpts) > 0 {
		if err := r.createEventMatches(ctx, tx, tenantId, createMatchOpts); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to register run completion matches: %w", err)
		}
	}

	if err := optTx.Commit(ctx); err != nil {
		return nil, nil, nil, err
	}

	return createdTasks, createdDags, celFailures, nil
}

func (r *durableEventsRepository) triggerPendingWaitFor(ctx context.Context, tenantId uuid.UUID, task *sqlcv1.FlattenExternalIdsRow, branchId, nodeId int64, waitForConditions []CreateExternalSignalConditionOpt) error {
	ctx, span := telemetry.NewSpan(ctx, "trigger-pending-durable-wait-for")
	defer span.End()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		return fmt.Errorf("failed to prepare tx: %w", err)
	}
	defer rollback()

	claimedSet, err := r.claimDurableEventLogEntriesForTrigger(ctx, tx, []DurableTaskEventLogEntryKey{{
		NodeID:                nodeId,
		BranchID:              branchId,
		DurableTaskID:         task.ID,
		DurableTaskInsertedAt: task.InsertedAt,
		DurableTaskExternalId: task.ExternalID,
	}})
	if err != nil {
		return fmt.Errorf("failed to claim durable wait for entry for trigger: %w", err)
	}

	if len(claimedSet) == 0 {
		return nil
	}

	if err := r.handleWaitFor(ctx, tx, tenantId, branchId, nodeId, waitForConditions, task); err != nil {
		return err
	}

	return commit(ctx)
}

type DurableTaskEventLogEntryKey struct {
	DurableTaskID         int64
	DurableTaskInsertedAt pgtype.Timestamptz
	DurableTaskExternalId uuid.UUID
	NodeID                int64
	BranchID              int64
}

type durableEventLogEntryClaim struct {
	DurableTaskID int64
	NodeID        int64
	BranchID      int64
}

func (k DurableTaskEventLogEntryKey) claim() durableEventLogEntryClaim {
	return durableEventLogEntryClaim{
		DurableTaskID: k.DurableTaskID,
		NodeID:        k.NodeID,
		BranchID:      k.BranchID,
	}
}

func (r *durableEventsRepository) claimDurableEventLogEntriesForTrigger(ctx context.Context, tx sqlcv1.DBTX, nodesToClaim []DurableTaskEventLogEntryKey) (map[durableEventLogEntryClaim]struct{}, error) {
	nodeIds := make([]int64, len(nodesToClaim))
	branchIds := make([]int64, len(nodesToClaim))
	durableTaskIds := make([]int64, len(nodesToClaim))
	durableTaskInsertedAts := make([]pgtype.Timestamptz, len(nodesToClaim))

	for i, k := range nodesToClaim {
		nodeIds[i] = k.NodeID
		branchIds[i] = k.BranchID
		durableTaskIds[i] = k.DurableTaskID
		durableTaskInsertedAts[i] = k.DurableTaskInsertedAt
	}

	claimed, err := r.queries.ClaimDurableEventLogEntriesForTrigger(ctx, tx, sqlcv1.ClaimDurableEventLogEntriesForTriggerParams{
		Durabletaskids:         durableTaskIds,
		Durabletaskinsertedats: durableTaskInsertedAts,
		Nodeids:                nodeIds,
		Branchids:              branchIds,
	})

	if err != nil {
		return nil, err
	}

	claimedSet := make(map[durableEventLogEntryClaim]struct{}, len(claimed))

	for _, c := range claimed {
		claimedSet[durableEventLogEntryClaim{
			DurableTaskID: c.DurableTaskID,
			NodeID:        c.NodeID,
			BranchID:      c.BranchID,
		}] = struct{}{}
	}

	return claimedSet, nil
}

func (r *durableEventsRepository) handleEventLookback(ctx context.Context, tenantId uuid.UUID, task *sqlcv1.FlattenExternalIdsRow, initialWaitForResult *IngestWaitForResult, waitForConditions []CreateExternalSignalConditionOpt) (*IngestWaitForResult, error) {
	if initialWaitForResult.IsSatisfied {
		return initialWaitForResult, nil
	}

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

	targetMatchID, err := r.queries.GetActiveMatchForDurableWait(ctx, lookbackTx, sqlcv1.GetActiveMatchForDurableWaitParams{
		Tenantid:              tenantId,
		Eventkeys:             lookbackParams.Keys,
		Eventscopes:           lookbackParams.Scopes,
		Durabletaskid:         task.ID,
		Durabletaskinsertedat: task.InsertedAt,
		Durabletaskexternalid: task.ExternalID,
		Nodeid:                initialWaitForResult.NodeId,
		Branchid:              initialWaitForResult.BranchId,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return initialWaitForResult, nil
		}

		return nil, fmt.Errorf("failed to find active match for durable wait: %w", err)
	}

	retrievePayloadOpts := make([]RetrievePayloadOpts, 0, len(previousEventsFound))

	for _, row := range previousEventsFound {
		retrievePayloadOpts = append(retrievePayloadOpts, RetrievePayloadOpts{
			Id:         row.ID,
			InsertedAt: row.SeenAt,
			Type:       sqlcv1.V1PayloadTypeUSEREVENTINPUT,
			TenantId:   tenantId,
			ExternalId: row.ExternalID,
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
			ExternalId: row.ExternalID,
		}

		payload, ok := retrieveOptsToPayload[retrieveOpts]

		if !ok {
			r.l.Warn().Ctx(ctx).Msgf("payload not found for recent user event with id %d and seen_at %s", row.ID, row.SeenAt.Time)
			payload = nil
		}

		var resourceHint *string
		if row.Scope.Valid {
			resourceHint = &row.Scope.String
		}

		retroCandidates = append(retroCandidates, CandidateEventMatch{
			ID:             row.ExternalID,
			EventTimestamp: row.SeenAt.Time,
			Key:            row.Key,
			ResourceHint:   resourceHint,
			Data:           payload,
		})
	}

	retroMatchResults, err := r.processEventMatchesForTarget(ctx, lookbackTx, tenantId, retroCandidates, sqlcv1.V1EventTypeUSER, &targetMatchID)

	if err != nil {
		return nil, fmt.Errorf("failed to process retroactive event matches: %w", err)
	}

	if len(retroMatchResults.SatisfiedDurableEventLogEntries) > 1 {
		return nil, fmt.Errorf("expected at most one satisfied durable wait from targeted lookback, got %d", len(retroMatchResults.SatisfiedDurableEventLogEntries))
	}

	if len(retroMatchResults.SatisfiedDurableEventLogEntries) == 1 {
		entry := retroMatchResults.SatisfiedDurableEventLogEntries[0]
		if entry.DurableTaskExternalId != task.ExternalID || entry.NodeId != initialWaitForResult.NodeId || entry.BranchId != initialWaitForResult.BranchId {
			return nil, fmt.Errorf("targeted lookback satisfied an unexpected durable wait")
		}

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
			SatisfiedOrder:  entry.SatisfiedOrder,
		}, nil
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

func (r *durableEventsRepository) latestSatisfiedOrderBeforeBranchPoint(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, nodeId, branchId int64, task *sqlcv1.FlattenExternalIdsRow, nextBranchIdToBranchPoint map[int64]*sqlcv1.V1DurableEventLogBranchPoint) (int64, error) {
	entries, err := r.queries.ListDurableEventLogEntriesBeforeNode(ctx, tx, sqlcv1.ListDurableEventLogEntriesBeforeNodeParams{
		Durabletaskid:         task.ID,
		Durabletaskinsertedat: task.InsertedAt,
		Tenantid:              tenantId,
		Nodeid:                nodeId,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to list durable event log entries before branch point: %w", err)
	}

	return latestSatisfiedOrderForBranchPrefix(entries, branchId, nextBranchIdToBranchPoint), nil
}

func latestSatisfiedOrderForBranchPrefix(entries []*sqlcv1.V1DurableEventLogEntry, branchId int64, nextBranchIdToBranchPoint map[int64]*sqlcv1.V1DurableEventLogBranchPoint) int64 {
	var latest int64

	for _, entry := range entries {
		if entry.BranchID != resolveBranchForNode(entry.NodeID, branchId, nextBranchIdToBranchPoint) {
			continue
		}

		if entry.SatisfiedOrder.Valid && entry.SatisfiedOrder.Int64 > latest {
			latest = entry.SatisfiedOrder.Int64
		}
	}

	return latest
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
	return r.handleBranch(ctx, tenantId, nodeId, branchId, task, nil)
}

func (r *durableEventsRepository) handleBranch(ctx context.Context, tenantId uuid.UUID, nodeId, branchId int64, task *sqlcv1.FlattenExternalIdsRow, replayChildExternalIds []uuid.UUID) (*HandleBranchResult, error) {
	optTx, err := r.PrepareOptimisticTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare tx: %w", err)
	}
	defer optTx.Rollback()

	tx := optTx.tx

	logFile, nextBranchIdToBranchPoint, err := r.getAndLockLogFile(ctx, tx, tenantId, task.ID, task.InsertedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to lock log file: %w", err)
	}

	newBranchId := logFile.LatestBranchID + 1
	zero := int64(0)

	latestSatisfiedOrder, err := r.latestSatisfiedOrderBeforeBranchPoint(ctx, tx, tenantId, nodeId, branchId, task, nextBranchIdToBranchPoint)
	if err != nil {
		return nil, err
	}

	logFile, err = r.queries.UpdateLogFile(ctx, tx, sqlcv1.UpdateLogFileParams{
		BranchId:             sqlchelpers.ToBigInt(&newBranchId),
		NodeId:               sqlchelpers.ToBigInt(&zero),
		LatestSatisfiedOrder: sqlchelpers.ToBigInt(&latestSatisfiedOrder),

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
		ReplayChildExternalIds: replayChildExternalIds,
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

func (r *durableEventsRepository) HandleBranchForDAGReplay(ctx context.Context, tenantId uuid.UUID, task *sqlcv1.FlattenExternalIdsRow, forcedChildExternalIds []uuid.UUID) (*HandleBranchResult, error) {
	logFiles, err := r.queries.GetDurableTaskLogFiles(ctx, r.pool, sqlcv1.GetDurableTaskLogFilesParams{
		Durabletaskids:         []int64{task.ID},
		Durabletaskinsertedats: []pgtype.Timestamptz{task.InsertedAt},
		Tenantids:              []uuid.UUID{tenantId},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get log file for replay branch: %w", err)
	}
	if len(logFiles) == 0 {
		return nil, nil
	}

	return r.handleBranch(ctx, tenantId, 0, logFiles[0].LatestBranchID, task, forcedChildExternalIds)
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
		taskInsertedAts[i] = sqlchelpers.TimestamptzFromUnixMicros(t.InsertedAtUnixMicros)
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
			ID:                   logFile.DurableTaskID,
			InsertedAtUnixMicros: logFile.DurableTaskInsertedAt.Time.UnixMicro(),
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
