package statusutils

import (
	"context"
	"fmt"

	"github.com/hatchet-dev/hatchet/internal/listutils"
	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type V1RunStatus string

const (
	V1RunStatusQueued    V1RunStatus = "QUEUED"
	V1RunStatusRunning   V1RunStatus = "RUNNING"
	V1RunStatusCancelled V1RunStatus = "CANCELLED"
	V1RunStatusFailed    V1RunStatus = "FAILED"
	V1RunStatusCompleted V1RunStatus = "COMPLETED"
)

func V1RunStatusFromProto(status contracts.RunStatus) (*V1RunStatus, error) {
	switch status {
	case contracts.RunStatus_QUEUED:
		q := V1RunStatusQueued
		return &q, nil
	case contracts.RunStatus_RUNNING:
		r := V1RunStatusRunning
		return &r, nil
	case contracts.RunStatus_CANCELLED:
		c := V1RunStatusCancelled
		return &c, nil
	case contracts.RunStatus_FAILED:
		f := V1RunStatusFailed
		return &f, nil
	case contracts.RunStatus_COMPLETED:
		c := V1RunStatusCompleted
		return &c, nil
	default:
		return nil, fmt.Errorf("unknown run status: %v", status)
	}
}

func (s *V1RunStatus) ToProto() (*contracts.RunStatus, error) {
	switch *s {
	case V1RunStatusQueued:
		r := contracts.RunStatus_QUEUED
		return &r, nil
	case V1RunStatusRunning:
		r := contracts.RunStatus_RUNNING
		return &r, nil
	case V1RunStatusCancelled:
		r := contracts.RunStatus_CANCELLED
		return &r, nil
	case V1RunStatusFailed:
		r := contracts.RunStatus_FAILED
		return &r, nil
	case V1RunStatusCompleted:
		r := contracts.RunStatus_COMPLETED
		return &r, nil
	default:
		return nil, fmt.Errorf("unknown run status: %v", *s)
	}
}

func V1RunStatusFromEventType(eventType sqlcv1.V1TaskEventType) (*V1RunStatus, error) {
	switch eventType {
	case sqlcv1.V1TaskEventTypeCANCELLED:
		q := V1RunStatusCancelled
		return &q, nil
	case sqlcv1.V1TaskEventTypeCOMPLETED:
		r := V1RunStatusCompleted
		return &r, nil
	case sqlcv1.V1TaskEventTypeFAILED:
		c := V1RunStatusFailed
		return &c, nil
	default:
		return nil, fmt.Errorf("unknown task event type: %v", eventType)
	}
}

func DeriveWorkflowRunStatus(ctx context.Context, statuses []V1RunStatus) (*V1RunStatus, error) {
	uniqueStatuses := listutils.Uniq(statuses)

	if len(uniqueStatuses) == 1 {
		return &uniqueStatuses[0], nil
	}

	if listutils.Any(uniqueStatuses, "FAILED") {
		f := V1RunStatusFailed
		return &f, nil
	}

	if listutils.Any(uniqueStatuses, "RUNNING") || listutils.Any(uniqueStatuses, "QUEUED") {
		r := V1RunStatusRunning
		return &r, nil
	}

	if listutils.Any(uniqueStatuses, "CANCELLED") {
		c := V1RunStatusCancelled
		return &c, nil
	}

	return nil, fmt.Errorf("cannot derive workflow run status from given statuses")
}
