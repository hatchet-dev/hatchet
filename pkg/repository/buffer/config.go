package buffer

// ConfigFileBuffer is the configuration for the buffer. We store it here to prevent circular dependencies
type ConfigFileBuffer struct {
	// FlushPeriodMilliseconds is the number of milliseconds before flush
	FlushPeriodMilliseconds int `mapstructure:"flushPeriodMilliseconds" json:"flushPeriodMilliseconds,omitempty" default:"10"`

	// FlushItemsThreshold is the number of items to hold in memory until flushing to the database
	FlushItemsThreshold int `mapstructure:"flushItemsThreshold" json:"flushItemsThreshold,omitempty" default:"100"`

	// SerialBuffer is a flag to determine if the buffer should be serial or bulk
	SerialBuffer bool `mapstructure:"serialBuffer" json:"serialBuffer,omitempty" default:"false"`
}
