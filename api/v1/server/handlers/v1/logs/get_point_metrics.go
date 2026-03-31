package logs

import (
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (t *LogsService) V1TenantLogLineGetPointMetrics(ctx echo.Context, request gen.V1TenantLogLineGetPointMetricsRequestObject) (gen.V1TenantLogLineGetPointMetricsResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	reqCtx, span := telemetry.NewSpan(ctx.Request().Context(), "GET /api/v1/stable/tenants/{tenant}/log-point-metrics")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant.id", Value: tenantId},
	)

	lowerBound := time.Now().UTC().Add(-24 * time.Hour).Truncate(30 * time.Minute)
	upperBound := time.Now().UTC()

	if request.Params.Since != nil {
		lowerBound = request.Params.Since.UTC()
	}
	if request.Params.Until != nil {
		upperBound = request.Params.Until.UTC()
	}

	// determine a bucket interval based on the time range
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

	var search *string
	if request.Params.Search != nil {
		search = request.Params.Search
	}

	var levels []string
	if request.Params.Levels != nil {
		for _, l := range *request.Params.Levels {
			levels = append(levels, string(l))
		}
	}

	var taskExternalIds []uuid.UUID
	if request.Params.TaskExternalIds != nil {
		taskExternalIds = append(taskExternalIds, *request.Params.TaskExternalIds...)
	}

	var stepIds []uuid.UUID
	if request.Params.StepIds != nil {
		stepIds = append(stepIds, *request.Params.StepIds...)
	}

	var workflowIds []uuid.UUID
	if request.Params.WorkflowIds != nil {
		workflowIds = append(workflowIds, *request.Params.WorkflowIds...)
	}

	rows, err := t.config.V1.Logs().GetLogLinePointMetrics(reqCtx, tenantId, &v1.GetLogLinePointMetricsOpts{
		StartTimestamp:  lowerBound,
		EndTimestamp:    upperBound,
		BucketInterval:  bucketInterval,
		Search:          search,
		Levels:          levels,
		TaskExternalIds: taskExternalIds,
		StepIds:         stepIds,
		WorkflowIds:     workflowIds,
	})
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	t.config.Analytics.Count(ctx.Request().Context(), analytics.Log, analytics.Get, analytics.Properties{
		"has_search":            search != nil,
		"has_levels":            len(levels) > 0,
		"has_task_external_ids": len(taskExternalIds) > 0,
	})

	converted := convertToLogMetrics(rows)
	filled := fillMissingLogBuckets(lowerBound, upperBound, converted, bucketInterval)

	return gen.V1TenantLogLineGetPointMetrics200JSONResponse{
		Results: &filled,
	}, nil
}

func convertToLogMetrics(rows []*sqlcv1.GetLogLinePointMetricsRow) []gen.V1LogsPointMetric {
	converted := make([]gen.V1LogsPointMetric, len(rows))

	for i, row := range rows {
		if row == nil || !row.MinuteBucket.Valid {
			continue
		}

		converted[i] = gen.V1LogsPointMetric{
			Time:  row.MinuteBucket.Time.UTC(),
			DEBUG: int(row.DebugCount),
			INFO:  int(row.InfoCount),
			WARN:  int(row.WarnCount),
			ERROR: int(row.ErrorCount),
		}
	}

	return converted
}

func fillMissingLogBuckets(lowerBound, upperBound time.Time, metrics []gen.V1LogsPointMetric, bucketInterval time.Duration) []gen.V1LogsPointMetric {
	result := []gen.V1LogsPointMetric{}

	metricMap := make(map[time.Time]gen.V1LogsPointMetric)
	for _, m := range metrics {
		if !m.Time.IsZero() {
			metricMap[m.Time.UTC()] = m
		}
	}

	for t := lowerBound; t.Before(upperBound) || t.Equal(upperBound); t = t.Add(bucketInterval) {
		if m, exists := metricMap[t]; exists {
			result = append(result, m)
		} else {
			tCopy := t
			result = append(result, gen.V1LogsPointMetric{
				Time:  tCopy,
				DEBUG: 0,
				INFO:  0,
				WARN:  0,
				ERROR: 0,
			})
		}
	}

	return result
}
