package prisma

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type snsRepository struct {
	client *db.PrismaClient
	v      validator.Validator
}

func NewSNSRepository(client *db.PrismaClient, v validator.Validator) repository.SNSRepository {
	return &snsRepository{
		client: client,
		v:      v,
	}
}

func (r *snsRepository) GetSNSIntegration(tenantId, topicArn string) (*db.SNSIntegrationModel, error) {
	return r.client.SNSIntegration.FindUnique(
		db.SNSIntegration.TenantIDTopicArn(
			db.SNSIntegration.TenantID.Equals(tenantId),
			db.SNSIntegration.TopicArn.Equals(topicArn),
		),
	).Exec(context.Background())
}

func (r *snsRepository) GetSNSIntegrationById(id string) (*db.SNSIntegrationModel, error) {
	return r.client.SNSIntegration.FindUnique(
		db.SNSIntegration.ID.Equals(id),
	).Exec(context.Background())
}

func (r *snsRepository) CreateSNSIntegration(tenantId string, opts *repository.CreateSNSIntegrationOpts) (*db.SNSIntegrationModel, error) {
	return r.client.SNSIntegration.CreateOne(
		db.SNSIntegration.Tenant.Link(
			db.Tenant.ID.Equals(tenantId),
		),
		db.SNSIntegration.TopicArn.Set(opts.TopicArn),
	).Exec(context.Background())
}

func (r *snsRepository) ListSNSIntegrations(tenantId string) ([]db.SNSIntegrationModel, error) {
	return r.client.SNSIntegration.FindMany(
		db.SNSIntegration.TenantID.Equals(tenantId),
	).Exec(context.Background())
}

func (r *snsRepository) DeleteSNSIntegration(tenantId, id string) error {
	_, err := r.client.SNSIntegration.FindUnique(
		db.SNSIntegration.ID.Equals(id),
	).Delete().Exec(context.Background())

	return err
}
