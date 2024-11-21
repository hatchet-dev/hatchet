package buffer

import "time"

// ConfigFileBuffer is the configuration for the buffer. We store it here to prevent circular dependencies
type ConfigFileBuffer struct {

	// WaitForFlush is the time to wait for the buffer to flush used for backpressure on writers
	WaitForFlush time.Duration `mapstructure:"waitForFlush" json:"waitForFlush,omitempty" default:"1ms"`

	// MaxConcurrent is the maximum number of concurrent flushes
	MaxConcurrent int `mapstructure:"maxConcurrent" json:"maxConcurrent,omitempty" default:"50"`

	// FlushPeriodMilliseconds is the number of milliseconds before flush
	FlushPeriodMilliseconds int `mapstructure:"flushPeriodMilliseconds" json:"flushPeriodMilliseconds,omitempty" default:"10"`

	// FlushItemsThreshold is the number of items to hold in memory until flushing to the database
	FlushItemsThreshold int `mapstructure:"flushItemsThreshold" json:"flushItemsThreshold,omitempty" default:"100"`

	// SerialBuffer is a flag to determine if the buffer should be serial or bulk
	SerialBuffer bool `mapstructure:"serialBuffer" json:"serialBuffer,omitempty" default:"false"`

	// FlushStrategy is the strategy to use for flushing the buffer
	FlushStrategy BuffStrategy `mapstructure:"flushStrategy" json:"flushStrategy" default:"DYNAMIC"`
}
