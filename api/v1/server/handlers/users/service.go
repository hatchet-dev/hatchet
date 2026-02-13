package users

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type UserService struct {
	config *server.ServerConfig
}

func NewUserService(config *server.ServerConfig) *UserService {
	return &UserService{
		config: config,
	}
}

const ErrInvalidCredentials = "invalid credentials"
const ErrRegistrationFailed = "registration failed"
