package server

import (
	"crypto/tls"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"

	"github.com/hatchet-dev/hatchet/internal/auth/cookie"
	"github.com/hatchet-dev/hatchet/internal/auth/token"
	"github.com/hatchet-dev/hatchet/internal/config/database"
	"github.com/hatchet-dev/hatchet/internal/config/shared"
	"github.com/hatchet-dev/hatchet/internal/encryption"
	"github.com/hatchet-dev/hatchet/internal/integrations/vcs"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
	"github.com/hatchet-dev/hatchet/internal/validator"
	"github.com/hatchet-dev/hatchet/pkg/client"
)

type ServerConfigFile struct {
	Auth ConfigFileAuth `mapstructure:"auth" json:"auth,omitempty"`

	Encryption EncryptionConfigFile `mapstructure:"encryption" json:"encryption,omitempty"`

	Runtime ConfigFileRuntime `mapstructure:"runtime" json:"runtime,omitempty"`

	TaskQueue TaskQueueConfigFile `mapstructure:"taskQueue" json:"taskQueue,omitempty"`

	Services []string `mapstructure:"services" json:"services,omitempty" default:"[\"ticker\", \"grpc\", \"eventscontroller\", \"jobscontroller\", \"workflowscontroller\", \"heartbeater\"]"`

	TLS shared.TLSConfigFile `mapstructure:"tls" json:"tls,omitempty"`

	Logger shared.LoggerConfigFile `mapstructure:"logger" json:"logger,omitempty"`

	OpenTelemetry shared.OpenTelemetryConfigFile `mapstructure:"otel" json:"otel,omitempty"`

	VCS ConfigFileVCS `mapstructure:"vcs" json:"vcs,omitempty"`
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

	// GRPCBroadcastAddress is the address that the grpc server broadcasts to, which is what clients should use when connecting.
	GRPCBroadcastAddress string `mapstructure:"grpcBroadcastAddress" json:"grpcBroadcastAddress,omitempty" default:"127.0.0.1:7070"`

	// GRPCInsecure controls whether the grpc server is insecure or uses certs
	GRPCInsecure bool `mapstructure:"grpcInsecure" json:"grpcInsecure,omitempty" default:"false"`

	// Whether the internal worker is enabled for this instance
	WorkerEnabled bool `mapstructure:"workerEnabled" json:"workerEnabled,omitempty" default:"false"`
}

// Encryption options
type EncryptionConfigFile struct {
	// MasterKeyset is the raw master keyset for the instance. This should be a base64-encoded JSON string. You must set
	// either MasterKeyset, MasterKeysetFile or cloudKms.enabled with CloudKMS credentials
	MasterKeyset string `mapstructure:"masterKeyset" json:"masterKeyset,omitempty"`

	// MasterKeysetFile is the path to the master keyset file for the instance.
	MasterKeysetFile string `mapstructure:"masterKeysetFile" json:"masterKeysetFile,omitempty"`

	JWT EncryptionConfigFileJWT `mapstructure:"jwt" json:"jwt,omitempty"`

	// CloudKMS is the configuration for Google Cloud KMS. You must set either MasterKeyset or cloudKms.enabled.
	CloudKMS EncryptionConfigFileCloudKMS `mapstructure:"cloudKms" json:"cloudKms,omitempty"`
}

type EncryptionConfigFileJWT struct {
	// PublicJWTKeyset is a base64-encoded JSON string containing the public keyset which has been encrypted
	// by the master key.
	PublicJWTKeyset string `mapstructure:"publicJWTKeyset" json:"publicJWTKeyset,omitempty"`

	// PublicJWTKeysetFile is the path to the public keyset file for the instance.
	PublicJWTKeysetFile string `mapstructure:"publicJWTKeysetFile" json:"publicJWTKeysetFile,omitempty"`

	// PrivateJWTKeyset is a base64-encoded JSON string containing the private keyset which has been encrypted
	// by the master key.
	PrivateJWTKeyset string `mapstructure:"privateJWTKeyset" json:"privateJWTKeyset,omitempty"`

	// PrivateJWTKeysetFile is the path to the private keyset file for the instance.
	PrivateJWTKeysetFile string `mapstructure:"privateJWTKeysetFile" json:"privateJWTKeysetFile,omitempty"`
}

type EncryptionConfigFileCloudKMS struct {
	// Enabled controls whether the Cloud KMS service is enabled for this Hatchet instance.
	Enabled bool `mapstructure:"enabled" json:"enabled,omitempty" default:"false"`

	// KeyURI is the URI of the key in Google Cloud KMS. This should be in the format of
	// gcp-kms://...
	KeyURI string `mapstructure:"keyURI" json:"keyURI,omitempty"`

	// CredentialsJSON is the JSON credentials for the Google Cloud KMS service account.
	CredentialsJSON string `mapstructure:"credentialsJSON" json:"credentialsJSON,omitempty"`
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

type ConfigFileVCS struct {
	Github ConfigFileGithub `mapstructure:"github" json:"github,omitempty"`
}

type ConfigFileGithub struct {
	Enabled                bool   `mapstructure:"enabled" json:"enabled"`
	GithubAppClientID      string `mapstructure:"appClientID" json:"appClientID,omitempty"`
	GithubAppClientSecret  string `mapstructure:"appClientSecret" json:"appClientSecret,omitempty"`
	GithubAppName          string `mapstructure:"appName" json:"appName,omitempty"`
	GithubAppWebhookSecret string `mapstructure:"appWebhookSecret" json:"appWebhookSecret,omitempty"`
	GithubAppWebhookURL    string `mapstructure:"appWebhookURL" json:"appWebhookURL,omitempty"`
	GithubAppID            string `mapstructure:"appID" json:"appID,omitempty"`
	GithubAppSecretPath    string `mapstructure:"appSecretPath" json:"appSecretPath,omitempty"`
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

	JWTManager token.JWTManager
}

type ServerConfig struct {
	*database.Config

	Auth AuthConfig

	Encryption encryption.EncryptionService

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

	VCSProviders map[vcs.VCSRepositoryKind]vcs.VCSProvider

	InternalClient client.Client
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
	_ = v.BindEnv("runtime.port", "SERVER_PORT")
	_ = v.BindEnv("runtime.url", "SERVER_URL")
	_ = v.BindEnv("runtime.grpcPort", "SERVER_GRPC_PORT")
	_ = v.BindEnv("runtime.grpcBindAddress", "SERVER_GRPC_BIND_ADDRESS")
	_ = v.BindEnv("runtime.grpcBroadcastAddress", "SERVER_GRPC_BROADCAST_ADDRESS")
	_ = v.BindEnv("runtime.grpcInsecure", "SERVER_GRPC_INSECURE")
	_ = v.BindEnv("runtime.workerEnabled", "SERVER_WORKER_ENABLED")
	_ = v.BindEnv("services", "SERVER_SERVICES")

	// encryption options
	_ = v.BindEnv("encryption.masterKeyset", "SERVER_ENCRYPTION_MASTER_KEYSET")
	_ = v.BindEnv("encryption.masterKeysetFile", "SERVER_ENCRYPTION_MASTER_KEYSET_FILE")
	_ = v.BindEnv("encryption.jwt.publicJWTKeyset", "SERVER_ENCRYPTION_JWT_PUBLIC_KEYSET")
	_ = v.BindEnv("encryption.jwt.publicJWTKeysetFile", "SERVER_ENCRYPTION_JWT_PUBLIC_KEYSET_FILE")
	_ = v.BindEnv("encryption.jwt.privateJWTKeyset", "SERVER_ENCRYPTION_JWT_PRIVATE_KEYSET")
	_ = v.BindEnv("encryption.jwt.privateJWTKeysetFile", "SERVER_ENCRYPTION_JWT_PRIVATE_KEYSET_FILE")
	_ = v.BindEnv("encryption.cloudKms.enabled", "SERVER_ENCRYPTION_CLOUDKMS_ENABLED")
	_ = v.BindEnv("encryption.cloudKms.keyURI", "SERVER_ENCRYPTION_CLOUDKMS_KEY_URI")
	_ = v.BindEnv("encryption.cloudKms.credentialsJSON", "SERVER_ENCRYPTION_CLOUDKMS_CREDENTIALS_JSON")

	// auth options
	_ = v.BindEnv("auth.restrictedEmailDomains", "SERVER_AUTH_RESTRICTED_EMAIL_DOMAINS")
	_ = v.BindEnv("auth.basicAuthEnabled", "SERVER_AUTH_BASIC_AUTH_ENABLED")
	_ = v.BindEnv("auth.setEmailVerified", "SERVER_AUTH_SET_EMAIL_VERIFIED")
	_ = v.BindEnv("auth.cookie.name", "SERVER_AUTH_COOKIE_NAME")
	_ = v.BindEnv("auth.cookie.domain", "SERVER_AUTH_COOKIE_DOMAIN")
	_ = v.BindEnv("auth.cookie.secrets", "SERVER_AUTH_COOKIE_SECRETS")
	_ = v.BindEnv("auth.cookie.insecure", "SERVER_AUTH_COOKIE_INSECURE")
	_ = v.BindEnv("auth.google.enabled", "SERVER_AUTH_GOOGLE_ENABLED")
	_ = v.BindEnv("auth.google.clientID", "SERVER_AUTH_GOOGLE_CLIENT_ID")
	_ = v.BindEnv("auth.google.clientSecret", "SERVER_AUTH_GOOGLE_CLIENT_SECRET")
	_ = v.BindEnv("auth.google.scopes", "SERVER_AUTH_GOOGLE_SCOPES")

	// task queue options
	_ = v.BindEnv("taskQueue.kind", "SERVER_TASKQUEUE_KIND")
	_ = v.BindEnv("taskQueue.rabbitmq.url", "SERVER_TASKQUEUE_RABBITMQ_URL")

	// tls options
	_ = v.BindEnv("tls.tlsStrategy", "SERVER_TLS_STRATEGY")
	_ = v.BindEnv("tls.tlsCert", "SERVER_TLS_CERT")
	_ = v.BindEnv("tls.tlsCertFile", "SERVER_TLS_CERT_FILE")
	_ = v.BindEnv("tls.tlsKey", "SERVER_TLS_KEY")
	_ = v.BindEnv("tls.tlsKeyFile", "SERVER_TLS_KEY_FILE")
	_ = v.BindEnv("tls.tlsRootCA", "SERVER_TLS_ROOT_CA")
	_ = v.BindEnv("tls.tlsRootCAFile", "SERVER_TLS_ROOT_CA_FILE")
	_ = v.BindEnv("tls.tlsServerName", "SERVER_TLS_SERVER_NAME")

	// logger options
	_ = v.BindEnv("logger.level", "SERVER_LOGGER_LEVEL")
	_ = v.BindEnv("logger.format", "SERVER_LOGGER_FORMAT")

	// otel options
	_ = v.BindEnv("otel.serviceName", "SERVER_OTEL_SERVICE_NAME")
	_ = v.BindEnv("otel.collectorURL", "SERVER_OTEL_COLLECTOR_URL")

	// vcs options
	_ = v.BindEnv("vcs.kind", "SERVER_VCS_KIND")
	_ = v.BindEnv("vcs.github.enabled", "SERVER_VCS_GITHUB_ENABLED")
	_ = v.BindEnv("vcs.github.appClientID", "SERVER_VCS_GITHUB_APP_CLIENT_ID")
	_ = v.BindEnv("vcs.github.appClientSecret", "SERVER_VCS_GITHUB_APP_CLIENT_SECRET")
	_ = v.BindEnv("vcs.github.appName", "SERVER_VCS_GITHUB_APP_NAME")
	_ = v.BindEnv("vcs.github.appWebhookSecret", "SERVER_VCS_GITHUB_APP_WEBHOOK_SECRET")
	_ = v.BindEnv("vcs.github.appWebhookURL", "SERVER_VCS_GITHUB_APP_WEBHOOK_URL")
	_ = v.BindEnv("vcs.github.appID", "SERVER_VCS_GITHUB_APP_ID")
	_ = v.BindEnv("vcs.github.appSecretPath", "SERVER_VCS_GITHUB_APP_SECRET_PATH")
}
