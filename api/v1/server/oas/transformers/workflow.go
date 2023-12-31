package transformers

import (
	"context"
	"fmt"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/iter"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/services/shared/defaults"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
)

func ToWorkflow(workflow *db.WorkflowModel, lastRun *db.WorkflowRunModel) (*gen.Workflow, error) {
	res := &gen.Workflow{
		Metadata: *toAPIMetadata(workflow.ID, workflow.CreatedAt, workflow.UpdatedAt),
		Name:     workflow.Name,
	}

	if lastRun != nil {
		var err error
		res.LastRun, err = ToWorkflowRun(lastRun)

		if err != nil {
			return nil, err
		}
	}

	if description, ok := workflow.Description(); ok {
		res.Description = &description
	}

	if workflow.RelationsWorkflow.Tags != nil {
		if tags := workflow.Tags(); tags != nil {
			apiTags := make([]gen.WorkflowTag, len(tags))

			for i, tag := range tags {
				apiTags[i] = gen.WorkflowTag{
					Name:  tag.Name,
					Color: tag.Color,
				}
			}

			res.Tags = &apiTags
		}
	}

	if workflow.RelationsWorkflow.Versions != nil {
		if versions := workflow.Versions(); versions != nil {
			apiVersions := make([]gen.WorkflowVersionMeta, len(versions))

			for i, version := range versions {
				versionCp := version
				apiVersions[i] = *ToWorkflowVersionMeta(&versionCp)
			}

			res.Versions = &apiVersions
		}
	}

	return res, nil
}

func ToWorkflowVersionMeta(version *db.WorkflowVersionModel) *gen.WorkflowVersionMeta {
	res := &gen.WorkflowVersionMeta{
		Metadata:   *toAPIMetadata(version.ID, version.CreatedAt, version.UpdatedAt),
		Version:    version.Version,
		WorkflowId: version.WorkflowID,
		Order:      int32(version.Order),
	}

	return res
}

func ToWorkflowVersion(workflow *db.WorkflowModel, version *db.WorkflowVersionModel) (*gen.WorkflowVersion, error) {
	res := &gen.WorkflowVersion{
		Metadata:   *toAPIMetadata(version.ID, version.CreatedAt, version.UpdatedAt),
		Version:    version.Version,
		WorkflowId: version.WorkflowID,
		Order:      int32(version.Order),
	}

	if version.RelationsWorkflowVersion.Jobs != nil {
		if jobs := version.Jobs(); jobs != nil {
			apiJobs := make([]gen.Job, len(jobs))

			for i, job := range jobs {
				jobCp := job
				apiJob, err := ToJob(&jobCp)

				if err != nil {
					return nil, err
				}
				apiJobs[i] = *apiJob
			}

			res.Jobs = &apiJobs
		}
	}

	if workflow == nil {
		workflow = version.RelationsWorkflowVersion.Workflow
	}

	if workflow != nil {
		var err error
		res.Workflow, err = ToWorkflow(workflow, nil)

		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func ToWorkflowYAMLBytes(workflow *db.WorkflowModel, version *db.WorkflowVersionModel) ([]byte, error) {
	res := &types.Workflow{
		Name:    workflow.Name,
		Version: version.Version,
	}

	if description, ok := workflow.Description(); ok {
		res.Description = description
	}

	if triggers, ok := version.Triggers(); ok && triggers != nil {
		triggersResp := types.WorkflowTriggers{}

		if crons := triggers.Crons(); crons != nil && len(crons) > 0 {
			triggersResp.Cron = make([]string, len(crons))

			for i, cron := range crons {
				triggersResp.Cron[i] = cron.Cron
			}
		}

		if events := triggers.Events(); events != nil && len(events) > 0 {
			triggersResp.Events = make([]string, len(events))

			for i, event := range events {
				triggersResp.Events[i] = event.EventKey
			}
		}

		res.Triggers = triggersResp
	}

	if jobs := version.Jobs(); jobs != nil {
		res.Jobs = make(map[string]types.WorkflowJob, len(jobs))

		for _, job := range jobs {
			jobCp := job

			jobRes := types.WorkflowJob{}

			if description, ok := jobCp.Description(); ok {
				jobRes.Description = description
			}

			if timeout, ok := jobCp.Timeout(); ok {
				jobRes.Timeout = timeout
			}

			if steps := jobCp.Steps(); steps != nil {
				jobRes.Steps = make([]types.WorkflowStep, 0)

				iter, err := iter.New(steps)

				if err != nil {
					return nil, fmt.Errorf("could not create step iterator: %w", err)
				}

				for {
					step, ok := iter.Next()

					if !ok {
						break
					}

					stepRes := types.WorkflowStep{
						ID:       step.ID,
						ActionID: step.ActionID,
					}

					if readableId, ok := step.ReadableID(); ok {
						stepRes.ID = readableId
					}

					if timeout, ok := step.Timeout(); ok {
						stepRes.Timeout = timeout
					}

					if inputs, ok := step.Inputs(); ok {
						withMap := map[string]interface{}{}
						err := datautils.FromJSONType(&inputs, &withMap)

						if err != nil {
							return nil, err
						}

						stepRes.With = withMap
					}

					jobRes.Steps = append(jobRes.Steps, stepRes)
				}

				res.Jobs[jobCp.Name] = jobRes
			}
		}
	}

	return types.ToYAML(context.Background(), res)
}

func ToJob(job *db.JobModel) (*gen.Job, error) {
	res := &gen.Job{
		Metadata:  *toAPIMetadata(job.ID, job.CreatedAt, job.UpdatedAt),
		Name:      job.Name,
		TenantId:  job.TenantID,
		VersionId: job.WorkflowVersionID,
	}

	if description, ok := job.Description(); ok {
		res.Description = &description
	}

	if timeout, ok := job.Timeout(); ok {
		res.Timeout = &timeout
	} else {
		res.Timeout = repository.StringPtr(defaults.DefaultJobRunTimeout)
	}

	if steps := job.Steps(); steps != nil {
		apiSteps := make([]gen.Step, 0)

		iter, err := iter.New(steps)

		if err != nil {
			return nil, fmt.Errorf("could not create step iterator: %w", err)
		}

		for {
			step, ok := iter.Next()

			if !ok {
				break
			}

			apiSteps = append(apiSteps, *ToStep(step))
		}

		res.Steps = apiSteps
	}

	return res, nil
}

func ToStep(step *db.StepModel) *gen.Step {
	res := &gen.Step{
		Metadata: *toAPIMetadata(step.ID, step.CreatedAt, step.UpdatedAt),
		Action:   step.ActionID,
		JobId:    step.JobID,
		TenantId: step.TenantID,
	}

	if readableId, ok := step.ReadableID(); ok {
		res.ReadableId = readableId
	}

	if timeout, ok := step.Timeout(); ok {
		res.Timeout = &timeout
	} else {
		res.Timeout = repository.StringPtr(defaults.DefaultStepRunTimeout)
	}

	if next, ok := step.NextID(); ok {
		res.NextId = next
	}

	return res
}

func ToWorkflowFromSQLC(row *dbsqlc.Workflow) *gen.Workflow {
	res := &gen.Workflow{
		Metadata:    *toAPIMetadata(pgUUIDToStr(row.ID), row.CreatedAt.Time, row.UpdatedAt.Time),
		Name:        row.Name,
		Description: &row.Description.String,
	}

	return res
}

func ToWorkflowVersionMetaFromSQLC(row *dbsqlc.WorkflowVersion, workflow *gen.Workflow) *gen.WorkflowVersionMeta {
	res := &gen.WorkflowVersionMeta{
		Metadata:   *toAPIMetadata(pgUUIDToStr(row.ID), row.CreatedAt.Time, row.UpdatedAt.Time),
		Version:    row.Version,
		WorkflowId: pgUUIDToStr(row.WorkflowId),
		Order:      int32(row.Order),
		Workflow:   workflow,
	}

	return res
}

func ToWorkflowVersionFromSQLC(row *dbsqlc.WorkflowVersion, workflow *gen.Workflow) *gen.WorkflowVersion {
	res := &gen.WorkflowVersion{
		Metadata:   *toAPIMetadata(pgUUIDToStr(row.ID), row.CreatedAt.Time, row.UpdatedAt.Time),
		Version:    row.Version,
		WorkflowId: pgUUIDToStr(row.WorkflowId),
		Order:      int32(row.Order),
		Workflow:   workflow,
	}

	return res
}
