package files

import (
	"github.com/hatchet-dev/hatchet/internal/integrations/blob_storage"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type FileService struct {
	config       *server.ServerConfig
	blob_storage blob_storage.BlobStorageService
}

func NewFileService(config *server.ServerConfig) *FileService {
	return &FileService{
		config: config,
	}
}
