package v1

import (
	"github.com/hatchet-dev/hatchet/pkg/client"
	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	v0Config "github.com/hatchet-dev/hatchet/pkg/config/client"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
)

type HatchetClient interface {
	V0() client.Client
}

type v1ClientImpl struct {
	v0 *client.Client
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

func NewHatchetClient(config ...Config) (HatchetClient, error) {

	cf := &v0Config.ClientConfigFile{}

	if len(config) > 0 {
		opts := config[0]
		cf := &v0Config.ClientConfigFile{}

		// Apply provided config to the internal configuration
		// Zero values won't override server defaults
		cf.TenantId = opts.TenantId
		cf.Token = opts.Token
		cf.HostPort = opts.HostPort
		cf.Namespace = opts.Namespace
		cf.NoGrpcRetry = opts.NoGrpcRetry
		cf.CloudRegisterID = &opts.CloudRegisterID
		cf.RawRunnableActions = opts.RawRunnableActions
		cf.AutoscalingTarget = opts.AutoscalingTarget

		if opts.TLS != nil {
			cf.TLS = v0Config.ClientTLSConfigFile{
				Base:          *opts.TLS.Base,
				TLSServerName: opts.TLS.TLSServerName,
			}
		}
	}

	client, err := v0Client.NewFromConfigFile(cf)
	if err != nil {
		return nil, err
	}

	return &v1ClientImpl{
		v0: &client,
	}, nil
}

func (c *v1ClientImpl) V0() client.Client {
	return *c.v0
}
