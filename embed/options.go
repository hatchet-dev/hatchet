package embed

import (
	"fmt"
	"strings"
)

type Config struct {
	postgresURL      string
	rabbitMQURL      string
	adminEmail       string
	adminPassword    string
	version          string
	logLevel         string
	masterKeyset     []byte
	privateJWTKeyset []byte
	publicJWTKeyset  []byte
	apiPort          int
	grpcPort         int
	runMigrations    bool
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{
		runMigrations: true,
		apiPort:       8080,
		grpcPort:      7070,
		logLevel:      "warn",
	}
}

func WithPostgres(url string) Option {
	return func(c *Config) { c.postgresURL = url }
}

func WithKeysets(master, privateJWT, publicJWT []byte) Option {
	return func(c *Config) {
		c.masterKeyset = master
		c.privateJWTKeyset = privateJWT
		c.publicJWTKeyset = publicJWT
	}
}

func WithRabbitMQ(url string) Option {
	return func(c *Config) { c.rabbitMQURL = url }
}

func WithoutMigrations() Option {
	return func(c *Config) { c.runMigrations = false }
}

func WithAPIPort(port int) Option {
	return func(c *Config) { c.apiPort = port }
}

func WithGRPCPort(port int) Option {
	return func(c *Config) { c.grpcPort = port }
}

func WithAdminUser(email, password string) Option {
	return func(c *Config) {
		c.adminEmail = email
		c.adminPassword = password
	}
}

func WithVersion(version string) Option {
	return func(c *Config) { c.version = version }
}

func WithLogLevel(level string) Option {
	return func(c *Config) { c.logLevel = level }
}

func (c *Config) validate() error {
	if strings.TrimSpace(c.postgresURL) == "" {
		return fmt.Errorf("a Postgres connection string is required: use WithPostgres(url)")
	}

	if len(c.masterKeyset) == 0 || len(c.privateJWTKeyset) == 0 || len(c.publicJWTKeyset) == 0 {
		return fmt.Errorf("keysets are required: generate them with `hatchet-admin keyset create-local-keys` and pass them via WithKeysets. All engines sharing a database must use the same keysets")
	}

	if c.apiPort == c.grpcPort {
		return fmt.Errorf("api port and grpc port must differ (both %d)", c.apiPort)
	}

	return nil
}

func (c *Config) usePostgresMQ() bool {
	return strings.TrimSpace(c.rabbitMQURL) == ""
}
