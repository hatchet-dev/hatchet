package admin

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/constants"
	grpcmiddleware "github.com/hatchet-dev/hatchet/pkg/grpc/middleware"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a *AdminServiceImpl) triggerWorkflowV1(ctx context.Context, req *contracts.TriggerWorkflowRequest) (*contracts.TriggerWorkflowResponse, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	canCreateTR, trLimit, err := a.repov1.TenantLimit().CanCreate(
		ctx,
		sqlcv1.LimitResourceTASKRUN,
		tenantId,
		// NOTE: this isn't actually the number of tasks per workflow run, but we're just checking to see
		// if we've exceeded the limit
		1,
	)

	if err != nil {
		return nil, fmt.Errorf("could not check tenant limit: %w", err)
	}

	if !canCreateTR {
		return nil, status.Error(
			codes.ResourceExhausted,
			fmt.Sprintf("tenant has reached %d%% of its task runs limit", trLimit),
		)
	}

	opt, err := a.newTriggerOpt(ctx, tenantId, req)

	if err != nil {
		return nil, fmt.Errorf("could not create trigger opt: %w", err)
	}

	if err := v1.ValidateJSONB(opt.Data, "payload"); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %s", err)
	}

	if err := v1.ValidateJSONB(opt.AdditionalMetadata, "additionalMetadata"); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %s", err)
	}

	err = a.generateExternalIds(ctx, tenantId, []*v1.WorkflowNameTriggerOpts{opt})

	if err != nil {
		return nil, fmt.Errorf("could not generate external ids: %w", err)
	}

	err = a.ingest(
		ctx,
		tenantId,
		opt,
	)

	if err != nil {
		return nil, fmt.Errorf("could not trigger workflow: %w", err)
	}

	additionalMeta := ""
	if req.AdditionalMetadata != nil {
		additionalMeta = *req.AdditionalMetadata
	}

	corrId := datautils.ExtractCorrelationId(additionalMeta)

	if corrId != nil {
		ctx = context.WithValue(ctx, constants.CorrelationIdKey, *corrId)
	}

	ctx = context.WithValue(ctx, constants.ResourceIdKey, opt.ExternalId)
	ctx = context.WithValue(ctx, constants.ResourceTypeKey, constants.ResourceTypeWorkflowRun)

	grpcmiddleware.TriggerCallback(ctx)

	return &contracts.TriggerWorkflowResponse{
		WorkflowRunId: opt.ExternalId.String(),
	}, nil
}

func (a *AdminServiceImpl) bulkTriggerWorkflowV1(ctx context.Context, req *contracts.BulkTriggerWorkflowRequest) (*contracts.BulkTriggerWorkflowResponse, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	opts := make([]*v1.WorkflowNameTriggerOpts, len(req.Workflows))

	for i, workflow := range req.Workflows {
		opt, err := a.newTriggerOpt(ctx, tenantId, workflow)

		if err != nil {
			return nil, fmt.Errorf("could not create trigger opt: %w", err)
		}

		if err := v1.ValidateJSONB(opt.Data, "payload"); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %s", err)
		}

		if err := v1.ValidateJSONB(opt.AdditionalMetadata, "additionalMetadata"); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %s", err)
		}

		opts[i] = opt
	}

	err := a.generateExternalIds(ctx, tenantId, opts)

	if err != nil {
		return nil, fmt.Errorf("could not generate external ids: %w", err)
	}

	err = a.ingest(
		ctx,
		tenantId,
		opts...,
	)

	if err != nil {
		return nil, err
	}

	runIds := make([]string, len(req.Workflows))

	for i, opt := range opts {
		runIds[i] = opt.ExternalId.String()
	}

	for i, runId := range runIds {
		additionalMeta := ""
		if req.Workflows[i].AdditionalMetadata != nil {
			additionalMeta = *req.Workflows[i].AdditionalMetadata
		}
		corrId := datautils.ExtractCorrelationId(additionalMeta)

		ctx = context.WithValue(ctx, constants.CorrelationIdKey, corrId)
		ctx = context.WithValue(ctx, constants.ResourceIdKey, runId)
		ctx = context.WithValue(ctx, constants.ResourceTypeKey, constants.ResourceTypeWorkflowRun)

		grpcmiddleware.TriggerCallback(ctx)
	}

	return &contracts.BulkTriggerWorkflowResponse{
		WorkflowRunIds: runIds,
	}, nil
}

func (i *AdminServiceImpl) newTriggerOpt(
	ctx context.Context,
	tenantId uuid.UUID,
	req *contracts.TriggerWorkflowRequest,
) (*v1.WorkflowNameTriggerOpts, error) {
	ctx, span := telemetry.NewSpan(ctx, "admin_service.new_trigger_opt")
	defer span.End()

	span.SetAttributes(
		attribute.String("admin_service.new_trigger_opt.workflow_name", req.Name),
		attribute.Int("admin_service.new_trigger_opt.payload_size", len(req.Input)),
		attribute.Bool("admin_service.new_trigger_opt.is_child_workflow", req.ParentStepRunId != nil),
	)

	additionalMeta := ""

	if req.AdditionalMetadata != nil {
		additionalMeta = *req.AdditionalMetadata
	}

	var desiredWorkerId *uuid.UUID
	if req.DesiredWorkerId != nil {
		workerId, err := uuid.Parse(*req.DesiredWorkerId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "desiredWorkerId must be a valid UUID: %s", err)
		}
		desiredWorkerId = &workerId
	}

	t := &v1.TriggerTaskData{
		WorkflowName:       req.Name,
		Data:               []byte(req.Input),
		AdditionalMetadata: []byte(additionalMeta),
		DesiredWorkerId:    desiredWorkerId,
		Priority:           req.Priority,
	}

	if req.Priority != nil {
		if *req.Priority < 1 || *req.Priority > 3 {
			return nil, status.Errorf(codes.InvalidArgument, "priority must be between 1 and 3, got %d", *req.Priority)
		}
		t.Priority = req.Priority
	}

	if req.ParentStepRunId != nil {
		parentStepRunId, err := uuid.Parse(*req.ParentStepRunId)

		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "parentStepRunId must be a valid UUID: %s", err)
		}

		// lookup the parent external id
		parentTask, err := i.repov1.Tasks().GetTaskByExternalId(
			ctx,
			tenantId,
			parentStepRunId,
			false,
		)

		if err != nil {
			return nil, fmt.Errorf("could not find parent task: %w", err)
		}

		parentExternalId := parentTask.ExternalID
		childIndex := int64(*req.ChildIndex)

		t.ParentExternalId = &parentExternalId
		t.ParentTaskId = &parentTask.ID
		t.ParentTaskInsertedAt = &parentTask.InsertedAt.Time
		t.ChildIndex = &childIndex
		t.ChildKey = req.ChildKey
	}

	return &v1.WorkflowNameTriggerOpts{
		TriggerTaskData: t,
	}, nil
}

func (i *AdminServiceImpl) generateExternalIds(ctx context.Context, tenantId uuid.UUID, opts []*v1.WorkflowNameTriggerOpts) error {
	return i.repov1.Triggers().PopulateExternalIdsForWorkflow(ctx, tenantId, opts)
}

func (i *AdminServiceImpl) ingest(ctx context.Context, tenantId uuid.UUID, opts ...*v1.WorkflowNameTriggerOpts) error {
	optsToSend := make([]*v1.WorkflowNameTriggerOpts, 0)

	for _, opt := range opts {
		if opt.ShouldSkip {
			continue
		}

		optsToSend = append(optsToSend, opt)
	}

	if len(optsToSend) > 0 {
		verifyErr := i.repov1.Triggers().PreflightVerifyWorkflowNameOpts(ctx, tenantId, optsToSend)

		if verifyErr != nil {
			namesNotFound := &v1.ErrNamesNotFound{}

			if errors.As(verifyErr, &namesNotFound) {
				return status.Error(
					codes.InvalidArgument,
					verifyErr.Error(),
				)
			}

			return fmt.Errorf("could not verify workflow name opts: %w", verifyErr)
		}

		msg, err := tasktypes.TriggerTaskMessage(
			tenantId,
			optsToSend...,
		)

		if err != nil {
			return fmt.Errorf("could not create event task: %w", err)
		}

		err = i.mqv1.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg)

		if err != nil {
			return fmt.Errorf("could not add event to task queue: %w", err)
		}
	}

	return nil
}
