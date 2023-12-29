// Adapted from: https://github.com/hatchet-dev/hatchet/blob/3c2c13168afa1af68d4baaf5ed02c9d49c5f0323/internal/config/loader/loader.go

package loader

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/creasty/defaults"
	"github.com/hatchet-dev/hatchet/internal/auth/cookie"
	"github.com/hatchet-dev/hatchet/internal/config/client"
	"github.com/hatchet-dev/hatchet/internal/config/database"
	"github.com/hatchet-dev/hatchet/internal/config/server"
	"github.com/hatchet-dev/hatchet/internal/config/shared"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/services/ingestor"
	"github.com/hatchet-dev/hatchet/internal/taskqueue/rabbitmq"
	"github.com/hatchet-dev/hatchet/internal/validator"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

// LoadDatabaseConfigFile loads the database config file via viper
func LoadDatabaseConfigFile(files ...[]byte) (*database.ConfigFile, error) {
	configFile := &database.ConfigFile{}
	f := database.BindAllEnv

	_, err := LoadConfigFromViper(f, configFile, files...)

	return configFile, err
}

// LoadServerConfigFile loads the server config file via viper
func LoadServerConfigFile(files ...[]byte) (*server.ServerConfigFile, error) {
	configFile := &server.ServerConfigFile{}
	f := server.BindAllEnv

	_, err := LoadConfigFromViper(f, configFile, files...)

	return configFile, err
}

// LoadClientConfigFile loads the worker config file via viper
func LoadClientConfigFile(files ...[]byte) (*client.ClientConfigFile, error) {
	configFile := &client.ClientConfigFile{}
	f := client.BindAllEnv

	_, err := LoadConfigFromViper(f, configFile, files...)

	return configFile, err
}

func LoadConfigFromViper(bindFunc func(v *viper.Viper), configFile interface{}, files ...[]byte) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	bindFunc(v)

	for _, f := range files {
		err := v.MergeConfig(bytes.NewBuffer(f))

		if err != nil {
			return nil, fmt.Errorf("could not load viper config: %w", err)
		}
	}

	defaults.Set(configFile)

	err := v.Unmarshal(configFile)

	if err != nil {
		return nil, fmt.Errorf("could not unmarshal viper config: %w", err)
	}

	return v, nil
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
	configFileBytes, err := getConfigBytes(sharedFilePath)

	if err != nil {
		return nil, err
	}

	cf, err := LoadDatabaseConfigFile(configFileBytes...)

	if err != nil {
		return nil, err
	}

	return GetDatabaseConfigFromConfigFile(cf)
}

// LoadServerConfig loads the server configuration
func (c *ConfigLoader) LoadServerConfig() (res *server.ServerConfig, err error) {
	sharedFilePath := filepath.Join(c.directory, "server.yaml")
	configFileBytes, err := getConfigBytes(sharedFilePath)

	if err != nil {
		return nil, err
	}

	dc, err := c.LoadDatabaseConfig()

	if err != nil {
		return nil, err
	}

	cf, err := LoadServerConfigFile(configFileBytes...)

	return GetServerConfigFromConfigfile(dc, cf)
}

// LoadClientConfig loads the client configuration
func (c *ConfigLoader) LoadClientConfig() (res *client.ClientConfig, err error) {
	sharedFilePath := filepath.Join(c.directory, "client.yaml")
	configFileBytes, err := getConfigBytes(sharedFilePath)

	if err != nil {
		return nil, err
	}

	cf, err := LoadClientConfigFile(configFileBytes...)

	if err != nil {
		return nil, err
	}

	return GetClientConfigFromConfigFile(cf)
}

func getConfigBytes(configFilePath string) ([][]byte, error) {
	configFileBytes := make([][]byte, 0)

	if fileExists(configFilePath) {
		fileBytes, err := ioutil.ReadFile(configFilePath) // #nosec G304 -- config files are meant to be read from user-supplied directory

		if err != nil {
			return nil, fmt.Errorf("could not read config file at path %s: %w", configFilePath, err)
		}

		configFileBytes = append(configFileBytes, fileBytes)
	}

	return configFileBytes, nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil && os.IsNotExist(err) {
		return false
	} else if err != nil {
		return false
	}

	return !info.IsDir()
}

func GetDatabaseConfigFromConfigFile(cf *database.ConfigFile) (res *database.Config, err error) {
	databaseUrl := fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		cf.PostgresUsername,
		cf.PostgresPassword,
		cf.PostgresHost,
		cf.PostgresPort,
		cf.PostgresDbName,
		cf.PostgresSSLMode,
	)

	os.Setenv("DATABASE_URL", databaseUrl)

	client := db.NewClient(
	// db.WithDatasourceURL(databaseUrl),
	)

	if err := client.Prisma.Connect(); err != nil {
		return nil, err
	}

	pool, err := pgxpool.New(context.Background(), databaseUrl)

	if err != nil {
		return nil, fmt.Errorf("could not connect to database: %w", err)
	}

	return &database.Config{
		Disconnect: client.Prisma.Disconnect,
		Repository: prisma.NewPrismaRepository(client, pool),
		Seed:       cf.Seed,
	}, nil
}

func GetServerConfigFromConfigfile(dc *database.Config, cf *server.ServerConfigFile) (res *server.ServerConfig, err error) {
	l := zerolog.New(os.Stderr)

	tls, err := loadServerTLSConfig(&cf.TLS)

	if err != nil {
		return nil, fmt.Errorf("could not load TLS config: %w", err)
	}

	runtime := server.ServerRuntimeConfig{
		ServerURL: cf.Runtime.ServerURL,
		Port:      cf.Runtime.Port,
	}

	ss, err := cookie.NewUserSessionStore(
		cookie.WithSessionRepository(dc.Repository.UserSession()),
		cookie.WithCookieAllowInsecure(cf.Auth.Cookie.Insecure),
		cookie.WithCookieDomain(cf.Auth.Cookie.Domain),
		cookie.WithCookieName(cf.Auth.Cookie.Name),
		cookie.WithCookieSecrets(getStrArr(cf.Auth.Cookie.Secrets)...),
	)

	if err != nil {
		return nil, fmt.Errorf("could not create session store: %w", err)
	}

	tq := rabbitmq.New(context.Background(), rabbitmq.WithURL(cf.TaskQueue.RabbitMQ.URL))

	ingestor, err := ingestor.NewIngestor(
		ingestor.WithEventRepository(dc.Repository.Event()),
		ingestor.WithTaskQueue(tq),
	)

	if err != nil {
		return nil, fmt.Errorf("could not create ingestor: %w", err)
	}

	return &server.ServerConfig{
		Runtime:      runtime,
		Auth:         cf.Auth,
		Config:       dc,
		TaskQueue:    rabbitmq.New(context.Background(), rabbitmq.WithURL(cf.TaskQueue.RabbitMQ.URL)),
		Services:     cf.Services,
		Logger:       &l,
		TLSConfig:    tls,
		SessionStore: ss,
		Validator:    validator.NewDefaultValidator(),
		Ingestor:     ingestor,
	}, nil
}

func GetClientConfigFromConfigFile(cf *client.ClientConfigFile) (res *client.ClientConfig, err error) {
	tlsConf, err := loadClientTLSConfig(&cf.TLS)

	if err != nil {
		return nil, fmt.Errorf("could not load TLS config: %w", err)
	}

	return &client.ClientConfig{
		TenantId:  cf.TenantId,
		TLSConfig: tlsConf,
	}, nil
}

func loadClientTLSConfig(tlsConfig *client.ClientTLSConfigFile) (*tls.Config, error) {
	res, ca, err := LoadBaseTLSConfig(&tlsConfig.Base)

	if err != nil {
		return nil, err
	}

	res.ServerName = tlsConfig.TLSServerName
	res.RootCAs = ca

	return res, nil
}

func loadServerTLSConfig(tlsConfig *shared.TLSConfigFile) (*tls.Config, error) {
	res, ca, err := LoadBaseTLSConfig(tlsConfig)

	if err != nil {
		return nil, err
	}

	res.ClientAuth = tls.RequireAndVerifyClientCert
	res.ClientCAs = ca

	return res, nil
}

func LoadBaseTLSConfig(tlsConfig *shared.TLSConfigFile) (*tls.Config, *x509.CertPool, error) {
	var x509Cert tls.Certificate
	var err error

	if tlsConfig.TLSCert != "" && tlsConfig.TLSKey != "" {
		x509Cert, err = tls.X509KeyPair([]byte(tlsConfig.TLSCert), []byte(tlsConfig.TLSKey))
	} else if tlsConfig.TLSCertFile != "" && tlsConfig.TLSKeyFile != "" {
		x509Cert, err = tls.LoadX509KeyPair(tlsConfig.TLSCertFile, tlsConfig.TLSKeyFile)
	} else {
		return nil, nil, fmt.Errorf("no cert or key provided")
	}

	var caBytes []byte

	if tlsConfig.TLSRootCA != "" {
		caBytes = []byte(tlsConfig.TLSRootCA)
	} else if tlsConfig.TLSRootCAFile != "" {
		caBytes, err = os.ReadFile(tlsConfig.TLSRootCAFile)
	} else {
		return nil, nil, fmt.Errorf("no root CA provided")
	}

	ca := x509.NewCertPool()

	if ok := ca.AppendCertsFromPEM(caBytes); !ok {
		return nil, nil, fmt.Errorf("could not append root CA to cert pool: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{x509Cert},
	}, ca, nil
}

func getStrArr(v string) []string {
	return strings.Split(v, " ")
}
