package sqlcv1

// The scheduler represents workflow-scoped and tenant-scoped concurrency strategies with a
// single descriptor type (V1StepConcurrency); tenant-scoped strategies carry zero-uuid
// workflow columns. These converters build that descriptor from the query rows that return
// strategies from both tables.

func (r *ListActiveConcurrencyStrategiesRow) ToV1StepConcurrency() *V1StepConcurrency {
	return &V1StepConcurrency{
		ID:                r.ID,
		ParentStrategyID:  r.ParentStrategyID,
		WorkflowID:        r.WorkflowID,
		WorkflowVersionID: r.WorkflowVersionID,
		StepID:            r.StepID,
		IsActive:          r.IsActive,
		LastActiveAt:      r.LastActiveAt,
		Strategy:          r.Strategy,
		Expression:        r.Expression,
		TenantID:          r.TenantID,
		MaxConcurrency:    r.MaxConcurrency,
	}
}

func (r *GetConcurrencyStrategyByIdRow) ToV1StepConcurrency() *V1StepConcurrency {
	return &V1StepConcurrency{
		ID:                r.ID,
		ParentStrategyID:  r.ParentStrategyID,
		WorkflowID:        r.WorkflowID,
		WorkflowVersionID: r.WorkflowVersionID,
		StepID:            r.StepID,
		IsActive:          r.IsActive,
		LastActiveAt:      r.LastActiveAt,
		Strategy:          r.Strategy,
		Expression:        r.Expression,
		TenantID:          r.TenantID,
		MaxConcurrency:    r.MaxConcurrency,
	}
}

func (r *ListConcurrencyStrategiesByStepIdRow) ToV1StepConcurrency() *V1StepConcurrency {
	return &V1StepConcurrency{
		ID:                r.ID,
		ParentStrategyID:  r.ParentStrategyID,
		WorkflowID:        r.WorkflowID,
		WorkflowVersionID: r.WorkflowVersionID,
		StepID:            r.StepID,
		IsActive:          r.IsActive,
		LastActiveAt:      r.LastActiveAt,
		Strategy:          r.Strategy,
		Expression:        r.Expression,
		TenantID:          r.TenantID,
		MaxConcurrency:    r.MaxConcurrency,
	}
}
