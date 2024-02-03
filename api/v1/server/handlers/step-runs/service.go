package stepruns

import (
	"github.com/hatchet-dev/hatchet/internal/config/server"
)

type StepRunService struct {
	config *server.ServerConfig
}

func NewStepRunService(config *server.ServerConfig) *StepRunService {
	return &StepRunService{
		config: config,
	}
}
