package apitokens

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type APITokenService struct {
	config *server.ServerConfig
}

func NewAPITokenService(config *server.ServerConfig) *APITokenService {
	return &APITokenService{
		config: config,
	}
}
