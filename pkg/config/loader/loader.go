// Adapted from: https://github.com/hatchet-dev/hatchet-v1-archived/blob/3c2c13168afa1af68d4baaf5ed02c9d49c5f0323/internal/config/loader/loader.go

package loader

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/exaring/otelpgx"
	pgxzero "github.com/jackc/pgx-zerolog"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"golang.org/x/oauth2"

	"github.com/hatchet-dev/hatchet/internal/integrations/alerting"
	"github.com/hatchet-dev/hatchet/internal/integrations/email"
	"github.com/hatchet-dev/hatchet/internal/integrations/email/postmark"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/msgqueue/postgres"
	"github.com/hatchet-dev/hatchet/internal/msgqueue/rabbitmq"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/analytics/posthog"
	"github.com/hatchet-dev/hatchet/pkg/auth/cookie"
	"github.com/hatchet-dev/hatchet/pkg/auth/oauth"
	"github.com/hatchet-dev/hatchet/pkg/auth/token"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/loader/loaderutils"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/errors/sentry"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/metered"
	postgresdb "github.com/hatchet-dev/hatchet/pkg/repository/postgres"
	v0 "github.com/hatchet-dev/hatchet/pkg/scheduling/v0"
	v1 "github.com/hatchet-dev/hatchet/pkg/scheduling/v1"
	"github.com/hatchet-dev/hatchet/pkg/security"
	"github.com/hatchet-dev/hatchet/pkg/validator"

	msgqueuev1 "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	pgmqv1 "github.com/hatchet-dev/hatchet/internal/msgqueue/v1/postgres"
	rabbitmqv1 "github.com/hatchet-dev/hatchet/internal/msgqueue/v1/rabbitmq"
	repov1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
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

type RepositoryOverrides struct {
	LogsEngineRepository repository.LogsEngineRepository
	LogsAPIRepository    repository.LogsAPIRepository
}

type ConfigLoader struct {
	directory           string
	RepositoryOverrides RepositoryOverrides
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

	scf, err := LoadServerConfigFile(configFileBytes...)

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

	config, err := pgxpool.ParseConfig(databaseUrl)
	if err != nil {
		return nil, err
	}

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

	config.MaxConnLifetime = 15 * 60 * time.Second

	if cf.Logger.Level == "debug" {
		debugger := &debugger{
			callerCounts: make(map[string]int),
			l:            &l,
		}

		config.BeforeAcquire = debugger.beforeAcquire
	}

	// a smaller pool for essential services like the heartbeat
	essentialConfig := config.Copy()
	essentialConfig.MinConns = 1

	essentialConfig.MaxConns /= 100
	if essentialConfig.MaxConns < 1 {
		essentialConfig.MaxConns = 1
	}

	config.MaxConns -= essentialConfig.MaxConns

	essentialPool, err := pgxpool.NewWithConfig(context.Background(), essentialConfig)

	if err != nil {
		return nil, fmt.Errorf("could not connect to database: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)

	if err != nil {
		return nil, fmt.Errorf("could not connect to database: %w", err)
	}

	ch := cache.New(cf.CacheDuration)

	entitlementRepo := postgresdb.NewEntitlementRepository(pool, &scf.Runtime, postgresdb.WithLogger(&l), postgresdb.WithCache(ch))

	meter := metered.NewMetered(entitlementRepo, &l)

	var opts []postgresdb.PostgresRepositoryOpt

	opts = append(opts, postgresdb.WithLogger(&l), postgresdb.WithCache(ch), postgresdb.WithMetered(meter))

	if c.RepositoryOverrides.LogsEngineRepository != nil {
		opts = append(opts, postgresdb.WithLogsEngineRepository(c.RepositoryOverrides.LogsEngineRepository))
	}

	cleanupEngine, engineRepo, err := postgresdb.NewEngineRepository(pool, essentialPool, &scf.Runtime, opts...)

	if err != nil {
		return nil, fmt.Errorf("could not create engine repository: %w", err)
	}

	if c.RepositoryOverrides.LogsAPIRepository != nil {
		opts = append(opts, postgresdb.WithLogsAPIRepository(c.RepositoryOverrides.LogsAPIRepository))
	}

	v1, cleanupV1 := repov1.NewRepository(pool, &l)

	apiRepo, cleanupApiRepo, err := postgresdb.NewAPIRepository(pool, &scf.Runtime, opts...)

	if err != nil {
		return nil, fmt.Errorf("could not create api repository: %w", err)
	}

	return &database.Layer{
		Disconnect: func() error {
			if err := cleanupEngine(); err != nil {
				return err
			}

			ch.Stop()
			meter.Stop()

			if err := cleanupV1(); err != nil {
				return err
			}

			return cleanupApiRepo()
		},
		Pool:                  pool,
		EssentialPool:         essentialPool,
		QueuePool:             pool,
		APIRepository:         apiRepo,
		EngineRepository:      engineRepo,
		EntitlementRepository: entitlementRepo,
		V1:                    v1,
		Seed:                  cf.Seed,
	}, nil

}

type ServerConfigFileOverride func(*server.ServerConfigFile)

// CreateServerFromConfig loads the server configuration and returns a server
func (c *ConfigLoader) CreateServerFromConfig(version string, overrides ...ServerConfigFileOverride) (cleanup func() error, res *server.ServerConfig, err error) {

	log.Printf("Loading server config from %s", c.directory)
	sharedFilePath := filepath.Join(c.directory, "server.yaml")
	log.Printf("Shared file path: %s", sharedFilePath)

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
		cookie.WithSessionRepository(dc.APIRepository.UserSession()),
		cookie.WithCookieAllowInsecure(cf.Auth.Cookie.Insecure),
		cookie.WithCookieDomain(cf.Auth.Cookie.Domain),
		cookie.WithCookieName(cf.Auth.Cookie.Name),
		cookie.WithCookieSecrets(getStrArr(cf.Auth.Cookie.Secrets)...),
	)

	if err != nil {
		return nil, nil, fmt.Errorf("could not create session store: %w", err)
	}

	var mq msgqueue.MessageQueue
	var mqv1 msgqueuev1.MessageQueue
	cleanup1 := func() error {
		return nil
	}

	var ing ingestor.Ingestor

	if cf.MessageQueue.Enabled {
		switch strings.ToLower(cf.MessageQueue.Kind) {
		case "postgres":
			l.Warn().Msg("Using a Postgres-backed message queue. This feature is still in beta.")

			mq = postgres.NewPostgresMQ(
				dc.EngineRepository.MessageQueue(),
				postgres.WithLogger(&l),
				postgres.WithQos(cf.MessageQueue.Postgres.Qos),
			)

			mqv1 = pgmqv1.NewPostgresMQ(
				dc.EngineRepository.MessageQueue(),
				pgmqv1.WithLogger(&l),
				pgmqv1.WithQos(cf.MessageQueue.Postgres.Qos),
			)
		case "rabbitmq":
			var cleanupv0 func() error
			var cleanupv1 func() error

			cleanupv0, mq = rabbitmq.New(
				rabbitmq.WithURL(cf.MessageQueue.RabbitMQ.URL),
				rabbitmq.WithLogger(&l),
				rabbitmq.WithQos(cf.MessageQueue.RabbitMQ.Qos),
				rabbitmq.WithDisableTenantExchangePubs(cf.Runtime.DisableTenantPubs),
			)

			cleanupv1, mqv1 = rabbitmqv1.New(
				rabbitmqv1.WithURL(cf.MessageQueue.RabbitMQ.URL),
				rabbitmqv1.WithLogger(&l),
				rabbitmqv1.WithQos(cf.MessageQueue.RabbitMQ.Qos),
				rabbitmqv1.WithDisableTenantExchangePubs(cf.Runtime.DisableTenantPubs),
			)

			cleanup1 = func() error {
				if err := cleanupv0(); err != nil {
					return err
				}

				return cleanupv1()
			}
		}

		ing, err = ingestor.NewIngestor(
			ingestor.WithEventRepository(dc.EngineRepository.Event()),
			ingestor.WithStreamEventsRepository(dc.EngineRepository.StreamEvent()),
			ingestor.WithLogRepository(dc.EngineRepository.Log()),
			ingestor.WithMessageQueue(mq),
			ingestor.WithEntitlementsRepository(dc.EntitlementRepository),
			ingestor.WithStepRunRepository(dc.EngineRepository.StepRun()),
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
		}, dc.APIRepository.SecurityCheck())

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

	encryptionSvc, err := loadEncryptionSvc(cf)

	if err != nil {
		return nil, nil, fmt.Errorf("could not load encryption service: %w", err)
	}

	// create a new JWT manager
	auth.JWTManager, err = token.NewJWTManager(encryptionSvc, dc.EngineRepository.APIToken(), &token.TokenOpts{
		Issuer:               cf.Runtime.ServerURL,
		Audience:             cf.Runtime.ServerURL,
		GRPCBroadcastAddress: cf.Runtime.GRPCBroadcastAddress,
		ServerURL:            cf.Runtime.ServerURL,
	})

	if err != nil {
		return nil, nil, fmt.Errorf("could not create JWT manager: %w", err)
	}

	var emailSvc email.EmailService = &email.NoOpService{}

	if cf.Email.Postmark.Enabled {
		emailSvc = postmark.NewPostmarkClient(
			cf.Email.Postmark.ServerKey,
			cf.Email.Postmark.FromEmail,
			cf.Email.Postmark.FromName,
			cf.Email.Postmark.SupportEmail,
		)
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

	schedulingPool, cleanupSchedulingPool, err := v0.NewSchedulingPool(
		dc.EngineRepository.Scheduler(),
		&queueLogger,
		cf.Runtime.SingleQueueLimit,
	)

	if err != nil {
		return nil, nil, fmt.Errorf("could not create scheduling pool: %w", err)
	}

	schedulingPoolV1, cleanupSchedulingPoolV1, err := v1.NewSchedulingPool(
		dc.V1.Scheduler(),
		&queueLogger,
		cf.Runtime.SingleQueueLimit,
	)

	if err != nil {
		return nil, nil, fmt.Errorf("could not create scheduling pool (v1): %w", err)
	}

	cleanup = func() error {
		log.Printf("cleaning up server config")

		if err := cleanupSchedulingPool(); err != nil {
			return fmt.Errorf("error cleaning up scheduling pool: %w", err)
		}

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

	if cf.Runtime.Monitoring.TLSRootCAFile == "" {
		cf.Runtime.Monitoring.TLSRootCAFile = cf.TLS.TLSRootCAFile
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
		MessageQueue:           mq,
		MessageQueueV1:         mqv1,
		Services:               services,
		Logger:                 &l,
		TLSConfig:              tls,
		SessionStore:           ss,
		Validator:              v,
		Ingestor:               ing,
		OpenTelemetry:          cf.OpenTelemetry,
		Email:                  emailSvc,
		TenantAlerter:          alerting.New(dc.EngineRepository, encryptionSvc, cf.Runtime.ServerURL, emailSvc),
		AdditionalOAuthConfigs: additionalOAuthConfigs,
		AdditionalLoggers:      cf.AdditionalLoggers,
		EnableDataRetention:    cf.EnableDataRetention,
		EnableWorkerRetention:  cf.EnableWorkerRetention,
		SchedulingPool:         schedulingPool,
		SchedulingPoolV1:       schedulingPoolV1,
		Version:                version,
	}, nil
}

func getStrArr(v string) []string {
	return strings.Split(v, " ")
}

func loadEncryptionSvc(cf *server.ServerConfigFile) (encryption.EncryptionService, error) {
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
