package database

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"

	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

type ConfigFile struct {
	PostgresHost     string `mapstructure:"host" json:"host,omitempty" default:"127.0.0.1"`
	PostgresPort     int    `mapstructure:"port" json:"port,omitempty" default:"5431"`
	PostgresUsername string `mapstructure:"username" json:"username,omitempty" default:"hatchet"`
	PostgresPassword string `mapstructure:"password" json:"password,omitempty" default:"hatchet"`
	PostgresDbName   string `mapstructure:"dbName" json:"dbName,omitempty" default:"hatchet"`
	PostgresSSLMode  string `mapstructure:"sslMode" json:"sslMode,omitempty" default:"disable"`

	ReadReplicaEnabled     bool   `mapstructure:"readReplicaEnabled" json:"readReplicaEnabled,omitempty" default:"false"`
	ReadReplicaDatabaseURL string `mapstructure:"readReplicaDatabaseUrl" json:"readReplicaDatabaseUrl,omitempty" default:""`
	ReadReplicaMaxConns    int    `mapstructure:"readReplicaMaxConns" json:"readReplicaMaxConns,omitempty" default:"50"`
	ReadReplicaMinConns    int    `mapstructure:"readReplicaMinConns" json:"readReplicaMinConns,omitempty" default:"10"`

	MaxConns int `mapstructure:"maxConns" json:"maxConns,omitempty" default:"50"`
	MinConns int `mapstructure:"minConns" json:"minConns,omitempty" default:"1"`

	// PgBouncerURL is an optional connection string for pgbouncer. When set, most queries are routed
	// through pgbouncer, other than DDL-modifying statements which use the separate direct connection pool.
	PgBouncerURL string `mapstructure:"pgbouncerUrl" json:"pgbouncerUrl,omitempty" default:""`

	DDLPoolMaxConns int `mapstructure:"ddlPoolMaxConns" json:"ddlPoolMaxConns,omitempty" default:"5"`
	DDLPoolMinConns int `mapstructure:"ddlPoolMinConns" json:"ddlPoolMinConns,omitempty" default:"1"`

	MaxConnLifetime time.Duration `mapstructure:"maxConnLifetime" json:"maxConnLifetime,omitempty" default:"15m"`
	MaxConnIdleTime time.Duration `mapstructure:"maxConnIdleTime" json:"maxConnIdleTime,omitempty" default:"1m"`

	// DialTimeout bounds the TCP connect of every new connection. During a failover the old
	// primary's address can drop packets, and a dial to it holds a pool slot until this fires.
	DialTimeout time.Duration `mapstructure:"dialTimeout" json:"dialTimeout,omitempty" default:"3s"`

	// ConnectTimeout bounds the connect (dial, TLS, auth) and, separately, the connection setup
	// queries. It overrides connect_timeout in the database URL. Keep it above the multi-second
	// stalls a live but overloaded server can show during startup, or those turn into errors.
	ConnectTimeout time.Duration `mapstructure:"connectTimeout" json:"connectTimeout,omitempty" default:"15s"`

	// FairpoolTenantMaxPercent is the share of the pool one tenant may hold.
	// 100 (the default) installs no cap. Values from 1 to 99 cap each tenant at that percent of MaxConns.
	FairpoolTenantMaxPercent int `mapstructure:"fairpoolTenantMaxPercent" json:"fairpoolTenantMaxPercent,omitempty" default:"100"`

	// FairpoolSharedMaxPercent is the share of the pool the shared bucket may hold. The shared bucket
	// carries engine work that is not tied to a tenant, such as the message queue, heartbeats and auth lookups.
	// 100 (the default) installs no cap. Values from 1 to 99 cap it at that percent of MaxConns.
	FairpoolSharedMaxPercent int `mapstructure:"fairpoolSharedMaxPercent" json:"fairpoolSharedMaxPercent,omitempty" default:"100"`

	// FairpoolTenantMaxWait is how long an acquire waits for a slot before it fails.
	// The wait is not applied to the query.
	FairpoolTenantMaxWait time.Duration `mapstructure:"fairpoolTenantMaxWait" json:"fairpoolTenantMaxWait,omitempty" default:"5s"`

	// ApplicationNamePrefix is prepended to the pgx application_name as "<prefix>:<otel service name>"
	// so connections in pg_stat_activity can be attributed to a deployment. In Kubernetes this is
	// populated with the pod namespace via the downward API (K8S_POD_NAMESPACE).
	ApplicationNamePrefix string `mapstructure:"applicationNamePrefix" json:"applicationNamePrefix,omitempty" default:""`

	Seed SeedConfigFile `mapstructure:"seed" json:"seed,omitempty"`

	Logger shared.LoggerConfigFile `mapstructure:"logger" json:"logger,omitempty"`

	LogQueries bool `mapstructure:"logQueries" json:"logQueries,omitempty" default:"false"`

	CacheDuration time.Duration `mapstructure:"cacheDuration" json:"cacheDuration,omitempty" default:"5s"`

	// EnforceUTCTimezone enforces that the database instance timezone is set to UTC.
	// If enabled and the database timezone is not UTC, the server will panic on startup.
	// To disable this check, set DATABASE_ENFORCE_UTC_TIMEZONE=false
	EnforceUTCTimezone bool `mapstructure:"enforceUtcTimezone" json:"enforceUtcTimezone,omitempty" default:"true"`

	// OLAP points the OLAP repository (run and event history, metrics, audit logs) at a separate
	// database. When DatabaseURL is unset the OLAP tables live in the main database.
	OLAP OLAPConfigFile `mapstructure:"olap" json:"olap,omitempty"`
}

// OLAPConfigFile configures the pool for a separate OLAP database. Every field other than
// DatabaseURL is an override of the matching main database setting and falls back to it when
// zero, so a deployment that only sets OLAP_DATABASE_URL gets the main pool's configuration.
type OLAPConfigFile struct {
	DatabaseURL string `mapstructure:"databaseUrl" json:"databaseUrl,omitempty" default:""`

	MaxConns int `mapstructure:"maxConns" json:"maxConns,omitempty" default:"0"`
	MinConns int `mapstructure:"minConns" json:"minConns,omitempty" default:"0"`

	MaxConnLifetime time.Duration `mapstructure:"maxConnLifetime" json:"maxConnLifetime,omitempty" default:"0"`
	MaxConnIdleTime time.Duration `mapstructure:"maxConnIdleTime" json:"maxConnIdleTime,omitempty" default:"0"`

	DialTimeout    time.Duration `mapstructure:"dialTimeout" json:"dialTimeout,omitempty" default:"0"`
	ConnectTimeout time.Duration `mapstructure:"connectTimeout" json:"connectTimeout,omitempty" default:"0"`

	FairpoolTenantMaxPercent int           `mapstructure:"fairpoolTenantMaxPercent" json:"fairpoolTenantMaxPercent,omitempty" default:"0"`
	FairpoolSharedMaxPercent int           `mapstructure:"fairpoolSharedMaxPercent" json:"fairpoolSharedMaxPercent,omitempty" default:"0"`
	FairpoolTenantMaxWait    time.Duration `mapstructure:"fairpoolTenantMaxWait" json:"fairpoolTenantMaxWait,omitempty" default:"0"`
}

type SeedConfigFile struct {
	AdminEmail    string `mapstructure:"adminEmail" json:"adminEmail,omitempty" default:"admin@example.com"`
	AdminPassword string `mapstructure:"adminPassword" json:"adminPassword,omitempty" default:"Admin123!!"`
	AdminName     string `mapstructure:"adminName" json:"adminName,omitempty" default:"Admin"`

	DefaultTenantName string `mapstructure:"defaultTenantName" json:"defaultTenantName,omitempty" default:"Default"`
	DefaultTenantSlug string `mapstructure:"defaultTenantSlug" json:"defaultTenantSlug,omitempty" default:"default"`
	DefaultTenantID   string `mapstructure:"defaultTenantId" json:"defaultTenantId,omitempty" default:"707d0855-80ab-4e1f-a156-f1c4546cbf52"`
}

type Layer struct {
	Disconnect func() error

	Pool *pgxpool.Pool

	// DirectDatabaseURL is the resolved direct (non-pgbouncer) connection URL.
	// LISTEN/NOTIFY does not survive transaction pooling, so consumers that
	// need LISTEN (e.g. the postgres PubSub) must build their pools from this.
	DirectDatabaseURL string

	// DialTimeout and ConnectTimeout bound new connections on the layer's pools. Pools built
	// from DirectDatabaseURL apply them too.
	DialTimeout    time.Duration
	ConnectTimeout time.Duration

	ReadReplicaPool *pgxpool.Pool

	// DDLPool is meant for DDL-modifying operations like DETACH PARTITION CONCURRENTLY, which
	// are critical and cannot run in an explicit transaction (and therefor cannot go through pgbouncer when it's configured)
	DDLPool *pgxpool.Pool

	V1 v1.Repository

	Seed SeedConfigFile

	DefaultTenantEntitlements v1.TenantEntitlements

	// Logger is the configured database logger. Consumers should read it via
	// GetLogger, which falls back to a disabled logger when the layer was
	// constructed without one.
	Logger *zerolog.Logger
}

// GetLogger returns the layer's configured logger, or a disabled logger if the
// layer was constructed without one. Library code writing diagnostics should
// use this rather than printing to stdout/stderr directly, so that embedders
// keep control over the output.
func (l *Layer) GetLogger() *zerolog.Logger {
	if l == nil || l.Logger == nil {
		nop := zerolog.Nop()

		return &nop
	}

	return l.Logger
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
	_ = v.BindEnv("maxConnLifetime", "DATABASE_MAX_CONN_LIFETIME")
	_ = v.BindEnv("maxConnIdleTime", "DATABASE_MAX_CONN_IDLE_TIME")
	_ = v.BindEnv("dialTimeout", "DATABASE_DIAL_TIMEOUT")
	_ = v.BindEnv("connectTimeout", "DATABASE_CONNECT_TIMEOUT")
	_ = v.BindEnv("fairpoolTenantMaxPercent", "DATABASE_FAIRPOOL_TENANT_MAX_PERCENT")
	_ = v.BindEnv("fairpoolSharedMaxPercent", "DATABASE_FAIRPOOL_SHARED_MAX_PERCENT")
	_ = v.BindEnv("fairpoolTenantMaxWait", "DATABASE_FAIRPOOL_TENANT_MAX_WAIT")
	_ = v.BindEnv("applicationNamePrefix", "K8S_POD_NAMESPACE")

	_ = v.BindEnv("pgbouncerUrl", "DATABASE_PGBOUNCER_URL")
	_ = v.BindEnv("ddlPoolMaxConns", "DATABASE_DDL_POOL_MAX_CONNS")
	_ = v.BindEnv("ddlPoolMinConns", "DATABASE_DDL_POOL_MIN_CONNS")

	_ = v.BindEnv("olap.databaseUrl", "OLAP_DATABASE_URL")
	_ = v.BindEnv("olap.maxConns", "OLAP_DATABASE_MAX_CONNS")
	_ = v.BindEnv("olap.minConns", "OLAP_DATABASE_MIN_CONNS")
	_ = v.BindEnv("olap.maxConnLifetime", "OLAP_DATABASE_MAX_CONN_LIFETIME")
	_ = v.BindEnv("olap.maxConnIdleTime", "OLAP_DATABASE_MAX_CONN_IDLE_TIME")
	_ = v.BindEnv("olap.dialTimeout", "OLAP_DATABASE_DIAL_TIMEOUT")
	_ = v.BindEnv("olap.connectTimeout", "OLAP_DATABASE_CONNECT_TIMEOUT")
	_ = v.BindEnv("olap.fairpoolTenantMaxPercent", "OLAP_DATABASE_FAIRPOOL_TENANT_MAX_PERCENT")
	_ = v.BindEnv("olap.fairpoolSharedMaxPercent", "OLAP_DATABASE_FAIRPOOL_SHARED_MAX_PERCENT")
	_ = v.BindEnv("olap.fairpoolTenantMaxWait", "OLAP_DATABASE_FAIRPOOL_TENANT_MAX_WAIT")

	_ = v.BindEnv("readReplicaEnabled", "READ_REPLICA_ENABLED")
	_ = v.BindEnv("readReplicaDatabaseUrl", "READ_REPLICA_DATABASE_URL")
	_ = v.BindEnv("readReplicaMaxConns", "READ_REPLICA_MAX_CONNS")
	_ = v.BindEnv("readReplicaMinConns", "READ_REPLICA_MIN_CONNS")

	_ = v.BindEnv("cacheDuration", "CACHE_DURATION")
	_ = v.BindEnv("enforceUtcTimezone", "DATABASE_ENFORCE_UTC_TIMEZONE")

	_ = v.BindEnv("seed.adminEmail", "ADMIN_EMAIL")
	_ = v.BindEnv("seed.adminPassword", "ADMIN_PASSWORD")
	_ = v.BindEnv("seed.adminName", "ADMIN_NAME")
	_ = v.BindEnv("seed.defaultTenantName", "DEFAULT_TENANT_NAME")
	_ = v.BindEnv("seed.defaultTenantSlug", "DEFAULT_TENANT_SLUG")
	_ = v.BindEnv("seed.defaultTenantId", "DEFAULT_TENANT_ID")

	_ = v.BindEnv("logger.level", "DATABASE_LOGGER_LEVEL")
	_ = v.BindEnv("logger.format", "DATABASE_LOGGER_FORMAT")
}
