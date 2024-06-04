package webhook

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type WebhookWorker struct {
	opts   WorkerOpts
	client client.Client
	prisma *db.PrismaClient
}

type WorkerOpts struct {
	ID       string
	Secret   string
	Url      string
	TenantID string
}

func NewWorker(opts WorkerOpts, client client.Client, prisma *db.PrismaClient) (*WebhookWorker, error) {
	return &WebhookWorker{
		opts:   opts,
		client: client,
		prisma: prisma,
	}, nil
}

func (w *WebhookWorker) Start() (func() error, error) {
	r, err := worker.NewWorker(worker.WithClient(w.client),
		worker.WithInternalPrisma(w.prisma),
	)
	if err != nil {
		return nil, fmt.Errorf("could not create webhook worker: %w", err)
	}

	cleanup, err := r.StartWebhook(w.opts.ID)
	if err != nil {
		return nil, fmt.Errorf("could not start webhook worker: %w", err)
	}

	return cleanup, nil
}
