// Package limitalerts holds the plugin seam for tenant resource limit alerts.
// The engine ticker calls the configured Hook for every newly created alert so
// a downstream build can forward it; OSS leaves the hook nil.
package limitalerts

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AlertType mirrors the TenantResourceLimitAlertType enum values.
type AlertType string

const (
	// AlertTypeAlarm means usage reached the alarm value (80% of the limit by default).
	AlertTypeAlarm AlertType = "Alarm"
	// AlertTypeExhausted means usage reached the limit itself.
	AlertTypeExhausted AlertType = "Exhausted"
)

// Alert is one newly created tenant resource limit alert.
type Alert struct {
	ID       uuid.UUID
	TenantID uuid.UUID
	// Resource is the LimitResource enum value, for example TASK_RUN.
	Resource  string
	AlertType AlertType
	// Value and Limit are the snapshot taken when the alert row was inserted.
	Value     int32
	Limit     int32
	CreatedAt time.Time
}

// Hook receives every newly created tenant resource limit alert, regardless of
// the tenant's own alerting settings. Nil means disabled (OSS / self-host).
// Errors are logged by the ticker and never retried.
type Hook func(ctx context.Context, alert Alert) error
