package logs

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type LogsService struct {
	config *server.ServerConfig
}

func NewLogsService(config *server.ServerConfig) *LogsService {
	return &LogsService{
		config: config,
	}
}
