package users

import (
	"github.com/hatchet-dev/hatchet/internal/config/server"
)

type UserService struct {
	config *server.ServerConfig
}

func NewUserService(config *server.ServerConfig) *UserService {
	return &UserService{
		config: config,
	}
}
