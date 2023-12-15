package database

import (
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/spf13/viper"
)

type ConfigFile struct {
	PostgresHost     string `mapstructure:"host" json:"host,omitempty" default:"127.0.0.1"`
	PostgresPort     int    `mapstructure:"port" json:"port,omitempty" default:"5433"`
	PostgresUsername string `mapstructure:"username" json:"username,omitempty" default:"hatchet"`
	PostgresPassword string `mapstructure:"password" json:"password,omitempty" default:"hatchet"`
	PostgresDbName   string `mapstructure:"dbName" json:"dbName,omitempty" default:"hatchet"`
	PostgresSSLMode  string `mapstructure:"sslMode" json:"sslMode,omitempty" default:"disable"`
}

type Config struct {
	Disconnect func() error

	Repository repository.Repository
}

func BindAllEnv(v *viper.Viper) {
	v.BindEnv("host", "DATABASE_POSTGRES_HOST")
	v.BindEnv("port", "DATABASE_POSTGRES_PORT")
	v.BindEnv("username", "DATABASE_POSTGRES_USERNAME")
	v.BindEnv("password", "DATABASE_POSTGRES_PASSWORD")
	v.BindEnv("dbName", "DATABASE_POSTGRES_DB_NAME")
	v.BindEnv("sslMode", "DATABASE_POSTGRES_SSL_MODE")
}
