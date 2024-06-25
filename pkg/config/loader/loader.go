// Adapted from: https://github.com/hatchet-dev/hatchet-v1-archived/blob/3c2c13168afa1af68d4baaf5ed02c9d49c5f0323/internal/config/loader/loader.go

package loader

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/exaring/otelpgx"
	pgxzero "github.com/jackc/pgx-zerolog"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"golang.org/x/oauth2"

	"github.com/hatchet-dev/hatchet/internal/integrations/alerting"
	"github.com/hatchet-dev/hatchet/internal/integrations/email"
	"github.com/hatchet-dev/hatchet/internal/integrations/email/postmark"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
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
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/metered"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/security"
	"github.com/hatchet-dev/hatchet/pkg/validator"
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
	return &ConfigLoader{directory}
}

// LoadDatabaseConfig loads the database configuration
func (c *ConfigLoader) LoadDatabaseConfig() (res *database.Config, err error) {
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

	return GetDatabaseConfigFromConfigFile(cf, &scf.Runtime)
}

type ServerConfigFileOverride func(*server.ServerConfigFile)

// LoadServerConfig loads the server configuration
func (c *ConfigLoader) LoadServerConfig(version string, overrides ...ServerConfigFileOverride) (cleanup func() error, res *server.ServerConfig, err error) {
	log.Printf("Loading server config from %s", c.directory)
	sharedFilePath := filepath.Join(c.directory, "server.yaml")
	log.Printf("Shared file path: %s", sharedFilePath)

	configFileBytes, err := loaderutils.GetConfigBytes(sharedFilePath)
	if err != nil {
		return nil, nil, err
	}

	dc, err := c.LoadDatabaseConfig()
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

	return GetServerConfigFromConfigfile(dc, cf, version)
}

func GetDatabaseConfigFromConfigFile(cf *database.ConfigFile, runtime *server.ConfigFileRuntime) (res *database.Config, err error) {
	l := logger.NewStdErr(&cf.Logger, "database")

	databaseUrl := fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		cf.PostgresUsername,
		cf.PostgresPassword,
		cf.PostgresHost,
		cf.PostgresPort,
		cf.PostgresDbName,
		cf.PostgresSSLMode,
	)

	// TODO db.WithDatasourceURL(databaseUrl) is not working
	_ = os.Setenv("DATABASE_URL", databaseUrl)

	c := db.NewClient()

	if err := c.Prisma.Connect(); err != nil {
		return nil, err
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

	config.MaxConns = int32(cf.MaxConns)
	config.MinConns = int32(cf.MaxConns)

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("could not connect to database: %w", err)
	}

	ch := cache.New(cf.CacheDuration)

	entitlementRepo := prisma.NewEntitlementRepository(pool, runtime, prisma.WithLogger(&l), prisma.WithCache(ch))

	meter := metered.NewMetered(entitlementRepo, &l)

	return &database.Config{
		Disconnect: func() error {
			ch.Stop()
			meter.Stop()
			return c.Prisma.Disconnect()
		},
		Pool:                  pool,
		APIRepository:         prisma.NewAPIRepository(c, pool, prisma.WithLogger(&l), prisma.WithCache(ch), prisma.WithMetered(meter)),
		EngineRepository:      prisma.NewEngineRepository(pool, prisma.WithLogger(&l), prisma.WithCache(ch), prisma.WithMetered(meter)),
		EntitlementRepository: entitlementRepo,
		Seed:                  cf.Seed,
	}, nil
}

func GetServerConfigFromConfigfile(dc *database.Config, cf *server.ServerConfigFile, version string) (cleanup func() error, res *server.ServerConfig, err error) {
	l := logger.NewStdErr(&cf.Logger, "server")

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
	cleanup1 := func() error {
		return nil
	}

	var ing ingestor.Ingestor

	if cf.MessageQueue.Enabled {
		cleanup1, mq = rabbitmq.New(
			rabbitmq.WithURL(cf.MessageQueue.RabbitMQ.URL),
			rabbitmq.WithLogger(&l),
		)

		ing, err = ingestor.NewIngestor(
			ingestor.WithEventRepository(dc.EngineRepository.Event()),
			ingestor.WithStreamEventsRepository(dc.EngineRepository.StreamEvent()),
			ingestor.WithLogRepository(dc.EngineRepository.Log()),
			ingestor.WithMessageQueue(mq),
			ingestor.WithEntitlementsRepository(dc.EntitlementRepository),
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
		ConfigFile: cf.Auth,
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

	cleanup = func() error {
		log.Printf("cleaning up server config")
		if err := cleanup1(); err != nil {
			return fmt.Errorf("error cleaning up rabbitmq: %w", err)
		}
		return nil
	}

	return cleanup, &server.ServerConfig{
		Alerter:                alerter,
		Analytics:              analyticsEmitter,
		FePosthog:              feAnalyticsConfig,
		Pylon:                  &pylon,
		Runtime:                cf.Runtime,
		Auth:                   auth,
		Encryption:             encryptionSvc,
		Config:                 dc,
		MessageQueue:           mq,
		Services:               cf.Services,
		Logger:                 &l,
		TLSConfig:              tls,
		SessionStore:           ss,
		Validator:              validator.NewDefaultValidator(),
		Ingestor:               ing,
		OpenTelemetry:          cf.OpenTelemetry,
		Email:                  emailSvc,
		TenantAlerter:          alerting.New(dc.EngineRepository, encryptionSvc, cf.Runtime.ServerURL, emailSvc),
		AdditionalOAuthConfigs: additionalOAuthConfigs,
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
