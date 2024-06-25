package server

import (
	"crypto/tls"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"

	"github.com/hatchet-dev/hatchet/internal/integrations/alerting"
	"github.com/hatchet-dev/hatchet/internal/integrations/email"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/auth/cookie"
	"github.com/hatchet-dev/hatchet/pkg/auth/token"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type ServerConfigFile struct {
	Auth ConfigFileAuth `mapstructure:"auth" json:"auth,omitempty"`

	Alerting AlertingConfigFile `mapstructure:"alerting" json:"alerting,omitempty"`

	Analytics AnalyticsConfigFile `mapstructure:"analytics" json:"analytics,omitempty"`

	Pylon PylonConfig `mapstructure:"pylon" json:"pylon,omitempty"`

	Encryption EncryptionConfigFile `mapstructure:"encryption" json:"encryption,omitempty"`

	Runtime ConfigFileRuntime `mapstructure:"runtime" json:"runtime,omitempty"`

	MessageQueue MessageQueueConfigFile `mapstructure:"msgQueue" json:"msgQueue,omitempty"`

	Services []string `mapstructure:"services" json:"services,omitempty" default:"[\"health\", \"ticker\", \"grpc\", \"eventscontroller\", \"jobscontroller\", \"workflowscontroller\", \"heartbeater\"]"`

	TLS shared.TLSConfigFile `mapstructure:"tls" json:"tls,omitempty"`

	Logger shared.LoggerConfigFile `mapstructure:"logger" json:"logger,omitempty"`

	OpenTelemetry shared.OpenTelemetryConfigFile `mapstructure:"otel" json:"otel,omitempty"`

	TenantAlerting ConfigFileTenantAlerting `mapstructure:"tenantAlerting" json:"tenantAlerting,omitempty"`

	Email ConfigFileEmail `mapstructure:"email" json:"email,omitempty"`
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

	// ShutdownWait is the time between the readiness probe being offline when a shutdown is triggered and the actual start of cleaning up resources.
	ShutdownWait time.Duration `mapstructure:"shutdownWait" json:"shutdownWait,omitempty" default:"20s"`

	// Enforce limits controls whether the server enforces tenant limits
	EnforceLimits bool `mapstructure:"enforceLimits" json:"enforceLimits,omitempty" default:"false"`

	Limits LimitConfigFile `mapstructure:"limits" json:"limits,omitempty"`
}

type LimitConfigFile struct {
	DefaultWorkflowRunLimit      int           `mapstructure:"defaultWorkflowRunLimit" json:"defaultWorkflowRunLimit,omitempty" default:"1000"`
	DefaultWorkflowRunAlarmLimit int           `mapstructure:"defaultWorkflowRunAlarmLimit" json:"defaultWorkflowRunAlarmLimit,omitempty" default:"750"`
	DefaultWorkflowRunWindow     time.Duration `mapstructure:"defaultWorkflowRunWindow" json:"defaultWorkflowRunWindow,omitempty" default:"24h"`

	DefaultWorkerLimit      int `mapstructure:"defaultWorkerLimit" json:"defaultWorkerLimit,omitempty" default:"4"`
	DefaultWorkerAlarmLimit int `mapstructure:"defaultWorkerAlarmLimit" json:"defaultWorkerAlarmLimit,omitempty" default:"2"`

	DefaultEventLimit      int           `mapstructure:"defaultEventLimit" json:"defaultEventLimit,omitempty" default:"1000"`
	DefaultEventAlarmLimit int           `mapstructure:"defaultEventAlarmLimit" json:"defaultEventAlarmLimit,omitempty" default:"750"`
	DefaultEventWindow     time.Duration `mapstructure:"defaultEventWindow" json:"defaultEventWindow,omitempty" default:"24h"`

	DefaultCronLimit      int `mapstructure:"defaultCronLimit" json:"defaultCronLimit,omitempty" default:"5"`
	DefaultCronAlarmLimit int `mapstructure:"defaultCronAlarmLimit" json:"defaultCronAlarmLimit,omitempty" default:"2"`

	DefaultScheduleLimit      int `mapstructure:"defaultScheduleLimit" json:"defaultScheduleLimit,omitempty" default:"1000"`
	DefaultScheduleAlarmLimit int `mapstructure:"defaultScheduleAlarmLimit" json:"defaultScheduleAlarmLimit,omitempty" default:"750"`
}

// Alerting options
type AlertingConfigFile struct {
	Sentry SentryConfigFile `mapstructure:"sentry" json:"sentry,omitempty"`
}

type SentryConfigFile struct {
	// Enabled controls whether the Sentry service is enabled for this Hatchet instance.
	Enabled bool `mapstructure:"enabled" json:"enabled,omitempty"`

	// DSN is the Data Source Name for the Sentry instance
	DSN string `mapstructure:"dsn" json:"dsn,omitempty"`

	// Environment is the environment that the instance is running in
	Environment string `mapstructure:"environment" json:"environment,omitempty" default:"development"`
}

type AnalyticsConfigFile struct {
	Posthog PosthogConfigFile `mapstructure:"posthog" json:"posthog,omitempty"`
}

type PosthogConfigFile struct {
	// Enabled controls whether the Posthog service is enabled for this Hatchet instance.
	Enabled bool `mapstructure:"enabled" json:"enabled,omitempty"`

	// APIKey is the API key for the Posthog instance
	ApiKey string `mapstructure:"apiKey" json:"apiKey,omitempty"`

	// Endpoint is the endpoint for the Posthog instance
	Endpoint string `mapstructure:"endpoint" json:"endpoint,omitempty"`

	// FeApiKey is the frontend API key for the Posthog instance
	FeApiKey string `mapstructure:"feApiKey" json:"feApiKey,omitempty"`

	// FeApiHost is the frontend API host for the Posthog instance
	FeApiHost string `mapstructure:"feApiHost" json:"feApiHost,omitempty"`
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

	Github ConfigFileAuthGithub `mapstructure:"github" json:"github,omitempty"`
}

type ConfigFileTenantAlerting struct {
	Slack ConfigFileSlack `mapstructure:"slack" json:"slack,omitempty"`
}

type ConfigFileSlack struct {
	Enabled bool `mapstructure:"enabled" json:"enabled,omitempty"`

	SlackAppClientID     string   `mapstructure:"clientID" json:"clientID,omitempty"`
	SlackAppClientSecret string   `mapstructure:"clientSecret" json:"clientSecret,omitempty"`
	SlackAppScopes       []string `mapstructure:"scopes" json:"scopes,omitempty" default:"[\"incoming-webhook\"]"`
}

type ConfigFileAuthGoogle struct {
	Enabled bool `mapstructure:"enabled" json:"enabled,omitempty" default:"false"`

	ClientID     string   `mapstructure:"clientID" json:"clientID,omitempty"`
	ClientSecret string   `mapstructure:"clientSecret" json:"clientSecret,omitempty"`
	Scopes       []string `mapstructure:"scopes" json:"scopes,omitempty" default:"[\"openid\", \"profile\", \"email\"]"`
}

type ConfigFileAuthGithub struct {
	Enabled bool `mapstructure:"enabled" json:"enabled,omitempty" default:"false"`

	ClientID     string   `mapstructure:"clientID" json:"clientID,omitempty"`
	ClientSecret string   `mapstructure:"clientSecret" json:"clientSecret,omitempty"`
	Scopes       []string `mapstructure:"scopes" json:"scopes,omitempty" default:"[\"read:user\", \"user:email\"]"`
}

type ConfigFileAuthCookie struct {
	Name     string `mapstructure:"name" json:"name,omitempty" default:"hatchet"`
	Domain   string `mapstructure:"domain" json:"domain,omitempty"`
	Secrets  string `mapstructure:"secrets" json:"secrets,omitempty"`
	Insecure bool   `mapstructure:"insecure" json:"insecure,omitempty" default:"false"`
}

type MessageQueueConfigFile struct {
	Enabled bool `mapstructure:"enabled" json:"enabled,omitempty" default:"true"`

	Kind string `mapstructure:"kind" json:"kind,omitempty" validate:"required"`

	RabbitMQ RabbitMQConfigFile `mapstructure:"rabbitmq" json:"rabbitmq,omitempty" validate:"required"`
}

type RabbitMQConfigFile struct {
	URL string `mapstructure:"url" json:"url,omitempty" validate:"required" default:"amqp://user:password@localhost:5672/"`
}

type ConfigFileEmail struct {
	Postmark PostmarkConfigFile `mapstructure:"postmark" json:"postmark,omitempty"`
}

type PostmarkConfigFile struct {
	Enabled bool `mapstructure:"enabled" json:"enabled,omitempty"`

	ServerKey    string `mapstructure:"serverKey" json:"serverKey,omitempty"`
	FromEmail    string `mapstructure:"fromEmail" json:"fromEmail,omitempty"`
	FromName     string `mapstructure:"fromName" json:"fromName,omitempty" default:"Hatchet Support"`
	SupportEmail string `mapstructure:"supportEmail" json:"supportEmail,omitempty"`
}

type AuthConfig struct {
	ConfigFile ConfigFileAuth

	GoogleOAuthConfig *oauth2.Config

	GithubOAuthConfig *oauth2.Config

	JWTManager token.JWTManager
}

type PylonConfig struct {
	Enabled bool   `mapstructure:"enabled" json:"enabled,omitempty"`
	AppID   string `mapstructure:"appID" json:"appID,omitempty"`
	Secret  string `mapstructure:"secret" json:"secret,omitempty"`
}

type FePosthogConfig struct {
	ApiKey  string
	ApiHost string
}

type ServerConfig struct {
	*database.Config

	Auth AuthConfig

	Alerter errors.Alerter

	Analytics analytics.Analytics

	Pylon *PylonConfig

	FePosthog *FePosthogConfig

	Encryption encryption.EncryptionService

	Runtime ConfigFileRuntime

	Services []string

	Namespaces []string

	MessageQueue msgqueue.MessageQueue

	Logger *zerolog.Logger

	TLSConfig *tls.Config

	SessionStore *cookie.UserSessionStore

	Validator validator.Validator

	Ingestor ingestor.Ingestor

	OpenTelemetry shared.OpenTelemetryConfigFile

	Email email.EmailService

	TenantAlerter *alerting.TenantAlertManager

	AdditionalOAuthConfigs map[string]*oauth2.Config
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
	_ = v.BindEnv("runtime.shutdownWait", "SERVER_SHUTDOWN_WAIT")
	_ = v.BindEnv("services", "SERVER_SERVICES")
	_ = v.BindEnv("runtime.enforceLimits", "SERVER_ENFORCE_LIMITS")

	// limit options
	_ = v.BindEnv("runtime.limits.defaultWorkflowRunLimit", "SERVER_LIMITS_DEFAULT_WORKFLOW_RUN_LIMIT")
	_ = v.BindEnv("runtime.limits.defaultWorkflowRunAlertLimit", "SERVER_LIMITS_DEFAULT_WORKFLOW_RUN_ALERT_LIMIT")
	_ = v.BindEnv("runtime.limits.defaultWorkflowRunWindow", "SERVER_LIMITS_DEFAULT_WORKFLOW_RUN_WINDOW")

	_ = v.BindEnv("runtime.limits.defaultWorkerLimit", "SERVER_LIMITS_DEFAULT_WORKER_LIMIT")
	_ = v.BindEnv("runtime.limits.defaultWorkerAlertLimit", "SERVER_LIMITS_DEFAULT_WORKER_ALERT_LIMIT")

	_ = v.BindEnv("runtime.limits.defaultEventLimit", "SERVER_LIMITS_DEFAULT_EVENT_LIMIT")
	_ = v.BindEnv("runtime.limits.defaultEventAlertLimit", "SERVER_LIMITS_DEFAULT_EVENT_ALERT_LIMIT")
	_ = v.BindEnv("runtime.limits.defaultEventWindow", "SERVER_LIMITS_DEFAULT_EVENT_WINDOW")

	_ = v.BindEnv("runtime.limits.defaultCronLimit", "SERVER_LIMITS_DEFAULT_CRON_LIMIT")
	_ = v.BindEnv("runtime.limits.defaultCronAlertLimit", "SERVER_LIMITS_DEFAULT_CRON_ALERT_LIMIT")

	_ = v.BindEnv("runtime.limits.defaultScheduleLimit", "SERVER_LIMITS_DEFAULT_SCHEDULE_LIMIT")
	_ = v.BindEnv("runtime.limits.defaultScheduleAlertLimit", "SERVER_LIMITS_DEFAULT_SCHEDULE_ALERT_LIMIT")

	// alerting options
	_ = v.BindEnv("alerting.sentry.enabled", "SERVER_ALERTING_SENTRY_ENABLED")
	_ = v.BindEnv("alerting.sentry.dsn", "SERVER_ALERTING_SENTRY_DSN")
	_ = v.BindEnv("alerting.sentry.environment", "SERVER_ALERTING_SENTRY_ENVIRONMENT")

	// analytics options
	_ = v.BindEnv("analytics.posthog.enabled", "SERVER_ANALYTICS_POSTHOG_ENABLED")
	_ = v.BindEnv("analytics.posthog.apiKey", "SERVER_ANALYTICS_POSTHOG_API_KEY")
	_ = v.BindEnv("analytics.posthog.endpoint", "SERVER_ANALYTICS_POSTHOG_ENDPOINT")
	_ = v.BindEnv("analytics.posthog.feApiHost", "SERVER_ANALYTICS_POSTHOG_FE_API_HOST")
	_ = v.BindEnv("analytics.posthog.feApiKey", "SERVER_ANALYTICS_POSTHOG_FE_API_KEY")

	// pylon options
	_ = v.BindEnv("pylon.enabled", "SERVER_PYLON_ENABLED")
	_ = v.BindEnv("pylon.appID", "SERVER_PYLON_APP_ID")
	_ = v.BindEnv("pylon.secret", "SERVER_PYLON_SECRET")

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
	_ = v.BindEnv("auth.github.enabled", "SERVER_AUTH_GITHUB_ENABLED")
	_ = v.BindEnv("auth.github.clientID", "SERVER_AUTH_GITHUB_CLIENT_ID")
	_ = v.BindEnv("auth.github.clientSecret", "SERVER_AUTH_GITHUB_CLIENT_SECRET")
	_ = v.BindEnv("auth.github.scopes", "SERVER_AUTH_GITHUB_SCOPES")

	// task queue options
	// legacy options
	_ = v.BindEnv("msgQueue.kind", "SERVER_TASKQUEUE_KIND")
	_ = v.BindEnv("msgQueue.rabbitmq.url", "SERVER_TASKQUEUE_RABBITMQ_URL")

	_ = v.BindEnv("msgQueue.kind", "SERVER_MSGQUEUE_KIND")
	_ = v.BindEnv("msgQueue.rabbitmq.url", "SERVER_MSGQUEUE_RABBITMQ_URL")

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

	// tenant alerting options
	_ = v.BindEnv("tenantAlerting.slack.enabled", "SERVER_TENANT_ALERTING_SLACK_ENABLED")
	_ = v.BindEnv("tenantAlerting.slack.clientID", "SERVER_TENANT_ALERTING_SLACK_CLIENT_ID")
	_ = v.BindEnv("tenantAlerting.slack.clientSecret", "SERVER_TENANT_ALERTING_SLACK_CLIENT_SECRET")
	_ = v.BindEnv("tenantAlerting.slack.scopes", "SERVER_TENANT_ALERTING_SLACK_SCOPES")

	// email options
	_ = v.BindEnv("email.postmark.enabled", "SERVER_EMAIL_POSTMARK_ENABLED")
	_ = v.BindEnv("email.postmark.serverKey", "SERVER_EMAIL_POSTMARK_SERVER_KEY")
	_ = v.BindEnv("email.postmark.fromEmail", "SERVER_EMAIL_POSTMARK_FROM_EMAIL")
	_ = v.BindEnv("email.postmark.fromName", "SERVER_EMAIL_POSTMARK_FROM_NAME")
	_ = v.BindEnv("email.postmark.supportEmail", "SERVER_EMAIL_POSTMARK_SUPPORT_EMAIL")
}
