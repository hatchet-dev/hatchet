package operatorsv1

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type V1OperatorsService struct {
	config *server.ServerConfig
}

func NewV1OperatorsService(config *server.ServerConfig) *V1OperatorsService {
	return &V1OperatorsService{
		config: config,
	}
}
