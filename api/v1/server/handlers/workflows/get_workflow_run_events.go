package workflows

import (
	"errors"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

func (t *WorkflowService) WorkflowRunEventsGetMetrics(ctx echo.Context, request gen.WorkflowRunEventsGetMetricsRequestObject) (gen.WorkflowRunEventsGetMetricsResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	lowerBound := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	upperBound := time.Now().UTC()

	if request.Params.CreatedAfter != nil {
		lowerBound = *request.Params.CreatedAfter
	}
	if request.Params.CreatedBefore != nil {
		upperBound = *request.Params.CreatedBefore
	}

	metrics, err := t.config.EngineRepository.WorkflowRunEvent().GetWorkflowRunEventMetrics(ctx.Request().Context(), tenant.ID, &lowerBound, &upperBound)

	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return gen.WorkflowRunEventsGetMetrics400JSONResponse(
				apierrors.NewAPIErrors("workflow not found"),
			), nil
		}

		return nil, err
	}
	convertedMetrics := ConvertToGenMetrics(metrics)

	return gen.WorkflowRunEventsGetMetrics200JSONResponse{

		Results: &convertedMetrics,
	}, nil

}

type WorkflowRunEventsMetrics struct {
	Results *[]struct {
		FAILED     *int       `json:"FAILED,omitempty"`
		PENDING    *int       `json:"PENDING,omitempty"`
		QUEUED     *int       `json:"QUEUED,omitempty"`
		QUEUEDEPTH *int       `json:"QUEUE_DEPTH,omitempty"`
		RUNNING    *int       `json:"RUNNING,omitempty"`
		SUCCEEDED  *int       `json:"SUCCEEDED,omitempty"`
		Time       *time.Time `json:"time,omitempty"`
	} `json:"results,omitempty"`
}

func int64ToPtrInt(value int64) *int {
	intValue := int(value)
	return &intValue
}

func ConvertToGenMetrics(metrics []*dbsqlc.WorkflowRunEventsMetricsRow) []struct {
	FAILED     *int       `json:"FAILED,omitempty"`
	PENDING    *int       `json:"PENDING,omitempty"`
	QUEUED     *int       `json:"QUEUED,omitempty"`
	QUEUEDEPTH *int       `json:"QUEUE_DEPTH,omitempty"`
	RUNNING    *int       `json:"RUNNING,omitempty"`
	SUCCEEDED  *int       `json:"SUCCEEDED,omitempty"`
	Time       *time.Time `json:"time,omitempty"`
} {
	converted := make([]struct {
		FAILED     *int       `json:"FAILED,omitempty"`
		PENDING    *int       `json:"PENDING,omitempty"`
		QUEUED     *int       `json:"QUEUED,omitempty"`
		QUEUEDEPTH *int       `json:"QUEUE_DEPTH,omitempty"`
		RUNNING    *int       `json:"RUNNING,omitempty"`
		SUCCEEDED  *int       `json:"SUCCEEDED,omitempty"`
		Time       *time.Time `json:"time,omitempty"`
	}, len(metrics))

	for i, metric := range metrics {
		if metric == nil {
			continue
		}
		if metric.Minute == nil {
			continue
		}

		if timeMinute, ok := metric.Minute.(time.Time); ok {

			converted[i] = struct {
				FAILED     *int       `json:"FAILED,omitempty"`
				PENDING    *int       `json:"PENDING,omitempty"`
				QUEUED     *int       `json:"QUEUED,omitempty"`
				QUEUEDEPTH *int       `json:"QUEUE_DEPTH,omitempty"`
				RUNNING    *int       `json:"RUNNING,omitempty"`
				SUCCEEDED  *int       `json:"SUCCEEDED,omitempty"`
				Time       *time.Time `json:"time,omitempty"`
			}{
				FAILED:     int64ToPtrInt(metric.FailedCount),
				PENDING:    int64ToPtrInt(metric.PendingCount),
				QUEUED:     int64ToPtrInt(metric.QueuedCount),
				QUEUEDEPTH: int64ToPtrInt(metric.QueueDepth),
				RUNNING:    int64ToPtrInt(metric.RunningCount),
				SUCCEEDED:  int64ToPtrInt(metric.SucceededCount),
				Time:       &timeMinute,
			}
		} else {
			continue
		}

	}

	return converted
}
