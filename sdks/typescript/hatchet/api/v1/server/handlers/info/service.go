package info

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type InfoService struct {
	config *server.ServerConfig
}

func NewInfoService(config *server.ServerConfig) *InfoService {
	return &InfoService{
		config: config,
	}
}
