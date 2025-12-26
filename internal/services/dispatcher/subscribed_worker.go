package dispatcher

import (
	"context"
	"sync"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

type subscribedWorker struct {
	// stream is the server side of the RPC stream
	stream contracts.Dispatcher_ListenServer

	// finished is used to signal closure of a client subscribing goroutine
	finished chan<- bool

	sendMu sync.Mutex

	workerId string

	backlogSize   int64
	backlogSizeMu sync.Mutex

	maxBacklogSize int64

	pubBuffer *msgqueue.MQPubBuffer
}

func newSubscribedWorker(
	stream contracts.Dispatcher_ListenServer,
	fin chan<- bool,
	workerId string,
	maxBacklogSize int64,
	pubBuffer *msgqueue.MQPubBuffer,
) *subscribedWorker {
	if maxBacklogSize <= 0 {
		maxBacklogSize = 20
	}

	return &subscribedWorker{
		stream:         stream,
		finished:       fin,
		workerId:       workerId,
		maxBacklogSize: maxBacklogSize,
		pubBuffer:      pubBuffer,
	}
}

func (worker *subscribedWorker) StartStepRun(
	ctx context.Context,
	tenantId string,
	stepRun *dbsqlc.GetStepRunForEngineRow,
	stepRunData *dbsqlc.GetStepRunDataForEngineRow,
) error {
	_, span := telemetry.NewSpan(ctx, "start-step-run")
	defer span.End()

	inputBytes := []byte{}

	if stepRunData.Input != nil {
		inputBytes = stepRunData.Input
	}

	stepName := stepRun.StepReadableId.String

	action := &contracts.AssignedAction{
		TenantId:      tenantId,
		JobId:         sqlchelpers.UUIDToStr(stepRun.JobId),
		JobName:       stepRun.JobName,
		JobRunId:      sqlchelpers.UUIDToStr(stepRun.JobRunId),
		StepId:        sqlchelpers.UUIDToStr(stepRun.StepId),
		StepRunId:     sqlchelpers.UUIDToStr(stepRun.SRID),
		ActionType:    contracts.ActionType_START_STEP_RUN,
		ActionId:      stepRun.ActionId,
		ActionPayload: string(inputBytes),
		StepName:      stepName,
		WorkflowRunId: sqlchelpers.UUIDToStr(stepRun.WorkflowRunId),
		RetryCount:    stepRun.SRRetryCount,
		// NOTE: This is the default because this method is unused
		Priority: 1,
	}

	if stepRunData.AdditionalMetadata != nil {
		metadataStr := string(stepRunData.AdditionalMetadata)
		action.AdditionalMetadata = &metadataStr
	}

	if stepRunData.ChildIndex.Valid {
		action.ChildWorkflowIndex = &stepRunData.ChildIndex.Int32
	}

	if stepRunData.ChildKey.Valid {
		action.ChildWorkflowKey = &stepRunData.ChildKey.String
	}

	if stepRunData.ParentId.Valid {
		parentId := sqlchelpers.UUIDToStr(stepRunData.ParentId)
		action.ParentWorkflowRunId = &parentId
	}

	worker.sendMu.Lock()
	defer worker.sendMu.Unlock()

	return worker.stream.Send(action)
}

func (worker *subscribedWorker) StartStepRunFromBulk(
	ctx context.Context,
	tenantId string,
	stepRun *dbsqlc.GetStepRunBulkDataForEngineRow,
) error {
	_, span := telemetry.NewSpan(ctx, "start-step-run-from-bulk")
	defer span.End()

	inputBytes := []byte{}

	if stepRun.Input != nil {
		inputBytes = stepRun.Input
	}

	stepName := stepRun.StepReadableId.String

	action := &contracts.AssignedAction{
		TenantId:      tenantId,
		JobId:         sqlchelpers.UUIDToStr(stepRun.JobId),
		JobName:       stepRun.JobName,
		JobRunId:      sqlchelpers.UUIDToStr(stepRun.JobRunId),
		StepId:        sqlchelpers.UUIDToStr(stepRun.StepId),
		StepRunId:     sqlchelpers.UUIDToStr(stepRun.SRID),
		ActionType:    contracts.ActionType_START_STEP_RUN,
		ActionId:      stepRun.ActionId,
		ActionPayload: string(inputBytes),
		StepName:      stepName,
		WorkflowRunId: sqlchelpers.UUIDToStr(stepRun.WorkflowRunId),
		RetryCount:    stepRun.SRRetryCount,
		Priority:      stepRun.Priority,
	}

	if stepRun.AdditionalMetadata != nil {
		metadataStr := string(stepRun.AdditionalMetadata)
		action.AdditionalMetadata = &metadataStr
	}

	if stepRun.ChildIndex.Valid {
		action.ChildWorkflowIndex = &stepRun.ChildIndex.Int32
	}

	if stepRun.ChildKey.Valid {
		action.ChildWorkflowKey = &stepRun.ChildKey.String
	}

	if stepRun.ParentId.Valid {
		parentId := sqlchelpers.UUIDToStr(stepRun.ParentId)
		action.ParentWorkflowRunId = &parentId
	}

	worker.sendMu.Lock()
	defer worker.sendMu.Unlock()

	return worker.stream.Send(action)
}

func (worker *subscribedWorker) StartGroupKeyAction(
	ctx context.Context,
	tenantId string,
	getGroupKeyRun *dbsqlc.GetGroupKeyRunForEngineRow,
) error {
	_, span := telemetry.NewSpan(ctx, "start-group-key-action")
	defer span.End()

	inputData := getGroupKeyRun.GetGroupKeyRun.Input
	workflowRunId := sqlchelpers.UUIDToStr(getGroupKeyRun.WorkflowRunId)
	getGroupKeyRunId := sqlchelpers.UUIDToStr(getGroupKeyRun.GetGroupKeyRun.ID)

	worker.sendMu.Lock()
	defer worker.sendMu.Unlock()

	return worker.stream.Send(&contracts.AssignedAction{
		TenantId:         tenantId,
		WorkflowRunId:    workflowRunId,
		GetGroupKeyRunId: getGroupKeyRunId,
		ActionType:       contracts.ActionType_START_GET_GROUP_KEY,
		ActionId:         getGroupKeyRun.ActionId,
		ActionPayload:    string(inputData),
	})
}

func (worker *subscribedWorker) CancelStepRun(
	ctx context.Context,
	tenantId string,
	stepRun *dbsqlc.GetStepRunForEngineRow,
) error {
	_, span := telemetry.NewSpan(ctx, "cancel-step-run")
	defer span.End()

	worker.sendMu.Lock()
	defer worker.sendMu.Unlock()

	return worker.stream.Send(&contracts.AssignedAction{
		TenantId:      tenantId,
		JobId:         sqlchelpers.UUIDToStr(stepRun.JobId),
		JobName:       stepRun.JobName,
		JobRunId:      sqlchelpers.UUIDToStr(stepRun.JobRunId),
		StepId:        sqlchelpers.UUIDToStr(stepRun.StepId),
		StepRunId:     sqlchelpers.UUIDToStr(stepRun.SRID),
		ActionType:    contracts.ActionType_CANCEL_STEP_RUN,
		StepName:      stepRun.StepReadableId.String,
		WorkflowRunId: sqlchelpers.UUIDToStr(stepRun.WorkflowRunId),
		RetryCount:    stepRun.SRRetryCount,
	})
}
