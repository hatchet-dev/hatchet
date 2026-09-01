package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/cel"
	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/digest"
	"github.com/hatchet-dev/hatchet/internal/listutils"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

var ErrDagParentNotFound = errors.New("dag parent not found")

type DagNode struct {
	// include both of these to make it easier to render the dag in other places
	ParentReadableIds []string
	ChildReadableIds  []string

	// todo: expand this to include conditions and other stuff so we can render dags from json?
}

type DagShape map[StepReadableId]DagNode

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
	Concurrency []CreateConcurrencyOpts `json:"concurrency,omitempty" validate:"omitempty,dive"`

	// (optional) sticky strategy
	Sticky *string `validate:"omitempty,oneof=SOFT HARD"`

	DefaultPriority *int32 `validate:"omitempty,min=1,max=3"`

	DefaultFilters []types.DefaultFilter `json:"defaultFilters,omitempty" validate:"omitempty,dive"`

	InputJsonSchema []byte `json:"inputJsonSchema,omitempty"`

	Idempotency *IdempotencyConfig `json:"idempotency,omitempty"`
}

type IdempotencyConfig struct {
	Expression string                   `json:"expression" validate:"required,celworkflowrunstr"`
	TTLMs      int64                    `json:"ttlMs" validate:"required,min=1"`
	Method     sqlcv1.IdempotencyMethod `json:"method" validate:"required,oneof=TTL STATUS"`
}

type CreateConcurrencyOpts struct {
	// (optional) the maximum number of concurrent workflow runs, default 1
	MaxRuns *int32 `json:"maxRuns,omitempty" validate:"omitempty,min=1"`

	// (optional) the strategy to use when the concurrency limit is reached, default CANCEL_IN_PROGRESS
	LimitStrategy *string `validate:"omitnil,oneof=CANCEL_IN_PROGRESS GROUP_ROUND_ROBIN CANCEL_NEWEST CANCEL_QUEUED_EXCEPT_NEWEST CANCEL_QUEUED_EXCEPT_OLDEST"`

	// (required) a concurrency expression for evaluating the concurrency key
	Expression string `json:"expression" validate:"celworkflowrunstr"`

	// (required when TenantScoped) the strategy name; unique per tenant for tenant-scoped
	// strategies
	Name string `json:"name,omitempty" validate:"omitempty,hatchetName"`

	// (optional) when true, the entry defines (or updates in place) a tenant-scoped
	// strategy shared across workflows, keyed by Name
	TenantScoped bool `json:"tenantScoped,omitempty"`
}

// IsTenantScoped reports whether the entry declares a tenant-scoped strategy.
func (c *CreateConcurrencyOpts) IsTenantScoped() bool {
	return c.TenantScoped
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

	// (optional) whether this step is durable
	IsDurable bool `json:"isDurable,omitempty"`

	// (optional) whether this step is the synthetic DAG orchestrator step
	IsDagOrchestrator bool `json:"isDagOrchestrator,omitempty"`

	// (optional) slot requests for this step (slot_type -> units)
	SlotRequests map[string]int32 `json:"slotRequests,omitempty" validate:"omitempty,dive,keys,required,endkeys,gt=0"`

	// (optional) a list of additional trigger conditions
	TriggerConditions []CreateStepMatchConditionOpt `validate:"omitempty,dive"`

	// (optional) the step concurrency options
	Concurrency []CreateConcurrencyOpts `json:"concurrency,omitempty" validate:"omitempty,dive"`

	// (optional) batch execution configuration
	BatchConfig *StepBatchConfig `json:"batchConfig,omitempty"`
}

type StepBatchConfig struct {
	BatchMaxSize      int32   `json:"batchMaxSize" validate:"required,min=1,max=100000"`
	BatchMaxInterval  *int32  `json:"batchMaxInterval,omitempty" validate:"omitempty,min=1,max=86400000"`
	BatchGroupKey     *string `json:"batchGroupKey,omitempty"`
	BatchGroupMaxRuns *int32  `json:"batchGroupMaxRuns,omitempty" validate:"omitempty,min=1,max=10000"`
	BroadcastOutput   bool    `json:"broadcastOutput,omitempty"`
}

type CreateStepMatchConditionOpt struct {
	SleepDuration      *string   `validate:"omitempty,duration"`
	EventKey           *string   `validate:"omitempty"`
	ParentReadableId   *string   `validate:"omitempty"`
	MatchConditionKind string    `validate:"required,oneof=PARENT_OVERRIDE USER_EVENT SLEEP"`
	ReadableDataKey    string    `validate:"required"`
	Action             string    `validate:"required,oneof=QUEUE CANCEL SKIP"`
	OrGroupId          uuid.UUID `json:"-" validate:"required"`
	Expression         string    `validate:"omitempty"`
	OrGroupIdIndex     int32
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
	Duration *string `validate:"omitnil"`
}

var allowedRateLimitDurations = []string{
	"SECOND",
	"MINUTE",
	"HOUR",
	"DAY",
	"WEEK",
	"MONTH",
	"YEAR",
}

type ListWorkflowsOpts struct {
	// (optional) number of workflows to skip
	Offset *int

	// (optional) number of workflows to return
	Limit *int

	// (optional) the workflow name to filter by
	Name *string
}

type ListWorkflowsResult struct {
	Rows  []*sqlcv1.Workflow
	Count int
}

type WorkflowMetrics struct {
	// the number of runs for a specific group key
	GroupKeyRunsCount int `json:"groupKeyRunsCount,omitempty"`

	// the total number of concurrency group keys
	GroupKeyCount int `json:"groupKeyCount,omitempty"`
}

type WorkflowRepository interface {
	ListWorkflowNamesByIds(ctx context.Context, tenantId uuid.UUID, workflowIds []uuid.UUID) (map[uuid.UUID]string, error)
	PutWorkflowVersion(ctx context.Context, tenantId uuid.UUID, opts *CreateWorkflowVersionOpts) (*sqlcv1.GetWorkflowVersionForEngineRow, error)
	GetWorkflowShape(ctx context.Context, workflowVersionId uuid.UUID) ([]*sqlcv1.GetWorkflowShapeRow, error)
	ListStepsByWorkflowVersionId(ctx context.Context, tenantId uuid.UUID, workflowVersionId uuid.UUID) ([]*sqlcv1.ListStepsByWorkflowVersionIdsRow, error)
	ListStepMatchConditions(ctx context.Context, tenantId uuid.UUID, stepIds []uuid.UUID) ([]*sqlcv1.V1StepMatchCondition, error)

	// ListWorkflows returns all workflows for a given tenant.
	ListWorkflows(tenantId uuid.UUID, opts *ListWorkflowsOpts) (*ListWorkflowsResult, error)

	// GetWorkflowById returns a workflow by its name. It will return db.ErrNotFound if the workflow does not exist.
	GetWorkflowById(ctx context.Context, workflowId uuid.UUID) (*sqlcv1.GetWorkflowByIdRow, error)

	// GetWorkflowVersionById returns a workflow version by its id. It will return db.ErrNotFound if the workflow
	// version does not exist.
	GetWorkflowVersionWithTriggers(ctx context.Context, tenantId uuid.UUID, workflowVersionId uuid.UUID) (*sqlcv1.GetWorkflowVersionByIdRow,
		[]*sqlcv1.WorkflowTriggerCronRef,
		[]*sqlcv1.WorkflowTriggerEventRef,
		[]*sqlcv1.WorkflowTriggerScheduledRef,
		[]*sqlcv1.ListConcurrencyStrategiesByWorkflowVersionIdRow,
		[]*sqlcv1.ListWorkflowConcurrencyByVersionIdRow,
		error)

	GetWorkflowVersionById(ctx context.Context, tenantId uuid.UUID, workflowId uuid.UUID) (*sqlcv1.GetWorkflowVersionForEngineRow, error)

	// DeleteWorkflow deletes a workflow for a given tenant.
	DeleteWorkflow(ctx context.Context, tenantId uuid.UUID, workflowId uuid.UUID) (*sqlcv1.Workflow, error)

	GetWorkflowByName(ctx context.Context, tenantId uuid.UUID, workflowName string) (*sqlcv1.Workflow, error)

	GetLatestWorkflowVersion(ctx context.Context, tenantId uuid.UUID, workflowId uuid.UUID) (*sqlcv1.GetWorkflowVersionForEngineRow, error)

	PauseWorkflow(ctx context.Context, workflowId uuid.UUID, opts PauseWorkflowOpts) (*sqlcv1.Workflow, error)

	UnpauseWorkflow(ctx context.Context, workflowId uuid.UUID) (*sqlcv1.Workflow, error)

	MovePausedWorkflowQueueItems(ctx context.Context, tenantId uuid.UUID, workflowIds []uuid.UUID) error

	RequeuePausedWorkflowQueueItems(ctx context.Context, tenantId uuid.UUID, workflowIds []uuid.UUID) (queueNames []string, concurrencyStrategyIds []int64, err error)

	ListUnpausedWorkflowsWithPausedQueueItems(ctx context.Context, tenantId uuid.UUID) ([]uuid.UUID, error)
}

type workflowRepository struct {
	*sharedRepository
}

func newWorkflowRepository(shared *sharedRepository) WorkflowRepository {
	return &workflowRepository{
		sharedRepository: shared,
	}
}

func (r *workflowRepository) ListWorkflowNamesByIds(ctx context.Context, tenantId uuid.UUID, workflowIds []uuid.UUID) (map[uuid.UUID]string, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-workflow-names-by-ids")
	defer span.End()

	workflowNames, err := r.queries.ListWorkflowNamesByIds(ctx, r.pool, workflowIds)

	if err != nil {
		return nil, err
	}

	workflowIdToNameMap := make(map[uuid.UUID]string)

	for _, row := range workflowNames {
		workflowIdToNameMap[row.ID] = row.Name
	}

	return workflowIdToNameMap, nil
}

func (r *workflowRepository) ListStepsByWorkflowVersionId(ctx context.Context, tenantId uuid.UUID, workflowVersionId uuid.UUID) ([]*sqlcv1.ListStepsByWorkflowVersionIdsRow, error) {
	steps, err := r.listStepsByWorkflowVersionIds(ctx, r.pool, tenantId, []uuid.UUID{workflowVersionId})

	if err != nil {
		return nil, err
	}

	return steps[workflowVersionId], nil
}

func (r *workflowRepository) ListStepMatchConditions(ctx context.Context, tenantId uuid.UUID, stepIds []uuid.UUID) ([]*sqlcv1.V1StepMatchCondition, error) {
	return r.listStepMatchConditions(ctx, r.pool, tenantId, stepIds)
}

func (r *workflowRepository) upsertTenantConcurrencyStrategy(ctx context.Context, dbtx sqlcv1.DBTX, tenantId uuid.UUID, opts *CreateConcurrencyOpts) (*sqlcv1.V1TenantConcurrency, error) {
	if opts.Name == "" {
		return nil, fmt.Errorf("tenant-scoped concurrency entries require a name")
	}

	var maxRuns int32 = 1

	if opts.MaxRuns != nil {
		maxRuns = *opts.MaxRuns
	}

	strategy := sqlcv1.V1ConcurrencyStrategyCANCELINPROGRESS

	if opts.LimitStrategy != nil {
		strategy = sqlcv1.V1ConcurrencyStrategy(*opts.LimitStrategy)
	}

	return r.queries.UpsertTenantConcurrencyStrategy(ctx, dbtx, sqlcv1.UpsertTenantConcurrencyStrategyParams{
		Tenantid:       tenantId,
		Name:           opts.Name,
		Strategy:       strategy,
		Expression:     opts.Expression,
		Maxconcurrency: maxRuns,
	})
}

// collectTenantConcurrencyDefs returns the tenant-scoped concurrency definitions (name +
// expression set) declared anywhere on the workflow, deduplicated by name (last
// declaration wins).
func collectTenantConcurrencyDefs(opts *CreateWorkflowVersionOpts) []*CreateConcurrencyOpts {
	byName := make(map[string]*CreateConcurrencyOpts)
	names := make([]string, 0)

	collect := func(entries []CreateConcurrencyOpts) {
		for i := range entries {
			entry := &entries[i]

			if !entry.IsTenantScoped() {
				continue
			}

			if _, seen := byName[entry.Name]; !seen {
				names = append(names, entry.Name)
			}

			byName[entry.Name] = entry
		}
	}

	collect(opts.Concurrency)

	for i := range opts.Tasks {
		collect(opts.Tasks[i].Concurrency)
	}

	if opts.OnFailure != nil {
		collect(opts.OnFailure.Concurrency)
	}

	defs := make([]*CreateConcurrencyOpts, 0, len(names))

	for _, name := range names {
		defs = append(defs, byName[name])
	}

	return defs
}

// getTenantStrategiesForEntries resolves the tenant-scoped entries of one step's
// concurrency list to their strategy rows (upserted earlier in this transaction),
// erroring on duplicate references, which would collide in the task's slot chain.
func (r *workflowRepository) getTenantStrategiesForEntries(ctx context.Context, dbtx sqlcv1.DBTX, tenantId uuid.UUID, stepReadableId string, entries []CreateConcurrencyOpts) (map[string]*sqlcv1.V1TenantConcurrency, error) {
	names := make([]string, 0)
	seen := make(map[string]bool)

	for _, entry := range entries {
		if !entry.IsTenantScoped() {
			continue
		}

		if seen[entry.Name] {
			return nil, fmt.Errorf("tenant-scoped concurrency strategy %s is referenced more than once by step %s", entry.Name, stepReadableId)
		}

		seen[entry.Name] = true
		names = append(names, entry.Name)
	}

	if len(names) == 0 {
		return nil, nil
	}

	strategies, err := r.queries.GetTenantConcurrencyStrategiesByNames(ctx, dbtx, sqlcv1.GetTenantConcurrencyStrategiesByNamesParams{
		Tenantid: tenantId,
		Names:    names,
	})

	if err != nil {
		return nil, err
	}

	strategiesByName := make(map[string]*sqlcv1.V1TenantConcurrency, len(strategies))

	for _, strategy := range strategies {
		strategiesByName[strategy.Name] = strategy
	}

	missingNames := make([]string, 0)

	for _, name := range names {
		if _, ok := strategiesByName[name]; !ok {
			missingNames = append(missingNames, name)
		}
	}

	if len(missingNames) > 0 {
		return nil, fmt.Errorf("unknown tenant-scoped concurrency strategies: %s (step %s)", strings.Join(missingNames, ", "), stepReadableId)
	}

	return strategiesByName, nil
}

// checkTenantConcurrencyOrdering rejects registrations whose chains order tenant-scoped
// strategies inconsistently, either within this registration or against other workflows'
// latest versions. Tasks hold earlier slots in their chain while queued on later ones, so
// a cycle in the combined ordering can deadlock until scheduling timeouts fire. Chains do
// not have to be identical; they only must not order the same strategies differently
// (checked as a cycle in the combined precedence graph, which also catches cycles spread
// across more than two chains).
func (r *workflowRepository) checkTenantConcurrencyOrdering(ctx context.Context, tx sqlcv1.DBTX, tenantId, workflowId uuid.UUID, opts *CreateWorkflowVersionOpts) error {
	type chain struct {
		label string
		names []string
	}

	chains := make([]chain, 0)
	allNames := make(map[string]bool)

	collect := func(label string, entries []CreateConcurrencyOpts) {
		names := make([]string, 0)

		for _, entry := range entries {
			if entry.IsTenantScoped() {
				names = append(names, entry.Name)
				allNames[entry.Name] = true
			}
		}

		if len(names) >= 2 {
			chains = append(chains, chain{label: label, names: names})
		}
	}

	for i := range opts.Tasks {
		collect(fmt.Sprintf("task %s", opts.Tasks[i].ReadableId), opts.Tasks[i].Concurrency)
	}

	if opts.OnFailure != nil {
		collect(fmt.Sprintf("task %s", opts.OnFailure.ReadableId), opts.OnFailure.Concurrency)
	}

	// a single tenant-scoped entry per chain imposes no ordering, and existing chains are
	// mutually consistent by induction, so there is nothing to check
	if len(chains) == 0 {
		return nil
	}

	names := make([]string, 0, len(allNames))

	for name := range allNames {
		names = append(names, name)
	}

	strategies, err := r.queries.GetTenantConcurrencyStrategiesByNames(ctx, tx, sqlcv1.GetTenantConcurrencyStrategiesByNamesParams{
		Tenantid: tenantId,
		Names:    names,
	})

	if err != nil {
		return err
	}

	idsByName := make(map[string]int64, len(strategies))
	namesById := make(map[int64]string, len(strategies))

	for _, strategy := range strategies {
		idsByName[strategy.Name] = strategy.ID
		namesById[strategy.ID] = strategy.Name
	}

	// edges[a][b] holds the label of a chain ordering a before b
	edges := make(map[int64]map[int64]string)

	addChain := func(label string, ids []int64) {
		for i := 0; i < len(ids); i++ {
			for j := i + 1; j < len(ids); j++ {
				if edges[ids[i]] == nil {
					edges[ids[i]] = make(map[int64]string)
				}

				if _, ok := edges[ids[i]][ids[j]]; !ok {
					edges[ids[i]][ids[j]] = label
				}
			}
		}
	}

	for _, c := range chains {
		ids := make([]int64, len(c.names))

		for i, name := range c.names {
			ids[i] = idsByName[name]
		}

		addChain(fmt.Sprintf("%s of workflow %s", c.label, opts.Name), ids)
	}

	existing, err := r.queries.ListTenantConcurrencyOrderings(ctx, tx, sqlcv1.ListTenantConcurrencyOrderingsParams{
		Tenantid:          tenantId,
		Excludeworkflowid: workflowId,
	})

	if err != nil {
		return err
	}

	existingIds := make([]int64, 0)

	for _, row := range existing {
		addChain(fmt.Sprintf("a step of workflow %s", row.WorkflowID), row.StrategyIds)

		for _, id := range row.StrategyIds {
			if _, ok := namesById[id]; !ok {
				existingIds = append(existingIds, id)
			}
		}
	}

	if len(existingIds) > 0 {
		existingStrategies, err := r.queries.GetTenantConcurrencyStrategiesByIds(ctx, tx, sqlcv1.GetTenantConcurrencyStrategiesByIdsParams{
			Tenantid: tenantId,
			Ids:      existingIds,
		})

		if err != nil {
			return err
		}

		for _, strategy := range existingStrategies {
			namesById[strategy.ID] = strategy.Name
		}
	}

	// DFS cycle detection over the combined precedence graph
	const (
		unvisited = 0
		inStack   = 1
		done      = 2
	)

	state := make(map[int64]int)

	var cycleFrom, cycleTo int64

	var visit func(id int64) bool
	visit = func(id int64) bool {
		state[id] = inStack

		for next := range edges[id] {
			switch state[next] {
			case inStack:
				cycleFrom, cycleTo = id, next
				return true
			case unvisited:
				if visit(next) {
					return true
				}
			}
		}

		state[id] = done

		return false
	}

	for id := range edges {
		if state[id] == unvisited && visit(id) {
			label := edges[cycleFrom][cycleTo]

			return fmt.Errorf(
				"tenant-scoped concurrency strategies %s and %s are ordered inconsistently across concurrency chains (%s orders %s before %s, which conflicts with the ordering elsewhere); chains sharing tenant-scoped strategies must order them consistently to avoid deadlocks",
				namesById[cycleFrom], namesById[cycleTo],
				label, namesById[cycleFrom], namesById[cycleTo],
			)
		}
	}

	return nil
}

type JobRunHasCycleError struct {
	JobName string
}

func (e *JobRunHasCycleError) Error() string {
	return fmt.Sprintf("job %s has a cycle", e.JobName)
}

func (r *workflowRepository) PutWorkflowVersion(ctx context.Context, tenantId uuid.UUID, opts *CreateWorkflowVersionOpts) (*sqlcv1.GetWorkflowVersionForEngineRow, error) {
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

	tx, commit, rollback, err := sqlchelpers.PrepareTxWithStatementTimeout(ctx, r.pool, r.l, 60000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	// Upsert tenant-scoped concurrency definitions (entries carrying both a name and an
	// expression) first, in the same transaction, so the entries created below can
	// reference them. This runs before the checksum early-return on purpose: definitions
	// are tenant-level state, so re-registering an unchanged workflow with a changed
	// definition must still update the strategy in place.
	for _, entry := range collectTenantConcurrencyDefs(opts) {
		if _, err := r.upsertTenantConcurrencyStrategy(ctx, tx, tenantId, entry); err != nil {
			return nil, fmt.Errorf("could not upsert tenant concurrency strategy %s: %w", entry.Name, err)
		}
	}

	var workflowId uuid.UUID
	var oldWorkflowVersion *sqlcv1.GetWorkflowVersionForEngineRow

	// check whether the workflow exists
	existingWorkflow, err := r.queries.GetWorkflowByName(ctx, tx, sqlcv1.GetWorkflowByNameParams{
		Tenantid: tenantId,
		Name:     opts.Name,
	})

	switch {
	case err != nil && errors.Is(err, pgx.ErrNoRows):
		// create the workflow
		workflowId = uuid.New()

		_, err = r.queries.CreateWorkflow(
			ctx,
			tx,
			sqlcv1.CreateWorkflowParams{
				ID:          workflowId,
				Tenantid:    tenantId,
				Name:        opts.Name,
				Description: *opts.Description,
			},
		)

		if err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	case existingWorkflow.ID == uuid.Nil:
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
			Tenantid:    tenantId,
			Workflowids: []uuid.UUID{workflowId},
		})

		if err != nil {
			return nil, err
		}

		if len(workflowVersionIds) != 1 {
			return nil, fmt.Errorf("expected 1 workflow version, got %d", len(workflowVersionIds))
		}

		workflowVersions, err := r.queries.GetWorkflowVersionForEngine(ctx, tx, sqlcv1.GetWorkflowVersionForEngineParams{
			Tenantid: tenantId,
			Ids:      []uuid.UUID{workflowVersionIds[0]},
		})

		if err != nil {
			return nil, err
		}

		if len(workflowVersions) != 1 {
			return nil, fmt.Errorf("expected 1 workflow version, got %d", len(workflowVersions))
		}

		oldWorkflowVersion = workflowVersions[0]
	}

	workflowVersionId, err := r.createWorkflowVersionTxs(ctx, tx, tenantId, workflowId, opts, oldWorkflowVersion)

	if err != nil {
		return nil, err
	}

	workflowVersion, err := r.queries.GetWorkflowVersionForEngine(ctx, tx, sqlcv1.GetWorkflowVersionForEngineParams{
		Tenantid: tenantId,
		Ids:      []uuid.UUID{*workflowVersionId},
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

// mergeWorkflowConcurrencyOntoSingleTask moves workflow-level concurrency onto the
// sole DEFAULT task when the workflow is a single-task DAG (one task, no on-failure).
// Workflow concurrency creates a parent strategy that forces the DAG concurrency path;
// a one-task workflow is a standalone task, so those strategies belong on the task
// (workflow strategies first, then any existing task strategies).
func mergeWorkflowConcurrencyOntoSingleTask(opts *CreateWorkflowVersionOpts) {
	if opts == nil || len(opts.Concurrency) == 0 {
		return
	}

	if len(opts.Tasks) != 1 || opts.OnFailure != nil {
		return
	}

	merged := make([]CreateConcurrencyOpts, 0, len(opts.Concurrency)+len(opts.Tasks[0].Concurrency))
	merged = append(merged, opts.Concurrency...)
	merged = append(merged, opts.Tasks[0].Concurrency...)
	opts.Tasks[0].Concurrency = merged
	opts.Concurrency = nil
}

func (r *workflowRepository) createWorkflowVersionTxs(ctx context.Context, tx sqlcv1.DBTX, tenantId, workflowId uuid.UUID, opts *CreateWorkflowVersionOpts, oldWorkflowVersion *sqlcv1.GetWorkflowVersionForEngineRow) (*uuid.UUID, error) {
	workflowVersionId := uuid.New()

	mergeWorkflowConcurrencyOntoSingleTask(opts)

	dagOperatorEnabled, err := r.isDagOperatorEnabled(ctx, tx, tenantId)

	if err != nil {
		return nil, err
	}

	// todo: maybe don't need `len` check here?
	isUsingDagOperator := dagOperatorEnabled && len(opts.Tasks) > 1

	if isUsingDagOperator {
		var retentionPeriod *string

		if rp := r.limitConfig.CorePartitionRetentionOrDefault(); rp != "" {
			retentionPeriod = &rp
		}

		// big number of retries to make it very unlikely we exhaust them
		numRetries := 10_000

		orchestrator := CreateStepOpts{
			ReadableId:        opts.Name,
			Action:            strings.ToLower(fmt.Sprintf("%s_orchestrator", opts.Name)),
			IsDurable:         true,
			IsDagOrchestrator: true,
			Timeout:           retentionPeriod,
			ScheduleTimeout:   retentionPeriod,
			Retries:           &numRetries,
		}

		// Tenant-scoped workflow-level entries become concurrency on the orchestrator
		// task (in array order); workflow-scoped entries keep the parent fan-out below,
		// which always gates first.
		remaining := make([]CreateConcurrencyOpts, 0, len(opts.Concurrency))

		for _, entry := range opts.Concurrency {
			if entry.IsTenantScoped() {
				orchestrator.Concurrency = append(orchestrator.Concurrency, entry)
			} else {
				remaining = append(remaining, entry)
			}
		}

		opts.Concurrency = remaining
		opts.Tasks = append(opts.Tasks, orchestrator)
	}

	// On the old DAG path there is no single task to attach tenant-scoped workflow-level
	// concurrency to, and the parent/child machinery does not support it.
	for _, entry := range opts.Concurrency {
		if entry.IsTenantScoped() {
			return nil, fmt.Errorf("tenant-scoped concurrency (name %s) is not supported at the workflow level unless the workflow is a single task or uses the DAG operator", entry.Name)
		}
	}

	if err := r.checkTenantConcurrencyOrdering(ctx, tx, tenantId, workflowId, opts); err != nil {
		return nil, err
	}

	cs, modifiedOpts, err := checksumV1(opts)

	if err != nil {
		return nil, err
	}

	// if the checksum matches and the version's DAG operator flag matches the current server
	// flag, reuse the existing version — no change needed.
	if oldWorkflowVersion != nil && oldWorkflowVersion.WorkflowVersion.Checksum == cs &&
		oldWorkflowVersion.WorkflowVersion.IsUsingDagOperator == isUsingDagOperator {
		return &oldWorkflowVersion.WorkflowVersion.ID, nil
	}

	optsJson, err := json.Marshal(modifiedOpts)

	if err != nil {
		return nil, err
	}

	createParams := sqlcv1.CreateWorkflowVersionParams{
		ID:                        workflowVersionId,
		Checksum:                  cs,
		Workflowid:                workflowId,
		CreateWorkflowVersionOpts: optsJson,
		InputJsonSchema:           opts.InputJsonSchema,
		IsUsingDagOperator:        sqlchelpers.BoolFromBoolean(isUsingDagOperator),
	}

	if opts.Sticky != nil {
		createParams.Sticky = sqlcv1.NullStickyStrategy{
			StickyStrategy: sqlcv1.StickyStrategy(*opts.Sticky),
			Valid:          true,
		}
	}

	if opts.Idempotency != nil {
		idempotency := *opts.Idempotency

		createParams.IdempotencyKeyExpression = pgtype.Text{
			String: idempotency.Expression,
			Valid:  true,
		}
		createParams.IdempotencyKeyTtlMs = sqlchelpers.ToBigInt(&idempotency.TTLMs)
		createParams.IdempotencyMethod = sqlcv1.NullIdempotencyMethod{
			IdempotencyMethod: idempotency.Method,
			Valid:             true,
		}
	}

	if opts.DefaultPriority != nil {
		createParams.DefaultPriority = pgtype.Int4{
			Int32: *opts.DefaultPriority,
			Valid: true,
		}
	}

	if isUsingDagOperator {
		shape := make(DagShape)

		for _, t := range opts.Tasks {
			if t.IsDagOrchestrator {
				continue
			}

			children := make([]string, 0)
			for _, maybeChild := range opts.Tasks {
				if slices.Contains(maybeChild.Parents, t.ReadableId) {
					children = append(children, maybeChild.ReadableId)
				}
			}

			shape[StepReadableId(t.ReadableId)] = DagNode{
				ParentReadableIds: t.Parents,
				ChildReadableIds:  children,
			}
		}

		dagShape, err := json.Marshal(shape)

		if err != nil {
			return nil, err
		}

		createParams.DagShape = dagShape
	}

	sqlcWorkflowVersion, err := r.queries.CreateWorkflowVersion(
		ctx,
		tx,
		createParams,
	)

	if err != nil {
		return nil, err
	}

	_, err = r.createJobTx(ctx, tx, tenantId, workflowId, sqlcWorkflowVersion.ID, sqlcv1.JobKindDEFAULT, opts.Tasks)

	if err != nil {
		return nil, err
	}

	// create the onFailure job if exists
	if opts.OnFailure != nil {
		jobId, err := r.createJobTx(ctx, tx, tenantId, workflowId, sqlcWorkflowVersion.ID, sqlcv1.JobKindONFAILURE, []CreateStepOpts{*opts.OnFailure})

		if err != nil {
			return nil, err
		}

		_, err = r.queries.LinkOnFailureJob(ctx, tx, sqlcv1.LinkOnFailureJobParams{
			Workflowversionid: sqlcWorkflowVersion.ID,
			Jobid:             *jobId,
		})

		if err != nil {
			return nil, err
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
			return nil, fmt.Errorf("could not create concurrency group: %w", err)
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
			return nil, fmt.Errorf("could not create concurrency group: %w", err)
		}
	}

	// create the workflow triggers
	workflowTriggersId := uuid.New()

	sqlcWorkflowTriggers, err := r.queries.CreateWorkflowTriggers(
		ctx,
		tx,
		sqlcv1.CreateWorkflowTriggersParams{
			ID:                workflowTriggersId,
			Workflowversionid: sqlcWorkflowVersion.ID,
			Tenantid:          tenantId,
		},
	)

	if err != nil {
		return nil, err
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
			return nil, err
		}
	}

	for _, cronTrigger := range opts.CronTriggers {

		var priority pgtype.Int4

		if opts.DefaultPriority != nil {
			priority = sqlchelpers.ToInt(opts.DefaultPriority)
		}

		var oldWorkflowVersionId uuid.UUID
		if oldWorkflowVersion != nil {
			oldWorkflowVersionId = oldWorkflowVersion.WorkflowVersion.ID
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
				Priority:             priority,
				OldWorkflowVersionId: &oldWorkflowVersionId,
			},
		)

		if err != nil {
			return nil, err
		}

	}

	if oldWorkflowVersion != nil {
		// move existing api crons to the new workflow version
		err = r.queries.MoveCronTriggerToNewWorkflowTriggers(ctx, tx, sqlcv1.MoveCronTriggerToNewWorkflowTriggersParams{
			Oldworkflowversionid: oldWorkflowVersion.WorkflowVersion.ID,
			Newworkflowtriggerid: sqlcWorkflowTriggers.ID,
		})

		if err != nil {
			return nil, fmt.Errorf("could not move existing cron triggers to new workflow triggers: %w", err)
		}

		// delete DEFAULT cron refs from the old version — they were re-created fresh above for the new version,
		// and leaving them accumulates dead rows that the PollCronSchedules query locks every 15 seconds
		err = r.queries.DeleteOldDefaultCronTriggersForWorkflowVersion(ctx, tx, oldWorkflowVersion.WorkflowVersion.ID)

		if err != nil {
			return nil, fmt.Errorf("could not delete old DEFAULT cron triggers: %w", err)
		}

		// move existing scheduled triggers to the new workflow version
		err = r.queries.MoveScheduledTriggerToNewWorkflowTriggers(ctx, tx, sqlcv1.MoveScheduledTriggerToNewWorkflowTriggersParams{
			Oldworkflowversionid: oldWorkflowVersion.WorkflowVersion.ID,
			Newworkflowtriggerid: sqlcWorkflowTriggers.ID,
		})

		if err != nil {
			return nil, fmt.Errorf("could not move existing scheduled triggers to new workflow triggers: %w", err)
		}
	}

	if len(opts.DefaultFilters) > 0 {
		filterScopes := make([]string, len(opts.DefaultFilters))
		filterExpressions := make([]string, len(opts.DefaultFilters))
		filterPayloads := make([][]byte, len(opts.DefaultFilters))

		for ix, filter := range opts.DefaultFilters {
			var payload []byte

			if filter.Payload != nil {
				payload, err = json.Marshal(filter.Payload)

				if err != nil {
					return nil, fmt.Errorf("could not marshal filter payload: %w", err)
				}
			}

			filterScopes[ix] = filter.Scope
			filterExpressions[ix] = filter.Expression
			filterPayloads[ix] = payload
		}

		err := r.queries.DeleteExistingDeclarativeFiltersForOverwrite(
			ctx,
			tx,
			sqlcv1.DeleteExistingDeclarativeFiltersForOverwriteParams{
				Tenantid:   tenantId,
				Workflowid: workflowId,
			},
		)

		if err != nil {
			return nil, fmt.Errorf("could not delete existing declarative filters: %w", err)
		}

		err = r.queries.BulkInsertDeclarativeFilters(
			ctx,
			tx,
			sqlcv1.BulkInsertDeclarativeFiltersParams{
				Tenantid:    tenantId,
				Workflowid:  workflowId,
				Scopes:      filterScopes,
				Expressions: filterExpressions,
				Payloads:    filterPayloads,
			},
		)

		if err != nil {
			return nil, fmt.Errorf("could not upsert declarative filters: %w", err)
		}
	}

	return &workflowVersionId, nil
}

func (r *workflowRepository) createJobTx(ctx context.Context, tx sqlcv1.DBTX, tenantId, workflowId, workflowVersionId uuid.UUID, jobKind sqlcv1.JobKind, steps []CreateStepOpts) (*uuid.UUID, error) {
	if len(steps) == 0 {
		return nil, errors.New("no steps provided")
	}

	jobName := steps[0].ReadableId
	jobId := uuid.New()

	sqlcJob, err := r.queries.CreateJob(
		ctx,
		tx,
		sqlcv1.CreateJobParams{
			ID:                jobId,
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
		return nil, err
	}

	for _, stepOpts := range steps {
		stepId := uuid.New()

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
			return nil, err
		}

		createStepParams := sqlcv1.CreateStepParams{
			ID:                stepId,
			Tenantid:          tenantId,
			Jobid:             jobId,
			Actionid:          stepOpts.Action,
			Timeout:           timeout,
			Readableid:        stepOpts.ReadableId,
			CustomUserData:    customUserData,
			Retries:           retries,
			IsDurable:         sqlchelpers.BoolFromBoolean(stepOpts.IsDurable),
			IsDagOrchestrator: sqlchelpers.BoolFromBoolean(stepOpts.IsDagOrchestrator),
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
			return nil, err
		}

		if stepOpts.BatchConfig != nil {
			batchCfgParams := sqlcv1.CreateStepBatchConfigParams{
				Stepid:          stepId,
				Batchmaxsize:    stepOpts.BatchConfig.BatchMaxSize,
				Broadcastoutput: stepOpts.BatchConfig.BroadcastOutput,
			}
			if stepOpts.BatchConfig.BatchMaxInterval != nil {
				batchCfgParams.BatchMaxInterval = pgtype.Int4{
					Int32: *stepOpts.BatchConfig.BatchMaxInterval,
					Valid: true,
				}
			}
			if stepOpts.BatchConfig.BatchGroupKey != nil {
				batchCfgParams.BatchGroupKey = pgtype.Text{
					String: *stepOpts.BatchConfig.BatchGroupKey,
					Valid:  true,
				}
			}
			if stepOpts.BatchConfig.BatchGroupMaxRuns != nil {
				batchCfgParams.BatchGroupMaxRuns = pgtype.Int4{
					Int32: *stepOpts.BatchConfig.BatchGroupMaxRuns,
					Valid: true,
				}
			}
			if err = r.queries.CreateStepBatchConfig(ctx, tx, batchCfgParams); err != nil {
				return nil, err
			}
		}

		slotRequests := stepOpts.SlotRequests
		if len(slotRequests) == 0 {
			if stepOpts.IsDurable {
				slotRequests = map[string]int32{SlotTypeDurable: 1}
			} else {
				slotRequests = map[string]int32{SlotTypeDefault: 1}
			}
		}

		slotTypes := make([]string, 0, len(slotRequests))
		units := make([]int32, 0, len(slotRequests))
		for slotType, unit := range slotRequests {
			if unit <= 0 {
				continue
			}
			slotTypes = append(slotTypes, slotType)
			units = append(units, unit)
		}

		if len(slotTypes) == 0 {
			slotTypes = append(slotTypes, SlotTypeDefault)
			units = append(units, 1)
		}

		err = r.queries.CreateStepSlotRequests(
			ctx,
			tx,
			sqlcv1.CreateStepSlotRequestsParams{
				Tenantid:  tenantId,
				Stepid:    stepId,
				Slottypes: slotTypes,
				Units:     units,
			},
		)

		if err != nil {
			return nil, err
		}

		// upsert the queue based on the action
		// note: we don't use the postCommit func, it just sets the queue in the cache which is not necessary for writing a
		// workflow version, only when we're inserting a bunch of tasks for that queue
		_, err = r.upsertQueues(ctx, tx, tenantId, []string{createStepParams.Actionid})

		if err != nil {
			return nil, err
		}

		if len(stepOpts.DesiredWorkerLabels) > 0 {
			for i := range stepOpts.DesiredWorkerLabels {
				key := (stepOpts.DesiredWorkerLabels)[i].Key
				value := (stepOpts.DesiredWorkerLabels)[i]

				if key == "" {
					continue
				}

				opts := sqlcv1.UpsertDesiredWorkerLabelParams{
					Stepid: stepId,
					Key:    key,
				}

				if value.IntValue != nil {
					opts.IntValue = sqlchelpers.ToInt(value.IntValue)
				}

				if value.StrValue != nil {
					opts.StrValue = sqlchelpers.TextFromStr(*value.StrValue)
				}

				if value.Weight != nil {
					opts.Weight = sqlchelpers.ToInt(value.Weight)
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
					return nil, err
				}
			}
		}

		if len(stepOpts.Parents) > 0 {
			err := r.queries.AddStepParents(
				ctx,
				tx,
				sqlcv1.AddStepParentsParams{
					ID:      stepId,
					Parents: stepOpts.Parents,
					Jobid:   sqlcJob.ID,
				},
			)

			if err != nil {
				return nil, err
			}
		}

		if len(stepOpts.RateLimits) > 0 {
			createStepExprParams := sqlcv1.CreateStepExpressionsParams{
				Stepid: stepId,
			}

			for _, rateLimit := range stepOpts.RateLimits {
				// if ANY of the step expressions are not nil, we create ALL options as expressions, but with static
				// keys for any nil expressions.
				if rateLimit.KeyExpr != nil || rateLimit.LimitExpr != nil || rateLimit.UnitsExpr != nil {
					var keyExpr, limitExpr, unitsExpr string

					windowExpr := cel.Str("MINUTE")

					if rateLimit.Duration != nil {
						if slices.Contains(allowedRateLimitDurations, strings.ToUpper(*rateLimit.Duration)) {
							windowExpr = cel.Str(strings.ToUpper(*rateLimit.Duration))
						} else {
							windowExpr = *rateLimit.Duration
						}
					}

					if rateLimit.KeyExpr != nil {
						keyExpr = *rateLimit.KeyExpr
					} else {
						keyExpr = cel.Str(rateLimit.Key)
					}

					if rateLimit.UnitsExpr != nil {
						unitsExpr = *rateLimit.UnitsExpr
					} else if rateLimit.Units != nil {
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
					rlUnits := int32(1)

					if rateLimit.Units != nil {
						rlUnits = int32(*rateLimit.Units) // nolint: gosec
					}

					_, err := r.queries.CreateStepRateLimit(
						ctx,
						tx,
						sqlcv1.CreateStepRateLimitParams{
							Stepid:       stepId,
							Ratelimitkey: rateLimit.Key,
							Units:        rlUnits, // nolint: gosec
							Tenantid:     tenantId,
							Kind:         sqlcv1.StepRateLimitKindSTATIC,
						},
					)

					if err != nil {
						return nil, fmt.Errorf("could not create step rate limit: %w", err)
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
					return nil, err
				}
			}
		}

		if len(stepOpts.Concurrency) > 0 {
			// Entries are created in array order; chain order at task-insert time follows
			// creation order (ascending strategy row id), so a tenant-scoped entry may
			// come before or after a workflow-scoped one.
			tenantStrategies, err := r.getTenantStrategiesForEntries(ctx, tx, tenantId, stepOpts.ReadableId, stepOpts.Concurrency)

			if err != nil {
				return nil, err
			}

			for _, concurrency := range stepOpts.Concurrency {
				var maxRuns int32 = 1

				if concurrency.MaxRuns != nil {
					maxRuns = *concurrency.MaxRuns
				}

				strategy := sqlcv1.ConcurrencyLimitStrategyCANCELINPROGRESS

				if concurrency.LimitStrategy != nil {
					strategy = sqlcv1.ConcurrencyLimitStrategy(*concurrency.LimitStrategy)
				}

				params := sqlcv1.CreateStepConcurrencyParams{
					Workflowid:        workflowId,
					Workflowversionid: workflowVersionId,
					Stepid:            stepId,
					Tenantid:          tenantId,
					Expression:        concurrency.Expression,
					Maxconcurrency:    maxRuns,
					Strategy:          sqlcv1.V1ConcurrencyStrategy(strategy),
				}

				if concurrency.IsTenantScoped() {
					// The referencing row carries a copy of the tenant strategy's current
					// definition (kept in sync by the v1_tenant_concurrency update trigger).
					tenantStrategy := tenantStrategies[concurrency.Name]
					params.Expression = tenantStrategy.Expression
					params.Maxconcurrency = tenantStrategy.MaxConcurrency
					params.Strategy = tenantStrategy.Strategy
					params.TenantStrategyId = pgtype.Int8{Int64: tenantStrategy.ID, Valid: true}
				}

				_, err := r.queries.CreateStepConcurrency(
					ctx,
					tx,
					params,
				)

				if err != nil {
					return nil, err
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
						Stepid:           stepId,
						Readabledatakey:  condition.ReadableDataKey,
						Action:           sqlcv1.V1MatchConditionAction(condition.Action),
						Orgroupid:        condition.OrGroupId,
						Expression:       sqlchelpers.TextFromStr(condition.Expression),
						Kind:             sqlcv1.V1StepMatchConditionKind(condition.MatchConditionKind),
						ParentReadableId: parentReadableId,
						EventKey:         eventKey,
						SleepDuration:    sleepDuration,
					},
				)

				if err != nil {
					return nil, err
				}
			}
		}

	}

	return &jobId, nil
}

func (r *workflowRepository) GetWorkflowShape(ctx context.Context, workflowVersionId uuid.UUID) ([]*sqlcv1.GetWorkflowShapeRow, error) {
	return r.queries.GetWorkflowShape(ctx, r.pool, workflowVersionId)
}

func (r *workflowRepository) ListWorkflows(tenantId uuid.UUID, opts *ListWorkflowsOpts) (*ListWorkflowsResult, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	res := &ListWorkflowsResult{}

	queryParams := sqlcv1.ListWorkflowsParams{
		Tenantid: tenantId,
	}

	countParams := sqlcv1.CountWorkflowsParams{
		TenantId: tenantId,
	}

	if opts.Offset != nil {
		queryParams.Offset = *opts.Offset
	}

	if opts.Limit != nil {
		queryParams.Limit = *opts.Limit
	}

	if opts.Name != nil {
		search := pgtype.Text{String: *opts.Name, Valid: true}
		queryParams.Search = search
		countParams.Search = search
	}

	orderByField := "createdAt"
	orderByDirection := "DESC"

	queryParams.Orderby = orderByField + " " + orderByDirection

	tx, err := r.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer sqlchelpers.DeferRollback(context.Background(), r.l, tx.Rollback)

	workflows, err := r.queries.ListWorkflows(context.Background(), tx, queryParams)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflows: %w", err)
	}

	count, err := r.queries.CountWorkflows(context.Background(), tx, countParams)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return nil, err
	}

	res.Count = int(count)

	sqlcWorkflows := make([]*sqlcv1.Workflow, len(workflows))

	for i := range workflows {
		sqlcWorkflows[i] = &workflows[i].Workflow
	}

	res.Rows = sqlcWorkflows

	return res, nil
}

func (r *workflowRepository) GetWorkflowById(ctx context.Context, workflowId uuid.UUID) (*sqlcv1.GetWorkflowByIdRow, error) {
	return r.queries.GetWorkflowById(context.Background(), r.pool, workflowId)
}

func (r *workflowRepository) GetWorkflowVersionWithTriggers(ctx context.Context, tenantId uuid.UUID, workflowVersionId uuid.UUID) (
	*sqlcv1.GetWorkflowVersionByIdRow,
	[]*sqlcv1.WorkflowTriggerCronRef,
	[]*sqlcv1.WorkflowTriggerEventRef,
	[]*sqlcv1.WorkflowTriggerScheduledRef,
	[]*sqlcv1.ListConcurrencyStrategiesByWorkflowVersionIdRow,
	[]*sqlcv1.ListWorkflowConcurrencyByVersionIdRow,
	error,
) {
	row, err := r.queries.GetWorkflowVersionById(
		ctx,
		r.pool,
		workflowVersionId,
	)

	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to fetch workflow version: %w", err)
	}

	crons, err := r.queries.GetWorkflowVersionCronTriggerRefs(
		ctx,
		r.pool,
		workflowVersionId,
	)

	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to fetch cron triggers: %w", err)
	}

	events, err := r.queries.GetWorkflowVersionEventTriggerRefs(
		ctx,
		r.pool,
		workflowVersionId,
	)

	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to fetch event triggers: %w", err)
	}

	scheduled, err := r.queries.GetWorkflowVersionScheduleTriggerRefs(
		ctx,
		r.pool,
		workflowVersionId,
	)

	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to fetch scheduled triggers: %w", err)
	}

	stepConcurrency, err := r.queries.ListConcurrencyStrategiesByWorkflowVersionId(ctx, r.pool, sqlcv1.ListConcurrencyStrategiesByWorkflowVersionIdParams{
		Tenantid:          tenantId,
		Workflowversionid: row.WorkflowVersion.ID,
		Workflowid:        row.Workflow.ID,
	})

	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to fetch step concurrency strategies: %w", err)
	}

	workflowConcurrency, err := r.queries.ListWorkflowConcurrencyByVersionId(ctx, r.pool, sqlcv1.ListWorkflowConcurrencyByVersionIdParams{
		Workflowversionid: row.WorkflowVersion.ID,
		Workflowid:        row.Workflow.ID,
	})

	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to fetch workflow concurrency strategies: %w", err)
	}

	return row, crons, events, scheduled, stepConcurrency, workflowConcurrency, nil
}

func (r *workflowRepository) GetWorkflowVersionById(ctx context.Context, tenantId, workflowId uuid.UUID) (*sqlcv1.GetWorkflowVersionForEngineRow, error) {
	versions, err := r.queries.GetWorkflowVersionForEngine(ctx, r.pool, sqlcv1.GetWorkflowVersionForEngineParams{
		Tenantid: tenantId,
		Ids:      []uuid.UUID{workflowId},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflow version: %w", err)
	}

	if len(versions) != 1 {
		return nil, fmt.Errorf("expected 1 workflow version when getting by id, got %d", len(versions))
	}

	return versions[0], nil
}

func (r *workflowRepository) DeleteWorkflow(ctx context.Context, tenantId uuid.UUID, workflowId uuid.UUID) (*sqlcv1.Workflow, error) {
	return r.queries.SoftDeleteWorkflow(ctx, r.pool, workflowId)
}

func (r *workflowRepository) GetWorkflowByName(ctx context.Context, tenantId uuid.UUID, workflowName string) (*sqlcv1.Workflow, error) {
	return r.queries.GetWorkflowByName(ctx, r.pool, sqlcv1.GetWorkflowByNameParams{
		Tenantid: tenantId,
		Name:     workflowName,
	})
}

func (r *workflowRepository) GetLatestWorkflowVersion(ctx context.Context, tenantId uuid.UUID, workflowId uuid.UUID) (*sqlcv1.GetWorkflowVersionForEngineRow, error) {
	versionId, err := r.queries.GetWorkflowLatestVersion(ctx, r.pool, workflowId)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest version: %w", err)
	}

	versions, err := r.queries.GetWorkflowVersionForEngine(ctx, r.pool, sqlcv1.GetWorkflowVersionForEngineParams{
		Tenantid: tenantId,
		Ids:      []uuid.UUID{versionId},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflow version: %w", err)
	}

	if len(versions) != 1 {
		return nil, fmt.Errorf("expected 1 workflow version for latest, got %d", len(versions))
	}

	return versions[0], nil
}

type WorkflowPauseScheduledCronRunQueueBehavior string

const (
	WorkflowPauseScheduledCronRunQueueBehaviorDROP  = "DROP"
	WorkflowPauseScheduledCronRunQueueBehaviorQUEUE = "QUEUE"
)

type PauseWorkflowOpts struct {
	CronRunQueueBehavior      WorkflowPauseScheduledCronRunQueueBehavior
	ScheduledRunQueueBehavior WorkflowPauseScheduledCronRunQueueBehavior
	QueueTTL                  string
}

func (r *workflowRepository) PauseWorkflow(ctx context.Context, workflowId uuid.UUID, opts PauseWorkflowOpts) (*sqlcv1.Workflow, error) {
	return r.queries.PauseWorkflow(ctx, r.pool, sqlcv1.PauseWorkflowParams{
		ID:                        workflowId,
		Cronrunqueuebehavior:      sqlcv1.WorkflowPauseQueueBehavior(opts.CronRunQueueBehavior),
		Scheduledrunqueuebehavior: sqlcv1.WorkflowPauseQueueBehavior(opts.ScheduledRunQueueBehavior),
		Queuettl:                  opts.QueueTTL,
	})
}

func (r *workflowRepository) UnpauseWorkflow(ctx context.Context, workflowId uuid.UUID) (*sqlcv1.Workflow, error) {
	return r.queries.UnpauseWorkflow(ctx, r.pool, workflowId)
}

func (r *workflowRepository) MovePausedWorkflowQueueItems(ctx context.Context, tenantId uuid.UUID, workflowIds []uuid.UUID) error {
	ctx, span := telemetry.NewSpan(ctx, "move-paused-workflow-queue-items")
	defer span.End()

	if len(workflowIds) == 0 {
		return nil
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		return err
	}

	defer rollback()

	if err := r.queries.MovePausedWorkflowQueueItems(ctx, tx, sqlcv1.MovePausedWorkflowQueueItemsParams{
		Workflowids: workflowIds,
		Tenantid:    tenantId,
	}); err != nil {
		return err
	}

	if err := r.queries.MovePausedWorkflowConcurrencySlots(ctx, tx, sqlcv1.MovePausedWorkflowConcurrencySlotsParams{
		Workflowids: workflowIds,
		Tenantid:    tenantId,
	}); err != nil {
		return err
	}

	if err := r.queries.MovePausedWorkflowRateLimitedQueueItems(ctx, tx, sqlcv1.MovePausedWorkflowRateLimitedQueueItemsParams{
		Workflowids: workflowIds,
		Tenantid:    tenantId,
	}); err != nil {
		return err
	}

	return commit(ctx)
}

func (r *workflowRepository) RequeuePausedWorkflowQueueItems(ctx context.Context, tenantId uuid.UUID, workflowIds []uuid.UUID) ([]string, []int64, error) {
	ctx, span := telemetry.NewSpan(ctx, "requeue-paused-workflow-queue-items")
	defer span.End()

	if len(workflowIds) == 0 {
		return nil, nil, nil
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		return nil, nil, err
	}

	defer rollback()

	queueNames, err := r.queries.RequeuePausedWorkflowQueueItems(ctx, tx, sqlcv1.RequeuePausedWorkflowQueueItemsParams{
		Workflowids: workflowIds,
		Tenantid:    tenantId,
	})

	if err != nil {
		return nil, nil, err
	}

	concurrencyStrategyIds, err := r.queries.RequeuePausedWorkflowConcurrencySlots(ctx, tx, sqlcv1.RequeuePausedWorkflowConcurrencySlotsParams{
		Workflowids: workflowIds,
		Tenantid:    tenantId,
	})

	if err != nil {
		return nil, nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, nil, err
	}

	return listutils.Uniq(queueNames), listutils.Uniq(concurrencyStrategyIds), nil
}

func (r *workflowRepository) ListUnpausedWorkflowsWithPausedQueueItems(ctx context.Context, tenantId uuid.UUID) ([]uuid.UUID, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-unpaused-workflows-with-paused-queue-items")
	defer span.End()

	return r.queries.ListUnpausedWorkflowsWithPausedQueueItems(ctx, r.pool, tenantId)
}

func checksumV1(opts *CreateWorkflowVersionOpts) (string, *CreateWorkflowVersionOpts, error) {
	var err error
	opts.Tasks, err = orderWorkflowStepsV1(opts.Tasks)

	if err != nil {
		return "", opts, err
	}

	// Generate a unique index for each or group id in the workflow, and add this to the trigger condition.
	// We would like to update the workflow version checksum only when the combination of or group ids changes.
	orGroupIdsToIndex := make(map[uuid.UUID]int32)

	for i, task := range opts.Tasks {
		for j, condition := range task.TriggerConditions {
			if condition.OrGroupId == uuid.Nil {
				// generate a new UUID for the or group id
				condition.OrGroupId = uuid.New()
			}

			// if the or group id is not in the map, add it
			if _, exists := orGroupIdsToIndex[condition.OrGroupId]; !exists {
				orGroupIdsToIndex[condition.OrGroupId] = int32(len(orGroupIdsToIndex)) // nolint: gosec
			}

			// set the index for the or group id
			condition.OrGroupIdIndex = orGroupIdsToIndex[condition.OrGroupId]
			opts.Tasks[i].TriggerConditions[j] = condition
		}
	}

	// Normalize fields for backwards-compatible checksums:
	// default values that didn't exist before this feature should not change the hash.
	for i := range opts.Tasks {
		// SlotRequests={"default": 1}is the new default; strip it so it doesn't affect the hash.
		sr := opts.Tasks[i].SlotRequests
		if len(sr) == 1 {
			if units, ok := sr[SlotTypeDefault]; ok && units == 1 {
				opts.Tasks[i].SlotRequests = nil
			}
		}
	}

	// Tenant-scoped concurrency definitions are tenant-level state, not workflow shape:
	// changing one must update the strategy in place, not mint a new workflow version. So
	// tenant-scoped entries hash by name and position only, with the definition fields
	// stripped from a copy.
	hashOpts := *opts
	hashOpts.Concurrency = stripTenantConcurrencyDefs(opts.Concurrency)
	hashOpts.Tasks = make([]CreateStepOpts, len(opts.Tasks))

	for i, task := range opts.Tasks {
		hashOpts.Tasks[i] = task
		hashOpts.Tasks[i].Concurrency = stripTenantConcurrencyDefs(task.Concurrency)
	}

	if opts.OnFailure != nil {
		onFailure := *opts.OnFailure
		onFailure.Concurrency = stripTenantConcurrencyDefs(opts.OnFailure.Concurrency)
		hashOpts.OnFailure = &onFailure
	}

	// compute a checksum for the workflow
	declaredValues, err := datautils.ToJSONMap(&hashOpts)

	if err != nil {
		return "", opts, err
	}

	workflowChecksum, err := digest.DigestValues(declaredValues)

	if err != nil {
		return "", opts, err
	}

	return workflowChecksum.String(), opts, nil
}

// stripTenantConcurrencyDefs blanks the definition fields of tenant-scoped entries so the
// workflow version checksum covers only the referenced name and its position in the chain.
func stripTenantConcurrencyDefs(entries []CreateConcurrencyOpts) []CreateConcurrencyOpts {
	if len(entries) == 0 {
		return entries
	}

	stripped := make([]CreateConcurrencyOpts, len(entries))

	for i, entry := range entries {
		stripped[i] = entry

		if entry.IsTenantScoped() {
			stripped[i].Expression = ""
			stripped[i].MaxRuns = nil
			stripped[i].LimitStrategy = nil
		}
	}

	return stripped
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

	// Queue for steps with no incoming edges, but use a slice to collect them first
	var noIncomingEdges []string
	for id, degree := range inDegree {
		if degree == 0 {
			noIncomingEdges = append(noIncomingEdges, id)
		}
	}

	// Sort the initial steps with no incoming edges
	sort.Strings(noIncomingEdges)

	// Now use these as the initial queue
	queue := noIncomingEdges

	var ordered []CreateStepOpts
	// Process the steps in topological order.
	for len(queue) > 0 {
		// Get and remove the first element
		id := queue[0]
		queue = queue[1:]

		ordered = append(ordered, stepMap[id])

		// Collect children that become ready
		var readyChildren []string
		for _, child := range graph[id] {
			inDegree[child]--
			if inDegree[child] == 0 {
				readyChildren = append(readyChildren, child)
			}
		}

		// Sort the children that are now ready
		sort.Strings(readyChildren)

		// Append sorted children to the queue
		queue = append(queue, readyChildren...)
	}

	// If not all steps are processed, there is a cycle.
	if len(ordered) != len(steps) {
		return nil, fmt.Errorf("cycle detected in workflow steps")
	}

	return ordered, nil
}
