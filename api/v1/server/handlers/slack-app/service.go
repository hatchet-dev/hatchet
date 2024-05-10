package slackapp

import "github.com/hatchet-dev/hatchet/internal/config/server"

type SlackAppService struct {
	config *server.ServerConfig
}

func NewSlackAppService(config *server.ServerConfig) *SlackAppService {
	return &SlackAppService{
		config: config,
	}
}
