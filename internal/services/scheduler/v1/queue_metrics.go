package scheduler

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	repov1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const (
	queueMetricsPollInterval = 15 * time.Second

	// queueMetricsPollTimeout bounds one full polling pass. The per-tenant aggregate
	// queries scan the tenant's queue items, so a pass over all tenants on the partition
	// can be slow when queues are deep.
	queueMetricsPollTimeout = 60 * time.Second
)

type queueSizeKey struct {
	queue        string
	workflowName string
}

type metadataQueueSizeKey struct {
	queue string
	key   string
	value string
}

// queueMetricsPoller periodically reports queue backlog gauges for the tenants on this
// scheduler partition. It is only accessed from a singleton gocron job, so its state
// needs no locking.
type queueMetricsPoller struct {
	tasks repov1.TaskRepository
	l     *zerolog.Logger

	knownQueueSizes    map[uuid.UUID]map[queueSizeKey]bool
	knownMetadataSizes map[uuid.UUID]map[metadataQueueSizeKey]bool
}

func newQueueMetricsPoller(tasks repov1.TaskRepository, l *zerolog.Logger) *queueMetricsPoller {
	return &queueMetricsPoller{
		tasks:              tasks,
		l:                  l,
		knownQueueSizes:    make(map[uuid.UUID]map[queueSizeKey]bool),
		knownMetadataSizes: make(map[uuid.UUID]map[metadataQueueSizeKey]bool),
	}
}

// diffStaleSeries diffs the series reported this poll against previously-known ones.
// known maps each series to whether it was live (true) or already zeroed (false) on the
// previous poll. A series which disappears is zeroed for one poll — so scrapers observe
// the drop to 0, e.g. for scale-to-zero autoscaling — and deleted the poll after. The
// returned map is the known set for the next poll.
func diffStaleSeries[K comparable](known map[K]bool, current map[K]struct{}) (zeroed, deleted []K, next map[K]bool) {
	next = make(map[K]bool, len(current))

	for k := range current {
		next[k] = true
	}

	for k, live := range known {
		if _, ok := current[k]; ok {
			continue
		}

		if live {
			zeroed = append(zeroed, k)
			next[k] = false
		} else {
			deleted = append(deleted, k)
		}
	}

	return zeroed, deleted, next
}

func (p *queueMetricsPoller) poll(ctx context.Context, tenantIds []uuid.UUID) {
	activeTenants := make(map[uuid.UUID]struct{}, len(tenantIds))

	for _, tenantId := range tenantIds {
		activeTenants[tenantId] = struct{}{}
		p.pollTenant(ctx, tenantId)
	}

	// tenants which left the partition or lost metrics entitlement drain to zero and are
	// then deleted, like any other disappeared series
	for tenantId := range p.knownQueueSizes {
		if _, ok := activeTenants[tenantId]; !ok {
			p.applyQueueSizes(tenantId, nil)
		}
	}

	for tenantId := range p.knownMetadataSizes {
		if _, ok := activeTenants[tenantId]; !ok {
			p.applyMetadataSizes(tenantId, nil)
		}
	}
}

// pollTenant queries and reports both queue size gauges for one tenant. On a query error
// the tenant's previously-reported values are kept rather than reporting false zeroes.
func (p *queueMetricsPoller) pollTenant(ctx context.Context, tenantId uuid.UUID) {
	sizes, err := p.tasks.GetQueueSizes(ctx, tenantId)

	if err != nil {
		p.l.Warn().Err(err).Str("tenant_id", tenantId.String()).Msg("could not poll queue sizes")
		return
	}

	p.applyQueueSizes(tenantId, sizes)

	metadata, err := p.tasks.GetQueueSizesByMetadata(ctx, tenantId)

	if err != nil {
		p.l.Warn().Err(err).Str("tenant_id", tenantId.String()).Msg("could not poll queue sizes by additional metadata")
		return
	}

	p.applyMetadataSizes(tenantId, metadata)
}

func (p *queueMetricsPoller) applyQueueSizes(tenantId uuid.UUID, rows []*sqlcv1.GetQueueSizesRow) {
	tenantIdStr := tenantId.String()
	current := make(map[queueSizeKey]struct{}, len(rows))

	for _, row := range rows {
		k := queueSizeKey{queue: row.Queue, workflowName: row.WorkflowName}
		current[k] = struct{}{}
		prometheus.TenantQueueSize.WithLabelValues(tenantIdStr, k.queue, k.workflowName).Set(float64(row.Count))
	}

	zeroed, deleted, next := diffStaleSeries(p.knownQueueSizes[tenantId], current)

	for _, k := range zeroed {
		prometheus.TenantQueueSize.WithLabelValues(tenantIdStr, k.queue, k.workflowName).Set(0)
	}

	for _, k := range deleted {
		prometheus.TenantQueueSize.DeleteLabelValues(tenantIdStr, k.queue, k.workflowName)
	}

	if len(next) == 0 {
		delete(p.knownQueueSizes, tenantId)
	} else {
		p.knownQueueSizes[tenantId] = next
	}
}

func (p *queueMetricsPoller) applyMetadataSizes(tenantId uuid.UUID, rows []*sqlcv1.GetQueueSizesByMetadataRow) {
	tenantIdStr := tenantId.String()
	current := make(map[metadataQueueSizeKey]struct{}, len(rows))

	for _, row := range rows {
		// GetQueueSizesByMetadata filters on the same prefix; it is re-checked here because
		// this is where series are allocated
		if !strings.HasPrefix(row.Key, repov1.PrometheusMetadataKeyPrefix) {
			continue
		}

		k := metadataQueueSizeKey{queue: row.Queue, key: row.Key, value: row.Value}
		current[k] = struct{}{}
		prometheus.TenantQueueSizeByMetadata.WithLabelValues(tenantIdStr, k.queue, k.key, k.value).Set(float64(row.Count))
	}

	zeroed, deleted, next := diffStaleSeries(p.knownMetadataSizes[tenantId], current)

	for _, k := range zeroed {
		prometheus.TenantQueueSizeByMetadata.WithLabelValues(tenantIdStr, k.queue, k.key, k.value).Set(0)
	}

	for _, k := range deleted {
		prometheus.TenantQueueSizeByMetadata.DeleteLabelValues(tenantIdStr, k.queue, k.key, k.value)
	}

	if len(next) == 0 {
		delete(p.knownMetadataSizes, tenantId)
	} else {
		p.knownMetadataSizes[tenantId] = next
	}
}

func (s *Scheduler) runPollQueueMetrics(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, queueMetricsPollTimeout)
		defer cancel()

		tenants, err := s.repov1.Tenant().ListTenantsBySchedulerPartition(ctx, s.p.GetSchedulerPartitionId())

		if err != nil {
			s.l.Error().Err(err).Msg("could not list tenants for queue metrics")
			return
		}

		tenantIds := make([]uuid.UUID, 0, len(tenants))

		for _, tenant := range tenants {
			if s.promGate.Enabled(ctx, tenant.ID) {
				tenantIds = append(tenantIds, tenant.ID)
			}
		}

		s.queueMetrics.poll(ctx, tenantIds)
	}
}
