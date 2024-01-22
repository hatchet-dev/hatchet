package client

import (
	"crypto/tls"

	"github.com/spf13/viper"

	"github.com/hatchet-dev/hatchet/internal/config/shared"
)

type ClientConfigFile struct {
	TenantId string `mapstructure:"tenantId" json:"tenantId,omitempty"`

	TLS ClientTLSConfigFile `mapstructure:"tls" json:"tls,omitempty"`
}

type ClientTLSConfigFile struct {
	Base shared.TLSConfigFile `mapstructure:"base" json:"tenantId,omitempty"`

	TLSServerName string `mapstructure:"tlsServerName" json:"tlsServerName,omitempty"`
}

type ClientConfig struct {
	TenantId string

	TLSConfig *tls.Config
}

func BindAllEnv(v *viper.Viper) {
	_ = v.BindEnv("tenantId", "HATCHET_CLIENT_TENANT_ID")

	// tls options
	_ = v.BindEnv("tls.base.tlsCertFile", "HATCHET_CLIENT_TLS_CERT_FILE")
	_ = v.BindEnv("tls.base.tlsKeyFile", "HATCHET_CLIENT_TLS_KEY_FILE")
	_ = v.BindEnv("tls.base.tlsRootCAFile", "HATCHET_CLIENT_TLS_ROOT_CA_FILE")
	_ = v.BindEnv("tls.base.tlsCert", "HATCHET_CLIENT_TLS_CERT")
	_ = v.BindEnv("tls.base.tlsKey", "HATCHET_CLIENT_TLS_KEY")
	_ = v.BindEnv("tls.base.tlsRootCA", "HATCHET_CLIENT_TLS_ROOT_CA")
	_ = v.BindEnv("tls.tlsServerName", "HATCHET_CLIENT_TLS_SERVER_NAME")
}
