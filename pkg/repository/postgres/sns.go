package postgres

import (
	"context"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

type snsRepository struct {
	*sharedRepository
}

func NewSNSRepository(shared *sharedRepository) repository.SNSRepository {
	return &snsRepository{
		sharedRepository: shared,
	}
}

func (r *snsRepository) GetSNSIntegration(ctx context.Context, tenantId, topicArn string) (*dbsqlc.SNSIntegration, error) {
	return r.queries.GetSNSIntegration(
		ctx,
		r.pool,
		dbsqlc.GetSNSIntegrationParams{
			Tenantid: uuid.MustParse(tenantId),
			Topicarn: topicArn,
		},
	)
}

func (r *snsRepository) GetSNSIntegrationById(ctx context.Context, id string) (*dbsqlc.SNSIntegration, error) {
	return r.queries.GetSNSIntegrationById(
		ctx,
		r.pool,
		uuid.MustParse(id),
	)
}

func (r *snsRepository) CreateSNSIntegration(ctx context.Context, tenantId string, opts *repository.CreateSNSIntegrationOpts) (*dbsqlc.SNSIntegration, error) {
	return r.queries.CreateSNSIntegration(
		ctx,
		r.pool,
		dbsqlc.CreateSNSIntegrationParams{
			Tenantid: uuid.MustParse(tenantId),
			Topicarn: opts.TopicArn,
		},
	)
}

func (r *snsRepository) ListSNSIntegrations(ctx context.Context, tenantId string) ([]*dbsqlc.SNSIntegration, error) {
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
		dbsqlc.DeleteSNSIntegrationParams{
			Tenantid: uuid.MustParse(tenantId),
			ID:       uuid.MustParse(id),
		},
	)
}
