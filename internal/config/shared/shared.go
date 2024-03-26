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
	Level string `mapstructure:"level" json:"level,omitempty" default:"debug"`

	// format can be "json" or "console"
	Format string `mapstructure:"format" json:"format,omitempty" default:"json"`
}

type OpenTelemetryConfigFile struct {
	CollectorURL string `mapstructure:"collectorURL" json:"collectorURL,omitempty"`
	ServiceName  string `mapstructure:"serviceName" json:"serviceName,omitempty" default:"server"`
}
