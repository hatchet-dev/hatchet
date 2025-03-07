package v1

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func getChildSignalEventKey(parentExternalId string, stepIndex, childIndex int64, childKeyArg *string) string {
	childKey := fmt.Sprintf("%d", childIndex)

	if childKeyArg != nil {
		childKey = *childKeyArg
	}

	return fmt.Sprintf("%s.%d.%s", parentExternalId, stepIndex, childKey)
}

type WorkflowNameTriggerOpts struct {
	*TriggerTaskData

	ExternalId string

	// Whether to skip the creation of the child workflow
	ShouldSkip bool
}

func (g *WorkflowNameTriggerOpts) childSpawnKey() string {
	if g.ParentExternalId == nil || g.ChildIndex == nil {
		return ""
	}

	return getChildSignalEventKey(*g.ParentExternalId, 0, *g.ChildIndex, g.ChildKey)
}

type ChildWorkflowSignalCreatedData struct {
	// The external id of the target child task
	ChildExternalId string `json:"external_id"`

	// The external id of the parent task
	ParentExternalId string `json:"parent_external_id"`

	// The index of the child task
	ChildIndex int64 `json:"child_index"`

	// The key of the child task
	ChildKey *string `json:"child_key"`
}

func newChildWorkflowSignalCreatedData(childExternalId string, opt *WorkflowNameTriggerOpts) *ChildWorkflowSignalCreatedData {
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

// GenerateExternalIdsForWorkflow generates external ids and additional looks up child workflows and whether they
// already exist.
func (s *sharedRepository) PopulateExternalIdsForWorkflow(ctx context.Context, tenantId string, opts []*WorkflowNameTriggerOpts) error {
	// get child workflow data first
	optsWithParents := make([]*WorkflowNameTriggerOpts, 0, len(opts))

	for i := range opts {
		opt := opts[i] // we don't want a copy here, we want the actual pointer as we modify in-place

		if opt.ParentExternalId != nil && opt.ChildIndex != nil {
			optsWithParents = append(optsWithParents, opt)
		} else {
			opt.ExternalId = uuid.NewString()
		}
	}

	if len(optsWithParents) > 0 {
		err := s.generateExternalIdsForChildWorkflows(ctx, tenantId, optsWithParents)

		if err != nil {
			return err
		}
	}

	return nil
}

func (s *sharedRepository) generateExternalIdsForChildWorkflows(ctx context.Context, tenantId string, opts []*WorkflowNameTriggerOpts) error {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, s.pool, s.l, 5000)

	if err != nil {
		return err
	}

	defer rollback()

	externalIds := make([]pgtype.UUID, 0, len(opts))
	spawnKeyToOpt := make(map[string]*WorkflowNameTriggerOpts)

	for i, opt := range opts {
		externalIds = append(externalIds, sqlchelpers.UUIDFromStr(*opt.ParentExternalId))

		spawnKeyToOpt[opt.childSpawnKey()] = opts[i] // we don't want a copy here, we want the actual pointer as we modify in-place
	}

	gotTasks, err := s.queries.LookupExternalIds(ctx, tx, sqlcv1.LookupExternalIdsParams{
		Externalids: externalIds,
		Tenantid:    sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return err
	}

	externalIdToLookupRow := make(map[string]*sqlcv1.V1LookupTable)

	for _, task := range gotTasks {
		externalIdToLookupRow[sqlchelpers.UUIDToStr(task.ExternalID)] = task
	}

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
		eventKeys = append(eventKeys, getChildSignalEventKey(*opt.ParentExternalId, 0, *opt.ChildIndex, opt.ChildKey))
	}

	lockedEvents, err := s.queries.LockSignalCreatedEvents(ctx, tx, sqlcv1.LockSignalCreatedEventsParams{
		Tenantid:        sqlchelpers.UUIDFromStr(tenantId),
		Taskids:         eventTaskIds,
		Taskinsertedats: eventTaskInsertedAts,
		Eventkeys:       eventKeys,
	})

	if err != nil {
		return err
	}

	// for each locked event, write the correct external id to the opt
	for _, lockedEvent := range lockedEvents {
		opt := spawnKeyToOpt[lockedEvent.EventKey.String]

		c, err := newChildWorkflowSignalCreatedDataFromBytes(lockedEvent.Data)

		if err != nil {
			return err
		}

		opt.ExternalId = c.ChildExternalId
		opt.ShouldSkip = true
	}

	taskIds := make([]TaskIdInsertedAtRetryCount, 0, len(opts))
	taskExternalIds := make([]string, 0, len(opts))
	datas := make([][]byte, 0, len(opts))
	newEventKeys := make([]string, 0, len(opts))

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

		generatedId := uuid.NewString()
		opt.ExternalId = generatedId

		data := newChildWorkflowSignalCreatedData(generatedId, opt)

		taskIds = append(taskIds, TaskIdInsertedAtRetryCount{
			Id:         lookupRow.TaskID.Int64,
			InsertedAt: lookupRow.InsertedAt,
			RetryCount: -1,
		})

		taskExternalIds = append(taskExternalIds, sqlchelpers.UUIDToStr(lookupRow.ExternalID))
		datas = append(datas, data.Bytes())
		newEventKeys = append(newEventKeys, getChildSignalEventKey(*opt.ParentExternalId, 0, *opt.ChildIndex, opt.ChildKey))
	}

	// create the relevant events
	_, err = s.createTaskEvents(
		ctx,
		tx,
		tenantId,
		taskIds,
		taskExternalIds,
		datas,
		sqlcv1.V1TaskEventTypeSIGNALCREATED,
		newEventKeys,
	)

	if err != nil {
		return err
	}

	if err := commit(ctx); err != nil {
		return err
	}

	return nil
}
