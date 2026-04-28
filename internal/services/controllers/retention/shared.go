package retention

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func GetDataRetentionExpiredTime(duration string) (time.Time, error) {
	d, err := time.ParseDuration(duration)

	if err != nil {
		return time.Time{}, fmt.Errorf("could not parse duration: %w", err)
	}

	return time.Now().UTC().Add(-d), nil
}

func (rc *RetentionControllerImpl) ForTenants(ctx context.Context, perTenantTimeout time.Duration, f func(ctx context.Context, tenant sqlcv1.Tenant) error) error {
	tenants, err := rc.p.ListTenantsForController(ctx)

	if err != nil {
		return fmt.Errorf("could not list tenants: %w", err)
	}

	g := new(errgroup.Group)
	g.SetLimit(50)

	var (
		mu   sync.Mutex
		errs []error
	)

	for _, tenant := range tenants {
		g.Go(func() error {
			tenantCtx, cancel := context.WithTimeout(ctx, perTenantTimeout)
			defer cancel()

			if err := f(tenantCtx, *tenant); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("tenant %s: %w", tenant.ID.String(), err))
				mu.Unlock()
			}
			return nil
		})
	}

	_ = g.Wait()

	return errors.Join(errs...)
}
