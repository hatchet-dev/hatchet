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
	CollectorURL string `mapstructure:"collectorURL" json:"collectorURL,omitempty"`
	ServiceName  string `mapstructure:"serviceName" json:"serviceName,omitempty" default:"server"`
	TraceIdRatio string `mapstructure:"traceIdRatio" json:"traceIdRatio,omitempty" default:"1"`
	Insecure     bool   `mapstructure:"insecure" json:"insecure,omitempty" default:"false"`
}

type PrometheusConfigFile struct {
	// PrometheusServerURL is the URL of the prometheus server
	PrometheusServerURL string `mapstructure:"prometheusServerURL" json:"prometheusServerURL,omitempty" default:""`

	// PrometheusServerUsername is the username for the prometheus server that supports basic auth
	PrometheusServerUsername string `mapstructure:"prometheusServerUsername" json:"prometheusServerUsername,omitempty" default:""`

	// PrometheusServerPassword is the password for the prometheus server that supports basic auth
	PrometheusServerPassword string `mapstructure:"prometheusServerPassword" json:"prometheusServerPassword,omitempty" default:""`

	// Address is the metrics endpoint address
	Address string `mapstructure:"address" json:"address,omitempty" default:":9090"`

	// Enabled is a boolean that enables or disables the prometheus server
	Enabled bool `mapstructure:"enabled" json:"enabled,omitempty" default:"false"`

	// Path is the path to bind the prometheus server to
	Path string `mapstructure:"path" json:"path,omitempty" default:"/metrics"`
}
