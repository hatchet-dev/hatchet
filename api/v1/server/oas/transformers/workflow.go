package transformers

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func ToWorkflow(
	workflow *sqlcv1.Workflow,
	version *sqlcv1.WorkflowVersion,
) *gen.Workflow {

	res := &gen.Workflow{
		Metadata: *toAPIMetadata(
			workflow.ID.String(),
			workflow.CreatedAt.Time,
			workflow.UpdatedAt.Time,
		),
		Name:     workflow.Name,
		TenantId: workflow.TenantId.String(),
	}

	res.IsPaused = &workflow.IsPaused.Bool

	res.Description = &workflow.Description.String

	if version != nil {
		apiVersions := make([]gen.WorkflowVersionMeta, 1)
		apiVersions[0] = *ToWorkflowVersionMeta(version, workflow)
		res.Versions = &apiVersions
	}

	return res
}

func ToWorkflowVersionMeta(version *sqlcv1.WorkflowVersion, workflow *sqlcv1.Workflow) *gen.WorkflowVersionMeta {
	res := &gen.WorkflowVersionMeta{
		Metadata: *toAPIMetadata(
			version.ID.String(),
			version.CreatedAt.Time,
			version.UpdatedAt.Time,
		),
		WorkflowId: version.WorkflowId.String(),
		Order:      int32(version.Order), // nolint: gosec
		Version:    version.Version.String,
	}

	return res
}

type WorkflowConcurrency struct {
	MaxRuns       pgtype.Int4
	LimitStrategy sqlcv1.NullV1ConcurrencyStrategy
	Expression    string
}

func ToWorkflowVersion(
	version *sqlcv1.WorkflowVersion,
	workflow *sqlcv1.Workflow,
	concurrency *WorkflowConcurrency,
	crons []*sqlcv1.WorkflowTriggerCronRef,
	events []*sqlcv1.WorkflowTriggerEventRef,
	schedules []*sqlcv1.WorkflowTriggerScheduledRef,
	stepConcurrency []*sqlcv1.ListConcurrencyStrategiesByWorkflowVersionIdRow,
) *gen.WorkflowVersion {
	wfConfig := make(map[string]interface{})

	if version.CreateWorkflowVersionOpts != nil {
		err := json.Unmarshal(version.CreateWorkflowVersionOpts, &wfConfig)

		if err != nil {
			return nil
		}
	}

	res := &gen.WorkflowVersion{
		Metadata: *toAPIMetadata(
			version.ID.String(),
			version.CreatedAt.Time,
			version.UpdatedAt.Time,
		),
		// WorkflowId:      version.WorkflowId.String(),
		Order:           int32(version.Order), // nolint: gosec
		Version:         version.Version.String,
		ScheduleTimeout: &version.ScheduleTimeout,
		DefaultPriority: &version.DefaultPriority.Int32,
		WorkflowConfig:  &wfConfig,
	}

	if version.Sticky.Valid {
		var stickyStrategy string

		switch version.Sticky.StickyStrategy {
		case sqlcv1.StickyStrategyHARD:
			stickyStrategy = "hard"
		case sqlcv1.StickyStrategySOFT:
			stickyStrategy = "soft"
		}

		res.Sticky = &stickyStrategy
	}

	if version.WorkflowId != uuid.Nil {
		res.Workflow = ToWorkflowFromSQLC(workflow)
	}

	if len(version.InputJsonSchema) > 0 {
		versionMap := make(map[string]interface{})

		err := json.Unmarshal(version.InputJsonSchema, &versionMap)

		if err == nil {
			res.InputJsonSchema = &versionMap
		}
	}

	triggersResp := gen.WorkflowTriggers{}

	if len(crons) > 0 {
		genCrons := make([]gen.WorkflowTriggerCronRef, 0)

		for _, cron := range crons {
			cronCp := cron
			parentId := cronCp.ParentId.String()
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
			if eventCp.ParentId != uuid.Nil {
				parentId := eventCp.ParentId.String()
				genEvents = append(genEvents, gen.WorkflowTriggerEventRef{
					EventKey: &eventCp.EventKey,
					ParentId: &parentId,
				})
			}
		}

		triggersResp.Events = &genEvents
	}

	res.Triggers = &triggersResp
	res.V1Concurrency = ToV1Concurrency(concurrency, stepConcurrency)

	return res
}

func ToV1Concurrency(workflowConcurrency *WorkflowConcurrency, taskConcurrencies []*sqlcv1.ListConcurrencyStrategiesByWorkflowVersionIdRow) *[]gen.ConcurrencySetting {
	res := make([]gen.ConcurrencySetting, 0, len(taskConcurrencies)+1)

	for _, c := range taskConcurrencies {
		res = append(res, gen.ConcurrencySetting{
			StepReadableId: &c.StepReadableID.String,
			Expression:     c.Expression,
			LimitStrategy:  gen.ConcurrencyLimitStrategy(c.Strategy),
			MaxRuns:        c.MaxConcurrency,
			Scope:          gen.ConcurrencyScopeTASK,
		})
	}

	if workflowConcurrency != nil && workflowConcurrency.LimitStrategy.Valid {
		res = append(res, gen.ConcurrencySetting{
			Expression:    workflowConcurrency.Expression,
			LimitStrategy: gen.ConcurrencyLimitStrategy(workflowConcurrency.LimitStrategy.V1ConcurrencyStrategy),
			MaxRuns:       workflowConcurrency.MaxRuns.Int32,
			Scope:         gen.ConcurrencyScopeWORKFLOW,
		})
	}

	return &res
}

func ToJob(job *sqlcv1.Job, steps []*sqlcv1.GetStepsForJobsRow) *gen.Job {
	res := &gen.Job{
		Metadata: *toAPIMetadata(
			job.ID.String(),
			job.CreatedAt.Time,
			job.UpdatedAt.Time,
		),
		Name:        job.Name,
		TenantId:    job.TenantId.String(),
		VersionId:   job.WorkflowVersionId.String(),
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

func ToStep(step *sqlcv1.Step, parents []uuid.UUID) *gen.Step {
	res := &gen.Step{
		Metadata: *toAPIMetadata(
			step.ID.String(),
			step.CreatedAt.Time,
			step.UpdatedAt.Time,
		),
		Action:     step.ActionId,
		JobId:      step.JobId.String(),
		TenantId:   step.TenantId.String(),
		ReadableId: step.ReadableId.String,
		Timeout:    &step.Timeout.String,
	}

	parentStr := make([]string, 0)

	for i := range parents {
		parentStr = append(parentStr, parents[i].String())
	}

	res.Parents = &parentStr

	children := []string{}

	res.Children = &children

	return res
}

func ToWorkflowFromSQLC(row *sqlcv1.Workflow) *gen.Workflow {
	res := &gen.Workflow{
		Metadata:    *toAPIMetadata(row.ID.String(), row.CreatedAt.Time, row.UpdatedAt.Time),
		Name:        row.Name,
		Description: &row.Description.String,
		IsPaused:    &row.IsPaused.Bool,
	}

	return res
}

func ToWorkflowVersionFromSQLC(row *sqlcv1.WorkflowVersion, workflow *gen.Workflow) *gen.WorkflowVersion {
	res := &gen.WorkflowVersion{
		Metadata:   *toAPIMetadata(row.ID.String(), row.CreatedAt.Time, row.UpdatedAt.Time),
		Version:    row.Version.String,
		WorkflowId: row.WorkflowId.String(),
		Order:      int32(row.Order), // nolint: gosec
		Workflow:   workflow,
	}

	return res
}
