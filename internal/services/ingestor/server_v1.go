package ingestor

import (
	"context"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func (i *IngestorImpl) putStreamEventV1(ctx context.Context, tenant *dbsqlc.Tenant, req *contracts.PutStreamEventRequest) (*contracts.PutStreamEventResponse, error) {
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// get single task
	task, err := i.getSingleTask(ctx, tenantId, req.StepRunId, false)

	if err != nil {
		return nil, err
	}

	msg, err := msgqueue.NewTenantMessage(
		tenantId,
		"task-stream-event",
		true,
		false,
		tasktypes.StreamEventPayload{
			WorkflowRunId: sqlchelpers.UUIDToStr(task.WorkflowRunID),
			StepRunId:     req.StepRunId,
			CreatedAt:     req.CreatedAt.AsTime(),
			Payload:       req.Message,
		},
	)

	if err != nil {
		return nil, err
	}

	q := msgqueue.TenantEventConsumerQueue(tenantId)

	err = i.mqv1.SendMessage(ctx, q, msg)

	if err != nil {
		return nil, err
	}

	return &contracts.PutStreamEventResponse{}, nil
}

func (i *IngestorImpl) getSingleTask(ctx context.Context, tenantId, taskExternalId string, skipCache bool) (*sqlcv1.FlattenExternalIdsRow, error) {
	return i.repov1.Tasks().GetTaskByExternalId(ctx, tenantId, taskExternalId, skipCache)
}
