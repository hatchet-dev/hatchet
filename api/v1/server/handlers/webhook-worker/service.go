package webhookworker

import "github.com/hatchet-dev/hatchet/pkg/config/server"

type WebhookWorkersService struct {
	config *server.ServerConfig
}

func NewWebhookWorkersService(config *server.ServerConfig) *WebhookWorkersService {
	return &WebhookWorkersService{
		config: config,
	}
}
