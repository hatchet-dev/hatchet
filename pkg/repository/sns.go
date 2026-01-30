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
	GetSNSIntegration(ctx context.Context, tenantId, topicArn string) (*sqlcv1.SNSIntegration, error)

	GetSNSIntegrationById(ctx context.Context, id string) (*sqlcv1.SNSIntegration, error)

	CreateSNSIntegration(ctx context.Context, tenantId string, opts *CreateSNSIntegrationOpts) (*sqlcv1.SNSIntegration, error)

	ListSNSIntegrations(ctx context.Context, tenantId string) ([]*sqlcv1.SNSIntegration, error)

	DeleteSNSIntegration(ctx context.Context, tenantId, id string) error
}

type snsRepository struct {
	*sharedRepository
}

func newSNSRepository(shared *sharedRepository) SNSRepository {
	return &snsRepository{
		sharedRepository: shared,
	}
}

func (r *snsRepository) GetSNSIntegration(ctx context.Context, tenantId, topicArn string) (*sqlcv1.SNSIntegration, error) {
	return r.queries.GetSNSIntegration(
		ctx,
		r.pool,
		sqlcv1.GetSNSIntegrationParams{
			Tenantid: uuid.MustParse(tenantId),
			Topicarn: topicArn,
		},
	)
}

func (r *snsRepository) GetSNSIntegrationById(ctx context.Context, id string) (*sqlcv1.SNSIntegration, error) {
	return r.queries.GetSNSIntegrationById(
		ctx,
		r.pool,
		uuid.MustParse(id),
	)
}

func (r *snsRepository) CreateSNSIntegration(ctx context.Context, tenantId string, opts *CreateSNSIntegrationOpts) (*sqlcv1.SNSIntegration, error) {
	return r.queries.CreateSNSIntegration(
		ctx,
		r.pool,
		sqlcv1.CreateSNSIntegrationParams{
			Tenantid: uuid.MustParse(tenantId),
			Topicarn: opts.TopicArn,
		},
	)
}

func (r *snsRepository) ListSNSIntegrations(ctx context.Context, tenantId string) ([]*sqlcv1.SNSIntegration, error) {
	return r.queries.ListSNSIntegrations(
		ctx,
		r.pool,
		uuid.MustParse(tenantId),
	)
}

func (r *snsRepository) DeleteSNSIntegration(ctx context.Context, tenantId, id string) error {
	return r.queries.DeleteSNSIntegration(
		ctx,
		r.pool,
		sqlcv1.DeleteSNSIntegrationParams{
			Tenantid: uuid.MustParse(tenantId),
			ID:       uuid.MustParse(id),
		},
	)
}
