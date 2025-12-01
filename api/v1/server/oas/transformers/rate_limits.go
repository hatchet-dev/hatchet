package transformers

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

func ToRateLimitFromSQLC(rl *dbsqlc.ListRateLimitsForTenantNoMutateRow) (*gen.RateLimit, error) {
	res := &gen.RateLimit{
		Key:        rl.Key,
		TenantId:   rl.TenantId.String(),
		LastRefill: rl.LastRefill.Time,
		LimitValue: int(rl.LimitValue),
		Value:      int(rl.Value),
		Window:     rl.Window,
	}

	return res, nil
}
