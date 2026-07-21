package transformers

import (
	"encoding/json"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func ToWorkflow(
	workflow *sqlcv1.Workflow,
	version *sqlcv1.WorkflowVersion,
) *gen.Workflow {

	res := &gen.Workflow{
		Metadata: *toAPIMetadata(
			workflow.ID,
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
			version.ID,
			version.CreatedAt.Time,
			version.UpdatedAt.Time,
		),
		WorkflowId: version.WorkflowId.String(),
		Order:      int32(version.Order), // nolint: gosec
		Version:    version.Version.String,
	}

	return res
}

func ToWorkflowVersion(
	version *sqlcv1.WorkflowVersion,
	workflow *sqlcv1.Workflow,
	workflowConcurrency []*sqlcv1.ListWorkflowConcurrencyByVersionIdRow,
	crons []*sqlcv1.WorkflowTriggerCronRef,
	events []*sqlcv1.WorkflowTriggerEventRef,
	schedules []*sqlcv1.WorkflowTriggerScheduledRef,
	stepConcurrency []*sqlcv1.ListConcurrencyStrategiesByWorkflowVersionIdRow,
	taskData *repository.WorkflowVersionTaskData,
) *gen.WorkflowVersion {
	wfConfig := make(map[string]interface{})

	if version.CreateWorkflowVersionOpts != nil {
		if err := json.Unmarshal(version.CreateWorkflowVersionOpts, &wfConfig); err != nil {
			return nil
		}
	}

	res := &gen.WorkflowVersion{
		Metadata: *toAPIMetadata(
			version.ID,
			version.CreatedAt.Time,
			version.UpdatedAt.Time,
		),
		WorkflowId:      version.WorkflowId.String(),
		Order:           int32(version.Order), // nolint: gosec
		Version:         version.Version.String,
		ScheduleTimeout: &version.ScheduleTimeout,
		DefaultPriority: &version.DefaultPriority.Int32,
		WorkflowConfig:  &wfConfig,
	}

	if workflow.Description.Valid && workflow.Description.String != "" {
		description := workflow.Description.String
		res.Description = &description
	}

	if version.IdempotencyKeyExpression.Valid {
		res.Idempotency = &gen.WorkflowVersionIdempotency{
			Expression: version.IdempotencyKeyExpression.String,
			TtlMs:      version.IdempotencyKeyTtlMs.Int64,
		}
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
	res.V1Concurrency = ToV1Concurrency(workflowConcurrency, stepConcurrency)
	res.Tasks = ToWorkflowVersionTasks(taskData)

	return res
}

func ToWorkflowVersionTasks(taskData *repository.WorkflowVersionTaskData) *[]gen.WorkflowVersionTask {
	if taskData == nil {
		empty := make([]gen.WorkflowVersionTask, 0)
		return &empty
	}

	readableIdByStepId := make(map[uuid.UUID]string, len(taskData.Steps))
	for _, step := range taskData.Steps {
		readableIdByStepId[step.ID] = step.ReadableId.String
	}

	staticRateLimitsByStep := make(map[uuid.UUID][]*sqlcv1.StepRateLimit)
	for _, rl := range taskData.RateLimits {
		staticRateLimitsByStep[rl.StepId] = append(staticRateLimitsByStep[rl.StepId], rl)
	}

	dynamicRateLimitsByStep := make(map[uuid.UUID]map[string][]*sqlcv1.StepExpression)
	keyOrderByStep := make(map[uuid.UUID][]string)
	for _, expr := range taskData.Expressions {
		byKey, ok := dynamicRateLimitsByStep[expr.StepId]
		if !ok {
			byKey = make(map[string][]*sqlcv1.StepExpression)
			dynamicRateLimitsByStep[expr.StepId] = byKey
		}

		if _, seen := byKey[expr.Key]; !seen {
			keyOrderByStep[expr.StepId] = append(keyOrderByStep[expr.StepId], expr.Key)
		}

		byKey[expr.Key] = append(byKey[expr.Key], expr)
	}

	workerLabelsByStep := make(map[uuid.UUID][]*sqlcv1.StepDesiredWorkerLabel)
	for _, label := range taskData.WorkerLabels {
		workerLabelsByStep[label.StepId] = append(workerLabelsByStep[label.StepId], label)
	}

	res := make([]gen.WorkflowVersionTask, 0, len(taskData.Steps))

	for _, step := range taskData.Steps {
		parents := make([]string, 0, len(step.Parents))
		for _, parentId := range step.Parents {
			if readableId, ok := readableIdByStepId[parentId]; ok {
				parents = append(parents, readableId)
			}
		}

		genTask := gen.WorkflowVersionTask{
			ReadableId:      step.ReadableId.String,
			Action:          step.ActionId,
			Parents:         parents,
			Retries:         step.Retries,
			ScheduleTimeout: &step.ScheduleTimeout,
			IsDurable:       &step.IsDurable,
			RateLimits: ToWorkflowVersionTaskRateLimits(
				staticRateLimitsByStep[step.ID],
				dynamicRateLimitsByStep[step.ID],
				keyOrderByStep[step.ID],
			),
			DesiredWorkerLabels: ToWorkflowVersionTaskDesiredWorkerLabels(workerLabelsByStep[step.ID]),
		}

		if step.Timeout.Valid && step.Timeout.String != "" {
			timeout := step.Timeout.String
			genTask.Timeout = &timeout
		}

		if step.RetryBackoffFactor.Valid {
			factor := step.RetryBackoffFactor.Float64
			genTask.RetryBackoffFactor = &factor
		}

		if step.RetryMaxBackoff.Valid {
			maxBackoff := step.RetryMaxBackoff.Int32
			genTask.RetryBackoffMaxSeconds = &maxBackoff
		}

		res = append(res, genTask)
	}

	return &res
}

func ToWorkflowVersionTaskRateLimits(
	static []*sqlcv1.StepRateLimit,
	dynamicByKey map[string][]*sqlcv1.StepExpression,
	keyOrder []string,
) *[]gen.WorkflowVersionTaskRateLimit {
	res := make([]gen.WorkflowVersionTaskRateLimit, 0, len(static)+len(keyOrder))

	for _, rl := range static {
		key := rl.RateLimitKey
		units := rl.Units
		res = append(res, gen.WorkflowVersionTaskRateLimit{
			Key:   &key,
			Units: &units,
		})
	}

	for _, key := range keyOrder {
		genRl := gen.WorkflowVersionTaskRateLimit{}

		for _, expr := range dynamicByKey[key] {
			expression := expr.Expression

			switch expr.Kind {
			case sqlcv1.StepExpressionKindDYNAMICRATELIMITKEY:
				genRl.KeyExpression = &expression
			case sqlcv1.StepExpressionKindDYNAMICRATELIMITVALUE:
				genRl.LimitExpression = &expression
			case sqlcv1.StepExpressionKindDYNAMICRATELIMITUNITS:
				genRl.UnitsExpression = &expression
			case sqlcv1.StepExpressionKindDYNAMICRATELIMITWINDOW:
				genRl.Duration = &expression
			}
		}

		res = append(res, genRl)
	}

	return &res
}

func ToWorkflowVersionTaskDesiredWorkerLabels(labels []*sqlcv1.StepDesiredWorkerLabel) *[]gen.WorkflowVersionTaskDesiredWorkerLabel {
	res := make([]gen.WorkflowVersionTaskDesiredWorkerLabel, 0, len(labels))

	for _, label := range labels {
		required := label.Required
		weight := label.Weight
		comparator := string(label.Comparator)

		genLabel := gen.WorkflowVersionTaskDesiredWorkerLabel{
			Key:        label.Key,
			Required:   &required,
			Weight:     &weight,
			Comparator: &comparator,
		}

		if label.StrValue.Valid {
			strValue := label.StrValue.String
			genLabel.StrValue = &strValue
		}

		if label.IntValue.Valid {
			intValue := label.IntValue.Int32
			genLabel.IntValue = &intValue
		}

		res = append(res, genLabel)
	}

	return &res
}

func ToV1Concurrency(workflowConcurrencies []*sqlcv1.ListWorkflowConcurrencyByVersionIdRow, taskConcurrencies []*sqlcv1.ListConcurrencyStrategiesByWorkflowVersionIdRow) *[]gen.ConcurrencySetting {
	res := make([]gen.ConcurrencySetting, 0, len(taskConcurrencies)+len(workflowConcurrencies))

	for _, c := range taskConcurrencies {
		res = append(res, gen.ConcurrencySetting{
			StepReadableId: &c.StepReadableID.String,
			Expression:     c.Expression,
			LimitStrategy:  gen.ConcurrencyLimitStrategy(c.Strategy),
			MaxRuns:        c.MaxConcurrency,
			Scope:          gen.ConcurrencyScopeTASK,
		})
	}

	for _, wc := range workflowConcurrencies {
		res = append(res, gen.ConcurrencySetting{
			Expression:    wc.Expression,
			LimitStrategy: gen.ConcurrencyLimitStrategy(wc.LimitStrategy),
			MaxRuns:       wc.MaxRuns,
			Scope:         gen.ConcurrencyScopeWORKFLOW,
		})
	}

	return &res
}

func ToJob(job *sqlcv1.Job, steps []*sqlcv1.GetStepsForJobsRow) *gen.Job {
	res := &gen.Job{
		Metadata: *toAPIMetadata(
			job.ID,
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
	isDurable := step.IsDurable
	res := &gen.Step{
		Metadata: *toAPIMetadata(
			step.ID,
			step.CreatedAt.Time,
			step.UpdatedAt.Time,
		),
		Action:     step.ActionId,
		JobId:      step.JobId.String(),
		TenantId:   step.TenantId.String(),
		ReadableId: step.ReadableId.String,
		Timeout:    &step.Timeout.String,
		IsDurable:  &isDurable,
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
		Metadata:    *toAPIMetadata(row.ID, row.CreatedAt.Time, row.UpdatedAt.Time),
		Name:        row.Name,
		Description: &row.Description.String,
		IsPaused:    &row.IsPaused.Bool,
	}

	return res
}

func ToWorkflowVersionFromSQLC(row *sqlcv1.WorkflowVersion, workflow *gen.Workflow) *gen.WorkflowVersion {
	res := &gen.WorkflowVersion{
		Metadata:   *toAPIMetadata(row.ID, row.CreatedAt.Time, row.UpdatedAt.Time),
		Version:    row.Version.String,
		WorkflowId: row.WorkflowId.String(),
		Order:      int32(row.Order), // nolint: gosec
		Workflow:   workflow,
	}

	return res
}
