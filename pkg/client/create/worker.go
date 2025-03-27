package create

import "github.com/rs/zerolog"

// WorkerLabels represents a map of labels that can be assigned to a worker
// for filtering and identification purposes.
type WorkerLabels map[string]interface{}

// CreateOpts defines the options for creating a new worker.
type WorkerOpts struct {
	// (required) the friendly name of the worker
	Name string

	// (optional) maximum number of concurrent runs on this worker, defaults to 100
	Slots int

	// (optional) labels to set on the worker
	Labels WorkerLabels

	// (optional) logger to use for the worker
	Logger *zerolog.Logger

	// (optional) log level
	LogLevel string

	// (optional) maximum number of concurrent runs for durable tasks, defaults to 1000
	DurableSlots int
}
