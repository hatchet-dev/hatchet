package server

import (
	"crypto/tls"

	"github.com/hatchet-dev/hatchet/internal/auth/cookie"
	"github.com/hatchet-dev/hatchet/internal/config/database"
	"github.com/hatchet-dev/hatchet/internal/config/shared"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
	"github.com/hatchet-dev/hatchet/internal/validator"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

type ServerConfigFile struct {
	Auth ConfigFileAuth `mapstructure:"auth" json:"auth,omitempty"`

	Runtime ConfigFileRuntime `mapstructure:"runtime" json:"runtime,omitempty"`

	TaskQueue TaskQueueConfigFile `mapstructure:"taskQueue" json:"taskQueue,omitempty"`

	Services []string `mapstructure:"services" json:"services,omitempty" default:"[\"ticker\", \"grpc\", \"eventscontroller\", \"jobscontroller\"]"`

	TLS shared.TLSConfigFile `mapstructure:"tls" json:"tls,omitempty"`
}

// General server runtime options
type ConfigFileRuntime struct {
	// Port is the port that the core server listens on
	Port int `mapstructure:"port" json:"port,omitempty" default:"8080"`

	// ServerURL is the full server URL of the instance, including protocol.
	ServerURL string `mapstructure:"url" json:"url,omitempty" default:"http://localhost:8080"`
}

type ConfigFileAuth struct {
	// RestrictedEmailDomains sets the restricted email domains for the instance.
	RestrictedEmailDomains []string `mapstructure:"restrictedEmailDomains" json:"restrictedEmailDomains,omitempty"`

	// BasedAuthEnabled controls whether email and password-based login is enabled for this
	// Hatchet instance
	BasicAuthEnabled bool `mapstructure:"basicAuthEnabled" json:"basicAuthEnabled,omitempty" default:"true"`

	// SetEmailVerified controls whether the user's email is automatically set to verified
	SetEmailVerified bool `mapstructure:"setEmailVerified" json:"setEmailVerified,omitempty" default:"false"`

	// Configuration options for the cookie
	Cookie ConfigFileAuthCookie `mapstructure:"cookie" json:"cookie,omitempty"`
}

type ConfigFileAuthCookie struct {
	Name     string `mapstructure:"name" json:"name,omitempty" default:"hatchet"`
	Domain   string `mapstructure:"domain" json:"domain,omitempty"`
	Secrets  string `mapstructure:"secrets" json:"secrets,omitempty"`
	Insecure bool   `mapstructure:"insecure" json:"insecure,omitempty" default:"false"`
}

type TaskQueueConfigFile struct {
	Kind string `mapstructure:"kind" json:"kind,omitempty" validate:"required"`

	RabbitMQ RabbitMQConfigFile `mapstructure:"rabbitmq" json:"rabbitmq,omitempty" validate:"required"`
}

type RabbitMQConfigFile struct {
	URL string `mapstructure:"url" json:"url,omitempty" validate:"required" default:"amqp://user:password@localhost:5672/"`
}

type ServerRuntimeConfig struct {
	ServerURL string
	Port      int
}

type ServerConfig struct {
	*database.Config

	Auth ConfigFileAuth

	Runtime ServerRuntimeConfig

	Services []string

	Namespaces []string

	TaskQueue taskqueue.TaskQueue

	Logger *zerolog.Logger

	TLSConfig *tls.Config

	SessionStore *cookie.UserSessionStore

	Validator validator.Validator

	Ingestor ingestor.Ingestor
}

func (c *ServerConfig) HasService(name string) bool {
	for _, s := range c.Services {
		if s == name {
			return true
		}
	}

	return false
}

func BindAllEnv(v *viper.Viper) {
	// runtime options
	v.BindEnv("runtime.port", "SERVER_PORT")
	v.BindEnv("runtime.url", "SERVER_URL")
	v.BindEnv("services", "SERVER_SERVICES")

	// auth options
	v.BindEnv("auth.restrictedEmailDomains", "SERVER_AUTH_RESTRICTED_EMAIL_DOMAINS")
	v.BindEnv("auth.basicAuthEnabled", "SERVER_AUTH_BASIC_AUTH_ENABLED")
	v.BindEnv("auth.setEmailVerified", "SERVER_AUTH_SET_EMAIL_VERIFIED")
	v.BindEnv("auth.cookie.name", "SERVER_AUTH_COOKIE_NAME")
	v.BindEnv("auth.cookie.domain", "SERVER_AUTH_COOKIE_DOMAIN")
	v.BindEnv("auth.cookie.secrets", "SERVER_AUTH_COOKIE_SECRETS")
	v.BindEnv("auth.cookie.insecure", "SERVER_AUTH_COOKIE_INSECURE")

	// task queue options
	v.BindEnv("taskQueue.kind", "SERVER_TASKQUEUE_KIND")
	v.BindEnv("taskQueue.rabbitmq.url", "SERVER_TASKQUEUE_RABBITMQ_URL")

	// tls options
	v.BindEnv("tls.tlsCert", "SERVER_TLS_CERT")
	v.BindEnv("tls.tlsCertFile", "SERVER_TLS_CERT_FILE")
	v.BindEnv("tls.tlsKey", "SERVER_TLS_KEY")
	v.BindEnv("tls.tlsKeyFile", "SERVER_TLS_KEY_FILE")
	v.BindEnv("tls.tlsRootCA", "SERVER_TLS_ROOT_CA")
	v.BindEnv("tls.tlsRootCAFile", "SERVER_TLS_ROOT_CA_FILE")
	v.BindEnv("tls.tlsServerName", "SERVER_TLS_SERVER_NAME")
}
