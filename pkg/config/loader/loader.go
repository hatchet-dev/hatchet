// Adapted from: https://github.com/hatchet-dev/hatchet-v1-archived/blob/3c2c13168afa1af68d4baaf5ed02c9d49c5f0323/internal/config/loader/loader.go

package loader

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/exaring/otelpgx"
	pgxzero "github.com/jackc/pgx-zerolog"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"

	"github.com/hatchet-dev/hatchet/internal/integrations/alerting"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/analytics/posthog"
	"github.com/hatchet-dev/hatchet/pkg/auth/cookie"
	"github.com/hatchet-dev/hatchet/pkg/auth/oauth"
	"github.com/hatchet-dev/hatchet/pkg/auth/token"
	"github.com/hatchet-dev/hatchet/pkg/config/client"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/loader/loaderutils"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/errors/sentry"
	"github.com/hatchet-dev/hatchet/pkg/integrations/email"
	"github.com/hatchet-dev/hatchet/pkg/integrations/email/postmark"
	"github.com/hatchet-dev/hatchet/pkg/integrations/email/smtp"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/debugger"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	v1 "github.com/hatchet-dev/hatchet/pkg/scheduling/v1"
	"github.com/hatchet-dev/hatchet/pkg/security"
	"github.com/hatchet-dev/hatchet/pkg/validator"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	pgmq "github.com/hatchet-dev/hatchet/internal/msgqueue/postgres"
	"github.com/hatchet-dev/hatchet/internal/msgqueue/rabbitmq"
	clientv1 "github.com/hatchet-dev/hatchet/pkg/client/v1"
	repov1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

// LoadDatabaseConfigFile loads the database config file via viper
func LoadDatabaseConfigFile(files ...[]byte) (*database.ConfigFile, error) {
	configFile := &database.ConfigFile{}
	f := database.BindAllEnv

	_, err := loaderutils.LoadConfigFromViper(f, configFile, files...)

	return configFile, err
}

// LoadServerConfigFile loads the server config file via viper
func LoadServerConfigFile(files ...[]byte) (*server.ServerConfigFile, error) {
	configFile := &server.ServerConfigFile{}
	f := server.BindAllEnv

	_, err := loaderutils.LoadConfigFromViper(f, configFile, files...)
	return configFile, err
}

type ConfigLoader struct {
	directory string
}

func NewConfigLoader(directory string) *ConfigLoader {
	return &ConfigLoader{directory: directory}
}

// InitDataLayer initializes the database layer from the configuration
func (c *ConfigLoader) InitDataLayer() (res *database.Layer, err error) {
	sharedFilePath := filepath.Join(c.directory, "database.yaml")
	configFileBytes, err := loaderutils.GetConfigBytes(sharedFilePath)

	if err != nil {
		return nil, err
	}

	cf, err := LoadDatabaseConfigFile(configFileBytes...)

	if err != nil {
		return nil, err
	}

	serverSharedFilePath := filepath.Join(c.directory, "server.yaml")
	serverConfigFileBytes, err := loaderutils.GetConfigBytes(serverSharedFilePath)

	if err != nil {
		return nil, err
	}

	scf, err := LoadServerConfigFile(serverConfigFileBytes...)

	if err != nil {
		return nil, err
	}

	l := logger.NewStdErr(&cf.Logger, "database")

	databaseUrl := os.Getenv("DATABASE_URL")

	if databaseUrl == "" {
		databaseUrl = fmt.Sprintf(
			"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
			cf.PostgresUsername,
			cf.PostgresPassword,
			cf.PostgresHost,
			cf.PostgresPort,
			cf.PostgresDbName,
			cf.PostgresSSLMode,
		)

		_ = os.Setenv("DATABASE_URL", databaseUrl)
	}

	pgxpoolConnAfterConnect := func(ctx context.Context, conn *pgx.Conn) error {
		// Set timezone to UTC for all connections
		if _, err := conn.Exec(ctx, "SET TIME ZONE 'UTC'"); err != nil {
			return err
		}

		// ref: https://github.com/jackc/pgx/issues/1549
		t, err := conn.LoadType(ctx, "v1_readable_status_olap")
		if err != nil {
			return err
		}

		conn.TypeMap().RegisterType(t)

		t, err = conn.LoadType(ctx, "_v1_readable_status_olap")
		if err != nil {
			return err
		}

		conn.TypeMap().RegisterType(t)

		t, err = conn.LoadType(ctx, "v1_log_line_level")
		if err != nil {
			return err
		}

		conn.TypeMap().RegisterType(t)

		t, err = conn.LoadType(ctx, "_v1_log_line_level")
		if err != nil {
			return err
		}

		conn.TypeMap().RegisterType(t)

		_, err = conn.Exec(ctx, "SET statement_timeout=30000")

		return err
	}

	config, err := pgxpool.ParseConfig(databaseUrl)
	if err != nil {
		return nil, err
	}

	config.AfterConnect = pgxpoolConnAfterConnect

	if cf.LogQueries {
		config.ConnConfig.Tracer = &tracelog.TraceLog{
			Logger:   pgxzero.NewLogger(l),
			LogLevel: tracelog.LogLevelDebug,
		}
	}

	config.ConnConfig.Tracer = otelpgx.NewTracer()

	if cf.MaxConns != 0 {
		config.MaxConns = int32(cf.MaxConns) // nolint: gosec
	}

	if cf.MinConns != 0 {
		config.MinConns = int32(cf.MinConns) // nolint: gosec
	}

	config.MaxConnLifetime = cf.MaxConnLifetime
	config.MaxConnIdleTime = cf.MaxConnIdleTime

	// Check database instance timezone if enforcement is enabled
	if cf.EnforceUTCTimezone {
		if err := checkDatabaseTimezone(config.ConnConfig, cf.PostgresDbName, "primary database", &l); err != nil {
			return nil, err
		}
	}

	var debug *debugger.Debugger

	if cf.Logger.Level == "debug" {
		debug = debugger.NewDebugger(&l)

		config.BeforeAcquire = debug.BeforeAcquire // nolint: staticcheck
		config.AfterRelease = debug.AfterRelease
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)

	if err != nil {
		return nil, fmt.Errorf("could not connect to database: %w", err)
	}

	if debug != nil {
		// pool needs the debugger hooks (BeforeAcquire/AfterRelease) but debugger needs the pool
		// to track active connections, so we add the pool later
		debug.Setup(pool)
	}

	// a pool for read replicas, if enabled
	var readReplicaPool *pgxpool.Pool

	if cf.ReadReplicaEnabled {
		if cf.ReadReplicaDatabaseURL == "" {
			return nil, fmt.Errorf("read replica database url is required if read replica is enabled")
		}

		readReplicaConfig, err := pgxpool.ParseConfig(cf.ReadReplicaDatabaseURL)

		if err != nil {
			return nil, fmt.Errorf("could not parse read replica database url: %w", err)
		}

		if cf.ReadReplicaMaxConns != 0 {
			readReplicaConfig.MaxConns = int32(cf.ReadReplicaMaxConns) // nolint: gosec
		}

		if cf.ReadReplicaMinConns != 0 {
			readReplicaConfig.MinConns = int32(cf.ReadReplicaMinConns) // nolint: gosec
		}

		readReplicaConfig.MaxConnLifetime = cf.MaxConnLifetime
		readReplicaConfig.MaxConnIdleTime = cf.MaxConnIdleTime
		readReplicaConfig.ConnConfig.Tracer = otelpgx.NewTracer()

		// Check read replica database instance timezone if enforcement is enabled
		if cf.EnforceUTCTimezone {
			if err := checkDatabaseTimezone(readReplicaConfig.ConnConfig, "", "read replica database", &l); err != nil {
				return nil, err
			}
		}

		readReplicaConfig.AfterConnect = pgxpoolConnAfterConnect

		readReplicaPool, err = pgxpool.NewWithConfig(context.Background(), readReplicaConfig)

		if err != nil {
			return nil, fmt.Errorf("could not connect to read replica database: %w", err)
		}
	}

	ch := cache.New(cf.CacheDuration)

	retentionPeriod, err := time.ParseDuration(scf.Runtime.Limits.DefaultTenantRetentionPeriod)

	if err != nil {
		return nil, fmt.Errorf("could not parse retention period %s: %w", scf.Runtime.Limits.DefaultTenantRetentionPeriod, err)
	}

	taskLimits := repov1.TaskOperationLimits{
		TimeoutLimit:      scf.Runtime.TaskOperationLimits.TimeoutLimit,
		ReassignLimit:     scf.Runtime.TaskOperationLimits.ReassignLimit,
		RetryQueueLimit:   scf.Runtime.TaskOperationLimits.RetryQueueLimit,
		DurableSleepLimit: scf.Runtime.TaskOperationLimits.DurableSleepLimit,
	}

	inlineStoreTTLDays := scf.PayloadStore.InlineStoreTTLDays

	if inlineStoreTTLDays <= 0 {
		return nil, fmt.Errorf("inline store TTL days must be greater than 0")
	}

	inlineStoreTTL := time.Duration(inlineStoreTTLDays) * 24 * time.Hour

	payloadStoreOpts := repov1.PayloadStoreRepositoryOpts{
		EnablePayloadDualWrites:              scf.PayloadStore.EnablePayloadDualWrites,
		EnableTaskEventPayloadDualWrites:     scf.PayloadStore.EnableTaskEventPayloadDualWrites,
		EnableOLAPPayloadDualWrites:          scf.PayloadStore.EnableOLAPPayloadDualWrites,
		EnableDagDataPayloadDualWrites:       scf.PayloadStore.EnableDagDataPayloadDualWrites,
		ExternalCutoverProcessInterval:       scf.PayloadStore.ExternalCutoverProcessInterval,
		ExternalCutoverBatchSize:             scf.PayloadStore.ExternalCutoverBatchSize,
		ExternalCutoverNumConcurrentOffloads: scf.PayloadStore.ExternalCutoverNumConcurrentOffloads,
		InlineStoreTTL:                       &inlineStoreTTL,
		EnableImmediateOffloads:              scf.PayloadStore.EnableImmediateOffloads,
	}

	statusUpdateOpts := repov1.StatusUpdateBatchSizeLimits{
		Task: int32(scf.OLAPStatusUpdates.TaskBatchSizeLimit),
		DAG:  int32(scf.OLAPStatusUpdates.DagBatchSizeLimit),
	}

	v1, cleanupV1 := repov1.NewRepository(
		pool,
		&l,
		cf.CacheDuration,
		retentionPeriod,
		retentionPeriod,
		scf.Runtime.MaxInternalRetryCount,
		taskLimits,
		payloadStoreOpts,
		statusUpdateOpts,
		scf.Runtime.Limits,
		scf.Runtime.EnforceLimits,
		scf.Runtime.EnableDurableUserEventLog,
	)

	if readReplicaPool != nil {
		v1.OLAP().SetReadReplicaPool(readReplicaPool)
	}

	return &database.Layer{
		Disconnect: func() error {
			ch.Stop()

			return cleanupV1()
		},
		Pool:      pool,
		QueuePool: pool,
		V1:        v1,
		Seed:      cf.Seed,
	}, nil

}

type ServerConfigFileOverride func(*server.ServerConfigFile)

// CreateServerFromConfig loads the server configuration and returns a server
func (c *ConfigLoader) CreateServerFromConfig(version string, overrides ...ServerConfigFileOverride) (cleanup func() error, res *server.ServerConfig, err error) {
	sharedFilePath := filepath.Join(c.directory, "server.yaml")

	configFileBytes, err := loaderutils.GetConfigBytes(sharedFilePath)

	if err != nil {
		return nil, nil, err
	}

	dc, err := c.InitDataLayer()
	if err != nil {
		return nil, nil, err
	}

	cf, err := LoadServerConfigFile(configFileBytes...)
	if err != nil {
		return nil, nil, err
	}

	for _, override := range overrides {
		override(cf)
	}

	return createControllerLayer(dc, cf, version)
}

func createControllerLayer(dc *database.Layer, cf *server.ServerConfigFile, version string) (cleanup func() error, res *server.ServerConfig, err error) {
	l := logger.NewStdErr(&cf.Logger, "server")
	queueLogger := logger.NewStdErr(&cf.AdditionalLoggers.Queue, "queue")

	tls, err := loaderutils.LoadServerTLSConfig(&cf.TLS)

	if err != nil {
		return nil, nil, fmt.Errorf("could not load TLS config: %w", err)
	}

	ss, err := cookie.NewUserSessionStore(
		cookie.WithSessionRepository(dc.V1.UserSession()),
		cookie.WithCookieAllowInsecure(cf.Auth.Cookie.Insecure),
		cookie.WithCookieDomain(cf.Auth.Cookie.Domain),
		cookie.WithCookieName(cf.Auth.Cookie.Name),
		cookie.WithCookieSecrets(getStrArr(cf.Auth.Cookie.Secrets)...),
	)

	if err != nil {
		return nil, nil, fmt.Errorf("could not create session store: %w", err)
	}

	var mqv1 msgqueue.MessageQueue
	cleanup1 := func() error {
		return nil
	}

	var ing ingestor.Ingestor

	if cf.MessageQueue.Enabled {
		switch strings.ToLower(cf.MessageQueue.Kind) {
		case "postgres":
			var cleanupv1 func() error

			cleanupv1, mqv1, err = pgmq.NewPostgresMQ(
				dc.V1.MessageQueue(),
				pgmq.WithLogger(&l),
				pgmq.WithQos(cf.MessageQueue.Postgres.Qos),
			)

			if err != nil {
				return nil, nil, fmt.Errorf("could not init postgres queue: %w", err)
			}

			cleanup1 = func() error {
				return cleanupv1()
			}
		case "rabbitmq":
			if cf.MessageQueue.RabbitMQ.URL == "" {
				return nil, nil, fmt.Errorf("using RabbitMQ as message queue requires a URL to be set")
			}

			var cleanupv1 func() error

			cleanupv1, mqv1, err = rabbitmq.New(
				rabbitmq.WithURL(cf.MessageQueue.RabbitMQ.URL),
				rabbitmq.WithLogger(&l),
				rabbitmq.WithQos(cf.MessageQueue.RabbitMQ.Qos),
				rabbitmq.WithDisableTenantExchangePubs(cf.Runtime.DisableTenantPubs),
				rabbitmq.WithMaxPubChannels(cf.MessageQueue.RabbitMQ.MaxPubChans),
				rabbitmq.WithMaxSubChannels(cf.MessageQueue.RabbitMQ.MaxSubChans),
				rabbitmq.WithGzipCompression(
					cf.MessageQueue.RabbitMQ.CompressionEnabled,
					cf.MessageQueue.RabbitMQ.CompressionThreshold,
				),
				rabbitmq.WithMessageRejection(cf.MessageQueue.RabbitMQ.EnableMessageRejection, cf.MessageQueue.RabbitMQ.MaxDeathCount),
			)

			if err != nil {
				return nil, nil, fmt.Errorf("could not init rabbitmq: %w", err)
			}

			cleanup1 = func() error {
				return cleanupv1()
			}
		}

		ing, err = ingestor.NewIngestor(
			ingestor.WithMessageQueueV1(mqv1),
			ingestor.WithRepositoryV1(dc.V1),
		)

		if err != nil {
			return nil, nil, fmt.Errorf("could not create ingestor: %w", err)
		}
	}

	var alerter errors.Alerter

	if cf.Alerting.Sentry.Enabled {
		alerter, err = sentry.NewSentryAlerter(&sentry.SentryAlerterOpts{
			DSN:         cf.Alerting.Sentry.DSN,
			Environment: cf.Alerting.Sentry.Environment,
			SampleRate:  cf.Alerting.Sentry.SampleRate,
		})

		if err != nil {
			return nil, nil, fmt.Errorf("could not create sentry alerter: %w", err)
		}
	} else {
		alerter = errors.NoOpAlerter{}
	}

	if cf.SecurityCheck.Enabled {
		securityCheck := security.NewSecurityCheck(&security.DefaultSecurityCheck{
			Enabled:  cf.SecurityCheck.Enabled,
			Endpoint: cf.SecurityCheck.Endpoint,
			Logger:   &l,
			Version:  version,
		}, dc.V1.SecurityCheck())

		defer securityCheck.Check()
	}

	var analyticsEmitter analytics.Analytics
	var feAnalyticsConfig *server.FePosthogConfig

	if cf.Analytics.Posthog.Enabled {
		analyticsEmitter, err = posthog.NewPosthogAnalytics(&posthog.PosthogAnalyticsOpts{
			ApiKey:   cf.Analytics.Posthog.ApiKey,
			Endpoint: cf.Analytics.Posthog.Endpoint,
		})

		if cf.Analytics.Posthog.FeApiKey != "" && cf.Analytics.Posthog.FeApiHost != "" {

			feAnalyticsConfig = &server.FePosthogConfig{
				ApiKey:  cf.Analytics.Posthog.FeApiKey,
				ApiHost: cf.Analytics.Posthog.FeApiHost,
			}
		}

		if err != nil {
			return nil, nil, fmt.Errorf("could not create posthog analytics: %w", err)
		}
	} else {
		analyticsEmitter = analytics.NoOpAnalytics{}
	}

	// Register analytics callbacks for user and tenant creation
	dc.V1.User().RegisterCreateCallback(func(opts *repov1.UserCreateCallbackOpts) error {
		// Determine provider from opts
		provider := "basic"
		if opts.CreateOpts.OAuth != nil {
			provider = opts.CreateOpts.OAuth.Provider
		}

		analyticsEmitter.Enqueue(
			"user:create",
			opts.ID.String(),
			nil,
			map[string]interface{}{
				"email":    opts.Email,
				"name":     opts.Name.String,
				"provider": provider,
			},
			nil,
		)
		return nil
	})

	dc.V1.Tenant().RegisterCreateCallback(func(tenant *sqlcv1.Tenant) error {
		tenantId := tenant.ID

		analyticsEmitter.Tenant(tenantId, map[string]interface{}{
			"name": tenant.Name,
			"slug": tenant.Slug,
		})

		analyticsEmitter.Enqueue(
			"tenant:create",
			"system",
			&tenantId,
			map[string]interface{}{
				"tenant_created": true,
			},
			map[string]interface{}{
				"name": tenant.Name,
				"slug": tenant.Slug,
			},
		)
		return nil
	})

	var pylon server.PylonConfig

	if cf.Pylon.Enabled {
		if cf.Pylon.AppID == "" {
			return nil, nil, fmt.Errorf("pylon app id is required")
		}

		pylon.AppID = cf.Pylon.AppID
		pylon.Secret = cf.Pylon.Secret
	}

	auth := server.AuthConfig{
		RestrictedEmailDomains: getStrArr(cf.Auth.RestrictedEmailDomains),
		ConfigFile:             cf.Auth,
	}

	if cf.Auth.Google.Enabled {
		if cf.Auth.Google.ClientID == "" {
			return nil, nil, fmt.Errorf("google client id is required")
		}

		if cf.Auth.Google.ClientSecret == "" {
			return nil, nil, fmt.Errorf("google client secret is required")
		}

		gClient := oauth.NewGoogleClient(&oauth.Config{
			ClientID:     cf.Auth.Google.ClientID,
			ClientSecret: cf.Auth.Google.ClientSecret,
			BaseURL:      cf.Runtime.ServerURL,
			Scopes:       cf.Auth.Google.Scopes,
		})

		auth.GoogleOAuthConfig = gClient
	}

	if cf.Auth.Github.Enabled {
		if cf.Auth.Github.ClientID == "" {
			return nil, nil, fmt.Errorf("github client id is required")
		}

		if cf.Auth.Github.ClientSecret == "" {
			return nil, nil, fmt.Errorf("github client secret is required")
		}

		auth.GithubOAuthConfig = oauth.NewGithubClient(&oauth.Config{
			ClientID:     cf.Auth.Github.ClientID,
			ClientSecret: cf.Auth.Github.ClientSecret,
			BaseURL:      cf.Runtime.ServerURL,
			Scopes:       cf.Auth.Github.Scopes,
		})
	}

	encryptionSvc, err := LoadEncryptionSvc(cf)

	if err != nil {
		return nil, nil, fmt.Errorf("could not load encryption service: %w", err)
	}

	// create a new JWT manager
	auth.JWTManager, err = token.NewJWTManager(encryptionSvc, dc.V1.APIToken(), &token.TokenOpts{
		Issuer:               cf.Runtime.ServerURL,
		Audience:             cf.Runtime.ServerURL,
		GRPCBroadcastAddress: cf.Runtime.GRPCBroadcastAddress,
		ServerURL:            cf.Runtime.ServerURL,
	})

	if err != nil {
		return nil, nil, fmt.Errorf("could not create JWT manager: %w", err)
	}

	var emailSvc email.EmailService = &email.NoOpService{}

	switch strings.ToLower(cf.Email.Kind) {
	case "postmark":
		if !cf.Email.Postmark.Enabled {
			break
		}
		emailSvc = postmark.NewPostmarkClient(
			cf.Email.Postmark.ServerKey,
			cf.Email.Postmark.FromEmail,
			cf.Email.Postmark.FromName,
			cf.Email.Postmark.SupportEmail,
		)

	case "smtp":
		if !cf.Email.SMTP.Enabled {
			break
		}
		emailSvc, err = smtp.NewSMTPService(
			cf.Email.SMTP.ServerAddr,
			cf.Email.SMTP.BasicAuth.Username,
			cf.Email.SMTP.BasicAuth.Password,
			cf.Email.SMTP.FromEmail,
			cf.Email.SMTP.FromName,
			cf.Email.SMTP.SupportEmail,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create SMTP service: %w", err)
		}
	default:
		return nil, nil, fmt.Errorf("invalid email provider of type %s, must be 'postmark' or 'smtp'", cf.Email.Kind)
	}

	additionalOAuthConfigs := make(map[string]*oauth2.Config)

	if cf.TenantAlerting.Slack.Enabled {
		additionalOAuthConfigs["slack"] = oauth.NewSlackClient(&oauth.Config{
			ClientID:     cf.TenantAlerting.Slack.SlackAppClientID,
			ClientSecret: cf.TenantAlerting.Slack.SlackAppClientSecret,
			BaseURL:      cf.Runtime.ServerURL,
			Scopes:       cf.TenantAlerting.Slack.SlackAppScopes,
		})
	}

	v := validator.NewDefaultValidator()

	schedulingPoolV1, cleanupSchedulingPoolV1, err := v1.NewSchedulingPool(
		dc.V1.Scheduler(),
		&queueLogger,
		cf.Runtime.SingleQueueLimit,
		cf.Runtime.SchedulerConcurrencyRateLimit,
		cf.Runtime.SchedulerConcurrencyPollingMinInterval,
		cf.Runtime.SchedulerConcurrencyPollingMaxInterval,
		cf.Runtime.OptimisticSchedulingEnabled,
		cf.Runtime.OptimisticSchedulingSlots,
	)

	if err != nil {
		return nil, nil, fmt.Errorf("could not create scheduling pool (v1): %w", err)
	}

	schedulingPoolV1.Extensions.Add(v1.NewPrometheusExtension())

	cleanup = func() error {
		log.Printf("cleaning up server config")

		if err := cleanupSchedulingPoolV1(); err != nil {
			return fmt.Errorf("error cleaning up scheduling pool (v1): %w", err)
		}

		if err := cleanup1(); err != nil {
			return fmt.Errorf("error cleaning up rabbitmq: %w", err)
		}
		return nil
	}

	services := cf.Services

	// edge case to support backwards-compatibility with the services array in the config file
	if cf.ServicesString != "" {
		services = strings.Split(cf.ServicesString, " ")
	}

	pausedControllers := make(map[string]bool)

	if cf.PausedControllers != "" {
		for _, controller := range strings.Split(cf.PausedControllers, " ") {
			pausedControllers[controller] = true
		}
	}

	if cf.Runtime.Monitoring.TLSRootCAFile == "" {
		cf.Runtime.Monitoring.TLSRootCAFile = cf.TLS.TLSRootCAFile
	}

	internalClientFactory, err := loadInternalClient(&l, &cf.InternalClient, cf.TLS, cf.Runtime.GRPCBroadcastAddress, cf.Runtime.GRPCInsecure)

	if err != nil {
		return nil, nil, fmt.Errorf("could not load internal client: %w", err)
	}

	return cleanup, &server.ServerConfig{
		Alerter:                alerter,
		Analytics:              analyticsEmitter,
		FePosthog:              feAnalyticsConfig,
		Pylon:                  &pylon,
		Runtime:                cf.Runtime,
		Auth:                   auth,
		Encryption:             encryptionSvc,
		Layer:                  dc,
		MessageQueueV1:         mqv1,
		Services:               services,
		PausedControllers:      pausedControllers,
		InternalClientFactory:  internalClientFactory,
		Logger:                 &l,
		TLSConfig:              tls,
		SessionStore:           ss,
		Validator:              v,
		Ingestor:               ing,
		OpenTelemetry:          cf.OpenTelemetry,
		Prometheus:             cf.Prometheus,
		Email:                  emailSvc,
		TenantAlerter:          alerting.New(dc.V1, encryptionSvc, cf.Runtime.ServerURL, emailSvc),
		AdditionalOAuthConfigs: additionalOAuthConfigs,
		AdditionalLoggers:      cf.AdditionalLoggers,
		EnableDataRetention:    cf.EnableDataRetention,
		EnableWorkerRetention:  cf.EnableWorkerRetention,
		SchedulingPoolV1:       schedulingPoolV1,
		Version:                version,
		Sampling:               cf.Sampling,
		Operations:             cf.OLAP,
		CronOperations:         cf.CronOperations,
		OLAPStatusUpdates:      cf.OLAPStatusUpdates,
	}, nil
}

func getStrArr(v string) []string {
	return strings.Split(v, " ")
}

func LoadEncryptionSvc(cf *server.ServerConfigFile) (encryption.EncryptionService, error) {
	var err error

	hasLocalMasterKeyset := cf.Encryption.MasterKeyset != "" || cf.Encryption.MasterKeysetFile != ""
	isCloudKMSEnabled := cf.Encryption.CloudKMS.Enabled

	if !hasLocalMasterKeyset && !isCloudKMSEnabled {
		return nil, fmt.Errorf("encryption is required")
	}

	if hasLocalMasterKeyset && isCloudKMSEnabled {
		return nil, fmt.Errorf("cannot use both encryption and cloud kms")
	}

	hasJWTKeys := (cf.Encryption.JWT.PublicJWTKeyset != "" || cf.Encryption.JWT.PublicJWTKeysetFile != "") &&
		(cf.Encryption.JWT.PrivateJWTKeyset != "" || cf.Encryption.JWT.PrivateJWTKeysetFile != "")

	if !hasJWTKeys {
		return nil, fmt.Errorf("jwt encryption is required")
	}

	privateJWT := cf.Encryption.JWT.PrivateJWTKeyset

	if cf.Encryption.JWT.PrivateJWTKeysetFile != "" {
		privateJWTBytes, err := loaderutils.GetFileBytes(cf.Encryption.JWT.PrivateJWTKeysetFile)

		if err != nil {
			return nil, fmt.Errorf("could not load private jwt keyset file: %w", err)
		}

		privateJWT = string(privateJWTBytes)
	}

	publicJWT := cf.Encryption.JWT.PublicJWTKeyset

	if cf.Encryption.JWT.PublicJWTKeysetFile != "" {
		publicJWTBytes, err := loaderutils.GetFileBytes(cf.Encryption.JWT.PublicJWTKeysetFile)

		if err != nil {
			return nil, fmt.Errorf("could not load public jwt keyset file: %w", err)
		}

		publicJWT = string(publicJWTBytes)
	}

	var encryptionSvc encryption.EncryptionService

	if hasLocalMasterKeyset {
		masterKeyset := cf.Encryption.MasterKeyset

		if cf.Encryption.MasterKeysetFile != "" {
			masterKeysetBytes, err := loaderutils.GetFileBytes(cf.Encryption.MasterKeysetFile)

			if err != nil {
				return nil, fmt.Errorf("could not load master keyset file: %w", err)
			}

			masterKeyset = string(masterKeysetBytes)
		}

		encryptionSvc, err = encryption.NewLocalEncryption(
			[]byte(masterKeyset),
			[]byte(privateJWT),
			[]byte(publicJWT),
		)

		if err != nil {
			return nil, fmt.Errorf("could not create raw keyset encryption service: %w", err)
		}
	}

	if isCloudKMSEnabled {
		encryptionSvc, err = encryption.NewCloudKMSEncryption(
			cf.Encryption.CloudKMS.KeyURI,
			[]byte(cf.Encryption.CloudKMS.CredentialsJSON),
			[]byte(privateJWT),
			[]byte(publicJWT),
		)

		if err != nil {
			return nil, fmt.Errorf("could not create CloudKMS encryption service: %w", err)
		}
	}

	return encryptionSvc, nil
}

func loadInternalClient(l *zerolog.Logger, conf *server.InternalClientTLSConfigFile, baseServerTLS shared.TLSConfigFile, grpcBroadcastAddress string, grpcInsecure bool) (*clientv1.GRPCClientFactory, error) {
	// get gRPC broadcast address
	broadcastAddress := grpcBroadcastAddress

	if conf.InternalGRPCBroadcastAddress != "" {
		broadcastAddress = conf.InternalGRPCBroadcastAddress
	}

	tlsServerName := conf.TLSServerName

	if tlsServerName == "" {
		// parse host from broadcast address
		host, _, err := net.SplitHostPort(broadcastAddress)

		if err != nil {
			return nil, fmt.Errorf("could not parse host from broadcast address %s: %w", broadcastAddress, err)
		}

		tlsServerName = host
	}

	// construct TLS config
	var base shared.TLSConfigFile

	if conf.InheritBase {
		base = baseServerTLS

		if grpcInsecure {
			base.TLSStrategy = "none"
		}
	} else {
		base = conf.Base
	}

	tlsConfig, err := loaderutils.LoadClientTLSConfig(&client.ClientTLSConfigFile{
		Base:          base,
		TLSServerName: tlsServerName,
	}, tlsServerName)

	if err != nil {
		return nil, fmt.Errorf("could not load client TLS config: %w", err)
	}

	return clientv1.NewGRPCClientFactory(
		clientv1.WithHostPort(broadcastAddress),
		clientv1.WithTLS(tlsConfig),
		clientv1.WithLogger(l),
	), nil
}

// checkDatabaseTimezone validates that the database instance timezone is set to UTC.
// It creates a temporary connection to check the timezone without using the AfterConnect hook.
func checkDatabaseTimezone(connConfig *pgx.ConnConfig, dbName string, dbLabel string, l *zerolog.Logger) error {
	tempConn, err := pgx.ConnectConfig(context.Background(), connConfig)
	if err != nil {
		return fmt.Errorf("could not create temporary connection to %s to check timezone: %w", dbLabel, err)
	}
	defer tempConn.Close(context.Background())

	var dbTimezone string
	if err := tempConn.QueryRow(context.Background(), "SHOW timezone").Scan(&dbTimezone); err != nil {
		return fmt.Errorf("could not query %s timezone: %w", dbLabel, err)
	}

	// Accept both "UTC" and "Etc/UTC" as valid UTC timezones
	if dbTimezone != "UTC" && dbTimezone != "Etc/UTC" {
		if dbName == "" {
			dbName = "<your_database_name>"
		}
		return fmt.Errorf(
			"%s instance timezone is set to '%s' but must be 'UTC' or 'Etc/UTC'\n"+
				"This check ensures time-based operations work correctly across all sessions\n"+
				"To fix this issue, you have two options:\n"+
				"  1. Set your PostgreSQL instance timezone to UTC by running: ALTER DATABASE %s SET TIMEZONE='UTC'\n"+
				"  2. Disable this check by setting the environment variable: DATABASE_ENFORCE_UTC_TIMEZONE=false\n"+
				"Note: Disabling this check is not recommended as it may lead to timezone-related issues",
			dbLabel, dbTimezone, dbName,
		)
	}

	l.Info().Msgf("%s instance timezone verified: %s", dbLabel, dbTimezone)
	return nil
}
