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

type UsageMetricsRepository interface {
	GetUsageMetrics(ctx context.Context) (*UsageMetrics, error)
}

type usageMetricsRepository struct {
	*sharedRepository
}

func newUsageMetricsRepository(shared *sharedRepository) UsageMetricsRepository {
	return &usageMetricsRepository{
		sharedRepository: shared,
	}
}

func (r *usageMetricsRepository) GetUsageMetrics(ctx context.Context) (*UsageMetrics, error) {
	metrics := &UsageMetrics{
		WorkersByLanguage: map[string]int64{},
	}

	tenantCount, err := r.queries.CountAllTenants(ctx, r.pool)
	if err != nil {
		return nil, err
	}
	metrics.TenantCount = tenantCount

	userCount, err := r.queries.CountAllUsers(ctx, r.pool)
	if err != nil {
		return nil, err
	}
	metrics.UserCount = userCount

	workflowCount, err := r.queries.CountAllWorkflows(ctx, r.pool)
	if err != nil {
		return nil, err
	}
	metrics.WorkflowCount = workflowCount

	workerCount, err := r.queries.CountAllWorkers(ctx, r.pool)
	if err != nil {
		return nil, err
	}
	metrics.WorkerCount = workerCount

	byLanguage, err := r.queries.CountAllWorkersByLanguage(ctx, r.pool)
	if err != nil {
		return nil, err
	}

	for _, row := range byLanguage {
		metrics.WorkersByLanguage[row.Language] = row.Count
	}

	return metrics, nil
}
