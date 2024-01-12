package prisma

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlctoprisma"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/internal/validator"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/steebchen/prisma-client-go/runtime/transaction"
)

type workflowRepository struct {
	client  *db.PrismaClient
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
}

func NewWorkflowRepository(client *db.PrismaClient, pool *pgxpool.Pool, v validator.Validator) repository.WorkflowRepository {
	queries := dbsqlc.New()

	return &workflowRepository{
		client:  client,
		v:       v,
		queries: queries,
		pool:    pool,
	}
}

func (r *workflowRepository) ListWorkflows(tenantId string, opts *repository.ListWorkflowsOpts) (*repository.ListWorkflowsResult, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	res := &repository.ListWorkflowsResult{}

	pgTenantId := &pgtype.UUID{}

	if err := pgTenantId.Scan(tenantId); err != nil {
		return nil, err
	}

	queryParams := dbsqlc.ListWorkflowsParams{
		TenantId: *pgTenantId,
	}

	latestRunParams := dbsqlc.ListWorkflowsLatestRunsParams{
		TenantId: *pgTenantId,
	}

	countParams := dbsqlc.CountWorkflowsParams{
		TenantId: *pgTenantId,
	}

	if opts.EventKey != nil {
		pgEventKey := &pgtype.Text{}

		if err := pgEventKey.Scan(*opts.EventKey); err != nil {
			return nil, err
		}

		queryParams.EventKey = *pgEventKey
		countParams.EventKey = *pgEventKey
		latestRunParams.EventKey = *pgEventKey
	}

	if opts.Offset != nil {
		queryParams.Offset = *opts.Offset
	}

	if opts.Limit != nil {
		queryParams.Limit = *opts.Limit
	}

	orderByField := "createdAt"
	orderByDirection := "DESC"

	queryParams.Orderby = orderByField + " " + orderByDirection

	tx, err := r.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer tx.Rollback(context.Background())

	workflows, err := r.queries.ListWorkflows(context.Background(), tx, queryParams)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch workflows: %w", err)
	}

	latestRuns, err := r.queries.ListWorkflowsLatestRuns(context.Background(), tx, latestRunParams)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest runs: %w", err)
	}

	latestRunsMap := map[string]*dbsqlc.WorkflowRun{}

	for i := range latestRuns {
		uuid := sqlchelpers.UUIDToStr(latestRuns[i].WorkflowId)
		latestRunsMap[uuid] = &latestRuns[i].WorkflowRun
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

	sqlcWorkflows := make([]*dbsqlc.Workflow, len(workflows))

	for i := range workflows {
		sqlcWorkflows[i] = &workflows[i].Workflow
	}

	prismaWorkflows := sqlctoprisma.NewConverter[dbsqlc.Workflow, db.WorkflowModel]().ToPrismaList(sqlcWorkflows)

	rows := make([]*repository.ListWorkflowsRow, 0)

	for i, workflow := range prismaWorkflows {

		var prismaRun *db.WorkflowRunModel

		if latestRun, exists := latestRunsMap[workflow.ID]; exists {
			prismaRun = sqlctoprisma.NewConverter[dbsqlc.WorkflowRun, db.WorkflowRunModel]().ToPrisma(latestRun)
		}

		rows = append(rows, &repository.ListWorkflowsRow{
			WorkflowModel: prismaWorkflows[i],
			LatestRun:     prismaRun,
		})
	}

	res.Rows = rows

	return res, nil
}

func (r *workflowRepository) CreateNewWorkflow(tenantId string, opts *repository.CreateWorkflowVersionOpts) (*db.WorkflowVersionModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	// preflight check to ensure the workflow doesn't already exist
	workflow, err := r.client.Workflow.FindUnique(
		db.Workflow.TenantIDName(
			db.Workflow.TenantID.Equals(tenantId),
			db.Workflow.Name.Equals(opts.Name),
		),
	).Exec(context.Background())

	if err != nil {
		if !errors.Is(err, db.ErrNotFound) {
			return nil, err
		}
	} else if workflow != nil {
		return nil, fmt.Errorf(
			"workflow with name '%s' already exists",
			opts.Name,
		)
	}

	// begin a transaction
	workflowId := uuid.New().String()

	txs := []transaction.Param{
		r.client.Workflow.CreateOne(
			db.Workflow.Tenant.Link(
				db.Tenant.ID.Equals(tenantId),
			),
			db.Workflow.Name.Set(opts.Name),
			db.Workflow.ID.Set(workflowId),
			db.Workflow.Description.SetIfPresent(opts.Description),
		).Tx(),
	}

	// create any tags
	if len(opts.Tags) > 0 {
		for _, tag := range opts.Tags {
			txs = append(txs, r.client.WorkflowTag.UpsertOne(
				db.WorkflowTag.TenantIDName(
					db.WorkflowTag.TenantID.Equals(tenantId),
					db.WorkflowTag.Name.Equals(tag.Name),
				),
			).Create(
				db.WorkflowTag.Tenant.Link(
					db.Tenant.ID.Equals(tenantId),
				),
				db.WorkflowTag.Name.Set(tag.Name),
				db.WorkflowTag.Workflows.Link(
					db.Workflow.ID.Equals(workflowId),
				),
				db.WorkflowTag.Color.SetIfPresent(tag.Color),
			).Update(
				db.WorkflowTag.Workflows.Link(
					db.Workflow.ID.Equals(workflowId),
				),
				db.WorkflowTag.Color.SetIfPresent(tag.Color),
			).Tx())
		}
	}

	workflowVersionId, versionTxs := r.createWorkflowVersionTxs(tenantId, workflowId, opts)

	txs = append(txs, versionTxs...)

	// execute the transaction
	err = r.client.Prisma.Transaction(txs...).Exec(context.Background())

	if err != nil {
		return nil, err
	}

	return r.client.WorkflowVersion.FindUnique(
		db.WorkflowVersion.ID.Equals(workflowVersionId),
	).With(
		defaultWorkflowVersionPopulator()...,
	).Exec(context.Background())
}

func (r *workflowRepository) CreateWorkflowVersion(tenantId string, opts *repository.CreateWorkflowVersionOpts) (*db.WorkflowVersionModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	// preflight check to ensure the workflow already exists
	workflow, err := r.client.Workflow.FindUnique(
		db.Workflow.TenantIDName(
			db.Workflow.TenantID.Equals(tenantId),
			db.Workflow.Name.Equals(opts.Name),
		),
	).Exec(context.Background())

	if err != nil {
		return nil, err
	}

	if workflow == nil {
		return nil, fmt.Errorf(
			"workflow with name '%s' does not exist",
			opts.Name,
		)
	}

	// begin a transaction
	workflowVersionId, txs := r.createWorkflowVersionTxs(tenantId, workflow.ID, opts)

	// execute the transaction
	err = r.client.Prisma.Transaction(txs...).Exec(context.Background())

	if err != nil {
		return nil, err
	}

	return r.client.WorkflowVersion.FindUnique(
		db.WorkflowVersion.ID.Equals(workflowVersionId),
	).With(
		defaultWorkflowVersionPopulator()...,
	).Exec(context.Background())
}

type createScheduleTxResult interface {
	Result() *db.WorkflowTriggerScheduledRefModel
}

func (r *workflowRepository) CreateSchedules(
	tenantId, workflowVersionId string,
	opts *repository.CreateWorkflowSchedulesOpts,
) ([]*db.WorkflowTriggerScheduledRefModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	txs := []transaction.Param{}
	results := []createScheduleTxResult{}

	for _, scheduledTrigger := range opts.ScheduledTriggers {
		createTx := r.client.WorkflowTriggerScheduledRef.CreateOne(
			db.WorkflowTriggerScheduledRef.Parent.Link(
				db.WorkflowVersion.ID.Equals(workflowVersionId),
			),
			db.WorkflowTriggerScheduledRef.TriggerAt.Set(scheduledTrigger),
			db.WorkflowTriggerScheduledRef.Input.SetIfPresent(opts.Input),
		).Tx()

		txs = append(txs, createTx)
		results = append(results, createTx)
	}

	err := r.client.Prisma.Transaction(txs...).Exec(context.Background())

	if err != nil {
		return nil, err
	}

	res := make([]*db.WorkflowTriggerScheduledRefModel, 0)

	for _, result := range results {
		res = append(res, result.Result())
	}

	return res, nil
}

func (r *workflowRepository) GetWorkflowById(workflowId string) (*db.WorkflowModel, error) {
	return r.client.Workflow.FindUnique(
		db.Workflow.ID.Equals(workflowId),
	).With(
		defaultWorkflowPopulator()...,
	).Exec(context.Background())
}

func (r *workflowRepository) GetWorkflowByName(tenantId, workflowName string) (*db.WorkflowModel, error) {
	return r.client.Workflow.FindUnique(
		db.Workflow.TenantIDName(
			db.Workflow.TenantID.Equals(tenantId),
			db.Workflow.Name.Equals(workflowName),
		),
	).With(
		defaultWorkflowPopulator()...,
	).Exec(context.Background())
}

func (r *workflowRepository) GetScheduledById(tenantId, scheduleTriggerId string) (*db.WorkflowTriggerScheduledRefModel, error) {
	return r.client.WorkflowTriggerScheduledRef.FindUnique(
		db.WorkflowTriggerScheduledRef.ID.Equals(scheduleTriggerId),
	).Exec(context.Background())
}

func (r *workflowRepository) ListWorkflowsForEvent(ctx context.Context, tenantId, eventKey string) ([]db.WorkflowVersionModel, error) {
	ctx, span := telemetry.NewSpan(ctx, "db-list-workflows-for-event")
	defer span.End()

	var rows []struct {
		ID string `json:"id"`
	}

	err := r.client.Prisma.QueryRaw(
		`
		SELECT DISTINCT ON("WorkflowVersion"."workflowId") "WorkflowVersion".id 
		FROM "WorkflowVersion"
		LEFT JOIN "Workflow" AS j1 ON j1.id = "WorkflowVersion"."workflowId"
		LEFT JOIN "WorkflowTriggers" AS j2 ON j2."workflowVersionId" = "WorkflowVersion"."id"
		WHERE
		(j1."tenantId"::text = $1 AND j1.id IS NOT NULL) 
		AND 
		(j2.id IN (
			SELECT t3."parentId"
			FROM "WorkflowTriggerEventRef" AS t3 
			WHERE t3."eventKey" = $2 AND t3."parentId" IS NOT NULL
		) AND j2.id IS NOT NULL)
		ORDER BY "WorkflowVersion"."workflowId", "WorkflowVersion"."order" DESC
		`,
		tenantId, eventKey,
	).Exec(ctx, &rows)

	if err != nil {
		return nil, err
	}

	workflowVersionIds := []string{}

	for _, row := range rows {
		workflowVersionIds = append(workflowVersionIds, row.ID)
	}

	return r.client.WorkflowVersion.FindMany(
		db.WorkflowVersion.Workflow.Where(
			db.Workflow.TenantID.Equals(tenantId),
		),
		db.WorkflowVersion.ID.In(workflowVersionIds),
	).With(
		defaultWorkflowVersionPopulator()...,
	).Exec(ctx)
}

func (r *workflowRepository) createWorkflowVersionTxs(tenantId, workflowId string, opts *repository.CreateWorkflowVersionOpts) (string, []transaction.Param) {
	txs := []transaction.Param{}

	// create the workflow version
	workflowVersionId := uuid.New().String()

	txs = append(txs, r.client.WorkflowVersion.CreateOne(
		db.WorkflowVersion.Version.Set(opts.Version),
		db.WorkflowVersion.Workflow.Link(
			db.Workflow.ID.Equals(workflowId),
		),
		db.WorkflowVersion.ID.Set(workflowVersionId),
	).Tx())

	// create the workflow jobs
	for _, jobOpts := range opts.Jobs {
		jobId := uuid.New().String()

		txs = append(txs, r.client.Job.CreateOne(
			db.Job.Tenant.Link(
				db.Tenant.ID.Equals(tenantId),
			),
			db.Job.Workflow.Link(
				db.WorkflowVersion.ID.Equals(workflowVersionId),
			),
			db.Job.Name.Set(jobOpts.Name),
			db.Job.ID.Set(jobId),
			db.Job.Description.SetIfPresent(jobOpts.Description),
			db.Job.Timeout.SetIfPresent(jobOpts.Timeout),
		).Tx())

		// create the workflow job steps
		var prev *string

		for _, stepOpts := range jobOpts.Steps {
			stepId := uuid.New().String()

			optionals := []db.StepSetParam{
				db.Step.Timeout.SetIfPresent(stepOpts.Timeout),
				db.Step.Inputs.SetIfPresent(stepOpts.Inputs),
				db.Step.ID.Set(stepId),
				db.Step.ReadableID.Set(stepOpts.ReadableId),
			}

			if prev != nil {
				optionals = append(optionals, db.Step.Prev.Link(
					db.Step.ID.Equals(*prev),
				))
			}

			txs = append(txs, r.client.Action.UpsertOne(
				db.Action.TenantIDID(
					db.Action.TenantID.Equals(tenantId),
					db.Action.ID.Equals(stepOpts.Action),
				),
			).Create(
				db.Action.ID.Set(stepOpts.Action),
				db.Action.Tenant.Link(
					db.Tenant.ID.Equals(tenantId),
				),
			).Update().Tx())

			txs = append(txs, r.client.Step.CreateOne(
				db.Step.Tenant.Link(
					db.Tenant.ID.Equals(tenantId),
				),
				db.Step.Job.Link(
					db.Job.ID.Equals(jobId),
				),
				db.Step.Action.Link(
					db.Action.TenantIDID(
						db.Action.TenantID.Equals(tenantId),
						db.Action.ID.Equals(stepOpts.Action),
					),
				),
				optionals...,
			).Tx())

			prev = &stepId
		}
	}

	// create the workflow triggers
	workflowTriggersId := uuid.New().String()

	txs = append(txs, r.client.WorkflowTriggers.CreateOne(
		db.WorkflowTriggers.Workflow.Link(
			db.WorkflowVersion.ID.Equals(workflowVersionId),
		),
		db.WorkflowTriggers.Tenant.Link(
			db.Tenant.ID.Equals(tenantId),
		),
		db.WorkflowTriggers.ID.Set(workflowTriggersId),
	).Tx())

	for _, eventTrigger := range opts.EventTriggers {
		txs = append(txs, r.client.WorkflowTriggerEventRef.CreateOne(
			db.WorkflowTriggerEventRef.Parent.Link(
				db.WorkflowTriggers.ID.Equals(workflowTriggersId),
			),
			db.WorkflowTriggerEventRef.EventKey.Set(eventTrigger),
		).Tx())
	}

	for _, cronTrigger := range opts.CronTriggers {
		txs = append(txs, r.client.WorkflowTriggerCronRef.CreateOne(
			db.WorkflowTriggerCronRef.Parent.Link(
				db.WorkflowTriggers.ID.Equals(workflowTriggersId),
			),
			db.WorkflowTriggerCronRef.Cron.Set(cronTrigger),
		).Tx())
	}

	for _, scheduledTrigger := range opts.ScheduledTriggers {
		txs = append(txs, r.client.WorkflowTriggerScheduledRef.CreateOne(
			db.WorkflowTriggerScheduledRef.Parent.Link(
				db.WorkflowVersion.ID.Equals(workflowVersionId),
			),
			db.WorkflowTriggerScheduledRef.TriggerAt.Set(scheduledTrigger),
		).Tx())
	}

	return workflowVersionId, txs
}

func (r *workflowRepository) DeleteWorkflow(tenantId, workflowId string) (*db.WorkflowModel, error) {
	return r.client.Workflow.FindUnique(
		db.Workflow.ID.Equals(workflowId),
	).With(
		defaultWorkflowPopulator()...,
	).Delete().Exec(context.Background())
}

func (r *workflowRepository) GetWorkflowVersionById(tenantId, workflowVersionId string) (*db.WorkflowVersionModel, error) {
	return r.client.WorkflowVersion.FindUnique(
		db.WorkflowVersion.ID.Equals(workflowVersionId),
	).With(
		defaultWorkflowVersionPopulator()...,
	).Exec(context.Background())
}

func defaultWorkflowPopulator() []db.WorkflowRelationWith {
	return []db.WorkflowRelationWith{
		db.Workflow.Tags.Fetch(),
		db.Workflow.Versions.Fetch().OrderBy(
			db.WorkflowVersion.Order.Order(db.SortOrderDesc),
		).With(
			defaultWorkflowVersionPopulator()...,
		),
	}
}

func defaultWorkflowVersionPopulator() []db.WorkflowVersionRelationWith {
	return []db.WorkflowVersionRelationWith{
		db.WorkflowVersion.Workflow.Fetch(),
		db.WorkflowVersion.Triggers.Fetch().With(
			db.WorkflowTriggers.Events.Fetch(),
			db.WorkflowTriggers.Crons.Fetch().With(
				db.WorkflowTriggerCronRef.Ticker.Fetch(),
			),
		),
		db.WorkflowVersion.Jobs.Fetch().With(
			db.Job.Steps.Fetch().With(
				db.Step.Action.Fetch(),
			),
		),
		db.WorkflowVersion.Scheduled.Fetch().With(
			db.WorkflowTriggerScheduledRef.Ticker.Fetch(),
		),
	}
}
