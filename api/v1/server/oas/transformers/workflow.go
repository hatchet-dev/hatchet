package transformers

import (
	"context"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func ToWorkflow(workflow *db.WorkflowModel) (*gen.Workflow, error) {
	res := &gen.Workflow{
		Metadata: *toAPIMetadata(workflow.ID, workflow.CreatedAt, workflow.UpdatedAt),
		Name:     workflow.Name,
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
		WorkflowId: version.WorkflowID,
		Order:      int32(version.Order),
	}

	if setVersion, ok := version.Version(); ok {
		res.Version = setVersion
	}

	return res
}

func ToWorkflowVersion(version *dbsqlc.WorkflowVersion, workflow *dbsqlc.Workflow) *gen.WorkflowVersion {
	res := &gen.WorkflowVersion{
		Metadata: *toAPIMetadata(
			sqlchelpers.UUIDToStr(version.ID),
			version.CreatedAt.Time,
			version.UpdatedAt.Time,
		),
		WorkflowId:      sqlchelpers.UUIDToStr(version.WorkflowId),
		Order:           int32(version.Order),
		Version:         version.Version.String,
		ScheduleTimeout: &version.ScheduleTimeout,
	}

	if version.WorkflowId.Valid {
		res.Workflow = ToWorkflowFromSQLC(workflow)
	}

	// if version.RelationsWorkflowVersion.Jobs != nil {
	// 	if jobs := version.Jobs(); jobs != nil {
	// 		apiJobs := make([]gen.Job, len(jobs))

	// 		for i, job := range jobs {
	// 			jobCp := job
	// 			apiJob, err := ToJob(&jobCp)

	// 			if err != nil {
	// 				return nil, err
	// 			}
	// 			apiJobs[i] = *apiJob
	// 		}

	// 		res.Jobs = &apiJobs
	// 	}
	// }

	// if version.RelationsWorkflowVersion.Concurrency != nil {
	// 	if concurrency, ok := version.Concurrency(); ok {
	// 		apiConcurrency, err := ToWorkflowVersionConcurrency(concurrency)

	// 		if err != nil {
	// 			return nil, err
	// 		}

	// 		res.Concurrency = apiConcurrency
	// 	}
	// }

	// if version.RelationsWorkflowVersion.Triggers != nil {
	// 	if triggers, ok := version.Triggers(); ok && triggers != nil {
	// 		triggersResp := gen.WorkflowTriggers{}

	// 		if crons := triggers.Crons(); len(crons) > 0 {
	// 			genCrons := make([]gen.WorkflowTriggerCronRef, len(crons))

	// 			for i, cron := range crons {
	// 				cronCp := cron
	// 				genCrons[i] = gen.WorkflowTriggerCronRef{
	// 					Cron:     &cronCp.Cron,
	// 					ParentId: &cronCp.ParentID,
	// 				}
	// 			}

	// 			triggersResp.Crons = &genCrons
	// 		}

	// 		if events := triggers.Events(); len(events) > 0 {
	// 			genEvents := make([]gen.WorkflowTriggerEventRef, len(events))

	// 			for i, event := range events {
	// 				eventCp := event
	// 				genEvents[i] = gen.WorkflowTriggerEventRef{
	// 					EventKey: &eventCp.EventKey,
	// 					ParentId: &eventCp.ParentID,
	// 				}
	// 			}

	// 			triggersResp.Events = &genEvents
	// 		}

	// 		res.Triggers = &triggersResp
	// 	}
	// }

	return res
}

func ToWorkflowVersionConcurrency(concurrency *db.WorkflowConcurrencyModel) (*gen.WorkflowConcurrency, error) {
	res := &gen.WorkflowConcurrency{
		MaxRuns:       int32(concurrency.MaxRuns),
		LimitStrategy: gen.WorkflowConcurrencyLimitStrategy(concurrency.LimitStrategy),
	}

	if getGroup, ok := concurrency.GetConcurrencyGroup(); ok {
		res.GetConcurrencyGroup = getGroup.ActionID
	}

	return res, nil
}

func ToWorkflowYAMLBytes(workflow *db.WorkflowModel, version *db.WorkflowVersionModel) ([]byte, error) {
	res := &types.Workflow{
		Name: workflow.Name,
	}

	if setVersion, ok := version.Version(); ok {
		res.Version = setVersion
	}

	if description, ok := workflow.Description(); ok {
		res.Description = description
	}

	if triggers, ok := version.Triggers(); ok && triggers != nil {
		triggersResp := types.WorkflowTriggers{}

		if crons := triggers.Crons(); len(crons) > 0 {
			triggersResp.Cron = make([]string, len(crons))

			for i, cron := range crons {
				triggersResp.Cron[i] = cron.Cron
			}
		}

		if events := triggers.Events(); len(events) > 0 {
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

			if steps := jobCp.Steps(); steps != nil {
				jobRes.Steps = make([]types.WorkflowStep, 0)

				for _, step := range steps {
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

					jobRes.Steps = append(jobRes.Steps, stepRes)
				}

				res.Jobs[jobCp.Name] = jobRes
			}
		}
	}

	return types.ToYAML(context.Background(), res)
}

func ToJob(job *dbsqlc.Job, steps []*dbsqlc.GetStepsForJobsRow) *gen.Job {
	res := &gen.Job{
		Metadata: *toAPIMetadata(
			sqlchelpers.UUIDToStr(job.ID),
			job.CreatedAt.Time,
			job.UpdatedAt.Time,
		),
		Name:        job.Name,
		TenantId:    sqlchelpers.UUIDToStr(job.TenantId),
		VersionId:   sqlchelpers.UUIDToStr(job.WorkflowVersionId),
		Description: &job.Description.String,
		Timeout:     &job.Timeout.String,
	}

	apiSteps := make([]gen.Step, 0)

	for _, step := range steps {
		stepCp := step
		if stepCp.Step.JobId == job.ID {
			apiSteps = append(apiSteps, *ToStep(&stepCp.Step, stepCp.Parents))
		}
	}

	res.Steps = apiSteps

	return res
}

func ToStep(step *dbsqlc.Step, parents []pgtype.UUID) *gen.Step {
	res := &gen.Step{
		Metadata: *toAPIMetadata(
			sqlchelpers.UUIDToStr(step.ID),
			step.CreatedAt.Time,
			step.UpdatedAt.Time,
		),
		Action:     step.ActionId,
		JobId:      sqlchelpers.UUIDToStr(step.JobId),
		TenantId:   sqlchelpers.UUIDToStr(step.TenantId),
		ReadableId: step.ReadableId.String,
		Timeout:    &step.Timeout.String,
	}

	parentStr := make([]string, 0)

	for i := range parents {
		parentStr = append(parentStr, sqlchelpers.UUIDToStr(parents[i]))
	}

	res.Parents = &parentStr

	children := []string{}

	res.Children = &children

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
		Version:    row.Version.String,
		WorkflowId: pgUUIDToStr(row.WorkflowId),
		Order:      int32(row.Order),
		Workflow:   workflow,
	}

	return res
}

func ToWorkflowVersionFromSQLC(row *dbsqlc.WorkflowVersion, workflow *gen.Workflow) *gen.WorkflowVersion {
	res := &gen.WorkflowVersion{
		Metadata:   *toAPIMetadata(pgUUIDToStr(row.ID), row.CreatedAt.Time, row.UpdatedAt.Time),
		Version:    row.Version.String,
		WorkflowId: pgUUIDToStr(row.WorkflowId),
		Order:      int32(row.Order),
		Workflow:   workflow,
	}

	return res
}
