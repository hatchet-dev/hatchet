package hatchetembed

import (
	"fmt"
	"strings"

	"github.com/hatchet-dev/hatchet/pkg/version"
)

// Config holds the resolved configuration for an embedded Hatchet instance. It is populated by the
// Option functions passed to Start.
type Config struct {
	postgresURL      string
	rabbitMQURL      string
	adminEmail       string
	adminPassword    string
	version          string
	logLevel         string
	apiPort          int
	grpcPort         int
	dashboardPort    int
	runMigrations    bool
	dashboardEnabled bool
}

// Option configures an embedded Hatchet instance.
type Option func(*Config)

// defaultEngineVersion is the Hatchet engine version reported by the embedded engine. It comes from
// the canonical pkg/version constant (the single source of truth kept in sync with the release
// tag), so the SDK client sees a real semver and doesn't fall back to legacy worker registration.
const defaultEngineVersion = version.Version

func defaultConfig() *Config {
	return &Config{
		runMigrations:    true,
		apiPort:          8080,
		grpcPort:         7070,
		dashboardPort:    8082,
		dashboardEnabled: true,
		version:          defaultEngineVersion,
		logLevel:         "warn",
	}
}

// WithPostgres sets the Postgres connection string Hatchet runs against. This is required. Unless
// WithRabbitMQ is also provided, this same database backs the message queue.
func WithPostgres(url string) Option {
	return func(c *Config) { c.postgresURL = url }
}

// WithRabbitMQ switches the message queue to RabbitMQ at the given AMQP URL. When omitted, the
// message queue is backed by the Postgres database supplied to WithPostgres.
func WithRabbitMQ(url string) Option {
	return func(c *Config) { c.rabbitMQURL = url }
}

// WithoutMigrations disables running database migrations on startup. Use this when migrations are
// managed externally.
func WithoutMigrations() Option {
	return func(c *Config) { c.runMigrations = false }
}

// WithAPIPort sets the port the REST API listens on. Defaults to 8080.
func WithAPIPort(port int) Option {
	return func(c *Config) { c.apiPort = port }
}

// WithGRPCPort sets the port the gRPC engine listens on. Defaults to 7070.
func WithGRPCPort(port int) Option {
	return func(c *Config) { c.grpcPort = port }
}

// WithDashboardPort sets the port the bundled dashboard is served on. Defaults to 8082.
func WithDashboardPort(port int) Option {
	return func(c *Config) { c.dashboardPort = port }
}

// WithoutDashboard disables serving the bundled dashboard, exposing only the API and engine.
func WithoutDashboard() Option {
	return func(c *Config) { c.dashboardEnabled = false }
}

// WithAdminUser overrides the seeded admin user email/password. In no-auth mode all dashboard/REST
// requests resolve to this user.
func WithAdminUser(email, password string) Option {
	return func(c *Config) {
		c.adminEmail = email
		c.adminPassword = password
	}
}

// WithVersion sets the version string reported by the engine and API.
func WithVersion(version string) Option {
	return func(c *Config) { c.version = version }
}

// WithLogLevel sets the log level for the returned client.
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

// usePostgresMQ reports whether the message queue should be backed by Postgres (the default when no
// RabbitMQ URL is provided).
func (c *Config) usePostgresMQ() bool {
	return strings.TrimSpace(c.rabbitMQURL) == ""
}
