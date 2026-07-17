package embed

import (
	"fmt"
	"strings"
)

type Config struct {
	postgresURL      string
	rabbitMQURL      *string
	adminEmail       *string
	adminPassword    *string
	version          *string
	logLevel         *string
	masterKeyset     *[]byte
	privateJWTKeyset *[]byte
	publicJWTKeyset  *[]byte
	apiPort          *int
	grpcPort         *int
	runMigrations    *bool
	startAPI         *bool
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{
		runMigrations: new(true),
		startAPI:      new(true),
		logLevel:      new("warn"),
	}
}

func WithPostgres(url string) Option {
	return func(c *Config) { c.postgresURL = url }
}

func WithKeysets(master, privateJWT, publicJWT []byte) Option {
	return func(c *Config) {
		c.masterKeyset = &master
		c.privateJWTKeyset = &privateJWT
		c.publicJWTKeyset = &publicJWT
	}
}

func WithRabbitMQ(url string) Option {
	return func(c *Config) { c.rabbitMQURL = &url }
}

func WithoutMigrations() Option {
	return func(c *Config) { c.runMigrations = new(false) }
}

func WithoutAPI() Option {
	return func(c *Config) { c.startAPI = new(false) }
}

func WithAPIPort(port int) Option {
	return func(c *Config) { c.apiPort = &port }
}

func WithGRPCPort(port int) Option {
	return func(c *Config) { c.grpcPort = &port }
}

func WithAdminUser(email, password string) Option {
	return func(c *Config) {
		c.adminEmail = &email
		c.adminPassword = &password
	}
}

func WithLogLevel(level string) Option {
	return func(c *Config) { c.logLevel = &level }
}

func (c *Config) validate() error {
	if strings.TrimSpace(c.postgresURL) == "" {
		return fmt.Errorf("a Postgres connection string is required: use WithPostgres(url)")
	}

	if c.apiPort != nil && c.grpcPort != nil && *c.apiPort == *c.grpcPort {
		return fmt.Errorf("api port and grpc port must differ (both %d)", *c.apiPort)
	}

	if c.rabbitMQURL != nil && *c.rabbitMQURL == "" {
		return fmt.Errorf("invalid RabbitMQ URL provided")
	}

	return nil
}

func (c *Config) usePostgresMQ() bool {
	if c.rabbitMQURL != nil && *c.rabbitMQURL != "" {
		return false
	}

	return true
}
