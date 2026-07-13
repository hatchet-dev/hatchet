//go:build !e2e && !load && !rampup && !integration

package v1

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	promclient "github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	repo "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func strLabel(key, value string) *sqlcv1.ListManyWorkerLabelsRow {
	return &sqlcv1.ListManyWorkerLabelsRow{
		Key:      key,
		StrValue: pgtype.Text{String: value, Valid: true},
	}
}

func intLabel(key string, value int32) *sqlcv1.ListManyWorkerLabelsRow {
	return &sqlcv1.ListManyWorkerLabelsRow{
		Key:      key,
		IntValue: pgtype.Int4{Int32: value, Valid: true},
	}
}

// collectLabelPairSeries gathers a gauge vec and returns the series for the
// given tenant keyed by (label_key, label_value, slot_type).
func collectLabelPairSeries(t *testing.T, vec *promclient.GaugeVec, tenantId string) map[LabelPairPromLabels]float64 {
	t.Helper()

	ch := make(chan promclient.Metric, 1024)
	vec.Collect(ch)
	close(ch)

	out := make(map[LabelPairPromLabels]float64)

	for m := range ch {
		var d dto.Metric
		require.NoError(t, m.Write(&d))

		labels := make(map[string]string, len(d.Label))
		for _, lp := range d.Label {
			labels[lp.GetName()] = lp.GetValue()
		}

		if labels["tenant_id"] != tenantId {
			continue
		}

		seriesLabels := LabelPairPromLabels{
			WorkerLabelPair: WorkerLabelPair{Key: labels["label_key"], Value: labels["label_value"]},
			SlotType:        labels["slot_type"],
		}

		out[seriesLabels] = d.GetGauge().GetValue()
	}

	return out
}

func defaultSeries(key, value string) LabelPairPromLabels {
	return LabelPairPromLabels{
		WorkerLabelPair: WorkerLabelPair{Key: key, Value: value},
		SlotType:        repo.SlotTypeDefault,
	}
}

func durableSeries(key, value string) LabelPairPromLabels {
	return LabelPairPromLabels{
		WorkerLabelPair: WorkerLabelPair{Key: key, Value: value},
		SlotType:        repo.SlotTypeDurable,
	}
}

// snapshotInput builds a SnapshotInput from per-worker, per-slot-type utilization,
// deriving the cross-type aggregate the same way the scheduler does.
func snapshotInput(workers map[uuid.UUID]*WorkerCp, utilizationByType map[uuid.UUID]map[string]*SlotUtilization) *SnapshotInput {
	utilization := make(map[uuid.UUID]*SlotUtilization, len(utilizationByType))

	for workerId, byType := range utilizationByType {
		aggregate := &SlotUtilization{}

		for _, u := range byType {
			aggregate.UtilizedSlots += u.UtilizedSlots
			aggregate.NonUtilizedSlots += u.NonUtilizedSlots
		}

		utilization[workerId] = aggregate
	}

	return &SnapshotInput{
		Workers:                     workers,
		WorkerSlotUtilization:       utilization,
		WorkerSlotUtilizationByType: utilizationByType,
	}
}

func TestPrometheusExtensionAggregatesSlotsByLabelPairAndSlotType(t *testing.T) {
	ext := NewPrometheusExtension(nil)

	tenantId := uuid.New()
	worker1 := uuid.New()
	worker2 := uuid.New()
	worker3 := uuid.New()

	workers := map[uuid.UUID]*WorkerCp{
		worker1: {
			WorkerId: worker1,
			Name:     "gpu-worker-1",
			Labels:   []*sqlcv1.ListManyWorkerLabelsRow{strLabel("pool", "gpu"), strLabel("region", "us-east")},
		},
		worker2: {
			WorkerId: worker2,
			Name:     "gpu-worker-2",
			Labels:   []*sqlcv1.ListManyWorkerLabelsRow{strLabel("pool", "gpu"), intLabel("priority", 5)},
		},
		worker3: {
			WorkerId: worker3,
			Name:     "unlabeled-worker",
		},
	}

	utilizationByType := map[uuid.UUID]map[string]*SlotUtilization{
		worker1: {
			repo.SlotTypeDefault: {UtilizedSlots: 3, NonUtilizedSlots: 1},
			repo.SlotTypeDurable: {UtilizedSlots: 2, NonUtilizedSlots: 8},
		},
		worker2: {
			repo.SlotTypeDefault: {UtilizedSlots: 2, NonUtilizedSlots: 4},
		},
		worker3: {
			repo.SlotTypeDefault: {UtilizedSlots: 1, NonUtilizedSlots: 1},
		},
	}

	ext.ReportSnapshot(context.Background(), tenantId, snapshotInput(workers, utilizationByType))

	total := collectLabelPairSeries(t, prometheus.TenantWorkerLabelSlots, tenantId.String())
	used := collectLabelPairSeries(t, prometheus.TenantUsedWorkerLabelSlots, tenantId.String())
	available := collectLabelPairSeries(t, prometheus.TenantAvailableWorkerLabelSlots, tenantId.String())

	require.Equal(t, map[LabelPairPromLabels]float64{
		defaultSeries("pool", "gpu"):       10, // worker1 (4) + worker2 (6)
		durableSeries("pool", "gpu"):       10, // worker1 only
		defaultSeries("region", "us-east"): 4,
		durableSeries("region", "us-east"): 10,
		defaultSeries("priority", "5"):     6,
	}, total)

	require.Equal(t, map[LabelPairPromLabels]float64{
		defaultSeries("pool", "gpu"):       5,
		durableSeries("pool", "gpu"):       2,
		defaultSeries("region", "us-east"): 3,
		durableSeries("region", "us-east"): 2,
		defaultSeries("priority", "5"):     2,
	}, used)

	require.Equal(t, map[LabelPairPromLabels]float64{
		defaultSeries("pool", "gpu"):       5,
		durableSeries("pool", "gpu"):       8,
		defaultSeries("region", "us-east"): 1,
		durableSeries("region", "us-east"): 8,
		defaultSeries("priority", "5"):     4,
	}, available)
}

func TestPrometheusExtensionDeletesStaleLabelPairs(t *testing.T) {
	ext := NewPrometheusExtension(nil)

	tenantId := uuid.New()
	worker1 := uuid.New()
	worker2 := uuid.New()

	workers := map[uuid.UUID]*WorkerCp{
		worker1: {
			WorkerId: worker1,
			Name:     "worker-1",
			Labels:   []*sqlcv1.ListManyWorkerLabelsRow{strLabel("pool", "gpu")},
		},
		worker2: {
			WorkerId: worker2,
			Name:     "worker-2",
			Labels:   []*sqlcv1.ListManyWorkerLabelsRow{strLabel("pool", "cpu")},
		},
	}

	utilizationByType := map[uuid.UUID]map[string]*SlotUtilization{
		worker1: {repo.SlotTypeDefault: {UtilizedSlots: 1, NonUtilizedSlots: 1}},
		worker2: {repo.SlotTypeDefault: {UtilizedSlots: 2, NonUtilizedSlots: 2}},
	}

	ext.ReportSnapshot(context.Background(), tenantId, snapshotInput(workers, utilizationByType))

	require.Len(t, collectLabelPairSeries(t, prometheus.TenantWorkerLabelSlots, tenantId.String()), 2)

	// worker2 disappears, so the (pool, cpu) series should be deleted
	delete(workers, worker2)
	delete(utilizationByType, worker2)

	ext.ReportSnapshot(context.Background(), tenantId, snapshotInput(workers, utilizationByType))

	total := collectLabelPairSeries(t, prometheus.TenantWorkerLabelSlots, tenantId.String())
	require.Equal(t, map[LabelPairPromLabels]float64{
		defaultSeries("pool", "gpu"): 2,
	}, total)
}

func TestPrometheusExtensionSkipsTransientZeroSlotLabelPairs(t *testing.T) {
	ext := NewPrometheusExtension(nil)

	tenantId := uuid.New()
	worker1 := uuid.New()

	workers := map[uuid.UUID]*WorkerCp{
		worker1: {
			WorkerId: worker1,
			Name:     "worker-1",
			Labels:   []*sqlcv1.ListManyWorkerLabelsRow{strLabel("pool", "gpu")},
		},
	}

	ext.ReportSnapshot(context.Background(), tenantId, snapshotInput(workers, map[uuid.UUID]map[string]*SlotUtilization{
		worker1: {repo.SlotTypeDefault: {UtilizedSlots: 3, NonUtilizedSlots: 1}},
	}))

	// a replenishment gap reports 0 slots; the previous values should be preserved
	// and the series should not be deleted
	ext.ReportSnapshot(context.Background(), tenantId, snapshotInput(workers, map[uuid.UUID]map[string]*SlotUtilization{
		worker1: {repo.SlotTypeDefault: {UtilizedSlots: 0, NonUtilizedSlots: 0}},
	}))

	total := collectLabelPairSeries(t, prometheus.TenantWorkerLabelSlots, tenantId.String())
	used := collectLabelPairSeries(t, prometheus.TenantUsedWorkerLabelSlots, tenantId.String())

	require.Equal(t, map[LabelPairPromLabels]float64{defaultSeries("pool", "gpu"): 4}, total)
	require.Equal(t, map[LabelPairPromLabels]float64{defaultSeries("pool", "gpu"): 3}, used)

	// a brand-new pair that has only ever seen 0 slots should not create a series
	worker2 := uuid.New()
	workers[worker2] = &WorkerCp{
		WorkerId: worker2,
		Name:     "worker-2",
		Labels:   []*sqlcv1.ListManyWorkerLabelsRow{strLabel("pool", "cpu")},
	}

	ext.ReportSnapshot(context.Background(), tenantId, snapshotInput(workers, map[uuid.UUID]map[string]*SlotUtilization{
		worker1: {repo.SlotTypeDefault: {UtilizedSlots: 3, NonUtilizedSlots: 1}},
		worker2: {repo.SlotTypeDefault: {UtilizedSlots: 0, NonUtilizedSlots: 0}},
	}))

	total = collectLabelPairSeries(t, prometheus.TenantWorkerLabelSlots, tenantId.String())
	require.NotContains(t, total, defaultSeries("pool", "cpu"))
}

func TestPrometheusExtensionCleanupTenantDeletesLabelPairs(t *testing.T) {
	ext := NewPrometheusExtension(nil)

	tenantId := uuid.New()
	otherTenantId := uuid.New()
	worker1 := uuid.New()
	worker2 := uuid.New()

	input := snapshotInput(
		map[uuid.UUID]*WorkerCp{
			worker1: {
				WorkerId: worker1,
				Name:     "worker-1",
				Labels:   []*sqlcv1.ListManyWorkerLabelsRow{strLabel("pool", "gpu")},
			},
		},
		map[uuid.UUID]map[string]*SlotUtilization{
			worker1: {repo.SlotTypeDefault: {UtilizedSlots: 1, NonUtilizedSlots: 1}},
		},
	)

	otherInput := snapshotInput(
		map[uuid.UUID]*WorkerCp{
			worker2: {
				WorkerId: worker2,
				Name:     "worker-2",
				Labels:   []*sqlcv1.ListManyWorkerLabelsRow{strLabel("pool", "gpu")},
			},
		},
		map[uuid.UUID]map[string]*SlotUtilization{
			worker2: {repo.SlotTypeDefault: {UtilizedSlots: 2, NonUtilizedSlots: 2}},
		},
	)

	ext.ReportSnapshot(context.Background(), tenantId, input)
	ext.ReportSnapshot(context.Background(), otherTenantId, otherInput)

	require.NoError(t, ext.CleanupTenant(tenantId))

	require.Empty(t, collectLabelPairSeries(t, prometheus.TenantWorkerLabelSlots, tenantId.String()))
	require.Empty(t, collectLabelPairSeries(t, prometheus.TenantUsedWorkerLabelSlots, tenantId.String()))
	require.Empty(t, collectLabelPairSeries(t, prometheus.TenantAvailableWorkerLabelSlots, tenantId.String()))

	// other tenants are unaffected
	require.Len(t, collectLabelPairSeries(t, prometheus.TenantWorkerLabelSlots, otherTenantId.String()), 1)

	require.NoError(t, ext.Cleanup())

	require.Empty(t, collectLabelPairSeries(t, prometheus.TenantWorkerLabelSlots, otherTenantId.String()))
}
