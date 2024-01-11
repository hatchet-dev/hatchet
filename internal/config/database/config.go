package database

import (
	"github.com/hatchet-dev/hatchet/internal/config/shared"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/spf13/viper"
)

type ConfigFile struct {
	PostgresHost     string `mapstructure:"host" json:"host,omitempty" default:"127.0.0.1"`
	PostgresPort     int    `mapstructure:"port" json:"port,omitempty" default:"5431"`
	PostgresUsername string `mapstructure:"username" json:"username,omitempty" default:"hatchet"`
	PostgresPassword string `mapstructure:"password" json:"password,omitempty" default:"hatchet"`
	PostgresDbName   string `mapstructure:"dbName" json:"dbName,omitempty" default:"hatchet"`
	PostgresSSLMode  string `mapstructure:"sslMode" json:"sslMode,omitempty" default:"disable"`

	Seed SeedConfigFile `mapstructure:"seed" json:"seed,omitempty"`

	Logger shared.LoggerConfigFile `mapstructure:"logger" json:"logger,omitempty"`
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

type Config struct {
	Disconnect func() error

	Repository repository.Repository

	Seed SeedConfigFile
}

func BindAllEnv(v *viper.Viper) {
	v.BindEnv("host", "DATABASE_POSTGRES_HOST")
	v.BindEnv("port", "DATABASE_POSTGRES_PORT")
	v.BindEnv("username", "DATABASE_POSTGRES_USERNAME")
	v.BindEnv("password", "DATABASE_POSTGRES_PASSWORD")
	v.BindEnv("dbName", "DATABASE_POSTGRES_DB_NAME")
	v.BindEnv("sslMode", "DATABASE_POSTGRES_SSL_MODE")

	v.BindEnv("seed.adminEmail", "ADMIN_EMAIL")
	v.BindEnv("seed.adminPassword", "ADMIN_PASSWORD")
	v.BindEnv("seed.adminName", "ADMIN_NAME")
	v.BindEnv("seed.defaultTenantName", "DEFAULT_TENANT_NAME")
	v.BindEnv("seed.defaultTenantSlug", "DEFAULT_TENANT_SLUG")
	v.BindEnv("seed.isDevelopment", "SEED_DEVELOPMENT")

	v.BindEnv("logger.level", "DATABASE_LOGGER_LEVEL")
	v.BindEnv("logger.format", "DATABASE_LOGGER_FORMAT")
}
