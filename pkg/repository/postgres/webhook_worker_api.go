package postgres

import (
	"github.com/hatchet-dev/hatchet/pkg/repository"
)

type webhookWorkerRepository struct {
	*sharedRepository
}

func NewWebhookWorkerRepository(shared *sharedRepository) repository.WebhookWorkerRepository {
	return &webhookWorkerRepository{
		sharedRepository: shared,
	}
}
