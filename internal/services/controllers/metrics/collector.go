package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/services/partition"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

// MetricsCollector collects and reports database and system metrics to OTel
type MetricsCollector interface {
	Start() (func() error, error)
}

type MetricsCollectorImpl struct {
	l        *zerolog.Logger
	repo     v1.Repository
	recorder *telemetry.MetricsRecorder
	s        gocron.Scheduler
	a        *hatcheterrors.Wrapped
	p        *partition.Partition
}

type MetricsCollectorOpt func(*MetricsCollectorOpts)

type MetricsCollectorOpts struct {
	l       *zerolog.Logger
	repo    v1.Repository
	alerter hatcheterrors.Alerter
	p       *partition.Partition
}

func defaultMetricsCollectorOpts() *MetricsCollectorOpts {
	l := logger.NewDefaultLogger("metrics-collector")
	alerter := hatcheterrors.NoOpAlerter{}

	return &MetricsCollectorOpts{
		l:       &l,
		alerter: alerter,
	}
}

func WithLogger(l *zerolog.Logger) MetricsCollectorOpt {
	return func(opts *MetricsCollectorOpts) {
		opts.l = l
	}
}

func WithRepository(r v1.Repository) MetricsCollectorOpt {
	return func(opts *MetricsCollectorOpts) {
		opts.repo = r
	}
}

func WithAlerter(a hatcheterrors.Alerter) MetricsCollectorOpt {
	return func(opts *MetricsCollectorOpts) {
		opts.alerter = a
	}
}

func WithPartition(p *partition.Partition) MetricsCollectorOpt {
	return func(opts *MetricsCollectorOpts) {
		opts.p = p
	}
}

func New(fs ...MetricsCollectorOpt) (*MetricsCollectorImpl, error) {
	opts := defaultMetricsCollectorOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.repo == nil {
		return nil, fmt.Errorf("repository is required. use WithRepository")
	}

	if opts.p == nil {
		return nil, fmt.Errorf("partition is required. use WithPartition")
	}

	newLogger := opts.l.With().Str("service", "metrics-collector").Logger()
	opts.l = &newLogger

	recorder, err := telemetry.NewMetricsRecorder(context.Background())
	if err != nil {
		return nil, fmt.Errorf("could not create metrics recorder: %w", err)
	}

	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))
	if err != nil {
		return nil, fmt.Errorf("could not create scheduler: %w", err)
	}

	a := hatcheterrors.NewWrapped(opts.alerter)
	a.WithData(map[string]interface{}{"service": "metrics-collector"})

	return &MetricsCollectorImpl{
		l:        opts.l,
		repo:     opts.repo,
		recorder: recorder,
		s:        s,
		a:        a,
		p:        opts.p,
	}, nil
}

func (mc *MetricsCollectorImpl) Start() (func() error, error) {
	mc.s.Start()

	ctx := context.Background()

	// Collect database health metrics every 60 seconds
	_, err := mc.s.NewJob(
		gocron.DurationJob(60*time.Second),
		gocron.NewTask(mc.collectDatabaseHealthMetrics(ctx)),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)
	if err != nil {
		return nil, fmt.Errorf("could not schedule database health metrics collection: %w", err)
	}

	// Collect OLAP metrics every 5 minutes
	_, err = mc.s.NewJob(
		gocron.DurationJob(5*time.Minute),
		gocron.NewTask(mc.collectOLAPMetrics(ctx)),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)
	if err != nil {
		return nil, fmt.Errorf("could not schedule OLAP metrics collection: %w", err)
	}

	// Collect worker metrics every 60 seconds
	_, err = mc.s.NewJob(
		gocron.DurationJob(60*time.Second),
		gocron.NewTask(mc.collectWorkerMetrics(ctx)),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)
	if err != nil {
		return nil, fmt.Errorf("could not schedule worker metrics collection: %w", err)
	}

	// Collect yesterday's run count once per day at midnight
	_, err = mc.s.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(0, 5, 0))),
		gocron.NewTask(mc.collectYesterdayRunCounts(ctx)),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)
	if err != nil {
		return nil, fmt.Errorf("could not schedule yesterday run counts collection: %w", err)
	}

	cleanup := func() error {
		if err := mc.s.Shutdown(); err != nil {
			return fmt.Errorf("could not shutdown scheduler: %w", err)
		}
		return nil
	}

	return cleanup, nil
}

func (mc *MetricsCollectorImpl) collectDatabaseHealthMetrics(ctx context.Context) func() {
	return func() {
		ctx, span := telemetry.NewSpan(ctx, "MetricsCollector.collectDatabaseHealthMetrics")
		defer span.End()

		// Only run on the engine instance that has control over the internal tenant
		tenant, err := mc.p.GetInternalTenantForController(ctx)
		if err != nil {
			mc.l.Error().Err(err).Msg("could not get internal tenant")
			return
		}

		if tenant == nil {
			// This engine instance doesn't have control over the internal tenant
			return
		}

		mc.l.Debug().Msg("collecting database health metrics")

		// Check bloat
		bloatStatus, bloatCount, err := mc.repo.PGHealth().CheckBloat(ctx)
		if err != nil {
			mc.l.Error().Err(err).Msg("failed to check database bloat")
		} else {
			mc.recorder.RecordDBBloat(ctx, int64(bloatCount), string(bloatStatus))
			mc.l.Debug().Int("count", bloatCount).Str("status", string(bloatStatus)).Msg("recorded database bloat metric")
		}

		// Get detailed bloat metrics per table
		bloatDetails, err := mc.repo.PGHealth().GetBloatDetails(ctx)
		if err != nil {
			mc.l.Error().Err(err).Msg("failed to get bloat details")
		} else if len(bloatDetails) > 0 {
			mc.l.Info().Int("table_count", len(bloatDetails)).Msg("recording bloat details per table")
			for _, row := range bloatDetails {
				if row.DeadPct.Valid {
					deadPct, err := row.DeadPct.Float64Value()
					if err == nil {
						tableName := row.Tablename.String
						mc.recorder.RecordDBBloatPercent(ctx, tableName, deadPct.Float64)
						mc.l.Debug().
							Str("table", tableName).
							Float64("dead_pct", deadPct.Float64).
							Msg("recorded bloat percent metric")
					}
				}
			}
		}

		// Check long-running queries
		_, longRunningCount, err := mc.repo.PGHealth().CheckLongRunningQueries(ctx)
		if err != nil {
			mc.l.Error().Err(err).Msg("failed to check long-running queries")
		} else {
			mc.recorder.RecordDBLongRunningQueries(ctx, int64(longRunningCount))
			mc.l.Debug().Int("count", longRunningCount).Msg("recorded long-running queries metric")
		}

		// Check query cache hit ratios
		tables, err := mc.repo.PGHealth().CheckQueryCaches(ctx)
		if err != nil {
			mc.l.Error().Err(err).Msg("failed to check query cache")
		} else if len(tables) == 0 {
			mc.l.Info().Msg("no query cache data available (pg_stat_statements may not be enabled)")
		} else {
			mc.l.Info().Int("table_count", len(tables)).Msg("recording query cache hit ratios")
			for _, table := range tables {
				tableName := table.Tablename.String
				hitRatio := table.CacheHitRatioPct
				mc.recorder.RecordDBQueryCacheHitRatio(ctx, tableName, hitRatio)
				mc.l.Debug().
					Str("table", tableName).
					Float64("hit_ratio", hitRatio).
					Msg("recorded query cache hit ratio metric")
			}
		}

		// Check long-running vacuum
		vacuumStatus, vacuumCount, err := mc.repo.PGHealth().CheckLongRunningVacuum(ctx)
		if err != nil {
			mc.l.Error().Err(err).Msg("failed to check long-running vacuum")
		} else {
			mc.recorder.RecordDBLongRunningVacuum(ctx, int64(vacuumCount), string(vacuumStatus))
			mc.l.Debug().Int("count", vacuumCount).Str("status", string(vacuumStatus)).Msg("recorded long-running vacuum metric")
		}

		autovacuumRows, err := mc.repo.PGHealth().CheckLastAutovacuumForPartitionedTables(ctx)
		if err != nil {
			mc.l.Error().Err(err).Msg("failed to check last autovacuum for partitioned tables (OLAP DB)")
		} else if len(autovacuumRows) == 0 {
			mc.l.Warn().Msg("no partitioned tables found for autovacuum tracking (OLAP DB)")
		} else {
			mc.l.Info().Int("table_count", len(autovacuumRows)).Msg("recording last autovacuum metrics (OLAP DB)")
			validCount := 0
			for _, row := range autovacuumRows {
				if row.SecondsSinceLastAutovacuum.Valid {
					seconds, err := row.SecondsSinceLastAutovacuum.Float64Value()
					if err == nil {
						tableName := row.Tablename.String
						mc.recorder.RecordDBLastAutovacuumSecondsSince(ctx, tableName, seconds.Float64)
						mc.l.Debug().
							Str("table", tableName).
							Float64("seconds_since", seconds.Float64).
							Msg("recorded last autovacuum metric (OLAP DB)")
						validCount++
					}
				}
			}
			if validCount == 0 {
				mc.l.Warn().Int("table_count", len(autovacuumRows)).Msg("found partitioned tables but none have been autovacuumed yet (OLAP DB)")
			}
		}

		autovacuumRowsCoreDB, err := mc.repo.PGHealth().CheckLastAutovacuumForPartitionedTablesCoreDB(ctx)
		if err != nil {
			mc.l.Error().Err(err).Msg("failed to check last autovacuum for partitioned tables (CORE DB)")
		} else if len(autovacuumRowsCoreDB) == 0 {
			mc.l.Warn().Msg("no partitioned tables found for autovacuum tracking (CORE DB)")
		} else {
			mc.l.Info().Int("table_count", len(autovacuumRowsCoreDB)).Msg("recording last autovacuum metrics (CORE DB)")
			validCount := 0
			for _, row := range autovacuumRowsCoreDB {
				if row.SecondsSinceLastAutovacuum.Valid {
					seconds, err := row.SecondsSinceLastAutovacuum.Float64Value()
					if err == nil {
						tableName := row.Tablename.String
						mc.recorder.RecordDBLastAutovacuumSecondsSince(ctx, tableName, seconds.Float64)
						mc.l.Debug().
							Str("table", tableName).
							Float64("seconds_since", seconds.Float64).
							Msg("recorded last autovacuum metric (CORE DB)")
						validCount++
					}
				}
			}
			if validCount == 0 {
				mc.l.Warn().Int("table_count", len(autovacuumRowsCoreDB)).Msg("found partitioned tables but none have been autovacuumed yet (CORE DB)")
			}
		}

		mc.l.Debug().Msg("finished collecting database health metrics")
	}
}

func (mc *MetricsCollectorImpl) collectOLAPMetrics(ctx context.Context) func() {
	return func() {
		ctx, span := telemetry.NewSpan(ctx, "MetricsCollector.collectOLAPMetrics")
		defer span.End()

		// Only run on the engine instance that has control over the internal tenant
		tenant, err := mc.p.GetInternalTenantForController(ctx)
		if err != nil {
			mc.l.Error().Err(err).Msg("could not get internal tenant")
			return
		}

		if tenant == nil {
			// This engine instance doesn't have control over the internal tenant
			return
		}

		mc.l.Debug().Msg("collecting OLAP metrics")

		// Count DAG status updates temp table size (instance-wide)
		dagSize, err := mc.repo.OLAP().CountOLAPTempTableSizeForDAGStatusUpdates(ctx)
		if err != nil {
			mc.l.Error().Err(err).Msg("failed to count DAG temp table size")
		} else {
			mc.recorder.RecordOLAPTempTableSizeDAG(ctx, dagSize)
			mc.l.Debug().Int64("size", dagSize).Msg("recorded DAG temp table size metric")
		}

		// Count task status updates temp table size (instance-wide)
		taskSize, err := mc.repo.OLAP().CountOLAPTempTableSizeForTaskStatusUpdates(ctx)
		if err != nil {
			mc.l.Error().Err(err).Msg("failed to count task temp table size")
		} else {
			mc.recorder.RecordOLAPTempTableSizeTask(ctx, taskSize)
			mc.l.Debug().Int64("size", taskSize).Msg("recorded task temp table size metric")
		}

		mc.l.Debug().Msg("finished collecting OLAP metrics")
	}
}

func (mc *MetricsCollectorImpl) collectYesterdayRunCounts(ctx context.Context) func() {
	return func() {
		ctx, span := telemetry.NewSpan(ctx, "MetricsCollector.collectYesterdayRunCounts")
		defer span.End()

		// Only run on the engine instance that has control over the internal tenant
		tenant, err := mc.p.GetInternalTenantForController(ctx)
		if err != nil {
			mc.l.Error().Err(err).Msg("could not get internal tenant")
			return
		}

		if tenant == nil {
			// This engine instance doesn't have control over the internal tenant
			return
		}

		mc.l.Debug().Msg("collecting yesterday's run counts")

		// Get yesterday's run counts by status (instance-wide)
		runCounts, err := mc.repo.OLAP().ListYesterdayRunCountsByStatus(ctx)
		if err != nil {
			mc.l.Error().Err(err).Msg("failed to get yesterday's run counts")
			return
		}

		for status, count := range runCounts {
			mc.recorder.RecordYesterdayRunCount(ctx, string(status), count)
			mc.l.Debug().Str("status", string(status)).Int64("count", count).Msg("recorded yesterday run count metric")
		}

		mc.l.Debug().Msg("finished collecting yesterday's run counts")
	}
}

func (mc *MetricsCollectorImpl) collectWorkerMetrics(ctx context.Context) func() {
	return func() {
		ctx, span := telemetry.NewSpan(ctx, "MetricsCollector.collectWorkerMetrics")
		defer span.End()

		// Only run on the engine instance that has control over the internal tenant
		tenant, err := mc.p.GetInternalTenantForController(ctx)
		if err != nil {
			mc.l.Error().Err(err).Msg("could not get internal tenant")
			return
		}

		if tenant == nil {
			// This engine instance doesn't have control over the internal tenant
			return
		}

		mc.l.Debug().Msg("collecting worker metrics")

		// Count active slots per tenant
		activeSlots, err := mc.repo.Workers().CountActiveSlotsPerTenant()
		if err != nil {
			mc.l.Error().Err(err).Msg("failed to count active slots per tenant")
		} else if len(activeSlots) == 0 {
			mc.l.Debug().Msg("no active worker slots found")
		} else {
			mc.l.Info().Int("tenant_count", len(activeSlots)).Msg("recording active slots metrics")
			for tenantId, count := range activeSlots {
				mc.recorder.RecordActiveSlots(ctx, tenantId, count)
				mc.l.Debug().Str("tenant_id", tenantId).Int64("count", count).Msg("recorded active slots metric")
			}
		}

		// Count active workers per tenant
		activeWorkers, err := mc.repo.Workers().CountActiveWorkersPerTenant()
		if err != nil {
			mc.l.Error().Err(err).Msg("failed to count active workers per tenant")
		} else if len(activeWorkers) == 0 {
			mc.l.Debug().Msg("no active workers found")
		} else {
			mc.l.Info().Int("tenant_count", len(activeWorkers)).Msg("recording active workers metrics")
			for tenantId, count := range activeWorkers {
				mc.recorder.RecordActiveWorkers(ctx, tenantId, count)
				mc.l.Debug().Str("tenant_id", tenantId).Int64("count", count).Msg("recorded active workers metric")
			}
		}

		// Count active SDKs per tenant
		activeSDKs, err := mc.repo.Workers().ListActiveSDKsPerTenant()
		if err != nil {
			mc.l.Error().Err(err).Msg("failed to list active SDKs per tenant")
		} else if len(activeSDKs) == 0 {
			mc.l.Debug().Msg("no active SDKs found")
		} else {
			mc.l.Info().Int("sdk_count", len(activeSDKs)).Msg("recording active SDKs metrics")
			for tuple, count := range activeSDKs {
				sdkInfo := telemetry.SDKInfo{
					OperatingSystem: tuple.SDK.OperatingSystem,
					Language:        tuple.SDK.Language,
					LanguageVersion: tuple.SDK.LanguageVersion,
					SdkVersion:      tuple.SDK.SdkVersion,
				}
				mc.recorder.RecordActiveSDKs(ctx, tuple.TenantId, sdkInfo, count)
				mc.l.Debug().
					Str("tenant_id", tuple.TenantId).
					Int64("count", count).
					Str("sdk_language", sdkInfo.Language).
					Str("sdk_version", sdkInfo.SdkVersion).
					Msg("recorded active SDKs metric")
			}
		}

		mc.l.Debug().Msg("finished collecting worker metrics")
	}
}
