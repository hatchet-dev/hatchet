package v1

import (
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	v0Config "github.com/hatchet-dev/hatchet/pkg/config/client"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
)

// Deprecated: Config is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type Config struct {
	TenantId           uuid.UUID
	Token              string
	HostPort           string
	ServerURL          string
	Namespace          string
	NoGrpcRetry        bool
	CloudRegisterID    string
	RawRunnableActions []string
	AutoscalingTarget  string
	TLS                *TLSConfig
	Logger             *zerolog.Logger
}

// Deprecated: TLSConfig is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type TLSConfig struct {
	Base          *shared.TLSConfigFile
	TLSServerName string
}

func mapConfigToCF(opts Config) *v0Config.ClientConfigFile {
	cf := &v0Config.ClientConfigFile{}

	// Apply provided config to the internal configuration
	// Zero values won't override server defaults
	cf.TenantId = opts.TenantId.String()
	cf.Token = opts.Token
	cf.HostPort = opts.HostPort
	cf.ServerURL = opts.ServerURL
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

	return cf
}
