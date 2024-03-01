package logs

import (
	"github.com/hatchet-dev/hatchet/internal/config/server"
)

type LogService struct {
	config *server.ServerConfig
}

func NewLogService(config *server.ServerConfig) *LogService {
	return &LogService{
		config: config,
	}
}
