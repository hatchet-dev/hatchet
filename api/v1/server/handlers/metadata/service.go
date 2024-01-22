package metadata

import (
	"github.com/hatchet-dev/hatchet/internal/config/server"
)

type MetadataService struct {
	config *server.ServerConfig
}

func NewMetadataService(config *server.ServerConfig) *MetadataService {
	return &MetadataService{
		config: config,
	}
}
