package users

import (
	"errors"
	"fmt"
	"strings"

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

func (u *UserService) checkUserRestrictionsForEmail(conf *server.ServerConfig, email string) error {
	if len(conf.Auth.ConfigFile.RestrictedEmailDomains) == 0 {
		return nil
	}

	// parse domain from email
	// make sure there's only one @ in the email
	if strings.Count(email, "@") != 1 {
		return errors.New("invalid email")
	}

	domain := strings.Split(email, "@")[1]

	return u.checkUserRestrictions(conf, domain)
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
