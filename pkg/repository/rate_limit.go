package repository

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

type UpsertRateLimitOpts struct {
	// The rate limit max value
	Limit int

	// The rate limit duration
	Duration *string `validate:"omitnil,oneof=SECOND MINUTE HOUR DAY WEEK MONTH YEAR"`
}

type RateLimitEngineRepository interface {
	// CreateRateLimit creates a new rate limit record
	UpsertRateLimit(ctx context.Context, tenantId string, key string, opts *UpsertRateLimitOpts) (*dbsqlc.RateLimit, error)
}
