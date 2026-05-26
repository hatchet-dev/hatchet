package retention

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/google/uuid"
)

func GetDataRetentionExpiredTime(duration string) (time.Time, error) {
	d, err := time.ParseDuration(duration)

	if err != nil {
		return time.Time{}, fmt.Errorf("could not parse duration: %w", err)
	}

	return time.Now().UTC().Add(-d), nil
}

func (rc *RetentionControllerImpl) ForTenants(ctx context.Context, perTenantTimeout time.Duration, f func(ctx context.Context, tenantId uuid.UUID) error) error {
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

	for _, tenantId := range tenants {
		g.Go(func() error {
			tenantCtx, cancel := context.WithTimeout(ctx, perTenantTimeout)
			defer cancel()

			if err := f(tenantCtx, tenantId); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("tenant %s: %w", tenantId.String(), err))
				mu.Unlock()
			}
			return nil
		})
	}

	_ = g.Wait()

	return errors.Join(errs...)
}
