package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/enumerables"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const dagStepSpawnStepIndex = -1

func getChildSignalEventKey(parentExternalId uuid.UUID, stepIndex, childIndex int64, childKeyArg *string) string {
	childKey := fmt.Sprintf("%d", childIndex)

	if childKeyArg != nil {
		childKey = *childKeyArg
	}

	return fmt.Sprintf("%s.%d.%s", parentExternalId, stepIndex, childKey)
}

type WorkflowNameTriggerOpts struct {
	*TriggerTaskData

	ExternalId uuid.UUID

	// Whether to skip the creation of the child workflow
	ShouldSkip bool

	// Whether the workflow has an idempotency key expression configured.
	HasIdempotencyKey bool

	// ReplayOrphanedChildren controls what happens when the child-spawn dedupe resolves to a
	// child whose log entry is only reachable on a branch orphaned by a replay reset. When set
	// (DAG operator steps), the existing child run is re-executed in place, preserving task
	// identity; otherwise a fresh child run is spawned on the active branch.
	ReplayOrphanedChildren bool

	// ParentReExecuted marks a DAG step whose parent re-executed this invocation, so it must
	// re-run during a replay even when its own orphaned entry is satisfied.
	ParentReExecuted bool

	// IsDagStepTrigger marks a run triggered by the DAG operator on behalf of one of its steps.
	IsDagStepTrigger bool `json:"is_dag_step_trigger"`
}

func (g *WorkflowNameTriggerOpts) childSpawnKey() string {
	if g.ParentExternalId == nil || g.ChildIndex == nil {
		return ""
	}

	stepIndex := int64(0)

	if g.IsDagStepTrigger {
		stepIndex = dagStepSpawnStepIndex
	}

	return getChildSignalEventKey(*g.ParentExternalId, stepIndex, *g.ChildIndex, g.ChildKey)
}

type ChildWorkflowSignalCreatedData struct {
	// The external id of the target child task
	ChildExternalId uuid.UUID `json:"external_id"`

	// The external id of the parent task
	ParentExternalId uuid.UUID `json:"parent_external_id"`

	// The index of the child task
	ChildIndex int64 `json:"child_index"`

	// The key of the child task
	ChildKey *string `json:"child_key"`
}

func newChildWorkflowSignalCreatedData(childExternalId uuid.UUID, opt *WorkflowNameTriggerOpts) *ChildWorkflowSignalCreatedData {
	return &ChildWorkflowSignalCreatedData{
		ChildExternalId:  childExternalId,
		ParentExternalId: *opt.ParentExternalId,
		ChildIndex:       *opt.ChildIndex,
		ChildKey:         opt.ChildKey,
	}
}

func newChildWorkflowSignalCreatedDataFromBytes(b []byte) (*ChildWorkflowSignalCreatedData, error) {
	var c ChildWorkflowSignalCreatedData

	err := json.Unmarshal(b, &c)

	if err != nil {
		return nil, err
	}

	return &c, nil
}

func (c *ChildWorkflowSignalCreatedData) Bytes() []byte {
	b, _ := json.Marshal(c) // nolint: errcheck
	return b
}

type childSpawn struct {
	parentTaskInsertedAt pgtype.Timestamptz
	spawnKey             string
	parentTaskId         int64
	tenantId             uuid.UUID
}

func deterministicChildExternalId(parentExternalId uuid.UUID, spawnKey string) uuid.UUID {
	return uuid.NewSHA1(parentExternalId, []byte(spawnKey))
}

// children spawned before deterministic ids were enabled keep the random id recorded on a SIGNAL_CREATED event
func (s *sharedRepository) listChildExternalIdsRecordedOnSignalCreatedEvents(ctx context.Context, db sqlcv1.DBTX, childSpawns []childSpawn) (map[taskIdSpawnKey]uuid.UUID, error) {
	params := sqlcv1.ListChildExternalIdsRecordedOnSignalCreatedEventsParams{
		Tenantids:               make([]uuid.UUID, 0, len(childSpawns)),
		Parenttaskids:           make([]int64, 0, len(childSpawns)),
		Parenttaskinsertedats:   make([]pgtype.Timestamptz, 0, len(childSpawns)),
		Eventkeys:               make([]string, 0, len(childSpawns)),
		Minparenttaskinsertedat: sqlchelpers.TimestamptzFromTime(time.Now().UTC()),
	}

	for _, spawn := range childSpawns {
		params.Tenantids = append(params.Tenantids, spawn.tenantId)
		params.Parenttaskids = append(params.Parenttaskids, spawn.parentTaskId)
		params.Parenttaskinsertedats = append(params.Parenttaskinsertedats, spawn.parentTaskInsertedAt)
		params.Eventkeys = append(params.Eventkeys, spawn.spawnKey)

		if spawn.parentTaskInsertedAt.Time.Before(params.Minparenttaskinsertedat.Time) {
			params.Minparenttaskinsertedat = spawn.parentTaskInsertedAt
		}
	}

	rows, err := s.queries.ListChildExternalIdsRecordedOnSignalCreatedEvents(ctx, db, params)

	if err != nil {
		return nil, fmt.Errorf("failed to list child external ids recorded on signal created events: %w", err)
	}

	return enumerables.ToMap(
		rows,
		func(row *sqlcv1.ListChildExternalIdsRecordedOnSignalCreatedEventsRow) taskIdSpawnKey {
			return taskIdSpawnKey{taskId: row.TaskID, spawnKey: row.EventKey.String}
		},
		func(row *sqlcv1.ListChildExternalIdsRecordedOnSignalCreatedEventsRow) uuid.UUID {
			return *row.ChildExternalID
		},
	), nil
}

func (s *sharedRepository) populateDeterministicExternalIdsForWorkflow(ctx context.Context, tenantId uuid.UUID, opts []*WorkflowNameTriggerOpts) error {
	optsByKey := make(map[taskIdSpawnKey][]*WorkflowNameTriggerOpts)
	childSpawns := make([]childSpawn, 0, len(opts))

	for _, opt := range opts {
		spawnKey := opt.childSpawnKey()

		if spawnKey == "" || opt.ParentTaskId == nil || opt.ParentTaskInsertedAt == nil {
			opt.ExternalId = uuid.New()
			continue
		}

		opt.ExternalId = deterministicChildExternalId(*opt.ParentExternalId, spawnKey)

		key := taskIdSpawnKey{taskId: *opt.ParentTaskId, spawnKey: spawnKey}

		// the trigger path only dedupes against runs committed earlier, not against others in this batch
		if _, seen := optsByKey[key]; seen {
			opt.ShouldSkip = true
		} else {
			childSpawns = append(childSpawns, childSpawn{
				tenantId:             tenantId,
				parentTaskId:         *opt.ParentTaskId,
				parentTaskInsertedAt: sqlchelpers.TimestamptzFromTime(*opt.ParentTaskInsertedAt),
				spawnKey:             spawnKey,
			})
		}

		optsByKey[key] = append(optsByKey[key], opt)
	}

	if len(childSpawns) == 0 {
		return nil
	}

	db := s.pool.ForTenant(tenantId)

	recordedIdByKey, err := s.listChildExternalIdsRecordedOnSignalCreatedEvents(ctx, db, childSpawns)

	if err != nil {
		return err
	}

	for key, recordedId := range recordedIdByKey {
		for _, opt := range optsByKey[key] {
			opt.ExternalId = recordedId
			opt.ShouldSkip = true
		}
	}

	deterministicChildExternalIds := make([]uuid.UUID, 0, len(optsByKey))

	for key, optsForKey := range optsByKey {
		if _, wasRecorded := recordedIdByKey[key]; !wasRecorded {
			deterministicChildExternalIds = append(deterministicChildExternalIds, optsForKey[0].ExternalId)
		}
	}

	if len(deterministicChildExternalIds) == 0 {
		return nil
	}

	alreadySpawnedChildren, err := s.queries.LookupExternalIds(ctx, db, sqlcv1.LookupExternalIdsParams{
		Tenantid:    tenantId,
		Externalids: deterministicChildExternalIds,
	})

	if err != nil {
		return fmt.Errorf("failed to list child runs which were already spawned: %w", err)
	}

	alreadySpawnedChildExternalIds := enumerables.ToSet(alreadySpawnedChildren, func(lookupRow *sqlcv1.V1LookupTable) uuid.UUID {
		return lookupRow.ExternalID
	})

	// a skipped child never reaches the trigger path, which would meter it as a new task run
	for _, opt := range opts {
		if _, wasAlreadySpawned := alreadySpawnedChildExternalIds[opt.ExternalId]; wasAlreadySpawned {
			opt.ShouldSkip = true
		}
	}

	return nil
}

// GenerateExternalIdsForWorkflow generates external ids and additional looks up child workflows and whether they
// already exist.
func (s *sharedRepository) PopulateExternalIdsForWorkflow(ctx context.Context, tenantId uuid.UUID, opts []*WorkflowNameTriggerOpts) error {
	if s.childRunDirectSpawn {
		return s.populateDeterministicExternalIdsForWorkflow(ctx, tenantId, opts)
	}

	db := s.pool.ForTenant(tenantId)

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, db, s.l)

	if err != nil {
		return err
	}

	defer rollback()

	if err := s.populateExternalIdsForWorkflow(ctx, tx, tenantId, opts); err != nil {
		return err
	}

	return commit(ctx)
}

func (s *sharedRepository) populateExternalIdsForWorkflow(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, opts []*WorkflowNameTriggerOpts) error {
	// get child workflow data first
	optsWithParents := make([]*WorkflowNameTriggerOpts, 0, len(opts))

	for i := range opts {
		opt := opts[i] // we don't want a copy here, we want the actual pointer as we modify in-place

		if opt.ParentExternalId != nil && opt.ChildIndex != nil {
			optsWithParents = append(optsWithParents, opt)
		} else {
			opt.ExternalId = uuid.New()
		}
	}

	if len(optsWithParents) > 0 {
		err := s.generateExternalIdsForChildWorkflows(ctx, tx, tenantId, optsWithParents)

		if err != nil {
			return err
		}
	}

	return nil
}

func (s *sharedRepository) generateExternalIdsForChildWorkflows(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, opts []*WorkflowNameTriggerOpts) error {
	externalIds := make([]uuid.UUID, 0, len(opts))
	spawnKeyToOpt := make(map[string]*WorkflowNameTriggerOpts)

	for i, opt := range opts {
		externalIds = append(externalIds, *opt.ParentExternalId)

		spawnKeyToOpt[opt.childSpawnKey()] = opts[i] // we don't want a copy here, we want the actual pointer as we modify in-place
	}

	gotTasks, err := s.queries.LookupExternalIds(ctx, tx, sqlcv1.LookupExternalIdsParams{
		Externalids: externalIds,
		Tenantid:    tenantId,
	})

	if err != nil {
		return err
	}

	externalIdToLookupRow := enumerables.KeyBy(gotTasks, func(task *sqlcv1.V1LookupTable) uuid.UUID {
		return task.ExternalID
	})

	eventTaskIds := make([]int64, 0, len(gotTasks))
	eventTaskInsertedAts := make([]pgtype.Timestamptz, 0, len(gotTasks))
	eventKeys := make([]string, 0, len(gotTasks))

	for _, opt := range opts {
		lookupRow, ok := externalIdToLookupRow[*opt.ParentExternalId]

		if !ok {
			continue
		}

		eventTaskIds = append(eventTaskIds, lookupRow.TaskID.Int64)
		eventTaskInsertedAts = append(eventTaskInsertedAts, lookupRow.InsertedAt)
		eventKeys = append(eventKeys, opt.childSpawnKey())
	}

	lockedEvents, err := s.queries.LockSignalCreatedEvents(ctx, tx, sqlcv1.LockSignalCreatedEventsParams{
		Tenantid:        tenantId,
		Taskids:         eventTaskIds,
		Taskinsertedats: eventTaskInsertedAts,
		Eventkeys:       eventKeys,
	})

	if err != nil {
		return err
	}

	// only need to hit the payload store for events created before the child_external_id column existed
	retrievePayloadOpts := make([]PayloadRef[ChildWorkflowSignalCreatedShape], 0, len(lockedEvents))

	for _, lockedEvent := range lockedEvents {
		if lockedEvent.ChildExternalID != nil {
			continue
		}

		retrievePayloadOpts = append(retrievePayloadOpts, PayloadRef[ChildWorkflowSignalCreatedShape]{
			Id:         lockedEvent.ID,
			InsertedAt: lockedEvent.InsertedAt,
			TenantId:   tenantId,
			ExternalId: lockedEvent.ExternalID,
		})
	}

	var payloads map[PayloadRef[ChildWorkflowSignalCreatedShape]]Payload[ChildWorkflowSignalCreatedShape]

	if len(retrievePayloadOpts) > 0 {
		payloads, err = s.payloadStore.ChildWorkflowSignalCreatedEvents().Retrieve(ctx, tx, retrievePayloadOpts...)

		if err != nil {
			return err
		}
	}

	// for each locked event, write the correct external id to the opt
	for _, lockedEvent := range lockedEvents {
		opt := spawnKeyToOpt[lockedEvent.EventKey.String]

		var childExternalId uuid.UUID

		if lockedEvent.ChildExternalID != nil {
			childExternalId = *lockedEvent.ChildExternalID
		} else {
			payload := RetrievePayloadOrFallback(
				payloads,
				PayloadRef[ChildWorkflowSignalCreatedShape]{
					Id:         lockedEvent.ID,
					InsertedAt: lockedEvent.InsertedAt,
					TenantId:   tenantId,
					ExternalId: lockedEvent.ExternalID,
				},
				lockedEvent.Data,
			)

			c, err := DecodeChildWorkflowSignalCreatedPayload(payload)

			if err != nil {
				return err
			}

			childExternalId = c.ChildExternalId
		}

		opt.ExternalId = childExternalId
		opt.ShouldSkip = true
	}

	taskIds := make([]TaskIdInsertedAtRetryCount, 0, len(opts))
	taskExternalIds := make([]uuid.UUID, 0, len(opts))
	datas := make([][]byte, 0, len(opts))
	newEventKeys := make([]string, 0, len(opts))
	childExternalIds := make([]*uuid.UUID, 0, len(opts))

	// for all other opts, write the events to the database
	for i := range opts {
		opt := opts[i] // we don't want a copy here, we want the actual pointer as we modify in-place
		lookupRow, ok := externalIdToLookupRow[*opt.ParentExternalId]

		if !ok {
			continue
		}

		if opt.ShouldSkip {
			continue
		}

		generatedId := deterministicChildExternalId(*opt.ParentExternalId, opt.childSpawnKey())
		opt.ExternalId = generatedId

		data := newChildWorkflowSignalCreatedData(generatedId, opt)

		taskIds = append(taskIds, TaskIdInsertedAtRetryCount{
			Id:         lookupRow.TaskID.Int64,
			InsertedAt: lookupRow.InsertedAt,
			RetryCount: -1,
		})

		taskExternalIds = append(taskExternalIds, lookupRow.ExternalID)
		datas = append(datas, data.Bytes())
		newEventKeys = append(newEventKeys, opt.childSpawnKey())
		childExternalIds = append(childExternalIds, &generatedId)
	}

	// create the relevant events
	_, err = s.createTaskEvents(
		ctx,
		tx,
		tenantId,
		taskIds,
		taskExternalIds,
		datas,
		makeEventTypeArr(sqlcv1.V1TaskEventTypeSIGNALCREATED, len(taskIds)),
		newEventKeys,
		nil,
		childExternalIds,
		nil,
	)

	if err != nil {
		return err
	}

	return nil
}
