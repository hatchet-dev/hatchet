// hatchet-serverless-operator runs the serverless operator core out of process: it talks to
// the engine database for leases and endpoints and, through the gRPC operator host, to the
// engine's OperatorService with per-tenant tokens.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	"github.com/hatchet-dev/hatchet/pkg/config/loader/loaderutils"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/operator/hostgrpc"
	"github.com/hatchet-dev/hatchet/pkg/operator/safeclient"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

// Version is linked by an ldflag during build.
var Version = "v0.94.16"

var printVersion bool

// configFile is the binary's environment, bound by bindEnv. Durations accept Go syntax (5s).
type configFile struct {
	DatabaseUrl string `mapstructure:"databaseUrl"`

	Encryption server.EncryptionConfigFile `mapstructure:"encryption"`

	// TokenExchange selects the tenant token source: "local" reads TokenFile; unset falls
	// back to ClientToken (HATCHET_CLIENT_TOKEN) for a single tenant.
	TokenExchange string `mapstructure:"tokenExchange"`
	TokenFile     string `mapstructure:"tokenFile"`
	ClientToken   string `mapstructure:"clientToken"`

	OperatorName string `mapstructure:"operatorName" default:"serverless"`
	DefaultSlots int32  `mapstructure:"defaultSlots" default:"10000"`
	DurableSlots int32  `mapstructure:"durableSlots" default:"10000"`

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

// bindEnv maps SERVERLESS_OPERATOR_* onto the config, with the engine's names for the
// encryption keyset and the SDK's HATCHET_CLIENT_TOKEN as the single-tenant fallback.
func bindEnv(v *viper.Viper) {
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

func loadConfig() (*configFile, error) {
	cf := &configFile{}

	if _, err := loaderutils.LoadConfigFromViper(bindEnv, cf); err != nil {
		return nil, err
	}

	if cf.DatabaseUrl == "" {
		return nil, fmt.Errorf("SERVERLESS_OPERATOR_DATABASE_URL (or DATABASE_URL) is required")
	}

	return cf, nil
}

func splitCSV(s string) []string {
	out := make([]string, 0)

	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}

	return out
}

func newExchange(cf *configFile, l *zerolog.Logger) (hostgrpc.TokenSource, func(), error) {
	switch strings.ToLower(strings.TrimSpace(cf.TokenExchange)) {
	case "local":
		if cf.TokenFile == "" {
			return nil, nil, fmt.Errorf("SERVERLESS_OPERATOR_TOKEN_FILE is required with TOKEN_EXCHANGE=local")
		}

		ex, err := hostgrpc.NewLocalExchange(cf.TokenFile, hostgrpc.WithLogger(l))

		if err != nil {
			return nil, nil, err
		}

		return ex, ex.Close, nil
	case "":
		if cf.ClientToken == "" {
			return nil, nil, fmt.Errorf("set SERVERLESS_OPERATOR_TOKEN_EXCHANGE=local with a token file, or HATCHET_CLIENT_TOKEN for a single tenant")
		}

		ex, err := hostgrpc.NewStaticExchange(cf.ClientToken)

		if err != nil {
			return nil, nil, err
		}

		l.Warn().Str("tenant_id", ex.TenantId().String()).Msg("using HATCHET_CLIENT_TOKEN for a single tenant; set TOKEN_EXCHANGE=local for production")

		return ex, func() {}, nil
	default:
		return nil, nil, fmt.Errorf("unknown token exchange %q", cf.TokenExchange)
	}
}

func run(ctx context.Context, cf *configFile) error {
	l := logger.NewStdErr(&cf.Logger, "serverless-operator")

	shutdownTracer, err := telemetry.InitTracer(&telemetry.TracerOpts{
		ServiceName:   cf.OTel.ServiceName,
		CollectorURL:  cf.OTel.CollectorURL,
		TraceIdRatio:  cf.OTel.TraceIdRatio,
		Insecure:      cf.OTel.Insecure,
		CollectorAuth: cf.OTel.CollectorAuth,
	})

	if err != nil {
		return fmt.Errorf("could not initialize tracer: %w", err)
	}

	defer func() { _ = shutdownTracer() }()

	pool, err := loader.NewPgxPool(ctx, cf.DatabaseUrl, loader.PgxPoolOpts{
		ApplicationName: "hatchet-serverless-operator",
	})

	if err != nil {
		return err
	}

	defer pool.Close()

	repo, cleanupRepo := repository.NewServerlessRepositoryFromPool(pool, &l)
	defer func() { _ = cleanupRepo() }()

	enc, err := loader.LoadDataEncryptionSvc(&cf.Encryption)

	if err != nil {
		return fmt.Errorf("could not load encryption service: %w", err)
	}

	exchange, closeExchange, err := newExchange(cf, &l)

	if err != nil {
		return err
	}

	defer closeExchange()

	infraCIDRs := splitCSV(cf.InfraBlockedCIDRs)

	sender, err := safeclient.New(safeclient.Config{
		InfraBlockedCIDRs:    infraCIDRs,
		AllowEmptyInfraCIDRs: cf.AllowEmptyInfraCIDRs || cf.InsecureDestinations,
		InsecureDestinations: cf.InsecureDestinations,
		MaxIdleConns:         cf.HTTPMaxIdleConns,
		MaxIdleConnsPerHost:  cf.HTTPMaxIdleConnsPerHost,
		IdleConnTimeout:      cf.HTTPIdleConnTimeout,
	}, &l)

	if err != nil {
		return fmt.Errorf("could not build request sender: %w", err)
	}

	defer sender.CloseIdleConnections()

	hostname, _ := os.Hostname()

	// The operator name is the core's: it registers each tenant's session under it.
	host := hostgrpc.New(exchange, hostgrpc.Options{Logger: &l})
	defer host.Close()

	return serverlessoperator.Run(ctx, serverlessoperator.Deps{
		Repo:       repo,
		Host:       host,
		Encryption: enc,
		Sender:     sender,
		Logger:     &l,
		Version:    Version,
		Hostname:   hostname,
		ProcessId:  uuid.New(),
		Config: serverlessoperator.Config{
			OperatorName:                 cf.OperatorName,
			LinkName:                     serverlessoperator.DefaultLinkName,
			DefaultSlots:                 cf.DefaultSlots,
			DurableSlots:                 cf.DurableSlots,
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
		},
	})
}

var rootCmd = &cobra.Command{
	Use:   "hatchet-serverless-operator",
	Short: "hatchet-serverless-operator delivers Hatchet tasks to serverless endpoints.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if printVersion {
			fmt.Println(Version)
			return nil
		}

		cf, err := loadConfig()

		if err != nil {
			return err
		}

		ctx, cancel := cmdutils.NewInterruptContext()
		defer cancel()

		return run(ctx, cf)
	},
}

func main() {
	rootCmd.PersistentFlags().BoolVar(&printVersion, "version", false, "print version and exit.")
	rootCmd.SilenceUsage = true

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
