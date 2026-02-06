package server

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/internal/integrations/alerting"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/auth/cookie"
	"github.com/hatchet-dev/hatchet/pkg/auth/token"
	client "github.com/hatchet-dev/hatchet/pkg/client/v1"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/limits"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/integrations/email"
	v1 "github.com/hatchet-dev/hatchet/pkg/scheduling/v1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type ServerConfigFile struct {
	InternalClient        InternalClientTLSConfigFile    `mapstructure:"internalClient" json:"internalClient,omitempty"`
	Encryption            EncryptionConfigFile           `mapstructure:"encryption" json:"encryption,omitempty"`
	TLS                   shared.TLSConfigFile           `mapstructure:"tls" json:"tls,omitempty"`
	Prometheus            shared.PrometheusConfigFile    `mapstructure:"prometheus" json:"prometheus,omitempty"`
	Analytics             AnalyticsConfigFile            `mapstructure:"analytics" json:"analytics,omitempty"`
	AdditionalLoggers     ConfigFileAdditionalLoggers    `mapstructure:"additionalLoggers" json:"additionalLoggers,omitempty"`
	Monitoring            ConfigFileMonitoring           `mapstructure:"monitoring" json:"monitoring,omitempty"`
	Pylon                 PylonConfig                    `mapstructure:"pylon" json:"pylon,omitempty"`
	Logger                shared.LoggerConfigFile        `mapstructure:"logger" json:"logger,omitempty"`
	SecurityCheck         SecurityCheckConfigFile        `mapstructure:"securityCheck" json:"securityCheck,omitempty"`
	PausedControllers     string                         `mapstructure:"pausedControllers" json:"pausedControllers,omitempty"`
	ServicesString        string                         `mapstructure:"servicesString" json:"servicesString,omitempty"`
	Auth                  ConfigFileAuth                 `mapstructure:"auth" json:"auth,omitempty"`
	Email                 ConfigFileEmail                `mapstructure:"email" json:"email,omitempty"`
	OpenTelemetry         shared.OpenTelemetryConfigFile `mapstructure:"otel" json:"otel,omitempty"`
	TenantAlerting        ConfigFileTenantAlerting       `mapstructure:"tenantAlerting" json:"tenantAlerting,omitempty"`
	Alerting              AlertingConfigFile             `mapstructure:"alerting" json:"alerting,omitempty"`
	Services              []string                       `mapstructure:"services" json:"services,omitempty" default:"[\"all\"]"`
	MessageQueue          MessageQueueConfigFile         `mapstructure:"msgQueue" json:"msgQueue,omitempty"`
	Runtime               ConfigFileRuntime              `mapstructure:"runtime" json:"runtime,omitempty"`
	CronOperations        CronOperationsConfigFile       `mapstructure:"cronOperations" json:"cronOperations,omitempty"`
	PayloadStore          PayloadStoreConfig             `mapstructure:"payloadStore" json:"payloadStore,omitempty"`
	Sampling              ConfigFileSampling             `mapstructure:"sampling" json:"sampling,omitempty"`
	OLAP                  ConfigFileOperations           `mapstructure:"olap" json:"olap,omitempty"`
	OLAPStatusUpdates     OLAPStatusUpdateConfigFile     `mapstructure:"statusUpdates" json:"statusUpdates,omitempty"`
	EnableDataRetention   bool                           `mapstructure:"enableDataRetention" json:"enableDataRetention,omitempty" default:"true"`
	EnableWorkerRetention bool                           `mapstructure:"enableWorkerRetention" json:"enableWorkerRetention,omitempty" default:"false"`
}

type ConfigFileAdditionalLoggers struct {
	// Queue is a custom logger config for the queue service
	Queue shared.LoggerConfigFile `mapstructure:"queue" json:"queue,omitempty"`

	// PgxStats is a custom logger config for the pgx stats service
	PgxStats shared.LoggerConfigFile `mapstructure:"pgxStats" json:"pgxStats,omitempty"`
}

type ConfigFileSampling struct {
	// Enabled controls whether sampling is enabled for this Hatchet instance.
	Enabled bool `mapstructure:"enabled" json:"enabled,omitempty" default:"false"`

	// SamplingRate is the rate at which to sample events. Default is 1.0 to sample all events.
	SamplingRate float64 `mapstructure:"samplingRate" json:"samplingRate,omitempty" default:"1.0"`
}

type ConfigFileOperations struct {
	// Jitter is the jitter duration for operations pools in milliseconds
	Jitter int `mapstructure:"jitter" json:"jitter,omitempty" default:"0"`

	// PollInterval is the polling interval for operations in seconds
	PollInterval int `mapstructure:"pollInterval" json:"pollInterval,omitempty" default:"2"`
}

type TaskOperationLimitsConfigFile struct {
	// TimeoutLimit is the limit for how many tasks to process in a single timeout operation
	TimeoutLimit int `mapstructure:"timeoutLimit" json:"timeoutLimit,omitempty" default:"1000"`

	// ReassignLimit is the limit for how many tasks to process in a single reassignment operation
	ReassignLimit int `mapstructure:"reassignLimit" json:"reassignLimit,omitempty" default:"1000"`

	// RetryQueueLimit is the limit for how many retry queue items to process in a single operation
	RetryQueueLimit int `mapstructure:"retryQueueLimit" json:"retryQueueLimit,omitempty" default:"1000"`

	// DurableSleepLimit is the limit for how many durable sleep items to process in a single operation
	DurableSleepLimit int `mapstructure:"durableSleepLimit" json:"durableSleepLimit,omitempty" default:"1000"`
}

// CronOperationsConfigFile is the configuration for the cron operations
type CronOperationsConfigFile struct {
	// TaskAnalyzeCronInterval is the interval for the task analyze cron operation
	TaskAnalyzeCronInterval time.Duration `mapstructure:"taskAnalyzeCronInterval" json:"taskAnalyzeCronInterval,omitempty" default:"3h"`

	// OLAPAnalyzeCronInterval is the interval for the olap analyze cron operation
	OLAPAnalyzeCronInterval time.Duration `mapstructure:"olapAnalyzeCronInterval" json:"olapAnalyzeCronInterval,omitempty" default:"3h"`

	// DBHealthMetricsInterval is the interval for collecting database health metrics
	DBHealthMetricsInterval time.Duration `mapstructure:"dbHealthMetricsInterval" json:"dbHealthMetricsInterval,omitempty" default:"60s"`

	// OLAPMetricsInterval is the interval for collecting OLAP metrics
	OLAPMetricsInterval time.Duration `mapstructure:"olapMetricsInterval" json:"olapMetricsInterval,omitempty" default:"5m"`

	// WorkerMetricsInterval is the interval for collecting worker metrics
	WorkerMetricsInterval time.Duration `mapstructure:"workerMetricsInterval" json:"workerMetricsInterval,omitempty" default:"60s"`

	// YesterdayRunCountHour is the hour (0-23) at which to collect yesterday's run count metrics
	YesterdayRunCountHour uint `mapstructure:"yesterdayRunCountHour" json:"yesterdayRunCountHour,omitempty" default:"0"`

	// YesterdayRunCountMinute is the minute (0-59) at which to collect yesterday's run count metrics
	YesterdayRunCountMinute uint `mapstructure:"yesterdayRunCountMinute" json:"yesterdayRunCountMinute,omitempty" default:"5"`
}

// OLAPStatusUpdateConfigFile is the configuration for OLAP status updates
type OLAPStatusUpdateConfigFile struct {
	// DagBatchSizeLimit is the limit for how many DAG status updates to process in a single batch update
	DagBatchSizeLimit int `mapstructure:"dagBatchSizeLimit" json:"dagBatchSizeLimit,omitempty" default:"1000"`

	// TaskBatchSizeLimit is the limit for how many task status updates to process in a single batch update
	TaskBatchSizeLimit int `mapstructure:"taskBatchSizeLimit" json:"taskBatchSizeLimit,omitempty" default:"1000"`
}

// General server runtime options
type ConfigFileRuntime struct {
	EnforceLimitsFunc                      func(ctx context.Context, tenantId string) (bool, error) `json:"-"`
	Monitoring                             ConfigFileMonitoring                                     `mapstructure:"monitoring" json:"monitoring,omitempty"`
	ServerURL                              string                                                   `mapstructure:"url" json:"url,omitempty" default:"http://localhost:8080"`
	DefaultEngineVersion                   string                                                   `mapstructure:"defaultEngineVersion" json:"defaultEngineVersion,omitempty" default:"V1"`
	GRPCBindAddress                        string                                                   `mapstructure:"grpcBindAddress" json:"grpcBindAddress,omitempty" default:"127.0.0.1"`
	GRPCBroadcastAddress                   string                                                   `mapstructure:"grpcBroadcastAddress" json:"grpcBroadcastAddress,omitempty" default:"127.0.0.1:7070"`
	Limits                                 limits.LimitConfigFile                                   `mapstructure:"limits" json:"limits,omitempty"`
	TaskOperationLimits                    TaskOperationLimitsConfigFile                            `mapstructure:"taskOperationLimits" json:"taskOperationLimits,omitempty"`
	FlushItemsThreshold                    int                                                      `mapstructure:"flushItemsThreshold" json:"flushItemsThreshold,omitempty" default:"100"`
	GRPCTriggerWriteSlots                  int                                                      `mapstructure:"grpcTriggerWriteSlots" json:"grpcTriggerWriteSlots,omitempty" default:"5"`
	WaitForFlush                           time.Duration                                            `mapstructure:"waitForFlush" json:"waitForFlush,omitempty" default:"1"`
	GRPCRateLimit                          float64                                                  `mapstructure:"grpcRateLimit" json:"grpcRateLimit,omitempty" default:"1000"`
	ShutdownWait                           time.Duration                                            `mapstructure:"shutdownWait" json:"shutdownWait,omitempty" default:"20s"`
	WorkflowRunBufferSize                  int                                                      `mapstructure:"workflowRunBufferSize" json:"workflowRunBufferSize,omitempty" default:"1000"`
	GRPCMaxMsgSize                         int                                                      `mapstructure:"grpcMaxMsgSize" json:"grpcMaxMsgSize,omitempty" default:"4194304"`
	SchedulerConcurrencyPollingMaxInterval time.Duration                                            `mapstructure:"schedulerConcurrencyPollingMaxInterval" json:"schedulerConcurrencyPollingMaxInterval,omitempty" default:"5s"`
	RequeueLimit                           int                                                      `mapstructure:"requeueLimit" json:"requeueLimit,omitempty" default:"100"`
	SingleQueueLimit                       int                                                      `mapstructure:"singleQueueLimit" json:"singleQueueLimit,omitempty" default:"100"`
	GRPCWorkerStreamMaxBacklogSize         int                                                      `mapstructure:"grpcWorkerStreamMaxBacklogSize" json:"grpcWorkerStreamMaxBacklogSize,omitempty" default:"20"`
	OptimisticSchedulingSlots              int                                                      `mapstructure:"optimisticSchedulingSlots" json:"optimisticSchedulingSlots,omitempty" default:"5"`
	SchedulerConcurrencyPollingMinInterval time.Duration                                            `mapstructure:"schedulerConcurrencyPollingMinInterval" json:"schedulerConcurrencyPollingMinInterval,omitempty" default:"500ms"`
	MaxConcurrent                          int                                                      `mapstructure:"maxConcurrent" json:"maxConcurrent,omitempty" default:"50"`
	UpdateHashFactor                       int                                                      `mapstructure:"updateHashFactor" json:"updateHashFactor,omitempty" default:"100"`
	UpdateConcurrentFactor                 int                                                      `mapstructure:"updateConcurrentFactor" json:"updateConcurrentFactor,omitempty" default:"10"`
	SchedulerConcurrencyRateLimit          int                                                      `mapstructure:"schedulerConcurrencyRateLimit" json:"schedulerConcurrencyRateLimit,omitempty" default:"20"`
	HealthcheckPort                        int                                                      `mapstructure:"healthcheckPort" json:"healthcheckPort,omitempty" default:"8733"`
	MaxPendingInvites                      int                                                      `mapstructure:"maxPendingInvites" json:"maxPendingInvites,omitempty" default:"100"`
	GRPCPort                               int                                                      `mapstructure:"grpcPort" json:"grpcPort,omitempty" default:"7070"`
	Port                                   int                                                      `mapstructure:"port" json:"port,omitempty" default:"8080"`
	APIRateLimit                           int                                                      `mapstructure:"apiRateLimit" json:"apiRateLimit,omitempty" default:"10"`
	APIRateLimitWindow                     time.Duration                                            `mapstructure:"apiRateLimitWindow" json:"apiRateLimitWindow,omitempty" default:"300s"`
	FlushPeriodMilliseconds                int                                                      `mapstructure:"flushPeriodMilliseconds" json:"flushPeriodMilliseconds,omitempty" default:"10"`
	MaxInternalRetryCount                  int32                                                    `mapstructure:"maxInternalRetryCount" json:"maxInternalRetryCount,omitempty" default:"10"`
	GRPCStaticStreamWindowSize             int32                                                    `mapstructure:"grpcStaticStreamWindowSize" json:"grpcStaticStreamWindowSize,omitempty" default:"10485760"`
	OptimisticSchedulingEnabled            bool                                                     `mapstructure:"optimisticSchedulingEnabled" json:"optimisticSchedulingEnabled,omitempty" default:"true"`
	DisableTenantPubs                      bool                                                     `mapstructure:"disableTenantPubs" json:"disableTenantPubs,omitempty"`
	AllowChangePassword                    bool                                                     `mapstructure:"allowChangePassword" json:"allowChangePassword,omitempty" default:"true"`
	AllowCreateTenant                      bool                                                     `mapstructure:"allowCreateTenant" json:"allowCreateTenant,omitempty" default:"true"`
	PreventTenantVersionUpgrade            bool                                                     `mapstructure:"preventTenantVersionUpgrade" json:"preventTenantVersionUpgrade,omitempty" default:"false"`
	AllowInvites                           bool                                                     `mapstructure:"allowInvites" json:"allowInvites,omitempty" default:"true"`
	ReplayEnabled                          bool                                                     `mapstructure:"replayEnabled" json:"replayEnabled,omitempty" default:"true"`
	AllowSignup                            bool                                                     `mapstructure:"allowSignup" json:"allowSignup,omitempty" default:"true"`
	GRPCTriggerWritesEnabled               bool                                                     `mapstructure:"grpcTriggerWritesEnabled" json:"grpcTriggerWritesEnabled,omitempty" default:"true"`
	GRPCInsecure                           bool                                                     `mapstructure:"grpcInsecure" json:"grpcInsecure,omitempty" default:"false"`
	LogIngestionEnabled                    bool                                                     `mapstructure:"logIngestionEnabled" json:"logIngestionEnabled,omitempty" default:"true"`
	Healthcheck                            bool                                                     `mapstructure:"healthcheck" json:"healthcheck,omitempty" default:"true"`
	EnableDurableUserEventLog              bool                                                     `mapstructure:"enableDurableUserEventLog" json:"enableDurableUserEventLog,omitempty" default:"false"`
	EnforceLimits                          bool                                                     `mapstructure:"enforceLimits" json:"enforceLimits,omitempty" default:"false"`
}

type InternalClientTLSConfigFile struct {
	Base                         shared.TLSConfigFile `mapstructure:"base" json:"base,omitempty"`
	InternalGRPCBroadcastAddress string               `mapstructure:"internalGRPCBroadcastAddress" json:"internalGRPCBroadcastAddress,omitempty"`
	TLSServerName                string               `mapstructure:"tlsServerName" json:"tlsServerName,omitempty"`
	InheritBase                  bool                 `mapstructure:"inheritBase" json:"inheritBase,omitempty" default:"true"`
}

type SecurityCheckConfigFile struct {
	Endpoint string `mapstructure:"endpoint" json:"endpoint,omitempty" default:"https://security.hatchet.run"`
	Enabled  bool   `mapstructure:"enabled" json:"enabled,omitempty" default:"true"`
}

// Alerting options
type AlertingConfigFile struct {
	Sentry SentryConfigFile `mapstructure:"sentry" json:"sentry,omitempty"`
}

type SentryConfigFile struct {
	DSN         string  `mapstructure:"dsn" json:"dsn,omitempty"`
	Environment string  `mapstructure:"environment" json:"environment,omitempty" default:"development"`
	SampleRate  float64 `mapstructure:"sampleRate" json:"sampleRate,omitempty" default:"1.0"`
	Enabled     bool    `mapstructure:"enabled" json:"enabled,omitempty"`
}

type AnalyticsConfigFile struct {
	Posthog PosthogConfigFile `mapstructure:"posthog" json:"posthog,omitempty"`
}

type PosthogConfigFile struct {
	ApiKey    string `mapstructure:"apiKey" json:"apiKey,omitempty"`
	Endpoint  string `mapstructure:"endpoint" json:"endpoint,omitempty"`
	FeApiKey  string `mapstructure:"feApiKey" json:"feApiKey,omitempty"`
	FeApiHost string `mapstructure:"feApiHost" json:"feApiHost,omitempty"`
	Enabled   bool   `mapstructure:"enabled" json:"enabled,omitempty"`
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
	CloudKMS EncryptionConfigFileCloudKMS `mapstructure:"cloudKms" json:"cloudKMS,omitempty"`
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
	KeyURI          string `mapstructure:"keyURI" json:"keyURI,omitempty"`
	CredentialsJSON string `mapstructure:"credentialsJSON" json:"credentialsJSON,omitempty"`
	Enabled         bool   `mapstructure:"enabled" json:"enabled,omitempty" default:"false"`
}

type ConfigFileAuth struct {
	RestrictedEmailDomains string               `mapstructure:"restrictedEmailDomains" json:"restrictedEmailDomains,omitempty"`
	Google                 ConfigFileAuthGoogle `mapstructure:"google" json:"google,omitempty"`
	Github                 ConfigFileAuthGithub `mapstructure:"github" json:"github,omitempty"`
	Cookie                 ConfigFileAuthCookie `mapstructure:"cookie" json:"cookie,omitempty"`
	BasicAuthEnabled       bool                 `mapstructure:"basicAuthEnabled" json:"basicAuthEnabled,omitempty" default:"true"`
	SetEmailVerified       bool                 `mapstructure:"setEmailVerified" json:"setEmailVerified,omitempty" default:"false"`
}

type ConfigFileTenantAlerting struct {
	Slack ConfigFileSlack `mapstructure:"slack" json:"slack,omitempty"`
}

type ConfigFileSlack struct {
	SlackAppClientID     string   `mapstructure:"clientID" json:"clientID,omitempty"`
	SlackAppClientSecret string   `mapstructure:"clientSecret" json:"clientSecret,omitempty"`
	SlackAppScopes       []string `mapstructure:"scopes" json:"scopes,omitempty" default:"[\"incoming-webhook\"]"`
	Enabled              bool     `mapstructure:"enabled" json:"enabled,omitempty"`
}

type ConfigFileAuthGoogle struct {
	ClientID     string   `mapstructure:"clientID" json:"clientID,omitempty"`
	ClientSecret string   `mapstructure:"clientSecret" json:"clientSecret,omitempty"`
	Scopes       []string `mapstructure:"scopes" json:"scopes,omitempty" default:"[\"openid\", \"profile\", \"email\"]"`
	Enabled      bool     `mapstructure:"enabled" json:"enabled,omitempty" default:"false"`
}

type ConfigFileAuthGithub struct {
	ClientID     string   `mapstructure:"clientID" json:"clientID,omitempty"`
	ClientSecret string   `mapstructure:"clientSecret" json:"clientSecret,omitempty"`
	Scopes       []string `mapstructure:"scopes" json:"scopes,omitempty" default:"[\"read:user\", \"user:email\"]"`
	Enabled      bool     `mapstructure:"enabled" json:"enabled,omitempty" default:"false"`
}

type ConfigFileAuthCookie struct {
	Name     string `mapstructure:"name" json:"name,omitempty" default:"hatchet"`
	Domain   string `mapstructure:"domain" json:"domain,omitempty"`
	Secrets  string `mapstructure:"secrets" json:"secrets,omitempty"`
	Insecure bool   `mapstructure:"insecure" json:"insecure,omitempty" default:"false"`
}

type MessageQueueConfigFile struct {
	Kind     string               `mapstructure:"kind" json:"kind,omitempty" validate:"required,oneof=rabbitmq postgres" default:"rabbitmq"`
	RabbitMQ RabbitMQConfigFile   `mapstructure:"rabbitmq" json:"rabbitmq,omitempty" validate:"required"`
	Postgres PostgresMQConfigFile `mapstructure:"postgres" json:"postgres,omitempty"`
	Enabled  bool                 `mapstructure:"enabled" json:"enabled,omitempty" default:"true"`
}

type PostgresMQConfigFile struct {
	Qos int `mapstructure:"qos" json:"qos,omitempty" default:"100"`
}

type RabbitMQConfigFile struct {
	URL                    string `mapstructure:"url" json:"url,omitempty" validate:"required"`
	Qos                    int    `mapstructure:"qos" json:"qos,omitempty" default:"100"`
	CompressionThreshold   int    `mapstructure:"compressionThreshold" json:"compressionThreshold,omitempty" default:"5120"`
	MaxDeathCount          int    `mapstructure:"maxDeathCount" json:"maxDeathCount,omitempty" default:"5"`
	MaxPubChans            int32  `mapstructure:"maxPubChans" json:"maxPubChans,omitempty" default:"20"`
	MaxSubChans            int32  `mapstructure:"maxSubChans" json:"maxSubChans,omitempty" default:"100"`
	CompressionEnabled     bool   `mapstructure:"compressionEnabled" json:"compressionEnabled,omitempty" default:"false"`
	EnableMessageRejection bool   `mapstructure:"enableMessageRejection" json:"enableMessageRejection,omitempty" default:"false"`
}

type ConfigFileEmail struct {
	Kind string `mapstructure:"kind" json:"kind,omitempty" default:"postmark"`

	Postmark PostmarkConfigFile `mapstructure:"postmark" json:"postmark,omitempty"`

	SMTP SMTPEmailConfig `mapstructure:"smtp" json:"smtp,omitempty"`
}

type ConfigFileMonitoring struct {
	TLSRootCAFile    string        `mapstructure:"tlsRootCAFile" json:"tlsRootCAFile,omitempty"`
	PermittedTenants []string      `mapstructure:"permittedTenants" json:"permittedTenants"`
	ProbeTimeout     time.Duration `mapstructure:"probeTimeout" json:"probeTimeout,omitempty" default:"30s"`
	Enabled          bool          `mapstructure:"enabled" json:"enabled,omitempty" default:"true"`
}

type PostmarkConfigFile struct {
	ServerKey    string `mapstructure:"serverKey" json:"serverKey,omitempty"`
	FromEmail    string `mapstructure:"fromEmail" json:"fromEmail,omitempty"`
	FromName     string `mapstructure:"fromName" json:"fromName,omitempty" default:"Hatchet Support"`
	SupportEmail string `mapstructure:"supportEmail" json:"supportEmail,omitempty"`
	Enabled      bool   `mapstructure:"enabled" json:"enabled,omitempty"`
}

type SMTPEmailConfig struct {
	BasicAuth    SMTPEmailConfigAuthBasic `mapstructure:"basicAuth" json:"basicAuth,omitempty"`
	ServerKey    string                   `mapstructure:"serverKey" json:"serverKey,omitempty"`
	ServerAddr   string                   `mapstructure:"serverAddr" json:"serverAddr,omitempty"`
	FromEmail    string                   `mapstructure:"fromEmail" json:"fromEmail,omitempty"`
	FromName     string                   `mapstructure:"fromName" json:"fromName,omitempty" default:"Hatchet Support"`
	SupportEmail string                   `mapstructure:"supportEmail" json:"supportEmail,omitempty"`
	Enabled      bool                     `mapstructure:"enabled" json:"enabled,omitempty"`
}

type SMTPEmailConfigAuthBasic struct {
	Username string `mapstructure:"username" json:"username,omitempty"`
	Password string `mapstructure:"password" json:"password,omitempty"`
}
type CustomAuthenticator interface {
	// Authenticate is called to authenticate for endpoints that support the customAuth security scheme
	Authenticate(c echo.Context) error
	// Authorize is called to authorize for endpoints that support the customAuth security scheme
	Authorize(c echo.Context, r *middleware.RouteInfo) error
	// CookieAuthorizerHook is called as part of cookie authorization
	CookieAuthorizerHook(c echo.Context, r *middleware.RouteInfo) error
}

type AuthConfig struct {
	JWTManager             token.JWTManager
	CustomAuthenticator    CustomAuthenticator
	GoogleOAuthConfig      *oauth2.Config
	GithubOAuthConfig      *oauth2.Config
	ConfigFile             ConfigFileAuth
	RestrictedEmailDomains []string
}

type PylonConfig struct {
	AppID   string `mapstructure:"appID" json:"appID,omitempty"`
	Secret  string `mapstructure:"secret" json:"secret,omitempty"`
	Enabled bool   `mapstructure:"enabled" json:"enabled,omitempty"`
}

type FePosthogConfig struct {
	ApiKey  string
	ApiHost string
}

type ServerConfig struct {
	Auth                   AuthConfig
	MessageQueueV1         msgqueue.MessageQueue
	Alerter                errors.Alerter
	Analytics              analytics.Analytics
	Email                  email.EmailService
	Ingestor               ingestor.Ingestor
	Encryption             encryption.EncryptionService
	Validator              validator.Validator
	TenantAlerter          *alerting.TenantAlertManager
	SessionStore           *cookie.UserSessionStore
	PausedControllers      map[string]bool
	SchedulingPoolV1       *v1.SchedulingPool
	AdditionalOAuthConfigs map[string]*oauth2.Config
	*database.Layer
	Logger                *zerolog.Logger
	FePosthog             *FePosthogConfig
	TLSConfig             *tls.Config
	InternalClientFactory *client.GRPCClientFactory
	Pylon                 *PylonConfig
	Prometheus            shared.PrometheusConfigFile
	AdditionalLoggers     ConfigFileAdditionalLoggers
	Version               string
	OpenTelemetry         shared.OpenTelemetryConfigFile
	Services              []string
	Namespaces            []string
	GRPCInterceptors      []grpc.UnaryServerInterceptor
	Runtime               ConfigFileRuntime
	CronOperations        CronOperationsConfigFile
	Sampling              ConfigFileSampling
	Operations            ConfigFileOperations
	OLAPStatusUpdates     OLAPStatusUpdateConfigFile
	EnableWorkerRetention bool
	EnableDataRetention   bool
}

type PayloadStoreConfig struct {
	EnablePayloadDualWrites              bool          `mapstructure:"enablePayloadDualWrites" json:"enablePayloadDualWrites,omitempty" default:"true"`
	EnableTaskEventPayloadDualWrites     bool          `mapstructure:"enableTaskEventPayloadDualWrites" json:"enableTaskEventPayloadDualWrites,omitempty" default:"true"`
	EnableDagDataPayloadDualWrites       bool          `mapstructure:"enableDagDataPayloadDualWrites" json:"enableDagDataPayloadDualWrites,omitempty" default:"true"`
	EnableOLAPPayloadDualWrites          bool          `mapstructure:"enableOLAPPayloadDualWrites" json:"enableOLAPPayloadDualWrites,omitempty" default:"true"`
	ExternalCutoverProcessInterval       time.Duration `mapstructure:"externalCutoverProcessInterval" json:"externalCutoverProcessInterval,omitempty" default:"15s"`
	ExternalCutoverBatchSize             int32         `mapstructure:"externalCutoverBatchSize" json:"externalCutoverBatchSize,omitempty" default:"1000"`
	ExternalCutoverNumConcurrentOffloads int32         `mapstructure:"externalCutoverNumConcurrentOffloads" json:"externalCutoverNumConcurrentOffloads,omitempty" default:"10"`
	InlineStoreTTLDays                   int32         `mapstructure:"inlineStoreTTLDays" json:"inlineStoreTTLDays,omitempty" default:"2"`
	EnableImmediateOffloads              bool          `mapstructure:"enableImmediateOffloads" json:"enableImmediateOffloads,omitempty" default:"false"`
}

func (c *ServerConfig) HasService(name string) bool {
	for _, s := range c.Services {
		if s == name {
			return true
		}
	}

	return false
}

func (c *ServerConfig) AddGRPCUnaryInterceptor(interceptor grpc.UnaryServerInterceptor) {
	c.GRPCInterceptors = append(c.GRPCInterceptors, interceptor)
}

func BindAllEnv(v *viper.Viper) {
	// runtime options
	_ = v.BindEnv("runtime.port", "SERVER_PORT")
	_ = v.BindEnv("runtime.url", "SERVER_URL")
	_ = v.BindEnv("runtime.healthcheck", "SERVER_HEALTHCHECK")
	_ = v.BindEnv("runtime.healthcheckPort", "SERVER_HEALTHCHECK_PORT")
	_ = v.BindEnv("runtime.grpcPort", "SERVER_GRPC_PORT")
	_ = v.BindEnv("runtime.grpcBindAddress", "SERVER_GRPC_BIND_ADDRESS")
	_ = v.BindEnv("runtime.grpcBroadcastAddress", "SERVER_GRPC_BROADCAST_ADDRESS")
	_ = v.BindEnv("runtime.grpcInsecure", "SERVER_GRPC_INSECURE")
	_ = v.BindEnv("runtime.grpcMaxMsgSize", "SERVER_GRPC_MAX_MSG_SIZE")
	_ = v.BindEnv("runtime.grpcWorkerStreamMaxBacklogSize", "SERVER_GRPC_WORKER_STREAM_MAX_BACKLOG_SIZE")
	_ = v.BindEnv("runtime.grpcStaticStreamWindowSize", "SERVER_GRPC_STATIC_STREAM_WINDOW_SIZE")
	_ = v.BindEnv("runtime.grpcRateLimit", "SERVER_GRPC_RATE_LIMIT")
	_ = v.BindEnv("runtime.schedulerConcurrencyRateLimit", "SCHEDULER_CONCURRENCY_RATE_LIMIT")
	_ = v.BindEnv("runtime.schedulerConcurrencyPollingMinInterval", "SCHEDULER_CONCURRENCY_POLLING_MIN_INTERVAL")
	_ = v.BindEnv("runtime.schedulerConcurrencyPollingMaxInterval", "SCHEDULER_CONCURRENCY_POLLING_MAX_INTERVAL")
	_ = v.BindEnv("runtime.shutdownWait", "SERVER_SHUTDOWN_WAIT")
	_ = v.BindEnv("servicesString", "SERVER_SERVICES")
	_ = v.BindEnv("pausedControllers", "SERVER_PAUSED_CONTROLLERS")
	_ = v.BindEnv("enableDataRetention", "SERVER_ENABLE_DATA_RETENTION")
	_ = v.BindEnv("enableWorkerRetention", "SERVER_ENABLE_WORKER_RETENTION")
	_ = v.BindEnv("runtime.enforceLimits", "SERVER_ENFORCE_LIMITS")
	_ = v.BindEnv("runtime.allowSignup", "SERVER_ALLOW_SIGNUP")
	_ = v.BindEnv("runtime.allowInvites", "SERVER_ALLOW_INVITES")
	_ = v.BindEnv("runtime.allowCreateTenant", "SERVER_ALLOW_CREATE_TENANT")
	_ = v.BindEnv("runtime.maxPendingInvites", "SERVER_MAX_PENDING_INVITES")
	_ = v.BindEnv("runtime.allowChangePassword", "SERVER_ALLOW_CHANGE_PASSWORD")
	_ = v.BindEnv("runtime.apiRateLimit", "SERVER_API_RATE_LIMIT")
	_ = v.BindEnv("runtime.apiRateLimitWindow", "SERVER_API_RATE_LIMIT_WINDOW")
	_ = v.BindEnv("runtime.disableTenantPubs", "SERVER_DISABLE_TENANT_PUBS")
	_ = v.BindEnv("runtime.maxInternalRetryCount", "SERVER_MAX_INTERNAL_RETRY_COUNT")
	_ = v.BindEnv("runtime.preventTenantVersionUpgrade", "SERVER_PREVENT_TENANT_VERSION_UPGRADE")
	_ = v.BindEnv("runtime.defaultEngineVersion", "SERVER_DEFAULT_ENGINE_VERSION")
	_ = v.BindEnv("runtime.replayEnabled", "SERVER_REPLAY_ENABLED")

	// security check options
	_ = v.BindEnv("securityCheck.enabled", "SERVER_SECURITY_CHECK_ENABLED")
	_ = v.BindEnv("securityCheck.endpoint", "SERVER_SECURITY_CHECK_ENDPOINT")

	// limit options
	_ = v.BindEnv("runtime.limits.defaultTenantRetentionPeriod", "SERVER_LIMITS_DEFAULT_TENANT_RETENTION_PERIOD")

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

	_ = v.BindEnv("runtime.limits.defaultIncomingWebhookLimit", "SERVER_LIMITS_DEFAULT_INCOMING_WEBHOOK_LIMIT")

	// buffer options
	_ = v.BindEnv("runtime.waitForFlush", "SERVER_WAIT_FOR_FLUSH")
	_ = v.BindEnv("runtime.maxConcurrent", "SERVER_MAX_CONCURRENT")
	_ = v.BindEnv("runtime.flushPeriodMilliseconds", "SERVER_FLUSH_PERIOD_MILLISECONDS")
	_ = v.BindEnv("runtime.flushItemsThreshold", "SERVER_FLUSH_ITEMS_THRESHOLD")
	_ = v.BindEnv("runtime.flushStrategy", "SERVER_FLUSH_STRATEGY")

	// log ingestion
	_ = v.BindEnv("runtime.logIngestionEnabled", "SERVER_LOG_INGESTION_ENABLED")

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
	_ = v.BindEnv("msgQueue.rabbitmq.maxPubChans", "SERVER_MSGQUEUE_RABBITMQ_MAX_PUB_CHANS")
	_ = v.BindEnv("msgQueue.rabbitmq.maxSubChans", "SERVER_MSGQUEUE_RABBITMQ_MAX_SUB_CHANS")
	_ = v.BindEnv("msgQueue.rabbitmq.compressionEnabled", "SERVER_MSGQUEUE_RABBITMQ_COMPRESSION_ENABLED")
	_ = v.BindEnv("msgQueue.rabbitmq.compressionThreshold", "SERVER_MSGQUEUE_RABBITMQ_COMPRESSION_THRESHOLD")
	_ = v.BindEnv("msgQueue.rabbitmq.enableMessageRejection", "SERVER_MSGQUEUE_RABBITMQ_ENABLE_MESSAGE_REJECTION")
	_ = v.BindEnv("msgQueue.rabbitmq.maxDeathCount", "SERVER_MSGQUEUE_RABBITMQ_MAX_DEATH_COUNT")

	// throughput options
	_ = v.BindEnv("msgQueue.rabbitmq.qos", "SERVER_MSGQUEUE_RABBITMQ_QOS")
	_ = v.BindEnv("runtime.requeueLimit", "SERVER_REQUEUE_LIMIT")
	_ = v.BindEnv("runtime.singleQueueLimit", "SERVER_SINGLE_QUEUE_LIMIT")
	_ = v.BindEnv("runtime.optimisticSchedulingEnabled", "SERVER_OPTIMISTIC_SCHEDULING_ENABLED")
	_ = v.BindEnv("runtime.optimisticSchedulingSlots", "SERVER_OPTIMISTIC_SCHEDULING_SLOTS")
	_ = v.BindEnv("runtime.grpcTriggerWritesEnabled", "SERVER_GRPC_TRIGGER_WRITES_ENABLED")
	_ = v.BindEnv("runtime.grpcTriggerWriteSlots", "SERVER_GRPC_TRIGGER_WRITE_SLOTS")
	_ = v.BindEnv("runtime.updateHashFactor", "SERVER_UPDATE_HASH_FACTOR")
	_ = v.BindEnv("runtime.updateConcurrentFactor", "SERVER_UPDATE_CONCURRENT_FACTOR")

	// enable durable user event log
	_ = v.BindEnv("runtime.enableDurableUserEventLog", "SERVER_ENABLE_DURABLE_USER_EVENT_LOG")

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
	_ = v.BindEnv("otel.collectorAuth", "SERVER_OTEL_COLLECTOR_AUTH")
	_ = v.BindEnv("otel.metricsEnabled", "SERVER_OTEL_METRICS_ENABLED")

	// prometheus options
	_ = v.BindEnv("prometheus.prometheusServerURL", "SERVER_PROMETHEUS_SERVER_URL")
	_ = v.BindEnv("prometheus.prometheusServerUsername", "SERVER_PROMETHEUS_SERVER_USERNAME")
	_ = v.BindEnv("prometheus.prometheusServerPassword", "SERVER_PROMETHEUS_SERVER_PASSWORD")
	_ = v.BindEnv("prometheus.enabled", "SERVER_PROMETHEUS_ENABLED")
	_ = v.BindEnv("prometheus.address", "SERVER_PROMETHEUS_ADDRESS")
	_ = v.BindEnv("prometheus.path", "SERVER_PROMETHEUS_PATH")

	// tenant alerting options
	_ = v.BindEnv("tenantAlerting.slack.enabled", "SERVER_TENANT_ALERTING_SLACK_ENABLED")
	_ = v.BindEnv("tenantAlerting.slack.clientID", "SERVER_TENANT_ALERTING_SLACK_CLIENT_ID")
	_ = v.BindEnv("tenantAlerting.slack.clientSecret", "SERVER_TENANT_ALERTING_SLACK_CLIENT_SECRET")
	_ = v.BindEnv("tenantAlerting.slack.scopes", "SERVER_TENANT_ALERTING_SLACK_SCOPES")

	// email options
	_ = v.BindEnv("email.kind", "SERVER_EMAIL_KIND")

	// postmark options
	_ = v.BindEnv("email.postmark.enabled", "SERVER_EMAIL_POSTMARK_ENABLED")
	_ = v.BindEnv("email.postmark.serverKey", "SERVER_EMAIL_POSTMARK_SERVER_KEY")
	_ = v.BindEnv("email.postmark.fromEmail", "SERVER_EMAIL_POSTMARK_FROM_EMAIL")
	_ = v.BindEnv("email.postmark.fromName", "SERVER_EMAIL_POSTMARK_FROM_NAME")
	_ = v.BindEnv("email.postmark.supportEmail", "SERVER_EMAIL_POSTMARK_SUPPORT_EMAIL")

	// smtp options
	_ = v.BindEnv("email.smtp.enabled", "SERVER_EMAIL_SMTP_ENABLED")
	_ = v.BindEnv("email.smtp.serverAddr", "SERVER_EMAIL_SMTP_SERVER_ADDR")
	_ = v.BindEnv("email.smtp.fromEmail", "SERVER_EMAIL_SMTP_FROM_EMAIL")
	_ = v.BindEnv("email.smtp.fromName", "SERVER_EMAIL_SMTP_FROM_NAME")
	_ = v.BindEnv("email.smtp.supportEmail", "SERVER_EMAIL_SMTP_SUPPORT_EMAIL")
	// allow basic auth credentials to be set
	_ = v.BindEnv("email.smtp.basicAuth.username", "SERVER_EMAIL_SMTP_AUTH_USERNAME")
	_ = v.BindEnv("email.smtp.basicAuth.password", "SERVER_EMAIL_SMTP_AUTH_PASSWORD")

	// monitoring options
	_ = v.BindEnv("runtime.monitoring.enabled", "SERVER_MONITORING_ENABLED")
	_ = v.BindEnv("runtime.monitoring.permittedTenants", "SERVER_MONITORING_PERMITTED_TENANTS")
	_ = v.BindEnv("runtime.monitoring.probeTimeout", "SERVER_MONITORING_PROBE_TIMEOUT")
	// we will fill this in from the server config if it is not set
	_ = v.BindEnv("runtime.monitoring.tlsRootCAFile", "SERVER_MONITORING_TLS_ROOT_CA_FILE")

	// sampling options
	_ = v.BindEnv("sampling.enabled", "SERVER_SAMPLING_ENABLED")
	_ = v.BindEnv("sampling.samplingRate", "SERVER_SAMPLING_RATE")

	// operations options
	_ = v.BindEnv("olap.jitter", "SERVER_OPERATIONS_JITTER")
	_ = v.BindEnv("olap.pollInterval", "SERVER_OPERATIONS_POLL_INTERVAL")

	// task operation limits options
	_ = v.BindEnv("taskOperationLimits.timeoutLimit", "SERVER_TASK_OPERATION_LIMITS_TIMEOUT_LIMIT")
	_ = v.BindEnv("taskOperationLimits.reassignLimit", "SERVER_TASK_OPERATION_LIMITS_REASSIGN_LIMIT")
	_ = v.BindEnv("taskOperationLimits.retryQueueLimit", "SERVER_TASK_OPERATION_LIMITS_RETRY_QUEUE_LIMIT")
	_ = v.BindEnv("taskOperationLimits.durableSleepLimit", "SERVER_TASK_OPERATION_LIMITS_DURABLE_SLEEP_LIMIT")

	// dispatcher options
	_ = v.BindEnv("runtime.workflowRunBufferSize", "SERVER_WORKFLOW_RUN_BUFFER_SIZE")

	// payload store options
	_ = v.BindEnv("payloadStore.enablePayloadDualWrites", "SERVER_PAYLOAD_STORE_ENABLE_PAYLOAD_DUAL_WRITES")
	_ = v.BindEnv("payloadStore.enableTaskEventPayloadDualWrites", "SERVER_PAYLOAD_STORE_ENABLE_TASK_EVENT_PAYLOAD_DUAL_WRITES")
	_ = v.BindEnv("payloadStore.enableDagDataPayloadDualWrites", "SERVER_PAYLOAD_STORE_ENABLE_DAG_DATA_PAYLOAD_DUAL_WRITES")
	_ = v.BindEnv("payloadStore.enableOLAPPayloadDualWrites", "SERVER_PAYLOAD_STORE_ENABLE_OLAP_PAYLOAD_DUAL_WRITES")
	_ = v.BindEnv("payloadStore.externalCutoverProcessInterval", "SERVER_PAYLOAD_STORE_EXTERNAL_CUTOVER_PROCESS_INTERVAL")
	_ = v.BindEnv("payloadStore.externalCutoverBatchSize", "SERVER_PAYLOAD_STORE_EXTERNAL_CUTOVER_BATCH_SIZE")
	_ = v.BindEnv("payloadStore.externalCutoverNumConcurrentOffloads", "SERVER_PAYLOAD_STORE_EXTERNAL_CUTOVER_NUM_CONCURRENT_OFFLOADS")
	_ = v.BindEnv("payloadStore.inlineStoreTTLDays", "SERVER_PAYLOAD_STORE_INLINE_STORE_TTL_DAYS")
	_ = v.BindEnv("payloadStore.enableImmediateOffloads", "SERVER_PAYLOAD_STORE_ENABLE_IMMEDIATE_OFFLOADS")

	// cron operations options
	_ = v.BindEnv("cronOperations.taskAnalyzeCronInterval", "SERVER_CRON_OPERATIONS_TASK_ANALYZE_CRON_INTERVAL")
	_ = v.BindEnv("cronOperations.olapAnalyzeCronInterval", "SERVER_CRON_OPERATIONS_OLAP_ANALYZE_CRON_INTERVAL")

	// OLAP status update options
	_ = v.BindEnv("statusUpdates.dagBatchSizeLimit", "SERVER_OLAP_STATUS_UPDATE_DAG_BATCH_SIZE_LIMIT")
	_ = v.BindEnv("statusUpdates.taskBatchSizeLimit", "SERVER_OLAP_STATUS_UPDATE_TASK_BATCH_SIZE_LIMIT")
}
