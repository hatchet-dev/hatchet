package shared

type TLSConfigFile struct {
	// TLSStrategy can be "tls", "mtls", or "none"
	TLSStrategy string `mapstructure:"tlsStrategy" json:"tlsStrategy,omitempty" default:"tls"`

	TLSCert       string `mapstructure:"tlsCert" json:"tlsCert,omitempty"`
	TLSCertFile   string `mapstructure:"tlsCertFile" json:"tlsCertFile,omitempty"`
	TLSKey        string `mapstructure:"tlsKey" json:"tlsKey,omitempty"`
	TLSKeyFile    string `mapstructure:"tlsKeyFile" json:"tlsKeyFile,omitempty"`
	TLSRootCA     string `mapstructure:"tlsRootCA" json:"tlsRootCA,omitempty"`
	TLSRootCAFile string `mapstructure:"tlsRootCAFile" json:"tlsRootCAFile,omitempty"`
}

type LoggerConfigFile struct {
	Level string `mapstructure:"level" json:"level,omitempty" default:"warn"`

	// format can be "json" or "console"
	Format string `mapstructure:"format" json:"format,omitempty" default:"console"`
}

type OpenTelemetryConfigFile struct {
	CollectorURL   string `mapstructure:"collectorURL" json:"collectorURL,omitempty"`
	ServiceName    string `mapstructure:"serviceName" json:"serviceName,omitempty" default:"server"`
	TraceIdRatio   string `mapstructure:"traceIdRatio" json:"traceIdRatio,omitempty" default:"1"`
	CollectorAuth  string `mapstructure:"collectorAuth" json:"collectorAuth,omitempty"`
	Insecure       bool   `mapstructure:"insecure" json:"insecure,omitempty" default:"false"`
	MetricsEnabled bool   `mapstructure:"metricsEnabled" json:"metricsEnabled,omitempty" default:"false"`
}

type PrometheusConfigFile struct {
	PrometheusServerURL      string `mapstructure:"prometheusServerURL" json:"prometheusServerURL,omitempty" default:""`
	PrometheusServerUsername string `mapstructure:"prometheusServerUsername" json:"prometheusServerUsername,omitempty" default:""`
	PrometheusServerPassword string `mapstructure:"prometheusServerPassword" json:"prometheusServerPassword,omitempty" default:""`
	Address                  string `mapstructure:"address" json:"address,omitempty" default:":9090"`
	Path                     string `mapstructure:"path" json:"path,omitempty" default:"/metrics"`
	Enabled                  bool   `mapstructure:"enabled" json:"enabled,omitempty" default:"false"`
}
