package metadata

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type MetadataService struct {
	config *server.ServerConfig
}

func NewMetadataService(config *server.ServerConfig) *MetadataService {
	return &MetadataService{
		config: config,
	}
}
