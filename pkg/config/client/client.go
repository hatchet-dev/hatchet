package client

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/spf13/viper"

	"github.com/hatchet-dev/hatchet/pkg/config/shared"
)

type ClientConfigFile struct {
	CloudRegisterID        *string             `mapstructure:"cloudRegisterID" json:"cloudRegisterID,omitempty"`
	TLS                    ClientTLSConfigFile `mapstructure:"tls" json:"tls,omitempty"`
	TenantId               string              `mapstructure:"tenantId" json:"tenantId,omitempty"`
	Token                  string              `mapstructure:"token" json:"token,omitempty"`
	HostPort               string              `mapstructure:"hostPort" json:"hostPort,omitempty"`
	ServerURL              string              `mapstructure:"serverURL" json:"serverURL,omitempty"`
	Namespace              string              `mapstructure:"namespace" json:"namespace,omitempty"`
	AutoscalingTarget      string              `mapstructure:"autoscalingTarget" json:"autoscalingTarget,omitempty"`
	RawRunnableActions     []string            `mapstructure:"runnableActions" json:"runnableActions,omitempty"`
	NoGrpcRetry            bool                `mapstructure:"noGrpcRetry" json:"noGrpcRetry,omitempty"`
	DisableGzipCompression bool                `mapstructure:"disableGzipCompression" json:"disableGzipCompression,omitempty"`
}

type ClientTLSConfigFile struct {
	Base shared.TLSConfigFile `mapstructure:"base" json:"base,omitempty"`

	TLSServerName string `mapstructure:"tlsServerName" json:"tlsServerName,omitempty"`
}

type ClientConfig struct {
	TLSConfig              *tls.Config
	CloudRegisterID        *string
	PresetWorkerLabels     map[string]string
	TenantId               string
	Token                  string
	ServerURL              string
	GRPCBroadcastAddress   string
	Namespace              string
	RunnableActions        []string
	NoGrpcRetry            bool
	DisableGzipCompression bool
}

func BindAllEnv(v *viper.Viper) {
	_ = v.BindEnv("tenantId", "HATCHET_CLIENT_TENANT_ID")
	_ = v.BindEnv("token", "HATCHET_CLIENT_TOKEN")
	_ = v.BindEnv("hostPort", "HATCHET_CLIENT_HOST_PORT")
	_ = v.BindEnv("serverURL", "HATCHET_CLIENT_SERVER_URL")
	_ = v.BindEnv("namespace", "HATCHET_CLIENT_NAMESPACE")

	_ = v.BindEnv("cloudRegisterID", "HATCHET_CLOUD_REGISTER_ID")
	_ = v.BindEnv("runnableActions", "HATCHET_CLOUD_ACTIONS")
	_ = v.BindEnv("noGrpcRetry", "HATCHET_CLIENT_NO_GRPC_RETRY")
	_ = v.BindEnv("autoscalingTarget", "HATCHET_CLIENT_AUTOSCALING_TARGET")
	_ = v.BindEnv("disableGzipCompression", "HATCHET_CLIENT_DISABLE_GZIP_COMPRESSION")

	// tls options
	_ = v.BindEnv("tls.base.tlsStrategy", "HATCHET_CLIENT_TLS_STRATEGY")
	_ = v.BindEnv("tls.base.tlsCertFile", "HATCHET_CLIENT_TLS_CERT_FILE")
	_ = v.BindEnv("tls.base.tlsKeyFile", "HATCHET_CLIENT_TLS_KEY_FILE")
	_ = v.BindEnv("tls.base.tlsRootCAFile", "HATCHET_CLIENT_TLS_ROOT_CA_FILE")
	_ = v.BindEnv("tls.base.tlsCert", "HATCHET_CLIENT_TLS_CERT")
	_ = v.BindEnv("tls.base.tlsKey", "HATCHET_CLIENT_TLS_KEY")
	_ = v.BindEnv("tls.base.tlsRootCA", "HATCHET_CLIENT_TLS_ROOT_CA")
	_ = v.BindEnv("tls.tlsServerName", "HATCHET_CLIENT_TLS_SERVER_NAME")
}

func ApplyNamespace(resourceName string, namespace *string) string {
	if namespace == nil || *namespace == "" {
		return resourceName
	}

	if strings.HasPrefix(resourceName, *namespace) {
		return resourceName
	}

	return fmt.Sprintf("%s%s", *namespace, resourceName)
}
