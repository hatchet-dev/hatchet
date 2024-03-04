package repository

import "github.com/hatchet-dev/hatchet/internal/repository/prisma/db"

type SNSRepository interface {
	GetSNSIntegration(tenantId, topicArn string) (*db.SNSIntegrationModel, error)
}
