package repository

import "github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"

type CreateSNSIntegrationOpts struct {
	TopicArn string `validate:"required,min=1,max=255"`
}

type SNSRepository interface {
	GetSNSIntegration(tenantId, topicArn string) (*db.SNSIntegrationModel, error)

	GetSNSIntegrationById(id string) (*db.SNSIntegrationModel, error)

	CreateSNSIntegration(tenantId string, opts *CreateSNSIntegrationOpts) (*db.SNSIntegrationModel, error)

	ListSNSIntegrations(tenantId string) ([]db.SNSIntegrationModel, error)

	DeleteSNSIntegration(tenantId, id string) error
}
