package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// acquirePartitionLease tries to acquire a system-wide partition lease using the existing
// Lease table.
func (r *sharedRepository) acquirePartitionLease(ctx context.Context, db sqlcv1.DBTX, leaseKey string) ([]*sqlcv1.Lease, error) {
	leases, err := r.queries.AcquireOrExtendLeases(ctx, db, sqlcv1.AcquireOrExtendLeasesParams{
		// 15-minute window
		LeaseDuration:    pgtype.Interval{Microseconds: 15 * 60 * 1_000_000, Valid: true},
		Kind:             sqlcv1.LeaseKindTABLEPARTITIONMAINTENANCE,
		Resourceids:      []string{leaseKey},
		Tenantid:         uuid.Nil,
		Existingleaseids: []int64{},
	})
	if err != nil {
		return nil, err
	}

	return leases, nil
}

func (r *sharedRepository) releasePartitionLease(ctx context.Context, db sqlcv1.DBTX, leases []*sqlcv1.Lease) error {
	leaseIds := make([]int64, len(leases))
	for i, l := range leases {
		leaseIds[i] = l.ID
	}
	_, err := r.queries.ReleaseLeases(ctx, db, leaseIds)
	return err
}
