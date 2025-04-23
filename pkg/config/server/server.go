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
	msgqueuev1 "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/auth/cookie"
	"github.com/hatchet-dev/hatchet/pkg/auth/token"
	client "github.com/hatchet-dev/hatchet/pkg/client/v1"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/repository/buffer"
	v0 "github.com/hatchet-dev/hatchet/pkg/scheduling/v0"
	v1 "github.com/hatchet-dev/hatchet/pkg/scheduling/v1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type ServerConfigFile struct {
	Auth ConfigFileAuth `mapstructure:"auth" json:"auth,omitempty"`

	Alerting AlertingConfigFile `mapstructure:"alerting" json:"alerting,omitempty"`

	Analytics AnalyticsConfigFile `mapstructure:"analytics" json:"analytics,omitempty"`

	Pylon PylonConfig `mapstructure:"pylon" json:"pylon,omitempty"`

	Decipher DecipherConfig `mapstructure:"decipher" json:"decipher,omitempty"`

	Encryption EncryptionConfigFile `mapstructure:"encryption" json:"encryption,omitempty"`

	Runtime ConfigFileRuntime `mapstructure:"runtime" json:"runtime,omitempty"`

	MessageQueue MessageQueueConfigFile `mapstructure:"msgQueue" json:"msgQueue,omitempty"`

	Services []string `mapstructure:"services" json:"services,omitempty" default:"[\"all\"]"`

	// Used to bind the environment variable, since the array is not well supported
	ServicesString string `mapstructure:"servicesString" json:"servicesString,omitempty"`

	EnableDataRetention bool `mapstructure:"enableDataRetention" json:"enableDataRetention,omitempty" default:"true"`

	EnableWorkerRetention bool `mapstructure:"enableWorkerRetention" json:"enableWorkerRetention,omitempty" default:"false"`

	TLS shared.TLSConfigFile `mapstructure:"tls" json:"tls,omitempty"`

	InternalClient InternalClientTLSConfigFile `mapstructure:"internalClient" json:"internalClient,omitempty"`

	Logger shared.LoggerConfigFile `mapstructure:"logger" json:"logger,omitempty"`

	AdditionalLoggers ConfigFileAdditionalLoggers `mapstructure:"additionalLoggers" json:"additionalLoggers,omitempty"`

	OpenTelemetry shared.OpenTelemetryConfigFile `mapstructure:"otel" json:"otel,omitempty"`

	Prometheus shared.PrometheusConfigFile `mapstructure:"prometheus" json:"prometheus,omitempty"`

	SecurityCheck SecurityCheckConfigFile `mapstructure:"securityCheck" json:"securityCheck,omitempty"`

	TenantAlerting ConfigFileTenantAlerting `mapstructure:"tenantAlerting" json:"tenantAlerting,omitempty"`

	Email ConfigFileEmail `mapstructure:"email" json:"email,omitempty"`

	Monitoring ConfigFileMonitoring `mapstructure:"monitoring" json:"monitoring,omitempty"`
}

type ConfigFileAdditionalLoggers struct {
	// Queue is a custom logger config for the queue service
	Queue shared.LoggerConfigFile `mapstructure:"queue" json:"queue,omitempty"`

	// PgxStats is a custom logger config for the pgx stats service
	PgxStats shared.LoggerConfigFile `mapstructure:"pgxStats" json:"pgxStats,omitempty"`
}

// General server runtime options
type ConfigFileRuntime struct {
	// Port is the port that the core server listens on
	Port int `mapstructure:"port" json:"port,omitempty" default:"8080"`

	// ServerURL is the full server URL of the instance, including protocol.
	ServerURL string `mapstructure:"url" json:"url,omitempty" default:"http://localhost:8080"`

	// Healthcheck controls whether the server has a healthcheck endpoint
	Healthcheck bool `mapstructure:"healthcheck" json:"healthcheck,omitempty" default:"true"`

	// GRPCPort is the port that the grpc service listens on
	GRPCPort int `mapstructure:"grpcPort" json:"grpcPort,omitempty" default:"7070"`

	// GRPCBindAddress is the address that the grpc server binds to. Should set to 0.0.0.0 if binding in docker container.
	GRPCBindAddress string `mapstructure:"grpcBindAddress" json:"grpcBindAddress,omitempty" default:"127.0.0.1"`

	// GRPCBroadcastAddress is the address that the grpc server broadcasts to, which is what clients should use when connecting.
	GRPCBroadcastAddress string `mapstructure:"grpcBroadcastAddress" json:"grpcBroadcastAddress,omitempty" default:"127.0.0.1:7070"`

	// GRPCInsecure controls whether the grpc server is insecure or uses certs
	GRPCInsecure bool `mapstructure:"grpcInsecure" json:"grpcInsecure,omitempty" default:"false"`

	// GRPCMaxMsgSize is the maximum message size that the grpc server will accept
	GRPCMaxMsgSize int `mapstructure:"grpcMaxMsgSize" json:"grpcMaxMsgSize,omitempty" default:"4194304"`

	// GRPCRateLimit is the rate limit for the grpc server. We count limits separately for the Workflow, Dispatcher and Events services. Workflow and Events service are set to this rate, Dispatcher is 10X this rate. The rate limit is per second, per engine, per api token.
	GRPCRateLimit float64 `mapstructure:"grpcRateLimit" json:"grpcRateLimit,omitempty" default:"1000"`

	// ShutdownWait is the time between the readiness probe being offline when a shutdown is triggered and the actual start of cleaning up resources.
	ShutdownWait time.Duration `mapstructure:"shutdownWait" json:"shutdownWait,omitempty" default:"20s"`

	// Enforce limits controls whether the server enforces tenant limits
	EnforceLimits bool `mapstructure:"enforceLimits" json:"enforceLimits,omitempty" default:"false"`

	// Default limit values
	Limits LimitConfigFile `mapstructure:"limits" json:"limits,omitempty"`

	// RequeueLimit is the number of times a message will be requeued in each attempt
	RequeueLimit int `mapstructure:"requeueLimit" json:"requeueLimit,omitempty" default:"100"`

	// QueueLimit is the limit of items to return from a single queue at a time
	SingleQueueLimit int `mapstructure:"singleQueueLimit" json:"singleQueueLimit,omitempty" default:"100"`

	// How many buckets to hash into for parallelizing updates
	UpdateHashFactor int `mapstructure:"updateHashFactor" json:"updateHashFactor,omitempty" default:"100"`

	// How many concurrent updates to allow
	UpdateConcurrentFactor int `mapstructure:"updateConcurrentFactor" json:"updateConcurrentFactor,omitempty" default:"10"`

	// Allow new tenants to be created
	AllowSignup bool `mapstructure:"allowSignup" json:"allowSignup,omitempty" default:"true"`

	// Allow new invites to be created
	AllowInvites bool `mapstructure:"allowInvites" json:"allowInvites,omitempty" default:"true"`

	// Maximum number of pending invites an inviter can have

	MaxPendingInvites int `mapstructure:"maxPendingInvites" json:"maxPendingInvites,omitempty" default:"100"`

	// Allow new tenants to be created
	AllowCreateTenant bool `mapstructure:"allowCreateTenant" json:"allowCreateTenant,omitempty" default:"true"`

	// Allow passwords to be changed
	AllowChangePassword bool `mapstructure:"allowChangePassword" json:"allowChangePassword,omitempty" default:"true"`

	// Buffer create workflow runs
	BufferCreateWorkflowRuns bool `mapstructure:"bufferCreateWorkflowRuns" json:"bufferCreateWorkflowRuns,omitempty" default:"true"`

	// DisableTenantPubs controls whether tenant pubsub is disabled
	DisableTenantPubs bool `mapstructure:"disableTenantPubs" json:"disableTenantPubs,omitempty"`

	// MaxInternalRetryCount is the maximum number of internal retries before a step run is considered failed (default: 10)
	MaxInternalRetryCount int32 `mapstructure:"maxInternalRetryCount" json:"maxInternalRetryCount,omitempty" default:"10"`

	// WaitForFlush is the time to wait for the buffer to flush used for exerting some back pressure on writers
	WaitForFlush time.Duration `mapstructure:"waitForFlush" json:"waitForFlush,omitempty" default:"1"`

	// MaxConcurrent is the maximum number of concurrent flushes
	MaxConcurrent int `mapstructure:"maxConcurrent" json:"maxConcurrent,omitempty" default:"50"`

	// FlushPeriodMilliseconds is the default number of milliseconds before flush
	FlushPeriodMilliseconds int `mapstructure:"flushPeriodMilliseconds" json:"flushPeriodMilliseconds,omitempty" default:"10"`

	// FlushItemsThreshold is the default number of items to hold in memory until flushing to the database
	FlushItemsThreshold int `mapstructure:"flushItemsThreshold" json:"flushItemsThreshold,omitempty" default:"100"`

	// FlushStrategy is the strategy to use for flushing the buffer
	FlushStrategy buffer.BuffStrategy `mapstructure:"flushStrategy" json:"flushStrategy" default:"DYNAMIC"`

	// WorkflowRunBuffer represents the buffer settings for workflow runs
	WorkflowRunBuffer buffer.ConfigFileBuffer `mapstructure:"workflowRunBuffer" json:"workflowRunBuffer,omitempty"`

	// EventBuffer represents the buffer settings for step run events
	EventBuffer buffer.ConfigFileBuffer `mapstructure:"eventBuffer" json:"eventBuffer,omitempty"`

	// ReleaseSemaphoreBuffer represents the buffer settings for releasing semaphore slots
	ReleaseSemaphoreBuffer buffer.ConfigFileBuffer `mapstructure:"releaseSemaphoreBuffer" json:"releaseSemaphoreBuffer,omitempty"`

	// QueueStepRunBuffer represents the buffer settings for inserting step runs into the queue
	QueueStepRunBuffer buffer.ConfigFileBuffer `mapstructure:"queueStepRunBuffer" json:"queueStepRunBuffer,omitempty"`

	Monitoring ConfigFileMonitoring `mapstructure:"monitoring" json:"monitoring,omitempty"`

	// PreventTenantVersionUpgrade controls whether the server prevents tenant version upgrades
	PreventTenantVersionUpgrade bool `mapstructure:"preventTenantVersionUpgrade" json:"preventTenantVersionUpgrade,omitempty" default:"false"`

	// DefaultEngineVersion is the default engine version to use for new tenants
	DefaultEngineVersion string `mapstructure:"defaultEngineVersion" json:"defaultEngineVersion,omitempty" default:"V0"`
}

type InternalClientTLSConfigFile struct {
	// InheritBase controls whether the internal client should inherit the base TLS config from the
	// server config. This will work if there's no gRPC proxy in between the externally-facing grpc server
	// and the API server. If there is a proxy, you should set this to false and configure the base TLS
	// config for the internal client.
	InheritBase bool `mapstructure:"inheritBase" json:"inheritBase,omitempty" default:"true"`

	// InternalGRPCBroadcastAddress is the address that the API endpoints can use to proxy to the gRPC server. If this
	// is not set, it defaults to the GRPC_BROADCAST_ADDRESS
	InternalGRPCBroadcastAddress string `mapstructure:"internalGRPCBroadcastAddress" json:"internalGRPCBroadcastAddress,omitempty"`

	// TLSServerName is the server name to use to verify the TLS connection. If this is not set, it defaults
	// to the host of the GRPC_BROADCAST_ADDRESS
	TLSServerName string `mapstructure:"tlsServerName" json:"tlsServerName,omitempty"`

	Base shared.TLSConfigFile `mapstructure:"base" json:"base,omitempty"`
}

type SecurityCheckConfigFile struct {
	Enabled  bool   `mapstructure:"enabled" json:"enabled,omitempty" default:"true"`
	Endpoint string `mapstructure:"endpoint" json:"endpoint,omitempty" default:"https://security.hatchet.run"`
}

type LimitConfigFile struct {
	DefaultTenantRetentionPeriod string `mapstructure:"defaultTenantRetentionPeriod" json:"defaultTenantRetentionPeriod,omitempty" default:"720h"`

	DefaultWorkflowRunLimit      int           `mapstructure:"defaultWorkflowRunLimit" json:"defaultWorkflowRunLimit,omitempty" default:"1000"`
	DefaultWorkflowRunAlarmLimit int           `mapstructure:"defaultWorkflowRunAlarmLimit" json:"defaultWorkflowRunAlarmLimit,omitempty" default:"750"`
	DefaultWorkflowRunWindow     time.Duration `mapstructure:"defaultWorkflowRunWindow" json:"defaultWorkflowRunWindow,omitempty" default:"24h"`

	DefaultTaskRunLimit      int           `mapstructure:"defaultTaskRunLimit" json:"defaultTaskRunLimit,omitempty" default:"2000"`
	DefaultTaskRunAlarmLimit int           `mapstructure:"defaultTaskRunAlarmLimit" json:"defaultTaskRunAlarmLimit,omitempty" default:"1500"`
	DefaultTaskRunWindow     time.Duration `mapstructure:"defaultTaskRunWindow" json:"defaultTaskRunWindow,omitempty" default:"24h"`

	DefaultWorkerLimit      int `mapstructure:"defaultWorkerLimit" json:"defaultWorkerLimit,omitempty" default:"4"`
	DefaultWorkerAlarmLimit int `mapstructure:"defaultWorkerAlarmLimit" json:"defaultWorkerAlarmLimit,omitempty" default:"2"`

	DefaultWorkerSlotLimit      int `mapstructure:"defaultWorkerSlotLimit" json:"defaultWorkerSlotLimit,omitempty" default:"4000"`
	DefaultWorkerSlotAlarmLimit int `mapstructure:"defaultWorkerSlotAlarmLimit" json:"defaultWorkerSlotAlarmLimit,omitempty" default:"3000"`

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

	// Sample rate is the rate at which to sample events. Default is 1.0 to sample all events.
	SampleRate float64 `mapstructure:"sampleRate" json:"sampleRate,omitempty" default:"1.0"`
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
	// NOTE: do not use this on the server from the config file.
	RestrictedEmailDomains string `mapstructure:"restrictedEmailDomains" json:"restrictedEmailDomains,omitempty"`

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

	Kind string `mapstructure:"kind" json:"kind,omitempty" validate:"required" default:"rabbitmq"`

	Postgres PostgresMQConfigFile `mapstructure:"postgres" json:"postgres,omitempty"`

	RabbitMQ RabbitMQConfigFile `mapstructure:"rabbitmq" json:"rabbitmq,omitempty" validate:"required"`
}

type PostgresMQConfigFile struct {
	Qos int `mapstructure:"qos" json:"qos,omitempty" default:"100"`
}

type RabbitMQConfigFile struct {
	URL string `mapstructure:"url" json:"url,omitempty" validate:"required" default:"amqp://user:password@localhost:5672/"`
	Qos int    `mapstructure:"qos" json:"qos,omitempty" default:"100"`
}

type ConfigFileEmail struct {
	Postmark PostmarkConfigFile `mapstructure:"postmark" json:"postmark,omitempty"`
}

type ConfigFileMonitoring struct {
	// Enabled controls whether the monitoring service is enabled for this Hatchet instance.
	Enabled bool `mapstructure:"enabled" json:"enabled,omitempty" default:"true"`

	// PermittedTenants is a list of tenant IDs that are allowed to use the monitoring service.
	PermittedTenants []string `mapstructure:"permittedTenants" json:"permittedTenants"`

	// ProbeTimeout is the time to wait for the probe to complete
	ProbeTimeout time.Duration `mapstructure:"probeTimeout" json:"probeTimeout,omitempty" default:"30s"`

	// TLSRootCAFile is the path to the root CA file for the monitoring service
	TLSRootCAFile string `mapstructure:"tlsRootCAFile" json:"tlsRootCAFile,omitempty"`
}

type PostmarkConfigFile struct {
	Enabled bool `mapstructure:"enabled" json:"enabled,omitempty"`

	ServerKey    string `mapstructure:"serverKey" json:"serverKey,omitempty"`
	FromEmail    string `mapstructure:"fromEmail" json:"fromEmail,omitempty"`
	FromName     string `mapstructure:"fromName" json:"fromName,omitempty" default:"Hatchet Support"`
	SupportEmail string `mapstructure:"supportEmail" json:"supportEmail,omitempty"`
}

type AuthConfig struct {
	RestrictedEmailDomains []string

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

type DecipherConfig struct {
	Enabled bool   `mapstructure:"enabled" json:"enabled,omitempty" default:"false"`
	Dsn     string `mapstructure:"dsn" json:"dsn,omitempty"`
}

type ServerConfig struct {
	*database.Layer

	Auth AuthConfig

	Alerter errors.Alerter

	Analytics analytics.Analytics

	Pylon *PylonConfig

	FePosthog *FePosthogConfig

	Decipher *DecipherConfig

	Encryption encryption.EncryptionService

	Runtime ConfigFileRuntime

	Services []string

	EnableDataRetention bool

	EnableWorkerRetention bool

	Namespaces []string

	MessageQueue msgqueue.MessageQueue

	MessageQueueV1 msgqueuev1.MessageQueue

	Logger *zerolog.Logger

	AdditionalLoggers ConfigFileAdditionalLoggers

	TLSConfig *tls.Config

	InternalClientFactory *client.GRPCClientFactory

	SessionStore *cookie.UserSessionStore

	Validator validator.Validator

	Ingestor ingestor.Ingestor

	OpenTelemetry shared.OpenTelemetryConfigFile

	Prometheus shared.PrometheusConfigFile

	Email email.EmailService

	TenantAlerter *alerting.TenantAlertManager

	AdditionalOAuthConfigs map[string]*oauth2.Config

	SchedulingPool *v0.SchedulingPool

	SchedulingPoolV1 *v1.SchedulingPool

	Version string
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
	_ = v.BindEnv("runtime.healthcheck", "SERVER_HEALTHCHECK")
	_ = v.BindEnv("runtime.grpcPort", "SERVER_GRPC_PORT")
	_ = v.BindEnv("runtime.grpcBindAddress", "SERVER_GRPC_BIND_ADDRESS")
	_ = v.BindEnv("runtime.grpcBroadcastAddress", "SERVER_GRPC_BROADCAST_ADDRESS")
	_ = v.BindEnv("runtime.grpcInsecure", "SERVER_GRPC_INSECURE")
	_ = v.BindEnv("runtime.grpcMaxMsgSize", "SERVER_GRPC_MAX_MSG_SIZE")
	_ = v.BindEnv("runtime.grpcRateLimit", "SERVER_GRPC_RATE_LIMIT")
	_ = v.BindEnv("runtime.shutdownWait", "SERVER_SHUTDOWN_WAIT")
	_ = v.BindEnv("servicesString", "SERVER_SERVICES")
	_ = v.BindEnv("enableDataRetention", "SERVER_ENABLE_DATA_RETENTION")
	_ = v.BindEnv("enableWorkerRetention", "SERVER_ENABLE_WORKER_RETENTION")
	_ = v.BindEnv("runtime.enforceLimits", "SERVER_ENFORCE_LIMITS")
	_ = v.BindEnv("runtime.allowSignup", "SERVER_ALLOW_SIGNUP")
	_ = v.BindEnv("runtime.allowInvites", "SERVER_ALLOW_INVITES")
	_ = v.BindEnv("runtime.allowCreateTenant", "SERVER_ALLOW_CREATE_TENANT")
	_ = v.BindEnv("runtime.maxPendingInvites", "SERVER_MAX_PENDING_INVITES")
	_ = v.BindEnv("runtime.allowChangePassword", "SERVER_ALLOW_CHANGE_PASSWORD")
	_ = v.BindEnv("runtime.bufferCreateWorkflowRuns", "SERVER_BUFFER_CREATE_WORKFLOW_RUNS")
	_ = v.BindEnv("runtime.disableTenantPubs", "SERVER_DISABLE_TENANT_PUBS")
	_ = v.BindEnv("runtime.maxInternalRetryCount", "SERVER_MAX_INTERNAL_RETRY_COUNT")
	_ = v.BindEnv("runtime.preventTenantVersionUpgrade", "SERVER_PREVENT_TENANT_VERSION_UPGRADE")
	_ = v.BindEnv("runtime.defaultEngineVersion", "SERVER_DEFAULT_ENGINE_VERSION")

	// security check options
	_ = v.BindEnv("securityCheck.enabled", "SERVER_SECURITY_CHECK_ENABLED")
	_ = v.BindEnv("securityCheck.endpoint", "SERVER_SECURITY_CHECK_ENDPOINT")

	// limit options
	_ = v.BindEnv("runtime.limits.defaultTenantRetentionPeriod", "SERVER_LIMITS_DEFAULT_TENANT_RETENTION_PERIOD")

	_ = v.BindEnv("runtime.limits.defaultWorkflowRunLimit", "SERVER_LIMITS_DEFAULT_WORKFLOW_RUN_LIMIT")
	_ = v.BindEnv("runtime.limits.defaultWorkflowRunAlarmLimit", "SERVER_LIMITS_DEFAULT_WORKFLOW_RUN_ALARM_LIMIT")
	_ = v.BindEnv("runtime.limits.defaultWorkflowRunWindow", "SERVER_LIMITS_DEFAULT_WORKFLOW_RUN_WINDOW")

	_ = v.BindEnv("runtime.limits.defaultTaskRunLimit", "SERVER_LIMITS_DEFAULT_TASK_RUN_LIMIT")
	_ = v.BindEnv("runtime.limits.defaultTaskRunAlarmLimit", "SERVER_LIMITS_DEFAULT_TASK_RUN_ALARM_LIMIT")
	_ = v.BindEnv("runtime.limits.defaultTaskRunWindow", "SERVER_LIMITS_DEFAULT_TASK_RUN_WINDOW")

	_ = v.BindEnv("runtime.limits.defaultWorkerLimit", "SERVER_LIMITS_DEFAULT_WORKER_LIMIT")
	_ = v.BindEnv("runtime.limits.defaultWorkerAlarmLimit", "SERVER_LIMITS_DEFAULT_WORKER_ALARM_LIMIT")

	_ = v.BindEnv("runtime.limits.defaultWorkerSlotLimit", "SERVER_LIMITS_DEFAULT_WORKER_SLOT_LIMIT")
	_ = v.BindEnv("runtime.limits.defaultWorkerSlotAlarmLimit", "SERVER_LIMITS_DEFAULT_WORKER_SLOT_ALARM_LIMIT")

	_ = v.BindEnv("runtime.limits.defaultEventLimit", "SERVER_LIMITS_DEFAULT_EVENT_LIMIT")
	_ = v.BindEnv("runtime.limits.defaultEventAlarmLimit", "SERVER_LIMITS_DEFAULT_EVENT_ALARM_LIMIT")
	_ = v.BindEnv("runtime.limits.defaultEventWindow", "SERVER_LIMITS_DEFAULT_EVENT_WINDOW")

	_ = v.BindEnv("runtime.limits.defaultCronLimit", "SERVER_LIMITS_DEFAULT_CRON_LIMIT")
	_ = v.BindEnv("runtime.limits.defaultCronAlarmLimit", "SERVER_LIMITS_DEFAULT_CRON_ALARM_LIMIT")

	_ = v.BindEnv("runtime.limits.defaultScheduleLimit", "SERVER_LIMITS_DEFAULT_SCHEDULE_LIMIT")
	_ = v.BindEnv("runtime.limits.defaultScheduleAlarmLimit", "SERVER_LIMITS_DEFAULT_SCHEDULE_ALARM_LIMIT")

	// buffer options
	_ = v.BindEnv("runtime.workflowRunBuffer.waitForFlush", "SERVER_WORKFLOWRUNBUFFER_WAIT_FOR_FLUSH")
	_ = v.BindEnv("runtime.workflowRunBuffer.maxConcurrent", "SERVER_WORKFLOWRUNBUFFER_MAX_CONCURRENT")
	_ = v.BindEnv("runtime.workflowRunBuffer.flushPeriodMilliseconds", "SERVER_WORKFLOWRUNBUFFER_FLUSH_PERIOD_MILLISECONDS")
	_ = v.BindEnv("runtime.workflowRunBuffer.flushItemsThreshold", "SERVER_WORKFLOWRUNBUFFER_FLUSH_ITEMS_THRESHOLD")
	_ = v.BindEnv("runtime.workflowRunBuffer.flushStrategy", "SERVER_WORKFLOWRUNBUFFER_FLUSH_STRATEGY")

	_ = v.BindEnv("runtime.eventBuffer.waitForFlush", "SERVER_EVENTBUFFER_WAIT_FOR_FLUSH")
	_ = v.BindEnv("runtime.eventBuffer.maxConcurrent", "SERVER_EVENTBUFFER_MAX_CONCURRENT")
	_ = v.BindEnv("runtime.eventBuffer.flushPeriodMilliseconds", "SERVER_EVENTBUFFER_FLUSH_PERIOD_MILLISECONDS")
	_ = v.BindEnv("runtime.eventBuffer.flushItemsThreshold", "SERVER_EVENTBUFFER_FLUSH_ITEMS_THRESHOLD")
	_ = v.BindEnv("runtime.eventBuffer.serialBuffer", "SERVER_EVENTBUFFER_SERIAL_BUFFER")
	_ = v.BindEnv("runtime.eventBuffer.flushStrategy", "SERVER_EVENTBUFFER_FLUSH_STRATEGY")

	_ = v.BindEnv(("runtime.releaseSemaphoreBuffer.waitForFlush"), "SERVER_RELEASESEMAPHOREBUFFER_WAIT_FOR_FLUSH")
	_ = v.BindEnv("runtime.releaseSemaphoreBuffer.maxConcurrent", "SERVER_RELEASESEMAPHOREBUFFER_MAX_CONCURRENT")
	_ = v.BindEnv("runtime.releaseSemaphoreBuffer.flushPeriodMilliseconds", "SERVER_RELEASESEMAPHOREBUFFER_FLUSH_PERIOD_MILLISECONDS")
	_ = v.BindEnv("runtime.releaseSemaphoreBuffer.flushItemsThreshold", "SERVER_RELEASESEMAPHOREBUFFER_FLUSH_ITEMS_THRESHOLD")
	_ = v.BindEnv("runtime.releaseSemaphoreBuffer.flushStrategy", "SERVER_RELEASESEMAPHOREBUFFER_FLUSH_STRATEGY")

	_ = v.BindEnv("runtime.queueStepRunBuffer.waitForFlush", "SERVER_QUEUESTEPRUNBUFFER_WAIT_FOR_FLUSH")
	_ = v.BindEnv("runtime.queueStepRunBuffer.maxConcurrent", "SERVER_QUEUESTEPRUNBUFFER_MAX_CONCURRENT")
	_ = v.BindEnv("runtime.queueStepRunBuffer.flushPeriodMilliseconds", "SERVER_QUEUESTEPRUNBUFFER_FLUSH_PERIOD_MILLISECONDS")
	_ = v.BindEnv("runtime.queueStepRunBuffer.flushItemsThreshold", "SERVER_QUEUESTEPRUNBUFFER_FLUSH_ITEMS_THRESHOLD")
	_ = v.BindEnv("runtime.queueStepRunBuffer.flushStrategy", "SERVER_QUEUESTEPRUNBUFFER_FLUSH_STRATEGY")

	_ = v.BindEnv("runtime.waitForFlush", "SERVER_WAIT_FOR_FLUSH")
	_ = v.BindEnv("runtime.maxConcurrent", "SERVER_MAX_CONCURRENT")
	_ = v.BindEnv("runtime.flushPeriodMilliseconds", "SERVER_FLUSH_PERIOD_MILLISECONDS")
	_ = v.BindEnv("runtime.flushItemsThreshold", "SERVER_FLUSH_ITEMS_THRESHOLD")
	_ = v.BindEnv("runtime.flushStrategy", "SERVER_FLUSH_STRATEGY")

	// alerting options
	_ = v.BindEnv("alerting.sentry.enabled", "SERVER_ALERTING_SENTRY_ENABLED")
	_ = v.BindEnv("alerting.sentry.dsn", "SERVER_ALERTING_SENTRY_DSN")
	_ = v.BindEnv("alerting.sentry.environment", "SERVER_ALERTING_SENTRY_ENVIRONMENT")
	_ = v.BindEnv("alerting.sentry.sampleRate", "SERVER_ALERTING_SENTRY_SAMPLE_RATE")

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

	// decipher options
	_ = v.BindEnv("decipher.enabled", "SERVER_DECIPHER_ENABLED")
	_ = v.BindEnv("decipher.dsn", "SERVER_DECIPHER_DSN")

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

	// throughput options
	_ = v.BindEnv("msgQueue.rabbitmq.qos", "SERVER_MSGQUEUE_RABBITMQ_QOS")
	_ = v.BindEnv("runtime.requeueLimit", "SERVER_REQUEUE_LIMIT")
	_ = v.BindEnv("runtime.singleQueueLimit", "SERVER_SINGLE_QUEUE_LIMIT")
	_ = v.BindEnv("runtime.updateHashFactor", "SERVER_UPDATE_HASH_FACTOR")
	_ = v.BindEnv("runtime.updateConcurrentFactor", "SERVER_UPDATE_CONCURRENT_FACTOR")

	// internal client options
	_ = v.BindEnv("internalClient.base.tlsStrategy", "SERVER_INTERNAL_CLIENT_BASE_STRATEGY")
	_ = v.BindEnv("internalClient.inheritBase", "SERVER_INTERNAL_CLIENT_BASE_INHERIT_BASE")
	_ = v.BindEnv("internalClient.base.tlsCert", "SERVER_INTERNAL_CLIENT_TLS_BASE_CERT")
	_ = v.BindEnv("internalClient.base.tlsCertFile", "SERVER_INTERNAL_CLIENT_TLS_BASE_CERT_FILE")
	_ = v.BindEnv("internalClient.base.tlsKey", "SERVER_INTERNAL_CLIENT_TLS_BASE_KEY")
	_ = v.BindEnv("internalClient.base.tlsKeyFile", "SERVER_INTERNAL_CLIENT_TLS_BASE_KEY_FILE")
	_ = v.BindEnv("internalClient.base.tlsRootCA", "SERVER_INTERNAL_CLIENT_TLS_BASE_ROOT_CA")
	_ = v.BindEnv("internalClient.base.tlsRootCAFile", "SERVER_INTERNAL_CLIENT_TLS_BASE_ROOT_CA_FILE")
	_ = v.BindEnv("internalClient.tlsServerName", "SERVER_INTERNAL_CLIENT_TLS_SERVER_NAME")
	_ = v.BindEnv("internalClient.internalGRPCBroadcastAddress", "SERVER_INTERNAL_CLIENT_INTERNAL_GRPC_BROADCAST_ADDRESS")

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

	// additional logger options
	_ = v.BindEnv("additionalLoggers.queue.level", "SERVER_ADDITIONAL_LOGGERS_QUEUE_LEVEL")
	_ = v.BindEnv("additionalLoggers.queue.format", "SERVER_ADDITIONAL_LOGGERS_QUEUE_FORMAT")
	_ = v.BindEnv("additionalLoggers.pgxStats.level", "SERVER_ADDITIONAL_LOGGERS_PGXSTATS_LEVEL")
	_ = v.BindEnv("additionalLoggers.pgxStats.format", "SERVER_ADDITIONAL_LOGGERS_PGXSTATS_FORMAT")

	// otel options
	_ = v.BindEnv("otel.serviceName", "SERVER_OTEL_SERVICE_NAME")
	_ = v.BindEnv("otel.collectorURL", "SERVER_OTEL_COLLECTOR_URL")
	_ = v.BindEnv("otel.traceIdRatio", "SERVER_OTEL_TRACE_ID_RATIO")
	_ = v.BindEnv("otel.insecure", "SERVER_OTEL_INSECURE")

	// prometheus options
	_ = v.BindEnv("prometheus.enabled", "SERVER_PROMETHEUS_ENABLED")
	_ = v.BindEnv("prometheus.address", "SERVER_PROMETHEUS_ADDRESS")
	_ = v.BindEnv("prometheus.path", "SERVER_PROMETHEUS_PATH")

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

	// monitoring options
	_ = v.BindEnv("runtime.monitoring.enabled", "SERVER_MONITORING_ENABLED")
	_ = v.BindEnv("runtime.monitoring.permittedTenants", "SERVER_MONITORING_PERMITTED_TENANTS")
	_ = v.BindEnv("runtime.monitoring.probeTimeout", "SERVER_MONITORING_PROBE_TIMEOUT")
	// we will fill this in from the server config if it is not set
	_ = v.BindEnv("runtime.monitoring.tlsRootCAFile", "SERVER_MONITORING_TLS_ROOT_CA_FILE")

}
