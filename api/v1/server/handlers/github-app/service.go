package githubapp

import "github.com/hatchet-dev/hatchet/pkg/config/server"

type GithubAppService struct {
	config *server.ServerConfig
}

func NewGithubAppService(config *server.ServerConfig) *GithubAppService {
	return &GithubAppService{
		config: config,
	}
}
