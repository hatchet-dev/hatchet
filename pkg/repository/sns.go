package repository

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

type CreateSNSIntegrationOpts struct {
	TopicArn string `validate:"required,min=1,max=255"`
}

type SNSRepository interface {
	GetSNSIntegration(ctx context.Context, tenantId, topicArn string) (*dbsqlc.SNSIntegration, error)

	GetSNSIntegrationById(ctx context.Context, id string) (*dbsqlc.SNSIntegration, error)

	CreateSNSIntegration(ctx context.Context, tenantId string, opts *CreateSNSIntegrationOpts) (*dbsqlc.SNSIntegration, error)

	ListSNSIntegrations(ctx context.Context, tenantId string) ([]*dbsqlc.SNSIntegration, error)

	DeleteSNSIntegration(ctx context.Context, tenantId, id string) error
}
