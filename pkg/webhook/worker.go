package webhook

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type WebhookWorker struct {
	opts   WorkerOpts
	client client.Client
}

type WorkerOpts struct {
	ID        string
	Secret    string
	URL       string
	TenantID  string
	Actions   []string
	Workflows []string
}

func NewWorker(opts WorkerOpts, client client.Client) (*WebhookWorker, error) {
	return &WebhookWorker{
		opts:   opts,
		client: client,
	}, nil
}

func (w *WebhookWorker) Start() (func() error, error) {
	r, err := worker.NewWorker(
		worker.WithClient(w.client),
		worker.WithInternalData(w.opts.Actions, w.opts.Workflows),
		worker.WithName("Webhook_"+w.opts.ID),
	)
	if err != nil {
		return nil, fmt.Errorf("could not create webhook worker: %w", err)
	}

	cleanup, err := r.StartWebhook(worker.WebhookWorkerOpts{
		URL:    w.opts.URL,
		Secret: w.opts.Secret,
	})
	if err != nil {
		return nil, fmt.Errorf("could not start webhook worker: %w", err)
	}

	return cleanup, nil
}
