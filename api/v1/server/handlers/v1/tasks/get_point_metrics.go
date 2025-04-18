package tasks

import (
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func (t *TasksService) V1TaskGetPointMetrics(ctx echo.Context, request gen.V1TaskGetPointMetricsRequestObject) (gen.V1TaskGetPointMetricsResponseObject, error) {
	tenant, err := populator.FromContext(ctx).GetTenant()
	if err != nil {
		return nil, err
	}
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// 24 hours ago, rounded to the nearest minute
	lowerBound := time.Now().UTC().Add(-24 * time.Hour).Truncate(30 * time.Minute)
	upperBound := time.Now().UTC()

	if request.Params.CreatedAfter != nil {
		lowerBound = request.Params.CreatedAfter.UTC()
	}
	if request.Params.FinishedBefore != nil {
		upperBound = request.Params.FinishedBefore.UTC()
	}

	// determine a bucket interval based on the time range. If the time range is less than 1 hour, use 1 minute intervals.
	// If the time range is less than 1 day, use 5 minute intervals. Otherwise, use 30 minute intervals.
	var bucketInterval time.Duration

	switch {
	case upperBound.Sub(lowerBound) < 61*time.Minute:
		bucketInterval = time.Minute
		lowerBound = lowerBound.Truncate(time.Minute)
	case upperBound.Sub(lowerBound) < 12*time.Hour:
		bucketInterval = 5 * time.Minute
		lowerBound = lowerBound.Truncate(5 * time.Minute)
	case upperBound.Sub(lowerBound) < 48*time.Hour:
		bucketInterval = 30 * time.Minute
		lowerBound = lowerBound.Truncate(30 * time.Minute)
	case upperBound.Sub(lowerBound) < 8*24*time.Hour:
		bucketInterval = 8 * time.Hour
		lowerBound = lowerBound.Truncate(8 * time.Hour)
	default:
		bucketInterval = 24 * time.Hour
		lowerBound = lowerBound.Truncate(24 * time.Hour)
	}

	metrics, err := t.config.V1.OLAP().GetTaskPointMetrics(ctx.Request().Context(), tenantId, &lowerBound, &upperBound, bucketInterval)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return gen.V1TaskGetPointMetrics400JSONResponse(
				apierrors.NewAPIErrors("workflow not found"),
			), nil
		}

		return nil, err
	}

	// Fill missing minutes with 0 values
	convertedMetrics := fillMissingMinutesWithZero(lowerBound, upperBound, convertToGenMetrics(metrics), bucketInterval)

	return gen.V1TaskGetPointMetrics200JSONResponse{
		Results: &convertedMetrics,
	}, nil
}

type WorkflowRunEventsMetrics struct {
	Results *[]gen.V1TaskPointMetric `json:"results,omitempty"`
}

func convertToGenMetrics(metrics []*sqlcv1.GetTaskPointMetricsRow) []gen.V1TaskPointMetric {
	converted := make([]gen.V1TaskPointMetric, len(metrics))

	for i, metric := range metrics {
		if metric == nil || !metric.Bucket2.Valid {
			continue
		}

		timeMinute := metric.Bucket2.Time.UTC()

		converted[i] = gen.V1TaskPointMetric{
			FAILED:    int(metric.FailedCount),
			SUCCEEDED: int(metric.CompletedCount),
			Time:      timeMinute,
		}
	}

	return converted
}

// fillMissingMinutesWithZero fills in missing minutes between lowerBound and upperBound with 0 values.
func fillMissingMinutesWithZero(lowerBound, upperBound time.Time, metrics []gen.V1TaskPointMetric, bucketInterval time.Duration) []gen.V1TaskPointMetric {
	result := []gen.V1TaskPointMetric{}

	metricMap := make(map[time.Time]gen.V1TaskPointMetric)

	for _, metric := range metrics {
		if !metric.Time.IsZero() {
			metricMap[(metric.Time).UTC()] = metric
		}
	}

	for t := lowerBound; t.Before(upperBound) || t.Equal(upperBound); t = t.Add(bucketInterval) {
		if metric, exists := metricMap[t]; exists {
			result = append(result, metric)
		} else {
			timeCopy := t
			result = append(result, gen.V1TaskPointMetric{
				FAILED:    int(0),
				SUCCEEDED: int(0),
				Time:      timeCopy,
			})
		}
	}

	return result
}
