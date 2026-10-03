// Package serverlessoperator is the configuration of the hatchet-serverless-operator binary:
// the file struct its environment is bound onto and the validation of the values that must
// agree with each other. The loader package turns a validated file into the binary's
// dependencies (pkg/config/loader).
package serverlessoperator

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator"
)

// Token exchange modes, the values of SERVERLESS_OPERATOR_TOKEN_EXCHANGE. Each mode reads its
// own variable and refuses the other mode's, so a deployment cannot carry a token it does not
// mean to use.
const (
	// TokenExchangeStatic serves the single tenant HATCHET_CLIENT_TOKEN belongs to.
	TokenExchangeStatic = "static"
	// TokenExchangeLocal serves the tenants listed in the file SERVERLESS_OPERATOR_TOKEN_FILE
	// names, reloaded when it changes.
	TokenExchangeLocal = "local"
)

// ConfigFile is the binary's environment, bound by BindAllEnv. Durations accept Go syntax (5s).
type ConfigFile struct {
	DatabaseUrl string `mapstructure:"databaseUrl"`

	Encryption server.EncryptionConfigFile `mapstructure:"encryption"`

	// TokenExchange selects the tenant token source: TokenExchangeStatic reads ClientToken,
	// TokenExchangeLocal reads TokenFile. The other mode's variable must be unset.
	TokenExchange string `mapstructure:"tokenExchange"`
	TokenFile     string `mapstructure:"tokenFile"`
	ClientToken   string `mapstructure:"clientToken"`

	OperatorName string `mapstructure:"operatorName" default:"serverless"`
	DefaultSlots int32  `mapstructure:"defaultSlots" default:"10000"`
	DurableSlots int32  `mapstructure:"durableSlots" default:"10000"`

	// EndpointPartitionCount is the endpoint_partition_count a tenant's v1_serverless_tenant row is
	// created with; see serverlessoperator.Config.EndpointPartitionCount for what it governs and
	// when a change applies.
	EndpointPartitionCount int32 `mapstructure:"endpointPartitionCount" default:"1"`

	LeaseTTL          time.Duration `mapstructure:"leaseTtl" default:"15s"`
	HeartbeatInterval time.Duration `mapstructure:"heartbeatInterval" default:"5s"`
	RebalanceInterval time.Duration `mapstructure:"rebalanceInterval" default:"5s"`
	ShedHysteresis    float64       `mapstructure:"shedHysteresis" default:"0.2"`
	DrainTimeout      time.Duration `mapstructure:"drainTimeout" default:"60s"`

	RoutingRefreshInterval time.Duration `mapstructure:"routingRefreshInterval" default:"10s"`

	HealthcheckTimeout           time.Duration `mapstructure:"healthcheckTimeout" default:"10s"`
	HealthcheckConcurrency       int           `mapstructure:"healthcheckConcurrency" default:"256"`
	HealthcheckTenantConcurrency int           `mapstructure:"healthcheckTenantConcurrency" default:"32"`
	HealthcheckApplyTimeout      time.Duration `mapstructure:"healthcheckApplyTimeout" default:"60s"`

	MaxWorkflowsPerEndpoint int `mapstructure:"maxWorkflowsPerEndpoint" default:"200"`
	MaxActionsPerEndpoint   int `mapstructure:"maxActionsPerEndpoint" default:"500"`

	MaintenanceConcurrency int   `mapstructure:"maintenanceConcurrency" default:"8"`
	LeaseMaxClaimPerTick   int32 `mapstructure:"leaseMaxClaimPerTick" default:"1024"`

	WSMaxFrameBytes int64         `mapstructure:"wsMaxFrameBytes" default:"4194304"`
	WSPingInterval  time.Duration `mapstructure:"wsPingInterval" default:"15s"`

	// Relay resource limits.
	WSMaxUpgradeHeaderBytes int64 `mapstructure:"wsMaxUpgradeHeaderBytes" default:"65536"`
	WSMaxQueuedBytes        int64 `mapstructure:"wsMaxQueuedBytes" default:"16777216"`
	WSMaxStreams            int   `mapstructure:"wsMaxStreams" default:"16"`

	// Outbound HTTP idle connection pool.
	HTTPMaxIdleConns        int           `mapstructure:"httpMaxIdleConns" default:"256"`
	HTTPMaxIdleConnsPerHost int           `mapstructure:"httpMaxIdleConnsPerHost" default:"4"`
	HTTPIdleConnTimeout     time.Duration `mapstructure:"httpIdleConnTimeout" default:"90s"`

	// InfraBlockedCIDRs is a comma-separated list of CIDRs added to safeclient's denylist.
	InfraBlockedCIDRs    string `mapstructure:"infraBlockedCidrs"`
	AllowEmptyInfraCIDRs bool   `mapstructure:"allowEmptyInfraCidrs"`
	InsecureDestinations bool   `mapstructure:"insecureDestinations"`

	HealthPort int `mapstructure:"healthPort" default:"8080"`

	Logger shared.LoggerConfigFile `mapstructure:"logger"`

	OTel shared.OpenTelemetryConfigFile `mapstructure:"otel"`
}

// BindAllEnv maps SERVERLESS_OPERATOR_* onto the config, with the engine's names for the
// encryption keyset and the SDK's HATCHET_CLIENT_TOKEN for the static token exchange.
func BindAllEnv(v *viper.Viper) {
	_ = v.BindEnv("databaseUrl", "SERVERLESS_OPERATOR_DATABASE_URL", "DATABASE_URL")

	_ = v.BindEnv("encryption.masterKeyset", "SERVER_ENCRYPTION_MASTER_KEYSET")
	_ = v.BindEnv("encryption.masterKeysetFile", "SERVER_ENCRYPTION_MASTER_KEYSET_FILE")
	_ = v.BindEnv("encryption.cloudKms.enabled", "SERVER_ENCRYPTION_CLOUDKMS_ENABLED")
	_ = v.BindEnv("encryption.cloudKms.keyURI", "SERVER_ENCRYPTION_CLOUDKMS_KEY_URI")
	_ = v.BindEnv("encryption.cloudKms.credentialsJSON", "SERVER_ENCRYPTION_CLOUDKMS_CREDENTIALS_JSON")

	_ = v.BindEnv("tokenExchange", "SERVERLESS_OPERATOR_TOKEN_EXCHANGE")
	_ = v.BindEnv("tokenFile", "SERVERLESS_OPERATOR_TOKEN_FILE")
	_ = v.BindEnv("clientToken", "HATCHET_CLIENT_TOKEN")

	_ = v.BindEnv("operatorName", "SERVERLESS_OPERATOR_OPERATOR_NAME")
	_ = v.BindEnv("defaultSlots", "SERVERLESS_OPERATOR_DEFAULT_SLOTS")
	_ = v.BindEnv("durableSlots", "SERVERLESS_OPERATOR_DURABLE_SLOTS")
	_ = v.BindEnv("endpointPartitionCount", "SERVERLESS_OPERATOR_ENDPOINT_PARTITION_COUNT")

	_ = v.BindEnv("leaseTtl", "SERVERLESS_OPERATOR_LEASE_TTL")
	_ = v.BindEnv("heartbeatInterval", "SERVERLESS_OPERATOR_HEARTBEAT_INTERVAL")
	_ = v.BindEnv("rebalanceInterval", "SERVERLESS_OPERATOR_REBALANCE_INTERVAL")
	_ = v.BindEnv("shedHysteresis", "SERVERLESS_OPERATOR_SHED_HYSTERESIS")
	_ = v.BindEnv("drainTimeout", "SERVERLESS_OPERATOR_DRAIN_TIMEOUT")

	_ = v.BindEnv("routingRefreshInterval", "SERVERLESS_OPERATOR_ROUTING_REFRESH_INTERVAL")

	_ = v.BindEnv("healthcheckTimeout", "SERVERLESS_OPERATOR_HEALTHCHECK_TIMEOUT")
	_ = v.BindEnv("healthcheckConcurrency", "SERVERLESS_OPERATOR_HEALTHCHECK_CONCURRENCY")
	_ = v.BindEnv("healthcheckTenantConcurrency", "SERVERLESS_OPERATOR_HEALTHCHECK_TENANT_CONCURRENCY")
	_ = v.BindEnv("healthcheckApplyTimeout", "SERVERLESS_OPERATOR_HEALTHCHECK_APPLY_TIMEOUT")
	_ = v.BindEnv("maxWorkflowsPerEndpoint", "SERVERLESS_OPERATOR_MAX_WORKFLOWS_PER_ENDPOINT")
	_ = v.BindEnv("maxActionsPerEndpoint", "SERVERLESS_OPERATOR_MAX_ACTIONS_PER_ENDPOINT")
	_ = v.BindEnv("maintenanceConcurrency", "SERVERLESS_OPERATOR_MAINTENANCE_CONCURRENCY")
	_ = v.BindEnv("leaseMaxClaimPerTick", "SERVERLESS_OPERATOR_LEASE_MAX_CLAIM_PER_TICK")

	_ = v.BindEnv("wsMaxFrameBytes", "SERVERLESS_OPERATOR_WS_MAX_FRAME_BYTES")
	_ = v.BindEnv("wsPingInterval", "SERVERLESS_OPERATOR_WS_PING_INTERVAL")

	_ = v.BindEnv("wsMaxUpgradeHeaderBytes", "SERVERLESS_OPERATOR_WS_MAX_UPGRADE_HEADER_BYTES")
	_ = v.BindEnv("wsMaxQueuedBytes", "SERVERLESS_OPERATOR_WS_MAX_QUEUED_BYTES")
	_ = v.BindEnv("wsMaxStreams", "SERVERLESS_OPERATOR_WS_MAX_STREAMS")

	_ = v.BindEnv("httpMaxIdleConns", "SERVERLESS_OPERATOR_HTTP_MAX_IDLE_CONNS")
	_ = v.BindEnv("httpMaxIdleConnsPerHost", "SERVERLESS_OPERATOR_HTTP_MAX_IDLE_CONNS_PER_HOST")
	_ = v.BindEnv("httpIdleConnTimeout", "SERVERLESS_OPERATOR_HTTP_IDLE_CONN_TIMEOUT")

	_ = v.BindEnv("infraBlockedCidrs", "SERVERLESS_OPERATOR_INFRA_BLOCKED_CIDRS")
	_ = v.BindEnv("allowEmptyInfraCidrs", "SERVERLESS_OPERATOR_ALLOW_EMPTY_INFRA_CIDRS")
	_ = v.BindEnv("insecureDestinations", "SERVERLESS_OPERATOR_INSECURE_DESTINATIONS")

	_ = v.BindEnv("healthPort", "SERVERLESS_OPERATOR_HEALTH_PORT")

	_ = v.BindEnv("logger.level", "SERVERLESS_OPERATOR_LOG_LEVEL")
	_ = v.BindEnv("logger.format", "SERVERLESS_OPERATOR_LOG_FORMAT")

	_ = v.BindEnv("otel.serviceName", "SERVERLESS_OPERATOR_OTEL_SERVICE_NAME", "SERVER_OTEL_SERVICE_NAME")
	_ = v.BindEnv("otel.collectorURL", "SERVERLESS_OPERATOR_OTEL_COLLECTOR_URL", "SERVER_OTEL_COLLECTOR_URL")
	_ = v.BindEnv("otel.insecure", "SERVERLESS_OPERATOR_OTEL_INSECURE", "SERVER_OTEL_INSECURE")
	_ = v.BindEnv("otel.traceIdRatio", "SERVERLESS_OPERATOR_OTEL_TRACE_ID_RATIO", "SERVER_OTEL_TRACE_ID_RATIO")
	_ = v.BindEnv("otel.collectorAuth", "SERVERLESS_OPERATOR_OTEL_COLLECTOR_AUTH", "SERVER_OTEL_COLLECTOR_AUTH")
}

// Validate checks the values that must be present or agree with each other: the database
// URL, the token exchange mode with its variable and without the other mode's, and the partition
// count bounds. Everything else has a default.
func (cf *ConfigFile) Validate() error {
	if cf.DatabaseUrl == "" {
		return fmt.Errorf("SERVERLESS_OPERATOR_DATABASE_URL (or DATABASE_URL) is required")
	}

	if err := cf.validateTokenExchange(); err != nil {
		return err
	}

	return serverlessoperator.ValidateEndpointPartitionCount(cf.EndpointPartitionCount)
}

// TokenExchangeMode is the normalized token exchange mode.
func (cf *ConfigFile) TokenExchangeMode() string {
	return strings.ToLower(strings.TrimSpace(cf.TokenExchange))
}

// validateTokenExchange requires an explicit mode, the mode's own variable, and the absence
// of the other mode's: a local exchange with HATCHET_CLIENT_TOKEN set, or a static one with
// a token file set, is refused rather than silently ignoring one of them.
func (cf *ConfigFile) validateTokenExchange() error {
	switch cf.TokenExchangeMode() {
	case TokenExchangeStatic:
		if cf.ClientToken == "" {
			return fmt.Errorf("HATCHET_CLIENT_TOKEN is required with SERVERLESS_OPERATOR_TOKEN_EXCHANGE=%s", TokenExchangeStatic)
		}

		if cf.TokenFile != "" {
			return fmt.Errorf("SERVERLESS_OPERATOR_TOKEN_FILE must not be set with SERVERLESS_OPERATOR_TOKEN_EXCHANGE=%s", TokenExchangeStatic)
		}
	case TokenExchangeLocal:
		if cf.TokenFile == "" {
			return fmt.Errorf("SERVERLESS_OPERATOR_TOKEN_FILE is required with SERVERLESS_OPERATOR_TOKEN_EXCHANGE=%s", TokenExchangeLocal)
		}

		if cf.ClientToken != "" {
			return fmt.Errorf("HATCHET_CLIENT_TOKEN must not be set with SERVERLESS_OPERATOR_TOKEN_EXCHANGE=%s", TokenExchangeLocal)
		}
	case "":
		return fmt.Errorf("SERVERLESS_OPERATOR_TOKEN_EXCHANGE is required: %s (HATCHET_CLIENT_TOKEN, one tenant) or %s (SERVERLESS_OPERATOR_TOKEN_FILE)", TokenExchangeStatic, TokenExchangeLocal)
	default:
		return fmt.Errorf("unknown SERVERLESS_OPERATOR_TOKEN_EXCHANGE %q: expected %s or %s", cf.TokenExchange, TokenExchangeStatic, TokenExchangeLocal)
	}

	return nil
}

// InfraBlockedCIDRList splits the comma-separated InfraBlockedCIDRs, dropping blanks.
func (cf *ConfigFile) InfraBlockedCIDRList() []string {
	out := make([]string, 0)

	for _, part := range strings.Split(cf.InfraBlockedCIDRs, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}

	return out
}

// CoreConfig maps the file onto the operator core's Config. The link name is the gRPC host's;
// the health port is the binary's own server.
func (cf *ConfigFile) CoreConfig() serverlessoperator.Config {
	return serverlessoperator.Config{
		OperatorName:                 cf.OperatorName,
		LinkName:                     serverlessoperator.DefaultLinkName,
		DefaultSlots:                 cf.DefaultSlots,
		DurableSlots:                 cf.DurableSlots,
		EndpointPartitionCount:       cf.EndpointPartitionCount,
		LeaseTTL:                     cf.LeaseTTL,
		HeartbeatInterval:            cf.HeartbeatInterval,
		RebalanceInterval:            cf.RebalanceInterval,
		ShedHysteresis:               cf.ShedHysteresis,
		DrainTimeout:                 cf.DrainTimeout,
		RoutingRefreshInterval:       cf.RoutingRefreshInterval,
		HealthcheckTimeout:           cf.HealthcheckTimeout,
		HealthcheckConcurrency:       cf.HealthcheckConcurrency,
		HealthcheckTenantConcurrency: cf.HealthcheckTenantConcurrency,
		HealthcheckApplyTimeout:      cf.HealthcheckApplyTimeout,
		MaxWorkflowsPerEndpoint:      cf.MaxWorkflowsPerEndpoint,
		MaxActionsPerEndpoint:        cf.MaxActionsPerEndpoint,
		MaintenanceConcurrency:       cf.MaintenanceConcurrency,
		LeaseMaxClaimPerTick:         cf.LeaseMaxClaimPerTick,
		WSMaxFrameBytes:              cf.WSMaxFrameBytes,
		WSPingInterval:               cf.WSPingInterval,
		HealthPort:                   cf.HealthPort,
		WSMaxUpgradeHeaderBytes:      cf.WSMaxUpgradeHeaderBytes,
		WSMaxQueuedBytes:             cf.WSMaxQueuedBytes,
		WSMaxStreams:                 cf.WSMaxStreams,
	}
}
