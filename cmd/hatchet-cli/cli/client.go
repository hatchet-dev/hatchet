package cli

import (
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/client"
	profileconfig "github.com/hatchet-dev/hatchet/pkg/config/cli"
	clientconfig "github.com/hatchet-dev/hatchet/pkg/config/client"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
)

// NewClientFromProfile creates a new Hatchet client from a profile configuration.
// It properly handles TLS settings, host/port, and authentication based on the profile.
func NewClientFromProfile(profile *profileconfig.Profile, logger *zerolog.Logger) (client.Client, error) {
	// Construct a ClientConfigFile from the profile
	configFile := &clientconfig.ClientConfigFile{
		TenantId:  profile.TenantId,
		Token:     profile.Token,
		HostPort:  profile.GrpcHostPort,
		ServerURL: profile.ApiServerURL,
		TLS: clientconfig.ClientTLSConfigFile{
			Base: shared.TLSConfigFile{
				TLSStrategy: profile.TLSStrategy,
			},
		},
	}

	// Create client with the config file and logger
	return client.NewFromConfigFile(configFile, client.WithLogger(logger))
}
