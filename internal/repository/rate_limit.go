package repository

import "github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"

type CreateRateLimitOpts struct {
	// The rate limit key
	Key string `validate:"required"`

	// The rate limit max value
	Max int `validate:"required"`

	// The rate limit unit
	Unit string `validate:"required,oneof=second minute hour"`
}

type RateLimitEngineRepository interface {
	// CreateRateLimit creates a new rate limit record
	CreateRateLimit(tenantId string, opts *CreateRateLimitOpts) (*dbsqlc.RateLimit, error)
}
