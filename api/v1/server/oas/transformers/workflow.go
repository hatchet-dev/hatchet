package transformers

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func ToWorkflow(
	workflow *dbsqlc.Workflow,
	version *dbsqlc.WorkflowVersion,
) *gen.Workflow {
	res := &gen.Workflow{
		Metadata: *toAPIMetadata(
			sqlchelpers.UUIDToStr(workflow.ID),
			workflow.CreatedAt.Time,
			workflow.UpdatedAt.Time,
		),
		Name: workflow.Name,
	}

	res.Description = &workflow.Description.String

	if version != nil {
		apiVersions := make([]gen.WorkflowVersionMeta, 1)
		apiVersions[0] = *ToWorkflowVersionMeta(version, workflow)
		res.Versions = &apiVersions
	}

	return res
}

func ToWorkflowVersionMeta(version *dbsqlc.WorkflowVersion, workflow *dbsqlc.Workflow) *gen.WorkflowVersionMeta {
	res := &gen.WorkflowVersionMeta{
		Metadata: *toAPIMetadata(
			sqlchelpers.UUIDToStr(version.ID),
			version.CreatedAt.Time,
			version.UpdatedAt.Time,
		),
		WorkflowId: sqlchelpers.UUIDToStr(version.WorkflowId),
		Order:      int32(version.Order),
		Version:    version.Version.String,
	}

	return res
}

type WorkflowConcurrency struct {
	ID                    pgtype.UUID
	GetConcurrencyGroupId pgtype.UUID
	MaxRuns               pgtype.Int4
	LimitStrategy         dbsqlc.NullConcurrencyLimitStrategy
}

func ToWorkflowVersion(
	version *dbsqlc.WorkflowVersion,
	workflow *dbsqlc.Workflow,
	concurrency *WorkflowConcurrency,
	crons []*dbsqlc.WorkflowTriggerCronRef,
	events []*dbsqlc.WorkflowTriggerEventRef,
	schedules []*dbsqlc.WorkflowTriggerScheduledRef,
) *gen.WorkflowVersion {
	res := &gen.WorkflowVersion{
		Metadata: *toAPIMetadata(
			sqlchelpers.UUIDToStr(version.ID),
			version.CreatedAt.Time,
			version.UpdatedAt.Time,
		),
		// WorkflowId:      sqlchelpers.UUIDToStr(version.WorkflowId),
		Order:           int32(version.Order),
		Version:         version.Version.String,
		ScheduleTimeout: &version.ScheduleTimeout,
		DefaultPriority: &version.DefaultPriority.Int32,
	}

	if version.Sticky.Valid {
		var stickyStrategy string

		switch version.Sticky.StickyStrategy {
		case dbsqlc.StickyStrategyHARD:
			stickyStrategy = "hard"
		case dbsqlc.StickyStrategySOFT:
			stickyStrategy = "soft"
		}

		res.Sticky = &stickyStrategy
	}

	if version.WorkflowId.Valid {
		res.Workflow = ToWorkflowFromSQLC(workflow)
	}

	if concurrency != nil {
		res.Concurrency = ToWorkflowVersionConcurrency(concurrency)
	}

	triggersResp := gen.WorkflowTriggers{}

	if len(crons) > 0 {
		genCrons := make([]gen.WorkflowTriggerCronRef, 0)

		for _, cron := range crons {
			cronCp := cron
			parentId := sqlchelpers.UUIDToStr(cronCp.ParentId)
			genCrons = append(genCrons, gen.WorkflowTriggerCronRef{
				Cron:     &cronCp.Cron,
				ParentId: &parentId,
			})
		}

		triggersResp.Crons = &genCrons
	}

	if len(events) > 0 {
		genEvents := make([]gen.WorkflowTriggerEventRef, 0)

		for _, event := range events {
			eventCp := event
			if eventCp.ParentId.Valid {
				parentId := sqlchelpers.UUIDToStr(eventCp.ParentId)
				genEvents = append(genEvents, gen.WorkflowTriggerEventRef{
					EventKey: &eventCp.EventKey,
					ParentId: &parentId,
				})
			}
		}

		triggersResp.Events = &genEvents
	}

	res.Triggers = &triggersResp

	return res
}

func ToWorkflowVersionConcurrency(concurrency *WorkflowConcurrency) *gen.WorkflowConcurrency {
	if !concurrency.LimitStrategy.Valid {
		return nil
	}

	res := &gen.WorkflowConcurrency{
		MaxRuns:             concurrency.MaxRuns.Int32,
		LimitStrategy:       gen.WorkflowConcurrencyLimitStrategy(concurrency.LimitStrategy.ConcurrencyLimitStrategy),
		GetConcurrencyGroup: sqlchelpers.UUIDToStr(concurrency.GetConcurrencyGroupId),
	}

	return res
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
