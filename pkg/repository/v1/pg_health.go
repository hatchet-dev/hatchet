package v1

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
)

type PGHealthError string

const (
	PGHealthAlert PGHealthError = "alert"
	PGHealthWarn  PGHealthError = "warn"
	PGHealthOK    PGHealthError = "ok"
)

type PGHealthRepository interface {
	PGStatStatementsEnabled(ctx context.Context) (bool, error)
	CheckBloat(ctx context.Context) (PGHealthError, int, error)
}

type pgHealthRepository struct {
	*sharedRepository
	pgStatStatementsCache *cache.Cache
}

const pgStatStatementsCacheKey = "pg_stat_statements_enabled"

func newPGHealthRepository(shared *sharedRepository) *pgHealthRepository {
	return &pgHealthRepository{
		sharedRepository:      shared,
		pgStatStatementsCache: cache.New(10 * time.Minute),
	}
}

func (h *pgHealthRepository) PGStatStatementsEnabled(ctx context.Context) (bool, error) {
	if cached, ok := h.pgStatStatementsCache.Get(pgStatStatementsCacheKey); ok {
		if enabled, ok := cached.(bool); ok {
			return enabled, nil
		}
	}

	cxt, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	count, err := h.queries.CheckPGStatStatementsEnabled(cxt, h.pool)
	if err != nil {
		return false, err
	}

	enabled := count > 0
	h.pgStatStatementsCache.Set(pgStatStatementsCacheKey, enabled)

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

	cxt, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := h.queries.CheckBloat(cxt, h.pool)
	if err != nil {
		return PGHealthOK, 0, err
	}

	if len(rows) == 0 {
		return PGHealthOK, 0, nil
	}

	return PGHealthWarn, len(rows), nil
}

func (h *pgHealthRepository) CheckLongRunningQueries(ctx context.Context) (PGHealthError, int, error) {

	enabled, err := h.PGStatStatementsEnabled(ctx)

	if err != nil {
		return PGHealthOK, 0, err
	}

	if !enabled {
		return PGHealthOK, 0, nil
	}

	cxt, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := h.queries.CheckLongRunningQueries(cxt, h.pool)
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

	cxt, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	tables, err := h.queries.CheckQueryCaches(cxt, h.pool)
	if err != nil {
		return PGHealthOK, 0, err
	}

	problemTables := make(map[string]float64)

	for _, table := range tables {
		hitRatio := table.CacheHitRatioPct
		if err != nil {
			return PGHealthOK, 0, err
		}
		if hitRatio < 95.0 && hitRatio > 10.0 {
			problemTables[table.Tablename.String] = hitRatio
		}
	}

	if len(problemTables) > 0 {
		return PGHealthWarn, len(problemTables), nil
	}

	return PGHealthOK, 0, nil
}

func (h *pgHealthRepository) CheckLongRunningVacuum(ctx context.Context) (PGHealthError, int, error) {

	enabled, err := h.PGStatStatementsEnabled(ctx)

	if err != nil {
		return PGHealthOK, 0, err
	}

	if !enabled {
		return PGHealthOK, 0, nil
	}

	cxt, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := h.queries.LongRunningVacuum(cxt, h.pool)
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
