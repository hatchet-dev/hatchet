package transformers

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func ToRateLimitFromSQLC(rl *dbsqlc.ListRateLimitsForTenantNoMutateRow) (*gen.RateLimit, error) {
	res := &gen.RateLimit{
		Key:        rl.Key,
		TenantId:   sqlchelpers.UUIDToStr(rl.TenantId),
		LastRefill: rl.LastRefill.Time,
		LimitValue: int(rl.LimitValue),
		Value:      int(rl.Value),
		Window:     rl.Window,
	}

	return res, nil
}
