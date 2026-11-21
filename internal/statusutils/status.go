package statusutils

import (
	"context"
	"fmt"

	"github.com/hatchet-dev/hatchet/internal/enumerables"
	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/enums"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type V1RunStatus string

const (
	V1RunStatusQueued    V1RunStatus = "QUEUED"
	V1RunStatusRunning   V1RunStatus = "RUNNING"
	V1RunStatusEvicted   V1RunStatus = "EVICTED"
	V1RunStatusCancelled V1RunStatus = "CANCELLED"
	V1RunStatusFailed    V1RunStatus = "FAILED"
	V1RunStatusCompleted V1RunStatus = "COMPLETED"
)

func V1RunStatusFromProto(status contracts.RunStatus) (*V1RunStatus, error) {
	s, err := enums.FromProto[V1RunStatus](status)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (s *V1RunStatus) ToProto() (*contracts.RunStatus, error) {
	if s == nil {
		return nil, fmt.Errorf("nil run status")
	}

	// clients treat an evicted run as still running
	if *s == V1RunStatusEvicted {
		r := contracts.RunStatus_RUNNING
		return &r, nil
	}

	r, err := enums.ToProto[contracts.RunStatus](*s)
	if err != nil {
		return nil, err
	}

	return &r, nil
}

func (s *V1RunStatus) IsEvicted() bool {
	return s != nil && *s == V1RunStatusEvicted
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
	case sqlcv1.V1TaskEventTypeSIGNALCREATED, sqlcv1.V1TaskEventTypeSIGNALCOMPLETED:
		return nil, fmt.Errorf("unknown task event type: %v", eventType)
	default:
		return nil, fmt.Errorf("unknown task event type: %v", eventType)
	}
}

func DeriveWorkflowRunStatus(ctx context.Context, statuses []V1RunStatus) (*V1RunStatus, error) {
	uniqueStatuses := enumerables.Uniq(statuses)

	if len(uniqueStatuses) == 1 {
		return &uniqueStatuses[0], nil
	}

	if enumerables.Any(uniqueStatuses, "FAILED") {
		f := V1RunStatusFailed
		return &f, nil
	}

	if enumerables.Any(uniqueStatuses, "RUNNING") || enumerables.Any(uniqueStatuses, "QUEUED") || enumerables.Any(uniqueStatuses, "EVICTED") {
		r := V1RunStatusRunning
		return &r, nil
	}

	if enumerables.Any(uniqueStatuses, "CANCELLED") {
		c := V1RunStatusCancelled
		return &c, nil
	}

	return nil, fmt.Errorf("cannot derive workflow run status from given statuses")
}
