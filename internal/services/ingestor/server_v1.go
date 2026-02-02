package ingestor

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (i *IngestorImpl) putStreamEventV1(ctx context.Context, tenant *sqlcv1.Tenant, req *contracts.PutStreamEventRequest) (*contracts.PutStreamEventResponse, error) {
	tenantId := tenant.ID

	taskRunId, err := uuid.Parse(req.TaskRunId)

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "task run id is not a valid uuid")
	}

	// get single task
	task, err := i.getSingleTask(ctx, tenantId, taskRunId, false)

	if err != nil {
		return nil, err
	}

	msg, err := msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDTaskStreamEvent,
		true,
		false,
		tasktypes.StreamEventPayload{
			WorkflowRunId: task.WorkflowRunID,
			TaskRunId:     taskRunId,
			CreatedAt:     req.CreatedAt.AsTime(),
			Payload:       req.Message,
			EventIndex:    req.EventIndex,
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

func (i *IngestorImpl) getSingleTask(ctx context.Context, tenantId, taskExternalId uuid.UUID, skipCache bool) (*sqlcv1.FlattenExternalIdsRow, error) {
	return i.repov1.Tasks().GetTaskByExternalId(ctx, tenantId, taskExternalId, skipCache)
}

func (i *IngestorImpl) putLogV1(ctx context.Context, tenant *sqlcv1.Tenant, req *contracts.PutLogRequest) (*contracts.PutLogResponse, error) {
	tenantId := tenant.ID
	taskRunId, err := uuid.Parse(req.TaskRunId)

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "task run id is not a valid uuid")
	}

	if !i.isLogIngestionEnabled {
		return &contracts.PutLogResponse{}, nil
	}

	task, err := i.getSingleTask(ctx, tenantId, taskRunId, false)

	if err != nil {
		return nil, err
	}

	var createdAt *time.Time

	if t := req.CreatedAt.AsTime(); !t.IsZero() {
		createdAt = &t
	}

	var metadata []byte

	if req.Metadata != "" {
		// Validate that metadata is valid JSON
		var metadataMap map[string]interface{}
		if err := json.Unmarshal([]byte(req.Metadata), &metadataMap); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid metadata JSON: %v", err)
		}

		// Re-marshal to ensure consistent formatting
		metadata, err = json.Marshal(metadataMap)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to marshal metadata: %v", err)
		}
	}

	var retryCount int

	if req.TaskRetryCount != nil {
		retryCount = int(*req.TaskRetryCount)
	} else {
		retryCount = int(task.RetryCount)
	}

	opts := &v1.CreateLogLineOpts{
		TaskExternalId: task.ExternalID,
		TaskId:         task.ID,
		TaskInsertedAt: task.InsertedAt,
		CreatedAt:      createdAt,
		Message:        req.Message,
		Level:          req.Level,
		Metadata:       metadata,
		RetryCount:     retryCount,
	}

	if apiErrors, err := i.v.ValidateAPI(opts); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %s", apiErrors.String())
	}

	if err := v1.ValidateJSONB(opts.Metadata, "additionalMetadata"); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %s", err)
	}

	err = i.repov1.Logs().PutLog(ctx, tenantId, opts)

	if err != nil {
		return nil, err
	}

	return &contracts.PutLogResponse{}, nil
}
