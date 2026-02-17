package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type CreateSNSIntegrationOpts struct {
	TopicArn string `validate:"required,min=1,max=255"`
}

type SNSRepository interface {
	GetSNSIntegration(ctx context.Context, tenantId uuid.UUID, topicArn string) (*sqlcv1.SNSIntegration, error)

	GetSNSIntegrationById(ctx context.Context, id uuid.UUID) (*sqlcv1.SNSIntegration, error)

	CreateSNSIntegration(ctx context.Context, tenantId uuid.UUID, opts *CreateSNSIntegrationOpts) (*sqlcv1.SNSIntegration, error)

	ListSNSIntegrations(ctx context.Context, tenantId uuid.UUID) ([]*sqlcv1.SNSIntegration, error)

	DeleteSNSIntegration(ctx context.Context, tenantId uuid.UUID, id uuid.UUID) error
}

type snsRepository struct {
	*sharedRepository
}

func newSNSRepository(shared *sharedRepository) SNSRepository {
	return &snsRepository{
		sharedRepository: shared,
	}
}

func (r *snsRepository) GetSNSIntegration(ctx context.Context, tenantId uuid.UUID, topicArn string) (*sqlcv1.SNSIntegration, error) {
	return r.queries.GetSNSIntegration(
		ctx,
		r.pool,
		sqlcv1.GetSNSIntegrationParams{
			Tenantid: tenantId,
			Topicarn: topicArn,
		},
	)
}

func (r *snsRepository) GetSNSIntegrationById(ctx context.Context, id uuid.UUID) (*sqlcv1.SNSIntegration, error) {
	return r.queries.GetSNSIntegrationById(
		ctx,
		r.pool,
		id,
	)
}

func (r *snsRepository) CreateSNSIntegration(ctx context.Context, tenantId uuid.UUID, opts *CreateSNSIntegrationOpts) (*sqlcv1.SNSIntegration, error) {
	return r.queries.CreateSNSIntegration(
		ctx,
		r.pool,
		sqlcv1.CreateSNSIntegrationParams{
			Tenantid: tenantId,
			Topicarn: opts.TopicArn,
		},
	)
}

func (r *snsRepository) ListSNSIntegrations(ctx context.Context, tenantId uuid.UUID) ([]*sqlcv1.SNSIntegration, error) {
	return r.queries.ListSNSIntegrations(
		ctx,
		r.pool,
		tenantId,
	)
}

func (r *snsRepository) DeleteSNSIntegration(ctx context.Context, tenantId uuid.UUID, id uuid.UUID) error {
	return r.queries.DeleteSNSIntegration(
		ctx,
		r.pool,
		sqlcv1.DeleteSNSIntegrationParams{
			Tenantid: tenantId,
			ID:       id,
		},
	)
}
