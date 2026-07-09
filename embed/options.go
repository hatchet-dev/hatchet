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
	dashboardDir     string
	apiPort          int
	grpcPort         int
	dashboardPort    int
	runMigrations    bool
	dashboardEnabled bool
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{
		runMigrations:    true,
		apiPort:          8080,
		grpcPort:         7070,
		dashboardPort:    8082,
		dashboardEnabled: true,
		version:          embedVersion,
		logLevel:         "warn",
	}
}

func WithPostgres(url string) Option {
	return func(c *Config) { c.postgresURL = url }
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

func WithDashboardPort(port int) Option {
	return func(c *Config) { c.dashboardPort = port }
}

func WithoutDashboard() Option {
	return func(c *Config) { c.dashboardEnabled = false }
}

func WithDashboardDir(dir string) Option {
	return func(c *Config) { c.dashboardDir = dir }
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

	if c.apiPort == c.grpcPort {
		return fmt.Errorf("api port and grpc port must differ (both %d)", c.apiPort)
	}

	if c.dashboardEnabled && (c.dashboardPort == c.apiPort || c.dashboardPort == c.grpcPort) {
		return fmt.Errorf("dashboard port %d must differ from api and grpc ports", c.dashboardPort)
	}

	return nil
}

func (c *Config) usePostgresMQ() bool {
	return strings.TrimSpace(c.rabbitMQURL) == ""
}
