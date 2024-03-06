package prisma

import (
	"context"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/validator"
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
