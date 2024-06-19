package repository

type HealthRepository interface {
	IsHealthy() bool
}
