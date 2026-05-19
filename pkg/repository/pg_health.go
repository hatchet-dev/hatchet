package repository

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// PGHealthRepository provides database health monitoring functionality.
//
// Important: Many health checks rely on PostgreSQL's statistics collector,
// which requires track_counts = on (enabled by default). If track_counts
// is disabled, queries against pg_stat_user_tables and pg_statio_user_tables
// will return empty results, and health checks will report no data available.
//
// To verify track_counts is enabled, run:
//   SHOW track_counts;
//
// To enable track_counts (requires superuser or appropriate permissions):
//   ALTER SYSTEM SET track_counts = on;
//   SELECT pg_reload_conf();

type PGHealthError string

const (
	PGHealthAlert PGHealthError = "alert"
	PGHealthWarn  PGHealthError = "warn"
	PGHealthOK    PGHealthError = "ok"
)

type PGHealthRepository interface {
	PGStatStatementsEnabled(ctx context.Context) (bool, error)
	TrackCountsEnabled(ctx context.Context) (bool, error)
	CheckBloat(ctx context.Context) (PGHealthError, int, error)
	GetBloatDetails(ctx context.Context) ([]*sqlcv1.CheckBloatRow, error)
	CheckLongRunningQueries(ctx context.Context) (PGHealthError, int, error)
	CheckQueryCache(ctx context.Context) (PGHealthError, int, error)
	CheckQueryCaches(ctx context.Context) ([]*sqlcv1.CheckQueryCachesRow, error)
	CheckLongRunningVacuum(ctx context.Context) (PGHealthError, int, error)
	CheckLastAutovacuumForPartitionedTables(ctx context.Context) ([]*sqlcv1.CheckLastAutovacuumForPartitionedTablesRow, error)
	CheckLastAutovacuumForPartitionedTablesCoreDB(ctx context.Context) ([]*sqlcv1.CheckLastAutovacuumForPartitionedTablesCoreDBRow, error)
}

type pgHealthRepository struct {
	*sharedRepository
	pgStatStatementsCache *cache.Cache
	trackCountsCache      *cache.Cache
}

const (
	pgStatStatementsCacheKey = "pg_stat_statements_enabled"
	trackCountsCacheKey      = "track_counts_enabled"
)

func newPGHealthRepository(shared *sharedRepository) *pgHealthRepository {
	return &pgHealthRepository{
		sharedRepository:      shared,
		pgStatStatementsCache: cache.New(10 * time.Minute),
		trackCountsCache:      cache.New(10 * time.Minute),
	}
}

func (h *pgHealthRepository) PGStatStatementsEnabled(ctx context.Context) (bool, error) {
	if cached, ok := h.pgStatStatementsCache.Get(pgStatStatementsCacheKey); ok {
		if enabled, ok := cached.(bool); ok {
			return enabled, nil
		}
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	count, err := h.queries.CheckPGStatStatementsEnabled(ctx, h.pool)
	if err != nil {
		return false, err
	}

	enabled := count > 0
	h.pgStatStatementsCache.Set(pgStatStatementsCacheKey, enabled)

	return enabled, nil
}

func (h *pgHealthRepository) TrackCountsEnabled(ctx context.Context) (bool, error) {
	if cached, ok := h.trackCountsCache.Get(trackCountsCacheKey); ok {
		if enabled, ok := cached.(bool); ok {
			return enabled, nil
		}
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var setting string
	err := h.pool.QueryRow(ctx, "SHOW track_counts").Scan(&setting)
	if err != nil {
		return false, err
	}

	enabled := setting == "on"
	h.trackCountsCache.Set(trackCountsCacheKey, enabled)

	return enabled, nil
}

func (h *pgHealthRepository) CheckBloat(ctx context.Context) (PGHealthError, int, error) {

	enabled, err := h.PGStatStatementsEnabled(ctx)

	if err != nil {
		return PGHealthOK, 0, err
	}

	if !enabled {
		return PGHealthOK, 0, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := h.queries.CheckBloat(ctx, h.pool)
	if err != nil {
		return PGHealthOK, 0, err
	}

	if len(rows) == 0 {
		return PGHealthOK, 0, nil
	}

	return PGHealthWarn, len(rows), nil
}

func (h *pgHealthRepository) GetBloatDetails(ctx context.Context) ([]*sqlcv1.CheckBloatRow, error) {
	enabled, err := h.PGStatStatementsEnabled(ctx)

	if err != nil {
		return nil, err
	}

	if !enabled {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return h.queries.CheckBloat(ctx, h.pool)
}

func (h *pgHealthRepository) CheckLongRunningQueries(ctx context.Context) (PGHealthError, int, error) {

	enabled, err := h.PGStatStatementsEnabled(ctx)

	if err != nil {
		return PGHealthOK, 0, err
	}

	if !enabled {
		return PGHealthOK, 0, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := h.queries.CheckLongRunningQueries(ctx, h.pool)
	if err != nil {
		return PGHealthOK, 0, err
	}

	return PGHealthOK, len(rows), nil
}

func (h *pgHealthRepository) CheckQueryCache(ctx context.Context) (PGHealthError, int, error) {
	enabled, err := h.PGStatStatementsEnabled(ctx)

	if err != nil {
		return PGHealthOK, 0, err
	}

	if !enabled {
		return PGHealthOK, 0, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	tables, err := h.queries.CheckQueryCaches(ctx, h.pool)
	if err != nil {
		return PGHealthOK, 0, err
	}

	problemTables := make(map[string]float64)

	for _, table := range tables {
		hitRatio := table.CacheHitRatioPct

		if hitRatio < 95.0 && hitRatio > 10.0 {
			problemTables[table.Tablename.String] = hitRatio
		}
	}

	if len(problemTables) > 0 {
		return PGHealthWarn, len(problemTables), nil
	}

	return PGHealthOK, 0, nil
}

func (h *pgHealthRepository) CheckQueryCaches(ctx context.Context) ([]*sqlcv1.CheckQueryCachesRow, error) {
	enabled, err := h.PGStatStatementsEnabled(ctx)

	if err != nil {
		return nil, err
	}

	if !enabled {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return h.queries.CheckQueryCaches(ctx, h.pool)
}

func (h *pgHealthRepository) CheckLongRunningVacuum(ctx context.Context) (PGHealthError, int, error) {

	enabled, err := h.PGStatStatementsEnabled(ctx)

	if err != nil {
		return PGHealthOK, 0, err
	}

	if !enabled {
		return PGHealthOK, 0, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := h.queries.LongRunningVacuum(ctx, h.pool)
	if err != nil {
		return PGHealthOK, 0, err
	}

	if len(rows) == 0 {
		return PGHealthOK, 0, nil
	}

	for _, row := range rows {
		if row.ElapsedTime > int32(time.Hour.Seconds()*10) {
			return PGHealthAlert, len(rows), nil
		}
	}

	return PGHealthWarn, len(rows), nil
}

func (h *pgHealthRepository) CheckLastAutovacuumForPartitionedTables(ctx context.Context) ([]*sqlcv1.CheckLastAutovacuumForPartitionedTablesRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return h.queries.CheckLastAutovacuumForPartitionedTables(ctx, h.pool)
}

func (h *pgHealthRepository) CheckLastAutovacuumForPartitionedTablesCoreDB(ctx context.Context) ([]*sqlcv1.CheckLastAutovacuumForPartitionedTablesCoreDBRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return h.queries.CheckLastAutovacuumForPartitionedTablesCoreDB(ctx, h.pool)
}
