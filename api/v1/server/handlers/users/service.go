package users

import (
	"fmt"

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

func (u *UserService) checkUserRestrictions(conf *server.ServerConfig, emailDomain string) error {
	if len(conf.Auth.ConfigFile.RestrictedEmailDomains) == 0 {
		return nil
	}

	for _, domain := range conf.Auth.ConfigFile.RestrictedEmailDomains {
		if domain == emailDomain {
			return nil
		}
	}

	return fmt.Errorf("email is not in the restricted domain group")
}
