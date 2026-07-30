package repository

import (
	"context"
)

type UsageMetrics struct {
	TenantCount       int64
	UserCount         int64
	WorkflowCount     int64
	WorkerCount       int64
	WorkersByLanguage map[string]int64
}

type SecurityCheckRepository interface {
	GetIdent() (string, error)
	GetUsageMetrics(ctx context.Context) (*UsageMetrics, error)
}

type securityCheckRepository struct {
	*sharedRepository
}

func newSecurityCheckRepository(shared *sharedRepository) SecurityCheckRepository {
	return &securityCheckRepository{
		sharedRepository: shared,
	}
}

func (a *securityCheckRepository) GetIdent() (string, error) {
	id, err := a.queries.GetSecurityCheckIdent(context.Background(), a.pool)

	if err != nil {
		return "", err
	}

	return id.String(), nil
}

func (a *securityCheckRepository) GetUsageMetrics(ctx context.Context) (*UsageMetrics, error) {
	metrics := &UsageMetrics{
		WorkersByLanguage: map[string]int64{},
	}

	if err := a.pool.QueryRow(ctx, `SELECT COUNT(*) FROM "Tenant"`).Scan(&metrics.TenantCount); err != nil {
		return nil, err
	}

	if err := a.pool.QueryRow(ctx, `SELECT COUNT(*) FROM "User"`).Scan(&metrics.UserCount); err != nil {
		return nil, err
	}

	if err := a.pool.QueryRow(ctx, `SELECT COUNT(*) FROM "Workflow" WHERE "deletedAt" IS NULL`).Scan(&metrics.WorkflowCount); err != nil {
		return nil, err
	}

	if err := a.pool.QueryRow(ctx, `SELECT COUNT(*) FROM "Worker" WHERE "deletedAt" IS NULL`).Scan(&metrics.WorkerCount); err != nil {
		return nil, err
	}

	rows, err := a.pool.Query(ctx, `
		SELECT COALESCE("language"::text, 'UNKNOWN') AS lang, COUNT(*)
		FROM "Worker"
		WHERE "deletedAt" IS NULL
		GROUP BY "language"
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var lang string
		var count int64

		if err := rows.Scan(&lang, &count); err != nil {
			return nil, err
		}

		metrics.WorkersByLanguage[lang] = count
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return metrics, nil
}
