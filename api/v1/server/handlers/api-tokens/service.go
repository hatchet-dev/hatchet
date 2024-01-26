package apitokens

import (
	"github.com/hatchet-dev/hatchet/internal/config/server"
)

type APITokenService struct {
	config *server.ServerConfig
}

func NewAPITokenService(config *server.ServerConfig) *APITokenService {
	return &APITokenService{
		config: config,
	}
}
