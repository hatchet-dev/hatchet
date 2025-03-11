package ingestor

import (
	"context"
	"time"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func (i *IngestorImpl) putLogV1(ctx context.Context, tenant *dbsqlc.Tenant, req *contracts.PutLogRequest) (*contracts.PutLogResponse, error) {
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	task, err := i.getSingleTask(ctx, tenantId, req.StepRunId, false)

	if err != nil {
		return nil, err
	}

	var createdAt *time.Time

	if t := req.CreatedAt.AsTime(); !t.IsZero() {
		createdAt = &t
	}

	var metadata []byte

	if req.Metadata != "" {
		metadata = []byte(req.Metadata)
	}

	opts := &v1.CreateLogLineOpts{
		TaskId:         task.ID,
		TaskInsertedAt: task.InsertedAt,
		CreatedAt:      createdAt,
		Message:        req.Message,
		Level:          req.Level,
		Metadata:       metadata,
	}

	if apiErrors, err := i.v.ValidateAPI(opts); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %s", apiErrors.String())
	}

	err = i.repov1.Logs().PutLog(ctx, tenantId, opts)

	if err != nil {
		return nil, err
	}

	return &contracts.PutLogResponse{}, nil
}
