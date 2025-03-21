package v1

import (
	"github.com/hatchet-dev/hatchet/pkg/client"
	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	v0Config "github.com/hatchet-dev/hatchet/pkg/config/client"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
)

type HatchetClient interface {
}

type v1ClientImpl struct {
	client *client.Client
}

// Config represents client configuration
type Config struct {
	TenantId           string
	Token              string
	HostPort           string
	Namespace          string
	NoGrpcRetry        bool
	CloudRegisterID    string
	RawRunnableActions []string
	AutoscalingTarget  string
	TLS                *TLSConfig
}

type TLSConfig struct {
	Base          *shared.TLSConfigFile
	TLSServerName string
}

func NewHatchetClient(config Config) (HatchetClient, error) {
	cf := &v0Config.ClientConfigFile{}

	// Apply provided config to the internal configuration
	// Zero values won't override server defaults
	cf.TenantId = config.TenantId
	cf.Token = config.Token
	cf.HostPort = config.HostPort
	cf.Namespace = config.Namespace
	cf.NoGrpcRetry = config.NoGrpcRetry
	cf.CloudRegisterID = &config.CloudRegisterID
	cf.RawRunnableActions = config.RawRunnableActions
	cf.AutoscalingTarget = config.AutoscalingTarget

	if config.TLS != nil {
		cf.TLS = v0Config.ClientTLSConfigFile{
			Base:          *config.TLS.Base,
			TLSServerName: config.TLS.TLSServerName,
		}
	}

	client, err := v0Client.NewFromConfigFile(cf)
	if err != nil {
		return nil, err
	}

	return &v1ClientImpl{
		client: &client,
	}, nil
}
