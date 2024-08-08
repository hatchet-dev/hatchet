package events

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type FileService struct {
	config *server.ServerConfig
}

func NewFileService(config *server.ServerConfig) *FileService {
	return &FileService{
		config: config,
	}
}
