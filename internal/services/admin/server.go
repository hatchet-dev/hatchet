package admin

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (a *AdminServiceImpl) GetWorkflowByName(ctx context.Context, req *contracts.GetWorkflowByNameRequest) (*contracts.Workflow, error) {
	workflow, err := a.repo.Workflow().GetWorkflowByName(
		req.TenantId,
		req.Name,
	)

	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, status.Error(
				codes.NotFound,
				"workflow not found",
			)
		}

		return nil, err
	}

	resp := toWorkflow(workflow)

	return resp, nil
}

func (a *AdminServiceImpl) PutWorkflow(ctx context.Context, req *contracts.PutWorkflowRequest) (*contracts.WorkflowVersion, error) {
	// TODO: authorize request

	createOpts, err := getCreateWorkflowOpts(req)

	if err != nil {
		return nil, err
	}

	// determine if workflow already exists
	var workflowVersion *db.WorkflowVersionModel
	var oldWorkflowVersion *db.WorkflowVersionModel

	currWorkflow, err := a.repo.Workflow().GetWorkflowByName(
		req.TenantId,
		req.Opts.Name,
	)

	if err != nil {
		if !errors.Is(err, db.ErrNotFound) {
			return nil, err
		}

		// workflow does not exist, create it
		workflowVersion, err = a.repo.Workflow().CreateNewWorkflow(
			req.TenantId,
			createOpts,
		)

		if err != nil {
			return nil, err
		}
	} else {
		oldWorkflowVersion = &currWorkflow.Versions()[0]

		// workflow exists, create a new version
		workflowVersion, err = a.repo.Workflow().CreateWorkflowVersion(
			req.TenantId,
			createOpts,
		)

		if err != nil {
			return nil, err
		}
	}

	// if this is a cron-based workflow, assign the workflow run to a ticker
	triggers, ok := workflowVersion.Triggers()

	if !ok {
		return nil, status.Error(
			codes.FailedPrecondition,
			"workflow version has no triggers",
		)
	}

	if crons := triggers.Crons(); len(crons) > 0 {
		within := time.Now().Add(-6 * time.Second)

		tickers, err := a.repo.Ticker().ListTickers(&repository.ListTickerOpts{
			LatestHeartbeatAt: &within,
		})

		if err != nil {
			return nil, err
		}

		if len(tickers) == 0 {
			return nil, status.Error(
				codes.FailedPrecondition,
				"no tickers available",
			)
		}

		numTickers := len(tickers)

		for i, cronTrigger := range crons {
			cronTriggerCp := cronTrigger
			ticker := tickers[i%numTickers]

			_, err := a.repo.Ticker().AddCron(
				ticker.ID,
				&cronTriggerCp,
			)

			if err != nil {
				return nil, err
			}

			task, err := cronScheduleTask(&ticker, &cronTriggerCp, workflowVersion)

			if err != nil {
				return nil, err
			}

			// send to task queue
			err = a.tq.AddTask(
				ctx,
				taskqueue.QueueTypeFromTicker(&ticker),
				task,
			)

			if err != nil {
				return nil, err
			}
		}
	}

	// cancel the old workflow version
	if oldWorkflowVersion != nil {
		oldTriggers, ok := oldWorkflowVersion.Triggers()

		if !ok {
			return nil, status.Error(
				codes.FailedPrecondition,
				"old workflow version has no triggers",
			)
		}

		if crons := oldTriggers.Crons(); len(crons) > 0 {
			for _, cronTrigger := range crons {
				cronTriggerCp := cronTrigger

				if ticker, ok := cronTrigger.Ticker(); ok {
					task, err := cronCancelTask(ticker, &cronTriggerCp, workflowVersion)

					if err != nil {
						return nil, err
					}

					// send to task queue
					err = a.tq.AddTask(
						ctx,
						taskqueue.QueueTypeFromTicker(ticker),
						task,
					)

					if err != nil {
						return nil, err
					}

					// remove cron
					_, err = a.repo.Ticker().RemoveCron(
						ticker.ID,
						&cronTriggerCp,
					)

					if err != nil {
						return nil, err
					}
				}
			}
		}
	}

	resp := toWorkflowVersion(workflowVersion)

	return resp, nil
}

func (a *AdminServiceImpl) DeleteWorkflow(ctx context.Context, req *contracts.DeleteWorkflowRequest) (*contracts.Workflow, error) {
	workflow, err := a.repo.Workflow().DeleteWorkflow(
		req.TenantId,
		req.WorkflowId,
	)

	if err != nil {
		return nil, err
	}

	resp := toWorkflow(workflow)

	return resp, nil
}

func (a *AdminServiceImpl) ListWorkflows(
	ctx context.Context,
	req *contracts.ListWorkflowsRequest,
) (*contracts.ListWorkflowsResponse, error) {
	listResp, err := a.repo.Workflow().ListWorkflows(
		req.TenantId,
		&repository.ListWorkflowsOpts{},
	)

	if err != nil {
		return nil, err
	}

	items := make([]*contracts.Workflow, len(listResp.Rows))

	for i := range listResp.Rows {
		items[i] = toWorkflow(listResp.Rows[i].WorkflowModel)
	}

	return &contracts.ListWorkflowsResponse{
		Workflows: items,
	}, nil
}

func (a *AdminServiceImpl) ListWorkflowsForEvent(
	ctx context.Context,
	req *contracts.ListWorkflowsForEventRequest,
) (*contracts.ListWorkflowsResponse, error) {
	listResp, err := a.repo.Workflow().ListWorkflows(
		req.TenantId,
		&repository.ListWorkflowsOpts{
			EventKey: &req.EventKey,
		},
	)

	if err != nil {
		return nil, err
	}

	items := make([]*contracts.Workflow, len(listResp.Rows))

	for i := range listResp.Rows {
		items[i] = toWorkflow(listResp.Rows[i].WorkflowModel)
	}

	return &contracts.ListWorkflowsResponse{
		Workflows: items,
	}, nil
}

func getCreateWorkflowOpts(req *contracts.PutWorkflowRequest) (*repository.CreateWorkflowVersionOpts, error) {
	jobs := make([]repository.CreateWorkflowJobOpts, len(req.Opts.Jobs))

	for i, job := range req.Opts.Jobs {
		jobCp := job

		steps := make([]repository.CreateWorkflowStepOpts, len(job.Steps))

		for j, step := range job.Steps {
			stepCp := step

			inputsMap := make(map[string]interface{})

			err := json.Unmarshal([]byte(stepCp.Inputs), &inputsMap)

			if err != nil {
				return nil, err
			}

			inputs, err := datautils.ToJSONType(inputsMap)

			if err != nil {
				return nil, err
			}

			steps[j] = repository.CreateWorkflowStepOpts{
				ReadableId: stepCp.ReadableId,
				Action:     stepCp.Action,
				Timeout:    &stepCp.Timeout,
				Inputs:     inputs,
			}
		}

		jobs[i] = repository.CreateWorkflowJobOpts{
			Name:        jobCp.Name,
			Description: &jobCp.Description,
			Timeout:     &jobCp.Timeout,
			Steps:       steps,
		}
	}

	return &repository.CreateWorkflowVersionOpts{
		Name:          req.Opts.Name,
		Description:   &req.Opts.Description,
		Version:       req.Opts.Version,
		EventTriggers: req.Opts.EventTriggers,
		CronTriggers:  req.Opts.CronTriggers,
		Jobs:          jobs,
	}, nil
}

func toWorkflow(workflow *db.WorkflowModel) *contracts.Workflow {
	w := &contracts.Workflow{
		Id:        workflow.ID,
		CreatedAt: timestamppb.New(workflow.CreatedAt),
		UpdatedAt: timestamppb.New(workflow.UpdatedAt),
		TenantId:  workflow.TenantID,
		Name:      workflow.Name,
	}

	if description, ok := workflow.Description(); ok {
		w.Description = wrapperspb.String(description)
	}

	versionModels := workflow.Versions()
	versions := make([]*contracts.WorkflowVersion, len(versionModels))

	for i, versionModel := range versionModels {
		versionModelCp := versionModel
		versions[i] = toWorkflowVersion(&versionModelCp)
	}

	w.Versions = versions

	return w
}

func toWorkflowVersion(workflowVersion *db.WorkflowVersionModel) *contracts.WorkflowVersion {
	version := &contracts.WorkflowVersion{
		Id:         workflowVersion.ID,
		CreatedAt:  timestamppb.New(workflowVersion.CreatedAt),
		UpdatedAt:  timestamppb.New(workflowVersion.UpdatedAt),
		Version:    workflowVersion.Version,
		Order:      int32(workflowVersion.Order),
		WorkflowId: workflowVersion.WorkflowID,
	}

	if triggers, ok := workflowVersion.Triggers(); ok {
		version.Triggers = toWorkflowVersionTriggers(triggers)
	}

	jobModels := workflowVersion.Jobs()
	jobs := make([]*contracts.Job, len(jobModels))

	for i, jobModel := range jobModels {
		jobModelCp := jobModel
		jobs[i] = toJob(&jobModelCp)
	}

	return version
}

func toWorkflowVersionTriggers(triggers *db.WorkflowTriggersModel) *contracts.WorkflowTriggers {
	t := &contracts.WorkflowTriggers{
		Id:                triggers.ID,
		CreatedAt:         timestamppb.New(triggers.CreatedAt),
		UpdatedAt:         timestamppb.New(triggers.UpdatedAt),
		WorkflowVersionId: triggers.WorkflowVersionID,
		TenantId:          triggers.TenantID,
	}

	eventTriggerModels := triggers.Events()
	eventTriggers := make([]*contracts.WorkflowTriggerEventRef, len(eventTriggerModels))

	for i, eventTriggerModel := range eventTriggerModels {
		eventTriggers[i] = &contracts.WorkflowTriggerEventRef{
			ParentId: eventTriggerModel.ParentID,
			EventKey: eventTriggerModel.EventKey,
		}
	}

	t.Events = eventTriggers

	cronTriggerModels := triggers.Crons()
	cronTriggers := make([]*contracts.WorkflowTriggerCronRef, len(cronTriggerModels))

	for i, cronTriggerModel := range cronTriggerModels {
		cronTriggers[i] = &contracts.WorkflowTriggerCronRef{
			ParentId: cronTriggerModel.ParentID,
			Cron:     cronTriggerModel.Cron,
		}
	}

	t.Crons = cronTriggers

	return t
}

func toJob(job *db.JobModel) *contracts.Job {
	j := &contracts.Job{
		Id:                job.ID,
		CreatedAt:         timestamppb.New(job.CreatedAt),
		UpdatedAt:         timestamppb.New(job.UpdatedAt),
		TenantId:          job.TenantID,
		WorkflowVersionId: job.WorkflowVersionID,
		Name:              job.Name,
	}

	if description, ok := job.Description(); ok {
		j.Description = wrapperspb.String(description)
	}

	if timeout, ok := job.Timeout(); ok {
		j.Timeout = wrapperspb.String(timeout)
	}

	stepModels := job.Steps()

	steps := make([]*contracts.Step, len(stepModels))

	for i, stepModel := range stepModels {
		stepModelCp := stepModel
		steps[i] = toStep(&stepModelCp)
	}

	j.Steps = steps

	return j
}

func toStep(step *db.StepModel) *contracts.Step {
	action := step.Action()

	s := &contracts.Step{
		Id:        step.ID,
		CreatedAt: timestamppb.New(step.CreatedAt),
		UpdatedAt: timestamppb.New(step.UpdatedAt),
		TenantId:  step.TenantID,
		JobId:     step.JobID,
		Action:    action.ID,
	}

	if readableId, ok := step.ReadableID(); ok {
		s.ReadableId = wrapperspb.String(readableId)
	}

	if timeout, ok := step.Timeout(); ok {
		s.Timeout = wrapperspb.String(timeout)
	}

	if nextId, ok := step.NextID(); ok {
		s.NextId = nextId
	}

	return s
}

func cronScheduleTask(ticker *db.TickerModel, cronTriggerRef *db.WorkflowTriggerCronRefModel, workflowVersion *db.WorkflowVersionModel) (*taskqueue.Task, error) {
	payload, _ := datautils.ToJSONMap(tasktypes.ScheduleCronTaskPayload{
		CronParentId:      cronTriggerRef.ParentID,
		Cron:              cronTriggerRef.Cron,
		WorkflowVersionId: workflowVersion.ID,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.ScheduleCronTaskMetadata{
		TenantId: workflowVersion.Workflow().TenantID,
	})

	return &taskqueue.Task{
		ID:       "schedule-cron",
		Queue:    taskqueue.QueueTypeFromTicker(ticker),
		Payload:  payload,
		Metadata: metadata,
	}, nil
}

func cronCancelTask(ticker *db.TickerModel, cronTriggerRef *db.WorkflowTriggerCronRefModel, workflowVersion *db.WorkflowVersionModel) (*taskqueue.Task, error) {
	payload, _ := datautils.ToJSONMap(tasktypes.CancelCronTaskPayload{
		CronParentId:      cronTriggerRef.ParentID,
		Cron:              cronTriggerRef.Cron,
		WorkflowVersionId: workflowVersion.ID,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.CancelCronTaskMetadata{
		TenantId: workflowVersion.Workflow().TenantID,
	})

	return &taskqueue.Task{
		ID:       "cancel-cron",
		Queue:    taskqueue.QueueTypeFromTicker(ticker),
		Payload:  payload,
		Metadata: metadata,
	}, nil
}
