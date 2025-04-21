package v1

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/cel"
	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/digest"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

var ErrDagParentNotFound = errors.New("dag parent not found")

type CreateWorkflowVersionOpts struct {
	// (required) the workflow name
	Name string `validate:"required,hatchetName"`

	// (optional) the workflow description
	Description *string `json:"description,omitempty"`

	// (optional) event triggers for the workflow
	EventTriggers []string

	// (optional) cron triggers for the workflow
	CronTriggers []string `validate:"dive,cron"`

	// (optional) the input bytes for the cron triggers
	CronInput []byte

	// (required) the tasks in the workflow
	Tasks []CreateStepOpts `validate:"required,min=1,dive"`

	OnFailure *CreateStepOpts `json:"onFailureJob,omitempty" validate:"omitempty"`

	// (optional) the workflow concurrency groups
	Concurrency []CreateConcurrencyOpts `json:"concurrency,omitempty" validator:"omitempty,dive"`

	// (optional) sticky strategy
	Sticky *string `validate:"omitempty,oneof=SOFT HARD"`

	DefaultPriority *int32 `validate:"omitempty,min=1,max=3"`
}

type CreateCronWorkflowTriggerOpts struct {
	// (required) the workflow id
	WorkflowId string `validate:"required,uuid"`

	// (required) the workflow name
	Name string `validate:"required"`

	Cron string `validate:"required,cron"`

	Input              map[string]interface{}
	AdditionalMetadata map[string]interface{}
}

type CreateConcurrencyOpts struct {
	// (optional) the maximum number of concurrent workflow runs, default 1
	MaxRuns *int32

	// (optional) the strategy to use when the concurrency limit is reached, default CANCEL_IN_PROGRESS
	LimitStrategy *string `validate:"omitnil,oneof=CANCEL_IN_PROGRESS GROUP_ROUND_ROBIN CANCEL_NEWEST"`

	// (required) a concurrency expression for evaluating the concurrency key
	Expression string `validate:"celworkflowrunstr"`
}

type CreateStepOpts struct {
	// (required) the task name
	ReadableId string `validate:"hatchetName"`

	// (required) the task action id
	Action string `validate:"required,actionId"`

	// (optional) the task timeout
	Timeout *string `validate:"omitnil,duration"`

	// (optional) the task scheduling timeout
	ScheduleTimeout *string `validate:"omitnil,duration"`

	// (optional) the parents that this step depends on
	Parents []string `validate:"dive,hatchetName"`

	// (optional) the step retry max
	Retries *int `validate:"omitempty,min=0"`

	// (optional) rate limits for this step
	RateLimits []CreateWorkflowStepRateLimitOpts `validate:"dive"`

	// (optional) desired worker affinity state for this step
	DesiredWorkerLabels map[string]DesiredWorkerLabelOpts `validate:"omitempty"`

	// (optional) the step retry backoff factor
	RetryBackoffFactor *float64 `validate:"omitnil,min=1,max=1000"`

	// (optional) the step retry backoff max seconds (can't be greater than 86400)
	RetryBackoffMaxSeconds *int `validate:"omitnil,min=1,max=86400"`

	// (optional) a list of additional trigger conditions
	TriggerConditions []CreateStepMatchConditionOpt `validate:"omitempty,dive"`

	// (optional) the step concurrency options
	Concurrency []CreateConcurrencyOpts `json:"concurrency,omitempty" validator:"omitnil"`
}

type CreateStepMatchConditionOpt struct {
	// (required) the type of match condition for triggering the step
	MatchConditionKind string `validate:"required,oneof=PARENT_OVERRIDE USER_EVENT SLEEP"`

	// (required) the key for the event data when the workflow is triggered
	ReadableDataKey string `validate:"required"`

	// (required) the initial state for the task when the match condition is satisfied
	Action string `validate:"required,oneof=QUEUE CANCEL SKIP"`

	// (required) the or group id for the match condition
	OrGroupId string `validate:"required,uuid"`

	// (optional) the expression for the match condition
	Expression string `validate:"omitempty"`

	// (optional) the sleep duration for the match condition, only set if this is a SLEEP
	SleepDuration *string `validate:"omitempty,duration"`

	// (optional) the event key for the match condition, only set if this is a USER_EVENT
	EventKey *string `validate:"omitempty"`

	// (optional) if this is a PARENT_OVERRIDE condition, this will be set to the parent readable_id for
	// the parent whose trigger behavior we're overriding
	ParentReadableId *string `validate:"omitempty"`
}

type DesiredWorkerLabelOpts struct {
	// (required) the label key
	Key string `validate:"required"`

	// (required if StringValue is nil) the label integer value
	IntValue *int32 `validate:"omitnil,required_without=StrValue"`

	// (required if StrValue is nil) the label string value
	StrValue *string `validate:"omitnil,required_without=IntValue"`

	// (optional) if the label is required
	Required *bool `validate:"omitempty"`

	// (optional) the weight of the label for scheduling (default: 100)
	Weight *int32 `validate:"omitempty"`

	// (optional) the label comparator for scheduling (default: EQUAL)
	Comparator *string `validate:"omitempty,oneof=EQUAL NOT_EQUAL GREATER_THAN LESS_THAN GREATER_THAN_OR_EQUAL LESS_THAN_OR_EQUAL"`
}

type CreateWorkflowStepRateLimitOpts struct {
	// (required) the rate limit key
	Key string `validate:"required"`

	// (optional) a CEL expression for the rate limit key
	KeyExpr *string `validate:"omitnil,celsteprunstr,required_without=Key"`

	// (optional) the rate limit units to consume
	Units *int `validate:"omitnil,required_without=UnitsExpr"`

	// (optional) a CEL expression for the rate limit units
	UnitsExpr *string `validate:"omitnil,celsteprunstr,required_without=Units"`

	// (optional) a CEL expression for a dynamic limit value for the rate limit
	LimitExpr *string `validate:"omitnil,celsteprunstr"`

	// (optional) the rate limit duration, defaults to MINUTE
	Duration *string `validate:"omitnil,oneof=SECOND MINUTE HOUR DAY WEEK MONTH YEAR"`
}

type WorkflowRepository interface {
	ListWorkflowNamesByIds(ctx context.Context, tenantId string, workflowIds []pgtype.UUID) (map[pgtype.UUID]string, error)
	PutWorkflowVersion(ctx context.Context, tenantId string, opts *CreateWorkflowVersionOpts) (*sqlcv1.GetWorkflowVersionForEngineRow, error)
}

type workflowRepository struct {
	*sharedRepository
}

func newWorkflowRepository(shared *sharedRepository) WorkflowRepository {
	return &workflowRepository{
		sharedRepository: shared,
	}
}

func (w *workflowRepository) ListWorkflowNamesByIds(ctx context.Context, tenantId string, workflowIds []pgtype.UUID) (map[pgtype.UUID]string, error) {
	workflowNames, err := w.queries.ListWorkflowNamesByIds(ctx, w.pool, workflowIds)

	if err != nil {
		return nil, err
	}

	workflowIdToNameMap := make(map[pgtype.UUID]string)

	for _, row := range workflowNames {
		workflowIdToNameMap[row.ID] = row.Name
	}

	return workflowIdToNameMap, nil
}

type JobRunHasCycleError struct {
	JobName string
}

func (e *JobRunHasCycleError) Error() string {
	return fmt.Sprintf("job %s has a cycle", e.JobName)
}

func (r *workflowRepository) PutWorkflowVersion(ctx context.Context, tenantId string, opts *CreateWorkflowVersionOpts) (*sqlcv1.GetWorkflowVersionForEngineRow, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	if hasCycleV1(opts.Tasks) {
		return nil, &JobRunHasCycleError{
			JobName: opts.Name,
		}
	}

	var err error
	opts.Tasks, err = orderWorkflowStepsV1(opts.Tasks)

	if err != nil {
		return nil, err
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 25000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)
	var workflowId pgtype.UUID
	var oldWorkflowVersion *sqlcv1.GetWorkflowVersionForEngineRow

	// check whether the workflow exists
	existingWorkflow, err := r.queries.GetWorkflowByName(ctx, r.pool, sqlcv1.GetWorkflowByNameParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Name:     opts.Name,
	})

	switch {
	case err != nil && errors.Is(err, pgx.ErrNoRows):
		// create the workflow
		workflowId = sqlchelpers.UUIDFromStr(uuid.New().String())

		_, err = r.queries.CreateWorkflow(
			ctx,
			tx,
			sqlcv1.CreateWorkflowParams{
				ID:          workflowId,
				Tenantid:    pgTenantId,
				Name:        opts.Name,
				Description: *opts.Description,
			},
		)

		if err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	case !existingWorkflow.ID.Valid:
		return nil, fmt.Errorf("invalid id for workflow %s", opts.Name)
	default:
		workflowId = existingWorkflow.ID

		// Lock the previous workflow version to prevent concurrent version creation
		_, err := r.queries.LockWorkflowVersion(ctx, tx, workflowId)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("failed to lock previous workflow version: %w", err)
		}

		// fetch the latest workflow version
		workflowVersionIds, err := r.queries.GetLatestWorkflowVersionForWorkflows(ctx, tx, sqlcv1.GetLatestWorkflowVersionForWorkflowsParams{
			Tenantid:    pgTenantId,
			Workflowids: []pgtype.UUID{workflowId},
		})

		if err != nil {
			return nil, err
		}

		if len(workflowVersionIds) != 1 {
			return nil, fmt.Errorf("expected 1 workflow version, got %d", len(workflowVersionIds))
		}

		workflowVersions, err := r.queries.GetWorkflowVersionForEngine(ctx, tx, sqlcv1.GetWorkflowVersionForEngineParams{
			Tenantid: pgTenantId,
			Ids:      []pgtype.UUID{workflowVersionIds[0]},
		})

		if err != nil {
			return nil, err
		}

		if len(workflowVersions) != 1 {
			return nil, fmt.Errorf("expected 1 workflow version, got %d", len(workflowVersions))
		}

		oldWorkflowVersion = workflowVersions[0]
	}

	workflowVersionId, err := r.createWorkflowVersionTxs(ctx, tx, pgTenantId, workflowId, opts, oldWorkflowVersion)

	if err != nil {
		return nil, err
	}

	workflowVersion, err := r.queries.GetWorkflowVersionForEngine(ctx, tx, sqlcv1.GetWorkflowVersionForEngineParams{
		Tenantid: pgTenantId,
		Ids:      []pgtype.UUID{sqlchelpers.UUIDFromStr(workflowVersionId)},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflow version: %w", err)
	}

	if len(workflowVersion) != 1 {
		return nil, fmt.Errorf("expected 1 workflow version when creating new, got %d", len(workflowVersion))
	}

	err = commit(ctx)

	if err != nil {
		return nil, err
	}

	return workflowVersion[0], nil
}

func (r *workflowRepository) createWorkflowVersionTxs(ctx context.Context, tx sqlcv1.DBTX, tenantId, workflowId pgtype.UUID, opts *CreateWorkflowVersionOpts, oldWorkflowVersion *sqlcv1.GetWorkflowVersionForEngineRow) (string, error) {
	workflowVersionId := uuid.New().String()

	cs, err := checksumV1(opts)

	if err != nil {
		return "", err
	}

	// if the checksum matches the old checksum, we don't need to create a new workflow version
	if oldWorkflowVersion != nil && oldWorkflowVersion.WorkflowVersion.Checksum == cs {
		return sqlchelpers.UUIDToStr(oldWorkflowVersion.WorkflowVersion.ID), nil
	}

	createParams := sqlcv1.CreateWorkflowVersionParams{
		ID:         sqlchelpers.UUIDFromStr(workflowVersionId),
		Checksum:   cs,
		Workflowid: workflowId,
	}

	if opts.Sticky != nil {
		createParams.Sticky = sqlcv1.NullStickyStrategy{
			StickyStrategy: sqlcv1.StickyStrategy(*opts.Sticky),
			Valid:          true,
		}
	}

	if opts.DefaultPriority != nil {
		createParams.DefaultPriority = pgtype.Int4{
			Int32: *opts.DefaultPriority,
			Valid: true,
		}
	}
	sqlcWorkflowVersion, err := r.queries.CreateWorkflowVersion(
		ctx,
		tx,
		createParams,
	)

	if err != nil {
		return "", err
	}

	_, err = r.createJobTx(ctx, tx, tenantId, workflowId, sqlcWorkflowVersion.ID, sqlcv1.JobKindDEFAULT, opts.Tasks)

	if err != nil {
		return "", err
	}

	// create the onFailure job if exists
	if opts.OnFailure != nil {
		jobId, err := r.createJobTx(ctx, tx, tenantId, workflowId, sqlcWorkflowVersion.ID, sqlcv1.JobKindONFAILURE, []CreateStepOpts{*opts.OnFailure})

		if err != nil {
			return "", err
		}

		_, err = r.queries.LinkOnFailureJob(ctx, tx, sqlcv1.LinkOnFailureJobParams{
			Workflowversionid: sqlcWorkflowVersion.ID,
			Jobid:             sqlchelpers.UUIDFromStr(jobId),
		})

		if err != nil {
			return "", err
		}
	}

	// create concurrency group
	// NOTE: we do this AFTER the creation of steps/jobs because we have a trigger which depends on the existence
	// of the jobs/steps to create the v1 concurrency groups
	for _, wfConcurrency := range opts.Concurrency {
		params := sqlcv1.CreateWorkflowConcurrencyV1Params{
			Workflowid:        workflowId,
			Workflowversionid: sqlcWorkflowVersion.ID,
			Expression:        wfConcurrency.Expression,
			Tenantid:          tenantId,
		}

		if wfConcurrency.MaxRuns != nil {
			params.MaxRuns = pgtype.Int4{
				Int32: *wfConcurrency.MaxRuns,
				Valid: true,
			}
		}

		var ls sqlcv1.V1ConcurrencyStrategy

		if wfConcurrency.LimitStrategy != nil && *wfConcurrency.LimitStrategy != "" {
			ls = sqlcv1.V1ConcurrencyStrategy(*wfConcurrency.LimitStrategy)
		} else {
			ls = sqlcv1.V1ConcurrencyStrategyCANCELINPROGRESS
		}

		params.Limitstrategy = ls

		wcs, err := r.queries.CreateWorkflowConcurrencyV1(
			ctx,
			tx,
			params,
		)

		if err != nil {
			return "", fmt.Errorf("could not create concurrency group: %w", err)
		}

		err = r.queries.UpdateWorkflowConcurrencyWithChildStrategyIds(
			ctx,
			tx,
			sqlcv1.UpdateWorkflowConcurrencyWithChildStrategyIdsParams{
				Workflowid:            workflowId,
				Workflowversionid:     sqlcWorkflowVersion.ID,
				Workflowconcurrencyid: wcs.ID,
				Childstrategyids:      wcs.ChildStrategyIds,
			},
		)

		if err != nil {
			return "", fmt.Errorf("could not create concurrency group: %w", err)
		}
	}

	// create the workflow triggers
	workflowTriggersId := uuid.New().String()

	sqlcWorkflowTriggers, err := r.queries.CreateWorkflowTriggers(
		ctx,
		tx,
		sqlcv1.CreateWorkflowTriggersParams{
			ID:                sqlchelpers.UUIDFromStr(workflowTriggersId),
			Workflowversionid: sqlcWorkflowVersion.ID,
			Tenantid:          tenantId,
		},
	)

	if err != nil {
		return "", err
	}

	for _, eventTrigger := range opts.EventTriggers {
		_, err := r.queries.CreateWorkflowTriggerEventRef(
			ctx,
			tx,
			sqlcv1.CreateWorkflowTriggerEventRefParams{
				Workflowtriggersid: sqlcWorkflowTriggers.ID,
				Eventtrigger:       eventTrigger,
			},
		)

		if err != nil {
			return "", err
		}
	}

	for _, cronTrigger := range opts.CronTriggers {

		var priority pgtype.Int4

		if opts.DefaultPriority != nil {
			priority = sqlchelpers.ToInt(*opts.DefaultPriority)
		}

		_, err := r.queries.CreateWorkflowTriggerCronRef(
			ctx,
			tx,
			sqlcv1.CreateWorkflowTriggerCronRefParams{
				Workflowtriggersid: sqlcWorkflowTriggers.ID,
				Crontrigger:        cronTrigger,
				Input:              opts.CronInput,
				Name: pgtype.Text{
					String: "",
					Valid:  true,
				},
				Priority: priority,
			},
		)

		if err != nil {
			return "", err
		}

	}

	if oldWorkflowVersion != nil {
		// move existing api crons to the new workflow version
		err = r.queries.MoveCronTriggerToNewWorkflowTriggers(ctx, tx, sqlcv1.MoveCronTriggerToNewWorkflowTriggersParams{
			Oldworkflowversionid: oldWorkflowVersion.WorkflowVersion.ID,
			Newworkflowtriggerid: sqlcWorkflowTriggers.ID,
		})

		if err != nil {
			return "", fmt.Errorf("could not move existing cron triggers to new workflow triggers: %w", err)
		}

		// move existing scheduled triggers to the new workflow version
		err = r.queries.MoveScheduledTriggerToNewWorkflowTriggers(ctx, tx, sqlcv1.MoveScheduledTriggerToNewWorkflowTriggersParams{
			Oldworkflowversionid: oldWorkflowVersion.WorkflowVersion.ID,
			Newworkflowtriggerid: sqlcWorkflowTriggers.ID,
		})

		if err != nil {
			return "", fmt.Errorf("could not move existing scheduled triggers to new workflow triggers: %w", err)
		}
	}

	return workflowVersionId, nil
}

func (r *workflowRepository) createJobTx(ctx context.Context, tx sqlcv1.DBTX, tenantId, workflowId, workflowVersionId pgtype.UUID, jobKind sqlcv1.JobKind, steps []CreateStepOpts) (string, error) {
	if len(steps) == 0 {
		return "", errors.New("no steps provided")
	}

	jobName := steps[0].ReadableId
	jobId := uuid.New().String()

	sqlcJob, err := r.queries.CreateJob(
		ctx,
		tx,
		sqlcv1.CreateJobParams{
			ID:                sqlchelpers.UUIDFromStr(jobId),
			Tenantid:          tenantId,
			Workflowversionid: workflowVersionId,
			Name:              jobName,
			Kind: sqlcv1.NullJobKind{
				Valid:   true,
				JobKind: jobKind,
			},
		},
	)

	if err != nil {
		return "", err
	}

	for _, stepOpts := range steps {
		stepId := uuid.New().String()

		var (
			timeout        pgtype.Text
			customUserData []byte
			retries        pgtype.Int4
		)

		if stepOpts.Timeout != nil {
			timeout = sqlchelpers.TextFromStr(*stepOpts.Timeout)
		}

		if stepOpts.Retries != nil {
			retries = pgtype.Int4{
				Valid: true,
				Int32: int32(*stepOpts.Retries), // nolint: gosec
			}
		}

		// upsert the action
		_, err := r.queries.UpsertAction(
			ctx,
			tx,
			sqlcv1.UpsertActionParams{
				Action:   stepOpts.Action,
				Tenantid: tenantId,
			},
		)

		if err != nil {
			return "", err
		}

		createStepParams := sqlcv1.CreateStepParams{
			ID:             sqlchelpers.UUIDFromStr(stepId),
			Tenantid:       tenantId,
			Jobid:          sqlchelpers.UUIDFromStr(jobId),
			Actionid:       stepOpts.Action,
			Timeout:        timeout,
			Readableid:     stepOpts.ReadableId,
			CustomUserData: customUserData,
			Retries:        retries,
		}

		if stepOpts.ScheduleTimeout != nil {
			createStepParams.ScheduleTimeout = sqlchelpers.TextFromStr(*stepOpts.ScheduleTimeout)
		}

		if stepOpts.RetryBackoffFactor != nil {
			createStepParams.RetryBackoffFactor = pgtype.Float8{
				Float64: *stepOpts.RetryBackoffFactor,
				Valid:   true,
			}
		}

		if stepOpts.RetryBackoffMaxSeconds != nil {
			createStepParams.RetryMaxBackoff = pgtype.Int4{
				Int32: int32(*stepOpts.RetryBackoffMaxSeconds), // nolint: gosec
				Valid: true,
			}
		}

		_, err = r.queries.CreateStep(
			ctx,
			tx,
			createStepParams,
		)

		if err != nil {
			return "", err
		}

		if len(stepOpts.DesiredWorkerLabels) > 0 {
			for i := range stepOpts.DesiredWorkerLabels {
				key := (stepOpts.DesiredWorkerLabels)[i].Key
				value := (stepOpts.DesiredWorkerLabels)[i]

				if key == "" {
					continue
				}

				opts := sqlcv1.UpsertDesiredWorkerLabelParams{
					Stepid: sqlchelpers.UUIDFromStr(stepId),
					Key:    key,
				}

				if value.IntValue != nil {
					opts.IntValue = sqlchelpers.ToInt(*value.IntValue)
				}

				if value.StrValue != nil {
					opts.StrValue = sqlchelpers.TextFromStr(*value.StrValue)
				}

				if value.Weight != nil {
					opts.Weight = sqlchelpers.ToInt(*value.Weight)
				}

				if value.Required != nil {
					opts.Required = sqlchelpers.BoolFromBoolean(*value.Required)
				}

				if value.Comparator != nil {
					opts.Comparator = sqlcv1.NullWorkerLabelComparator{
						WorkerLabelComparator: sqlcv1.WorkerLabelComparator(*value.Comparator),
						Valid:                 true,
					}
				}

				_, err = r.queries.UpsertDesiredWorkerLabel(
					ctx,
					tx,
					opts,
				)

				if err != nil {
					return "", err
				}
			}
		}

		if len(stepOpts.Parents) > 0 {
			err := r.queries.AddStepParents(
				ctx,
				tx,
				sqlcv1.AddStepParentsParams{
					ID:      sqlchelpers.UUIDFromStr(stepId),
					Parents: stepOpts.Parents,
					Jobid:   sqlcJob.ID,
				},
			)

			if err != nil {
				return "", err
			}
		}

		if len(stepOpts.RateLimits) > 0 {
			createStepExprParams := sqlcv1.CreateStepExpressionsParams{
				Stepid: sqlchelpers.UUIDFromStr(stepId),
			}

			for _, rateLimit := range stepOpts.RateLimits {
				// if ANY of the step expressions are not nil, we create ALL options as expressions, but with static
				// keys for any nil expressions.
				if rateLimit.KeyExpr != nil || rateLimit.LimitExpr != nil || rateLimit.UnitsExpr != nil {
					var keyExpr, limitExpr, unitsExpr string

					windowExpr := cel.Str("MINUTE")

					if rateLimit.Duration != nil {
						windowExpr = fmt.Sprintf(`"%s"`, *rateLimit.Duration)
					}

					if rateLimit.KeyExpr != nil {
						keyExpr = *rateLimit.KeyExpr
					} else {
						keyExpr = cel.Str(rateLimit.Key)
					}

					if rateLimit.UnitsExpr != nil {
						unitsExpr = *rateLimit.UnitsExpr
					} else {
						unitsExpr = cel.Int(*rateLimit.Units)
					}

					// create the key expression
					createStepExprParams.Kinds = append(createStepExprParams.Kinds, string(sqlcv1.StepExpressionKindDYNAMICRATELIMITKEY))
					createStepExprParams.Keys = append(createStepExprParams.Keys, rateLimit.Key)
					createStepExprParams.Expressions = append(createStepExprParams.Expressions, keyExpr)

					// create the limit value expression, if it's set
					if rateLimit.LimitExpr != nil {
						limitExpr = *rateLimit.LimitExpr

						createStepExprParams.Kinds = append(createStepExprParams.Kinds, string(sqlcv1.StepExpressionKindDYNAMICRATELIMITVALUE))
						createStepExprParams.Keys = append(createStepExprParams.Keys, rateLimit.Key)
						createStepExprParams.Expressions = append(createStepExprParams.Expressions, limitExpr)
					}

					// create the units value expression
					createStepExprParams.Kinds = append(createStepExprParams.Kinds, string(sqlcv1.StepExpressionKindDYNAMICRATELIMITUNITS))
					createStepExprParams.Keys = append(createStepExprParams.Keys, rateLimit.Key)
					createStepExprParams.Expressions = append(createStepExprParams.Expressions, unitsExpr)

					// create the window expression
					createStepExprParams.Kinds = append(createStepExprParams.Kinds, string(sqlcv1.StepExpressionKindDYNAMICRATELIMITWINDOW))
					createStepExprParams.Keys = append(createStepExprParams.Keys, rateLimit.Key)
					createStepExprParams.Expressions = append(createStepExprParams.Expressions, windowExpr)
				} else {
					_, err := r.queries.CreateStepRateLimit(
						ctx,
						tx,
						sqlcv1.CreateStepRateLimitParams{
							Stepid:       sqlchelpers.UUIDFromStr(stepId),
							Ratelimitkey: rateLimit.Key,
							Units:        int32(*rateLimit.Units), // nolint: gosec
							Tenantid:     tenantId,
							Kind:         sqlcv1.StepRateLimitKindSTATIC,
						},
					)

					if err != nil {
						return "", fmt.Errorf("could not create step rate limit: %w", err)
					}
				}
			}

			if len(createStepExprParams.Kinds) > 0 {
				err := r.queries.CreateStepExpressions(
					ctx,
					tx,
					createStepExprParams,
				)

				if err != nil {
					return "", err
				}
			}
		}

		if len(stepOpts.Concurrency) > 0 {
			for _, concurrency := range stepOpts.Concurrency {
				var maxRuns int32 = 1

				if concurrency.MaxRuns != nil {
					maxRuns = *concurrency.MaxRuns
				}

				strategy := sqlcv1.ConcurrencyLimitStrategyCANCELINPROGRESS

				if concurrency.LimitStrategy != nil {
					strategy = sqlcv1.ConcurrencyLimitStrategy(*concurrency.LimitStrategy)
				}

				_, err := r.queries.CreateStepConcurrency(
					ctx,
					tx,
					sqlcv1.CreateStepConcurrencyParams{
						Workflowid:        workflowId,
						Workflowversionid: workflowVersionId,
						Stepid:            sqlchelpers.UUIDFromStr(stepId),
						Tenantid:          tenantId,
						Expression:        concurrency.Expression,
						Maxconcurrency:    maxRuns,
						Strategy:          sqlcv1.V1ConcurrencyStrategy(strategy),
					},
				)

				if err != nil {
					return "", err
				}
			}
		}

		if len(stepOpts.TriggerConditions) > 0 {
			for _, condition := range stepOpts.TriggerConditions {
				var parentReadableId pgtype.Text

				if condition.ParentReadableId != nil {
					parentReadableId = sqlchelpers.TextFromStr(*condition.ParentReadableId)
				}

				var eventKey pgtype.Text

				if condition.EventKey != nil {
					eventKey = sqlchelpers.TextFromStr(*condition.EventKey)
				}

				var sleepDuration pgtype.Text

				if condition.SleepDuration != nil {
					sleepDuration = sqlchelpers.TextFromStr(*condition.SleepDuration)
				}

				_, err := r.queries.CreateStepMatchCondition(
					ctx,
					tx,
					sqlcv1.CreateStepMatchConditionParams{
						Tenantid:         tenantId,
						Stepid:           sqlchelpers.UUIDFromStr(stepId),
						Readabledatakey:  condition.ReadableDataKey,
						Action:           sqlcv1.V1MatchConditionAction(condition.Action),
						Orgroupid:        sqlchelpers.UUIDFromStr(condition.OrGroupId),
						Expression:       sqlchelpers.TextFromStr(condition.Expression),
						Kind:             sqlcv1.V1StepMatchConditionKind(condition.MatchConditionKind),
						ParentReadableId: parentReadableId,
						EventKey:         eventKey,
						SleepDuration:    sleepDuration,
					},
				)

				if err != nil {
					return "", err
				}
			}
		}

	}

	return jobId, nil
}

func checksumV1(opts *CreateWorkflowVersionOpts) (string, error) {
	var err error
	opts.Tasks, err = orderWorkflowStepsV1(opts.Tasks)

	if err != nil {
		return "", err
	}

	// compute a checksum for the workflow
	declaredValues, err := datautils.ToJSONMap(opts)

	if err != nil {
		return "", err
	}

	workflowChecksum, err := digest.DigestValues(declaredValues)

	if err != nil {
		return "", err
	}

	return workflowChecksum.String(), nil
}

func hasCycleV1(steps []CreateStepOpts) bool {
	graph := make(map[string][]string)
	for _, step := range steps {
		graph[step.ReadableId] = step.Parents
	}

	visited := make(map[string]bool)
	var dfs func(string) bool

	dfs = func(node string) bool {
		if seen, ok := visited[node]; ok && seen {
			return true
		}
		if _, ok := graph[node]; !ok {
			return false
		}
		visited[node] = true
		for _, parent := range graph[node] {
			if dfs(parent) {
				return true
			}
		}
		visited[node] = false
		return false
	}

	for _, step := range steps {
		if dfs(step.ReadableId) {
			return true
		}
	}
	return false
}

func orderWorkflowStepsV1(steps []CreateStepOpts) ([]CreateStepOpts, error) {
	// Build a map of step id to step for quick lookup.
	stepMap := make(map[string]CreateStepOpts)
	for _, step := range steps {
		stepMap[step.ReadableId] = step
	}

	// Initialize in-degree map and adjacency list graph.
	inDegree := make(map[string]int)
	graph := make(map[string][]string)
	for _, step := range steps {
		inDegree[step.ReadableId] = 0
	}

	// Build the graph and compute in-degrees.
	for _, step := range steps {
		for _, parent := range step.Parents {
			if _, exists := stepMap[parent]; !exists {
				return nil, fmt.Errorf("unknown parent step: %s", parent)
			}
			graph[parent] = append(graph[parent], step.ReadableId)
			inDegree[step.ReadableId]++
		}
	}

	// Queue for steps with no incoming edges.
	var queue []string
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	var ordered []CreateStepOpts
	// Process the steps in topological order.
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		ordered = append(ordered, stepMap[id])
		for _, child := range graph[id] {
			inDegree[child]--
			if inDegree[child] == 0 {
				queue = append(queue, child)
			}
		}
	}

	// If not all steps are processed, there is a cycle.
	if len(ordered) != len(steps) {
		return nil, fmt.Errorf("cycle detected in workflow steps")
	}

	return ordered, nil
}
