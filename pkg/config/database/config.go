package database

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"

	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	v2 "github.com/hatchet-dev/hatchet/pkg/repository/v2"
)

type ConfigFile struct {
	PostgresHost     string `mapstructure:"host" json:"host,omitempty" default:"127.0.0.1"`
	PostgresPort     int    `mapstructure:"port" json:"port,omitempty" default:"5431"`
	PostgresUsername string `mapstructure:"username" json:"username,omitempty" default:"hatchet"`
	PostgresPassword string `mapstructure:"password" json:"password,omitempty" default:"hatchet"`
	PostgresDbName   string `mapstructure:"dbName" json:"dbName,omitempty" default:"hatchet"`
	PostgresSSLMode  string `mapstructure:"sslMode" json:"sslMode,omitempty" default:"disable"`

	MaxConns int `mapstructure:"maxConns" json:"maxConns,omitempty" default:"50"`
	MinConns int `mapstructure:"minConns" json:"minConns,omitempty" default:"10"`

	MaxQueueConns int `mapstructure:"maxQueueConns" json:"maxQueueConns,omitempty" default:"50"`
	MinQueueConns int `mapstructure:"minQueueConns" json:"minQueueConns,omitempty" default:"10"`

	Seed SeedConfigFile `mapstructure:"seed" json:"seed,omitempty"`

	Logger shared.LoggerConfigFile `mapstructure:"logger" json:"logger,omitempty"`

	LogQueries bool `mapstructure:"logQueries" json:"logQueries,omitempty" default:"false"`

	CacheDuration time.Duration `mapstructure:"cacheDuration" json:"cacheDuration,omitempty" default:"60s"`
}

type SeedConfigFile struct {
	AdminEmail    string `mapstructure:"adminEmail" json:"adminEmail,omitempty" default:"admin@example.com"`
	AdminPassword string `mapstructure:"adminPassword" json:"adminPassword,omitempty" default:"Admin123!!"`
	AdminName     string `mapstructure:"adminName" json:"adminName,omitempty" default:"Admin"`

	DefaultTenantName string `mapstructure:"defaultTenantName" json:"defaultTenantName,omitempty" default:"Default"`
	DefaultTenantSlug string `mapstructure:"defaultTenantSlug" json:"defaultTenantSlug,omitempty" default:"default"`
	DefaultTenantID   string `mapstructure:"defaultTenantId" json:"defaultTenantId,omitempty" default:"707d0855-80ab-4e1f-a156-f1c4546cbf52"`

	IsDevelopment bool `mapstructure:"isDevelopment" json:"isDevelopment,omitempty" default:"false"`
}

type Layer struct {
	Disconnect func() error

	Pool *pgxpool.Pool

	EssentialPool *pgxpool.Pool

	QueuePool *pgxpool.Pool

	APIRepository repository.APIRepository

	OLAPRepository repository.OLAPEventRepository

	EngineRepository repository.EngineRepository

	V2 v2.Repository

	EntitlementRepository repository.EntitlementsRepository

	Seed SeedConfigFile
}

func BindAllEnv(v *viper.Viper) {
	_ = v.BindEnv("host", "DATABASE_POSTGRES_HOST")
	_ = v.BindEnv("port", "DATABASE_POSTGRES_PORT")
	_ = v.BindEnv("username", "DATABASE_POSTGRES_USERNAME")
	_ = v.BindEnv("password", "DATABASE_POSTGRES_PASSWORD")
	_ = v.BindEnv("dbName", "DATABASE_POSTGRES_DB_NAME")
	_ = v.BindEnv("sslMode", "DATABASE_POSTGRES_SSL_MODE")
	_ = v.BindEnv("logQueries", "DATABASE_LOG_QUERIES")
	_ = v.BindEnv("maxConns", "DATABASE_MAX_CONNS")
	_ = v.BindEnv("minConns", "DATABASE_MIN_CONNS")
	_ = v.BindEnv("maxQueueConns", "DATABASE_MAX_QUEUE_CONNS")
	_ = v.BindEnv("minQueueConns", "DATABASE_MIN_QUEUE_CONNS")

	_ = v.BindEnv("cacheDuration", "CACHE_DURATION")

	_ = v.BindEnv("seed.adminEmail", "ADMIN_EMAIL")
	_ = v.BindEnv("seed.adminPassword", "ADMIN_PASSWORD")
	_ = v.BindEnv("seed.adminName", "ADMIN_NAME")
	_ = v.BindEnv("seed.defaultTenantName", "DEFAULT_TENANT_NAME")
	_ = v.BindEnv("seed.defaultTenantSlug", "DEFAULT_TENANT_SLUG")
	_ = v.BindEnv("seed.isDevelopment", "SEED_DEVELOPMENT")

	_ = v.BindEnv("logger.level", "DATABASE_LOGGER_LEVEL")
	_ = v.BindEnv("logger.format", "DATABASE_LOGGER_FORMAT")
}
