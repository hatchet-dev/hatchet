package repository

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

type ListRateLimitOpts struct {
	// (optional) a search query for the key
	Search *string

	// (optional) number of events to skip
	Offset *int

	// (optional) number of events to return
	Limit *int

	// (optional) the order by field
	OrderBy *string `validate:"omitempty,oneof=key value limitValue"`

	// (optional) the order direction
	OrderDirection *string `validate:"omitempty,oneof=ASC DESC"`
}

type ListRateLimitsResult struct {
	Rows  []*dbsqlc.ListRateLimitsForTenantNoMutateRow
	Count int
}

type UpsertRateLimitOpts struct {
	// The rate limit max value
	Limit int

	// The rate limit duration
	Duration *string `validate:"omitnil,oneof=SECOND MINUTE HOUR DAY WEEK MONTH YEAR"`
}

type RateLimitEngineRepository interface {
	ListRateLimits(ctx context.Context, tenantId string, opts *ListRateLimitOpts) (*ListRateLimitsResult, error)

	// CreateRateLimit creates a new rate limit record
	UpsertRateLimit(ctx context.Context, tenantId string, key string, opts *UpsertRateLimitOpts) (*dbsqlc.RateLimit, error)
}
