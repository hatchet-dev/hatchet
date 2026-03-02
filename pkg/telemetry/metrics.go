package telemetry

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// MetricsRecorder provides a centralized way to record OTel metrics
type MetricsRecorder struct {
	meter metric.Meter

	// Database health metrics
	dbBloatGauge                      metric.Int64Gauge
	dbBloatPercentGauge               metric.Float64Gauge
	dbLongRunningQueriesGauge         metric.Int64Gauge
	dbQueryCacheHitRatioGauge         metric.Float64Gauge
	dbLongRunningVacuumGauge          metric.Int64Gauge
	dbLastAutovacuumSecondsSinceGauge metric.Float64Gauge

	// OLAP metrics
	olapTempTableSizeDAGGauge  metric.Int64Gauge
	olapTempTableSizeTaskGauge metric.Int64Gauge
	yesterdayRunCountGauge     metric.Int64Gauge

	// Worker metrics
	activeSlotsGauge      metric.Int64Gauge
	activeSlotsByKeyGauge metric.Int64Gauge
	activeWorkersGauge    metric.Int64Gauge
	activeSDKsGauge       metric.Int64Gauge
}

// NewMetricsRecorder creates a new metrics recorder with all instruments registered
func NewMetricsRecorder(ctx context.Context) (*MetricsRecorder, error) {
	meter := otel.Meter("hatchet.run/metrics")

	// Database health metrics
	dbBloatGauge, err := meter.Int64Gauge(
		"hatchet.db.bloat.count",
		metric.WithDescription("Number of bloated tables detected in the database"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create db bloat gauge: %w", err)
	}

	dbBloatPercentGauge, err := meter.Float64Gauge(
		"hatchet.db.bloat.dead_tuple_percent",
		metric.WithDescription("Percentage of dead tuples per table"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create db bloat percent gauge: %w", err)
	}

	dbLongRunningQueriesGauge, err := meter.Int64Gauge(
		"hatchet.db.long_running_queries.count",
		metric.WithDescription("Number of long-running queries detected in the database"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create long running queries gauge: %w", err)
	}

	dbQueryCacheHitRatioGauge, err := meter.Float64Gauge(
		"hatchet.db.query_cache.hit_ratio",
		metric.WithDescription("Query cache hit ratio percentage for tables"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create query cache hit ratio gauge: %w", err)
	}

	dbLongRunningVacuumGauge, err := meter.Int64Gauge(
		"hatchet.db.long_running_vacuum.count",
		metric.WithDescription("Number of long-running vacuum operations detected in the database"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create long running vacuum gauge: %w", err)
	}

	dbLastAutovacuumSecondsSinceGauge, err := meter.Float64Gauge(
		"hatchet.db.last_autovacuum.seconds_since",
		metric.WithDescription("Seconds since last autovacuum for partitioned tables"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create last autovacuum gauge: %w", err)
	}

	// OLAP metrics (instance-wide)
	olapTempTableSizeDAGGauge, err := meter.Int64Gauge(
		"hatchet.olap.temp_table_size.dag_status_updates",
		metric.WithDescription("Size of temporary table for DAG status updates (instance-wide)"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OLAP DAG temp table size gauge: %w", err)
	}

	olapTempTableSizeTaskGauge, err := meter.Int64Gauge(
		"hatchet.olap.temp_table_size.task_status_updates",
		metric.WithDescription("Size of temporary table for task status updates (instance-wide)"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OLAP task temp table size gauge: %w", err)
	}

	yesterdayRunCountGauge, err := meter.Int64Gauge(
		"hatchet.olap.yesterday_run_count",
		metric.WithDescription("Number of workflow runs from yesterday by status (instance-wide)"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create yesterday run count gauge: %w", err)
	}

	// Worker metrics
	activeSlotsGauge, err := meter.Int64Gauge(
		"hatchet.workers.active_slots",
		metric.WithDescription("Number of active worker slots per tenant"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create active slots gauge: %w", err)
	}

	activeSlotsByKeyGauge, err := meter.Int64Gauge(
		"hatchet.workers.active_slots.by_key",
		metric.WithDescription("Number of active worker slots per tenant and slot key"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create active slots by key gauge: %w", err)
	}

	activeWorkersGauge, err := meter.Int64Gauge(
		"hatchet.workers.active_count",
		metric.WithDescription("Number of active workers per tenant"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create active workers gauge: %w", err)
	}

	activeSDKsGauge, err := meter.Int64Gauge(
		"hatchet.workers.active_sdks",
		metric.WithDescription("Number of active SDKs per tenant and SDK version"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create active SDKs gauge: %w", err)
	}

	return &MetricsRecorder{
		meter:                             meter,
		dbBloatGauge:                      dbBloatGauge,
		dbBloatPercentGauge:               dbBloatPercentGauge,
		dbLongRunningQueriesGauge:         dbLongRunningQueriesGauge,
		dbQueryCacheHitRatioGauge:         dbQueryCacheHitRatioGauge,
		dbLongRunningVacuumGauge:          dbLongRunningVacuumGauge,
		dbLastAutovacuumSecondsSinceGauge: dbLastAutovacuumSecondsSinceGauge,
		olapTempTableSizeDAGGauge:         olapTempTableSizeDAGGauge,
		olapTempTableSizeTaskGauge:        olapTempTableSizeTaskGauge,
		yesterdayRunCountGauge:            yesterdayRunCountGauge,
		activeSlotsGauge:                  activeSlotsGauge,
		activeSlotsByKeyGauge:             activeSlotsByKeyGauge,
		activeWorkersGauge:                activeWorkersGauge,
		activeSDKsGauge:                   activeSDKsGauge,
	}, nil
}

// RecordDBBloat records the number of bloated tables detected
func (m *MetricsRecorder) RecordDBBloat(ctx context.Context, count int64, healthStatus string) {
	m.dbBloatGauge.Record(ctx, count,
		metric.WithAttributes(attribute.String("health_status", healthStatus)))
}

// RecordDBBloatPercent records the dead tuple percentage for a specific table
func (m *MetricsRecorder) RecordDBBloatPercent(ctx context.Context, tableName string, deadPercent float64) {
	m.dbBloatPercentGauge.Record(ctx, deadPercent,
		metric.WithAttributes(attribute.String("table_name", tableName)))
}

// RecordDBLongRunningQueries records the number of long-running queries
func (m *MetricsRecorder) RecordDBLongRunningQueries(ctx context.Context, count int64) {
	m.dbLongRunningQueriesGauge.Record(ctx, count)
}

// RecordDBQueryCacheHitRatio records the query cache hit ratio for a table
func (m *MetricsRecorder) RecordDBQueryCacheHitRatio(ctx context.Context, tableName string, hitRatio float64) {
	m.dbQueryCacheHitRatioGauge.Record(ctx, hitRatio,
		metric.WithAttributes(attribute.String("table_name", tableName)))
}

// RecordDBLongRunningVacuum records the number of long-running vacuum operations
func (m *MetricsRecorder) RecordDBLongRunningVacuum(ctx context.Context, count int64, healthStatus string) {
	m.dbLongRunningVacuumGauge.Record(ctx, count,
		metric.WithAttributes(attribute.String("health_status", healthStatus)))
}

// RecordDBLastAutovacuumSecondsSince records seconds since last autovacuum for a partitioned table
func (m *MetricsRecorder) RecordDBLastAutovacuumSecondsSince(ctx context.Context, tableName string, seconds float64) {
	m.dbLastAutovacuumSecondsSinceGauge.Record(ctx, seconds,
		metric.WithAttributes(attribute.String("table_name", tableName)))
}

// RecordOLAPTempTableSizeDAG records the size of the OLAP DAG status updates temp table (instance-wide)
func (m *MetricsRecorder) RecordOLAPTempTableSizeDAG(ctx context.Context, size int64) {
	m.olapTempTableSizeDAGGauge.Record(ctx, size)
}

// RecordOLAPTempTableSizeTask records the size of the OLAP task status updates temp table (instance-wide)
func (m *MetricsRecorder) RecordOLAPTempTableSizeTask(ctx context.Context, size int64) {
	m.olapTempTableSizeTaskGauge.Record(ctx, size)
}

// RecordYesterdayRunCount records the number of workflow runs from yesterday (instance-wide)
func (m *MetricsRecorder) RecordYesterdayRunCount(ctx context.Context, status string, count int64) {
	m.yesterdayRunCountGauge.Record(ctx, count,
		metric.WithAttributes(attribute.String("status", status)))
}

// RecordActiveSlots records the number of active worker slots
func (m *MetricsRecorder) RecordActiveSlots(ctx context.Context, tenantId uuid.UUID, count int64) {
	m.activeSlotsGauge.Record(ctx, count,
		metric.WithAttributes(attribute.String("tenant_id", tenantId.String())))
}

// RecordActiveSlotsByKey records the number of active worker slots by key
func (m *MetricsRecorder) RecordActiveSlotsByKey(ctx context.Context, tenantId uuid.UUID, slotKey string, count int64) {
	m.activeSlotsByKeyGauge.Record(ctx, count,
		metric.WithAttributes(
			attribute.String("tenant_id", tenantId.String()),
			attribute.String("slot_key", slotKey),
		))
}

// RecordActiveWorkers records the number of active workers
func (m *MetricsRecorder) RecordActiveWorkers(ctx context.Context, tenantId uuid.UUID, count int64) {
	m.activeWorkersGauge.Record(ctx, count,
		metric.WithAttributes(attribute.String("tenant_id", tenantId.String())))
}

// RecordActiveSDKs records the number of active SDKs
func (m *MetricsRecorder) RecordActiveSDKs(ctx context.Context, tenantId uuid.UUID, sdk SDKInfo, count int64) {
	m.activeSDKsGauge.Record(ctx, count,
		metric.WithAttributes(
			attribute.String("tenant_id", tenantId.String()),
			attribute.String("sdk_language", sdk.Language),
			attribute.String("sdk_version", sdk.SdkVersion),
			attribute.String("sdk_os", sdk.OperatingSystem),
			attribute.String("sdk_language_version", sdk.LanguageVersion),
		))
}

// SDKInfo contains information about an SDK
type SDKInfo struct {
	OperatingSystem string
	Language        string
	LanguageVersion string
	SdkVersion      string
}
