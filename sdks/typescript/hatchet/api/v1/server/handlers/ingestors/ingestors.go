package ingestors

import "github.com/hatchet-dev/hatchet/pkg/config/server"

type IngestorsService struct {
	config *server.ServerConfig
}

func NewIngestorsService(config *server.ServerConfig) *IngestorsService {
	return &IngestorsService{
		config: config,
	}
}
