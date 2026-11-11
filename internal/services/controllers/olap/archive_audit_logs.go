package olap

import (
	"context"
	"errors"
	"time"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

const (
	auditLogTimeInDatabaseAfterPartitionEnds = 24 * time.Hour
	auditLogArchiveLeaseDuration             = 5 * time.Minute
)

func (oc *OLAPControllerImpl) runArchiveAuditLogs(ctx context.Context) func() {
	return func() {
		tenant, err := oc.p.GetInternalTenantForController(ctx)

		if err != nil {
			oc.l.Error().Ctx(ctx).Err(err).Msg("could not get internal tenant")
			return
		}

		if tenant == nil {
			return
		}

		err = oc.repo.OLAP().ArchiveAuditLogPartitions(ctx, v1.ArchiveAuditLogPartitionsOpts{
			PartitionsEndingBefore: time.Now().UTC().Add(-auditLogTimeInDatabaseAfterPartitionEnds),
			LeaseDuration:          auditLogArchiveLeaseDuration,
		})

		if errors.Is(err, v1.ErrPartitionLockConflict) {
			oc.l.Warn().Ctx(ctx).Msg("audit log archive: lock conflict with concurrent table operation, will retry at next interval")
			return
		}

		if err != nil {
			oc.l.Error().Ctx(ctx).Err(err).Msg("could not archive audit logs")
		}
	}
}
