package filtersv1

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type V1FiltersService struct {
	config *server.ServerConfig
}

func NewV1FiltersService(config *server.ServerConfig) *V1FiltersService {

	return &V1FiltersService{
		config: config,
	}
}
