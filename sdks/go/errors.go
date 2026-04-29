package hatchet

import "fmt"

// NonDeterminismError is returned when a durable task replay detects non-deterministic behavior.
type NonDeterminismError struct {
	TaskExternalID  string
	Message         string
	NodeID          int64
	InvocationCount int32
}

func (e *NonDeterminismError) Error() string {
	return fmt.Sprintf(
		"non-determinism detected: task=%s invocation=%d node=%d: %s",
		e.TaskExternalID, e.InvocationCount, e.NodeID, e.Message,
	)
}

// IsNonDeterminismError checks if the error is a NonDeterminismError
// and returns it if so.
func IsNonDeterminismError(err error) (*NonDeterminismError, bool) {
	if nde, ok := err.(*NonDeterminismError); ok {
		return nde, true
	}
	return nil, false
}

// EvictionNotSupportedError is returned when an eviction policy is configured against
// an engine version that does not support durable-task eviction.
type EvictionNotSupportedError struct {
	EngineVersion string
}

func (e *EvictionNotSupportedError) Error() string {
	suffix := ""
	if e.EngineVersion != "" {
		suffix = fmt.Sprintf(" (engine version: %s)", e.EngineVersion)
	}
	return fmt.Sprintf(
		"eviction policies require engine >= %s%s. "+
			"Please upgrade your Hatchet engine or remove the eviction policy from your task.",
		MinEngineVersion.DurableEviction, suffix,
	)
}

// IsEvictionNotSupportedError reports whether err is an EvictionNotSupportedError.
func IsEvictionNotSupportedError(err error) (*EvictionNotSupportedError, bool) {
	if e, ok := err.(*EvictionNotSupportedError); ok {
		return e, true
	}
	return nil, false
}
