package scheduler

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	repov1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

const (
	tenantBacklogPollInterval = 15 * time.Second

	// tenantBacklogPollTimeout bounds one pass. Each count stops at the ceiling, so a pass
	// over every tenant on the partition stays short however deep its queues are.
	tenantBacklogPollTimeout = 15 * time.Second

	// tenantBacklogCeiling caps each state's count in a tenant's backlog measurement, which
	// bounds its cost however large the pile is.
	tenantBacklogCeiling = 250_000

	// tenantBacklogFloor is the smallest total backlog that gets hatchet_tenant_backlog series.
	// Most tenants sit below it, which keeps the gauge sparse.
	tenantBacklogFloor = 1_000
)

// tenantBacklogPoller measures the backlog of every tenant on this scheduler partition and
// reports it as hatchet_tenant_backlog. It is only accessed from a singleton gocron job, so
// its state needs no locking.
type tenantBacklogPoller struct {
	tasks repov1.TaskRepository
	l     *zerolog.Logger

	// reported is the set of hatchet_tenant_backlog series.
	reported map[backlogSeriesKey]struct{}
}

type backlogSeriesKey struct {
	tenantId uuid.UUID
	state    string
}

func newTenantBacklogPoller(tasks repov1.TaskRepository, l *zerolog.Logger) *tenantBacklogPoller {
	return &tenantBacklogPoller{
		tasks:    tasks,
		l:        l,
		reported: make(map[backlogSeriesKey]struct{}),
	}
}

// poll measures every given tenant's backlog in one repository call and reports it. On an
// error the previous series stay until a later pass succeeds.
func (p *tenantBacklogPoller) poll(ctx context.Context, tenantIds []uuid.UUID) {
	backlogs, err := p.tasks.GetTenantBacklogs(ctx, tenantIds, tenantBacklogCeiling)

	if err != nil {
		p.l.Error().Err(err).Msg("could not poll tenant backlogs")
		return
	}

	p.apply(backlogs)
}

// apply leaves exactly one series per non-empty state of each tenant in backlogs whose total
// is at or above the floor. Every other series is deleted: a tenant that dropped below the
// floor or left the partition, and a state that emptied.
func (p *tenantBacklogPoller) apply(backlogs map[uuid.UUID]repov1.TenantBacklog) {
	reported := make(map[backlogSeriesKey]struct{})

	for tenantId, backlog := range backlogs {
		if backlog.Total() < tenantBacklogFloor {
			continue
		}

		for _, s := range backlog.States() {
			if s.Count == 0 {
				continue
			}

			reported[backlogSeriesKey{tenantId: tenantId, state: s.State}] = struct{}{}
			prometheus.TenantBacklog.WithLabelValues(tenantId.String(), s.State).Set(float64(s.Count))
		}
	}

	for k := range p.reported {
		if _, ok := reported[k]; !ok {
			prometheus.TenantBacklog.DeleteLabelValues(k.tenantId.String(), k.state)
		}
	}

	p.reported = reported
}

// runPollTenantBacklogs measures every tenant on the partition, whether or not its
// hatchet_tenant_* metrics are enabled: the backlog is an operator signal.
func (s *Scheduler) runPollTenantBacklogs(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, tenantBacklogPollTimeout)
		defer cancel()

		tenants, err := s.repov1.Tenant().ListTenantsBySchedulerPartition(ctx, s.p.GetSchedulerPartitionId())

		if err != nil {
			s.l.Error().Err(err).Msg("could not list tenants for tenant backlogs")
			return
		}

		tenantIds := make([]uuid.UUID, 0, len(tenants))

		for _, tenant := range tenants {
			tenantIds = append(tenantIds, tenant.ID)
		}

		s.tenantBacklogs.poll(ctx, tenantIds)
	}
}
