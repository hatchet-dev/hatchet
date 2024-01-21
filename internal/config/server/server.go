package server

import (
	"crypto/tls"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"

	"github.com/hatchet-dev/hatchet/internal/auth/cookie"
	"github.com/hatchet-dev/hatchet/internal/config/database"
	"github.com/hatchet-dev/hatchet/internal/config/shared"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type ServerConfigFile struct {
	Auth ConfigFileAuth `mapstructure:"auth" json:"auth,omitempty"`

	Runtime ConfigFileRuntime `mapstructure:"runtime" json:"runtime,omitempty"`

	TaskQueue TaskQueueConfigFile `mapstructure:"taskQueue" json:"taskQueue,omitempty"`

	Services []string `mapstructure:"services" json:"services,omitempty" default:"[\"ticker\", \"grpc\", \"eventscontroller\", \"jobscontroller\", \"heartbeater\"]"`

	TLS shared.TLSConfigFile `mapstructure:"tls" json:"tls,omitempty"`

	Logger shared.LoggerConfigFile `mapstructure:"logger" json:"logger,omitempty"`

	OpenTelemetry shared.OpenTelemetryConfigFile `mapstructure:"otel" json:"otel,omitempty"`
}

// General server runtime options
type ConfigFileRuntime struct {
	// Port is the port that the core server listens on
	Port int `mapstructure:"port" json:"port,omitempty" default:"8080"`

	// ServerURL is the full server URL of the instance, including protocol.
	ServerURL string `mapstructure:"url" json:"url,omitempty" default:"http://localhost:8080"`

	// GRPCPort is the port that the grpc service listens on
	GRPCPort int `mapstructure:"grpcPort" json:"grpcPort,omitempty" default:"7070"`

	// GRPCBindAddress is the address that the grpc server binds to. Should set to 0.0.0.0 if binding in docker container.
	GRPCBindAddress string `mapstructure:"grpcBindAddress" json:"grpcBindAddress,omitempty" default:"127.0.0.1"`

	// GRPCInsecure controls whether the grpc server is insecure or uses certs
	GRPCInsecure bool `mapstructure:"grpcInsecure" json:"grpcInsecure,omitempty" default:"false"`
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

	Google ConfigFileAuthGoogle `mapstructure:"google" json:"google,omitempty"`
}

type ConfigFileAuthGoogle struct {
	Enabled bool `mapstructure:"enabled" json:"enabled,omitempty" default:"false"`

	ClientID     string   `mapstructure:"clientID" json:"clientID,omitempty"`
	ClientSecret string   `mapstructure:"clientSecret" json:"clientSecret,omitempty"`
	Scopes       []string `mapstructure:"scopes" json:"scopes,omitempty" default:"[\"openid\", \"profile\", \"email\"]"`
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

type AuthConfig struct {
	ConfigFile ConfigFileAuth

	GoogleOAuthConfig *oauth2.Config
}

type ServerConfig struct {
	*database.Config

	Auth AuthConfig

	Runtime ConfigFileRuntime

	Services []string

	Namespaces []string

	TaskQueue taskqueue.TaskQueue

	Logger *zerolog.Logger

	TLSConfig *tls.Config

	SessionStore *cookie.UserSessionStore

	Validator validator.Validator

	Ingestor ingestor.Ingestor

	OpenTelemetry shared.OpenTelemetryConfigFile
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
	v.BindEnv("runtime.grpcPort", "SERVER_GRPC_PORT")
	v.BindEnv("runtime.grpcBindAddress", "SERVER_GRPC_BIND_ADDRESS")
	v.BindEnv("runtime.grpcInsecure", "SERVER_GRPC_INSECURE")
	v.BindEnv("services", "SERVER_SERVICES")

	// auth options
	v.BindEnv("auth.restrictedEmailDomains", "SERVER_AUTH_RESTRICTED_EMAIL_DOMAINS")
	v.BindEnv("auth.basicAuthEnabled", "SERVER_AUTH_BASIC_AUTH_ENABLED")
	v.BindEnv("auth.setEmailVerified", "SERVER_AUTH_SET_EMAIL_VERIFIED")
	v.BindEnv("auth.cookie.name", "SERVER_AUTH_COOKIE_NAME")
	v.BindEnv("auth.cookie.domain", "SERVER_AUTH_COOKIE_DOMAIN")
	v.BindEnv("auth.cookie.secrets", "SERVER_AUTH_COOKIE_SECRETS")
	v.BindEnv("auth.cookie.insecure", "SERVER_AUTH_COOKIE_INSECURE")
	v.BindEnv("auth.google.enabled", "SERVER_AUTH_GOOGLE_ENABLED")
	v.BindEnv("auth.google.clientID", "SERVER_AUTH_GOOGLE_CLIENT_ID")
	v.BindEnv("auth.google.clientSecret", "SERVER_AUTH_GOOGLE_CLIENT_SECRET")
	v.BindEnv("auth.google.scopes", "SERVER_AUTH_GOOGLE_SCOPES")

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

	// logger options
	v.BindEnv("logger.level", "SERVER_LOGGER_LEVEL")
	v.BindEnv("logger.format", "SERVER_LOGGER_FORMAT")

	// otel options
	v.BindEnv("otel.serviceName", "SERVER_OTEL_SERVICE_NAME")
	v.BindEnv("otel.collectorURL", "SERVER_OTEL_COLLECTOR_URL")
}
