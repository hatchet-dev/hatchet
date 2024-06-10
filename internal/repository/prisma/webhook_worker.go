package prisma

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type webhookWorkerRepository struct {
	db *db.PrismaClient
	v  validator.Validator
	l  *zerolog.Logger
}

func NewWebhookWorkerRepository(db *db.PrismaClient, v validator.Validator, l *zerolog.Logger) repository.WebhookWorkerRepository {
	return &webhookWorkerRepository{
		db: db,
		v:  v,
		l:  l,
	}
}

func (r *webhookWorkerRepository) ListWebhookWorkers(ctx context.Context, tenantId string) ([]db.WebhookWorkerModel, error) {
	return r.db.WebhookWorker.FindMany(
		db.WebhookWorker.TenantID.Equals(tenantId),
	).Exec(ctx)
}

func (r *webhookWorkerRepository) UpsertWebhookWorker(ctx context.Context, opts *repository.CreateWebhookWorkerOpts) (*db.WebhookWorkerModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	ww, err := r.db.WebhookWorker.UpsertOne(
		db.WebhookWorker.URL.Equals(opts.URL),
	).Create(
		db.WebhookWorker.Secret.Set(opts.Secret),
		db.WebhookWorker.URL.Set(opts.URL),
		db.WebhookWorker.Tenant.Link(
			db.Tenant.ID.Equals(opts.TenantId),
		),
	).Update(
		db.WebhookWorker.Secret.Set(opts.Secret),
		db.WebhookWorker.URL.Set(opts.URL),
	).Exec(ctx)
	if err != nil {
		return nil, err
	}

	var txn []db.PrismaTransaction
	for _, wfIdOrName := range opts.Workflows {
		var params []db.WorkflowWhereParam
		_, err := uuid.Parse(wfIdOrName)
		if err != nil {
			params = append(params, db.Workflow.Name.Equals(wfIdOrName))
		} else {
			params = append(params, db.Workflow.ID.Equals(wfIdOrName))
		}
		workflow, err := r.db.Workflow.FindFirst(
			db.Workflow.Or(params...),
		).Exec(ctx)
		if err != nil {
			return nil, err
		}

		tx := r.db.WebhookWorkerWorkflow.UpsertOne(
			db.WebhookWorkerWorkflow.WebhookWorkerIDWorkflowID(
				db.WebhookWorkerWorkflow.WebhookWorkerID.Equals(ww.ID),
				db.WebhookWorkerWorkflow.WorkflowID.Equals(workflow.ID),
			),
		).Create(
			db.WebhookWorkerWorkflow.WebhookWorker.Link(
				db.WebhookWorker.ID.Equals(ww.ID),
			),
			db.WebhookWorkerWorkflow.Workflow.Link(
				db.Workflow.ID.Equals(workflow.ID),
			),
		).Update().Tx()
		txn = append(txn, tx)
	}

	if err := r.db.Prisma.Transaction(txn...).Exec(ctx); err != nil {
		return nil, err
	}

	return ww, nil
}

func (r *webhookWorkerRepository) GetActionNames(ctx context.Context, webhookWorkerId string) ([]string, error) {
	ww, err := r.db.WebhookWorker.FindUnique(db.WebhookWorker.ID.Equals(webhookWorkerId)).With(
		db.WebhookWorker.WebhookWorkerWorkflows.Fetch().With(
			db.WebhookWorkerWorkflow.Workflow.Fetch().With(
				db.Workflow.Versions.Fetch().OrderBy(
					db.WorkflowVersion.Order.Order(db.SortOrderDesc),
				).Take(1).With(
					db.WorkflowVersion.Jobs.Fetch().With(
						db.Job.Steps.Fetch().With(
							db.Step.Action.Fetch(),
						),
					),
				),
			),
		),
	).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not find webhook worker: %w", err)
	}

	var actionNames []string
	for _, wwf := range ww.WebhookWorkerWorkflows() {
		for _, version := range wwf.Workflow().Versions() {
			for _, job := range version.Jobs() {
				for _, step := range job.Steps() {
					actionNames = append(actionNames, step.Action().ActionID)
				}
			}
		}
	}
	if len(actionNames) == 0 {
		return nil, fmt.Errorf("no actions found for webhook worker %s", webhookWorkerId)
	}

	return actionNames, nil
}
