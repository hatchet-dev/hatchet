package ingestors

import "github.com/hatchet-dev/hatchet/internal/config/server"

type IngestorsService struct {
	config *server.ServerConfig
}

func NewIngestorsService(config *server.ServerConfig) *IngestorsService {
	return &IngestorsService{
		config: config,
	}
}
