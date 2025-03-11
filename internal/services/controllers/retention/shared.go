package retention

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

func GetDataRetentionExpiredTime(duration string) (time.Time, error) {
	d, err := time.ParseDuration(duration)

	if err != nil {
		return time.Time{}, fmt.Errorf("could not parse duration: %w", err)
	}

	return time.Now().UTC().Add(-d), nil
}

func (wc *RetentionControllerImpl) ForTenants(ctx context.Context, f func(ctx context.Context, tenant dbsqlc.Tenant) error) error {

	// list all tenants
	tenants, err := wc.p.ListTenantsForController(ctx, dbsqlc.TenantMajorEngineVersionV0)

	if err != nil {
		return fmt.Errorf("could not list tenants: %w", err)
	}

	g := new(errgroup.Group)

	for i := range tenants {
		index := i
		g.Go(func() error {
			return f(ctx, *tenants[index])
		})
	}

	err = g.Wait()

	if err != nil {
		return fmt.Errorf("could not run for tenants: %w", err)
	}

	return nil
}
