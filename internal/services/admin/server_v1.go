package admin

import (
	"context"
	"errors"
	"fmt"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a *AdminServiceImpl) triggerWorkflowV1(ctx context.Context, req *contracts.TriggerWorkflowRequest) (*contracts.TriggerWorkflowResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	canCreateWR, wrLimit, err := a.entitlements.TenantLimit().CanCreate(
		ctx,
		dbsqlc.LimitResourceWORKFLOWRUN,
		tenantId,
		1,
	)

	if err != nil {
		return nil, fmt.Errorf("could not check tenant limit: %w", err)
	}

	if !canCreateWR {
		return nil, status.Error(
			codes.ResourceExhausted,
			fmt.Sprintf("tenant has reached %d%% of its workflow runs limit", wrLimit),
		)
	}

	canCreateTR, trLimit, err := a.entitlements.TenantLimit().CanCreate(
		ctx,
		dbsqlc.LimitResourceTASKRUN,
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

	return &contracts.TriggerWorkflowResponse{
		WorkflowRunId: opt.ExternalId,
	}, nil
}

func (a *AdminServiceImpl) bulkTriggerWorkflowV1(ctx context.Context, req *contracts.BulkTriggerWorkflowRequest) (*contracts.BulkTriggerWorkflowResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	opts := make([]*v1.WorkflowNameTriggerOpts, len(req.Workflows))

	for i, workflow := range req.Workflows {
		opt, err := a.newTriggerOpt(ctx, tenantId, workflow)

		if err != nil {
			return nil, fmt.Errorf("could not create trigger opt: %w", err)
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
		runIds[i] = opt.ExternalId
	}

	return &contracts.BulkTriggerWorkflowResponse{
		WorkflowRunIds: runIds,
	}, nil
}

func (i *AdminServiceImpl) newTriggerOpt(
	ctx context.Context,
	tenantId string,
	req *contracts.TriggerWorkflowRequest,
) (*v1.WorkflowNameTriggerOpts, error) {
	additionalMeta := ""

	if req.AdditionalMetadata != nil {
		additionalMeta = *req.AdditionalMetadata
	}

	t := &v1.TriggerTaskData{
		WorkflowName:       req.Name,
		Data:               []byte(req.Input),
		AdditionalMetadata: []byte(additionalMeta),
		DesiredWorkerId:    req.DesiredWorkerId,
	}

	if req.Priority != nil {
		t.Priority = req.Priority
	}

	if req.ParentStepRunId != nil {
		// lookup the parent external id
		parentTask, err := i.repov1.Tasks().GetTaskByExternalId(
			ctx,
			tenantId,
			*req.ParentStepRunId,
			false,
		)

		if err != nil {
			return nil, fmt.Errorf("could not find parent task: %w", err)
		}

		parentExternalId := sqlchelpers.UUIDToStr(parentTask.ExternalID)
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

func (i *AdminServiceImpl) generateExternalIds(ctx context.Context, tenantId string, opts []*v1.WorkflowNameTriggerOpts) error {
	return i.repov1.Triggers().PopulateExternalIdsForWorkflow(ctx, tenantId, opts)
}

func (i *AdminServiceImpl) ingest(ctx context.Context, tenantId string, opts ...*v1.WorkflowNameTriggerOpts) error {
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
