package tasks

import (
	"errors"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/v2/timescalev2"
)

func (t *TasksService) V2TaskGetPointMetrics(ctx echo.Context, request gen.V2TaskGetPointMetricsRequestObject) (gen.V2TaskGetPointMetricsResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	// 24 hours ago, rounded to the nearest minute
	lowerBound := time.Now().UTC().Add(-24 * time.Hour).Truncate(30 * time.Minute)
	upperBound := time.Now().UTC()

	// fmt.Println(request.Params.CreatedAfter, request.Params.CreatedBefore, lowerBound, upperBound)

	if request.Params.CreatedAfter != nil {
		lowerBound = request.Params.CreatedAfter.UTC()
	}
	if request.Params.FinishedBefore != nil {
		upperBound = request.Params.FinishedBefore.UTC()
	}

	// determine a bucket interval based on the time range. If the time range is less than 1 hour, use 1 minute intervals.
	// If the time range is less than 1 day, use 5 minute intervals. Otherwise, use 30 minute intervals.
	var bucketInterval time.Duration

	if upperBound.Sub(lowerBound) < 61*time.Minute {
		bucketInterval = time.Minute
		lowerBound = lowerBound.Truncate(time.Minute)
	} else if upperBound.Sub(lowerBound) < 12*time.Hour {
		bucketInterval = 5 * time.Minute
		lowerBound = lowerBound.Truncate(5 * time.Minute)
	} else if upperBound.Sub(lowerBound) < 48*time.Hour {
		bucketInterval = 30 * time.Minute
		lowerBound = lowerBound.Truncate(30 * time.Minute)
	} else if upperBound.Sub(lowerBound) < 8*24*time.Hour {
		bucketInterval = 8 * time.Hour
		lowerBound = lowerBound.Truncate(8 * time.Hour)
	} else {
		bucketInterval = 24 * time.Hour
		lowerBound = lowerBound.Truncate(24 * time.Hour)
	}

	metrics, err := t.config.EngineRepository.OLAP().GetTaskPointMetrics(ctx.Request().Context(), tenant.ID, &lowerBound, &upperBound, bucketInterval)

	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return gen.V2TaskGetPointMetrics400JSONResponse(
				apierrors.NewAPIErrors("workflow not found"),
			), nil
		}

		return nil, err
	}

	// Fill missing minutes with 0 values
	convertedMetrics := fillMissingMinutesWithZero(lowerBound, upperBound, convertToGenMetrics(metrics), bucketInterval)

	return gen.V2TaskGetPointMetrics200JSONResponse{
		Results: &convertedMetrics,
	}, nil
}

type WorkflowRunEventsMetrics struct {
	Results *[]gen.V2TaskPointMetric `json:"results,omitempty"`
}

func convertToGenMetrics(metrics []*timescalev2.GetTaskPointMetricsRow) []gen.V2TaskPointMetric {
	converted := make([]gen.V2TaskPointMetric, len(metrics))

	for i, metric := range metrics {
		if metric == nil || !metric.Bucket.Valid {
			continue
		}

		timeMinute := metric.Bucket.Time.UTC()

		converted[i] = gen.V2TaskPointMetric{
			FAILED:    int(metric.FailedCount),
			SUCCEEDED: int(metric.CompletedCount),
			Time:      timeMinute,
		}
	}

	return converted
}

// fillMissingMinutesWithZero fills in missing minutes between lowerBound and upperBound with 0 values.
func fillMissingMinutesWithZero(lowerBound, upperBound time.Time, metrics []gen.V2TaskPointMetric, bucketInterval time.Duration) []gen.V2TaskPointMetric {
	result := []gen.V2TaskPointMetric{}

	metricMap := make(map[time.Time]gen.V2TaskPointMetric)

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
			result = append(result, gen.V2TaskPointMetric{
				FAILED:    int(0),
				SUCCEEDED: int(0),
				Time:      timeCopy,
			})
		}
	}

	return result
}
