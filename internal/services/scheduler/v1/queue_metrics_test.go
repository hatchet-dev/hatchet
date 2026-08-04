package scheduler

import (
	"context"
	"testing"

	"github.com/google/uuid"
	promclient "github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestDiffStaleSeries(t *testing.T) {
	current := func(keys ...string) map[string]struct{} {
		m := make(map[string]struct{}, len(keys))
		for _, k := range keys {
			m[k] = struct{}{}
		}
		return m
	}

	t.Run("new series are tracked as live", func(t *testing.T) {
		zeroed, deleted, next := diffStaleSeries(map[string]bool{}, current("a", "b"))

		assert.Empty(t, zeroed)
		assert.Empty(t, deleted)
		assert.Equal(t, map[string]bool{"a": true, "b": true}, next)
	})

	t.Run("disappeared series are zeroed for one poll", func(t *testing.T) {
		zeroed, deleted, next := diffStaleSeries(map[string]bool{"a": true, "b": true}, current("a"))

		assert.Equal(t, []string{"b"}, zeroed)
		assert.Empty(t, deleted)
		assert.Equal(t, map[string]bool{"a": true, "b": false}, next)
	})

	t.Run("zeroed series are deleted on the following poll", func(t *testing.T) {
		zeroed, deleted, next := diffStaleSeries(map[string]bool{"a": true, "b": false}, current("a"))

		assert.Empty(t, zeroed)
		assert.Equal(t, []string{"b"}, deleted)
		assert.Equal(t, map[string]bool{"a": true}, next)
	})

	t.Run("zeroed series that reappear become live again", func(t *testing.T) {
		zeroed, deleted, next := diffStaleSeries(map[string]bool{"a": true, "b": false}, current("a", "b"))

		assert.Empty(t, zeroed)
		assert.Empty(t, deleted)
		assert.Equal(t, map[string]bool{"a": true, "b": true}, next)
	})

	t.Run("empty current zeroes then deletes everything", func(t *testing.T) {
		zeroed, deleted, next := diffStaleSeries(map[string]bool{"a": true}, current())

		assert.Equal(t, []string{"a"}, zeroed)
		assert.Empty(t, deleted)
		assert.Equal(t, map[string]bool{"a": false}, next)

		zeroed, deleted, next = diffStaleSeries(next, current())

		assert.Empty(t, zeroed)
		assert.Equal(t, []string{"a"}, deleted)
		assert.Empty(t, next)
	})
}

// collectGaugeSeries gathers a gauge vec and returns the series for the given tenant,
// keyed by the remaining label values in alphabetical label-name order (padded with
// empty strings for gauges with fewer than three non-tenant labels).
func collectGaugeSeries(t *testing.T, vec *promclient.GaugeVec, tenantId string) map[[3]string]float64 {
	t.Helper()

	ch := make(chan promclient.Metric, 1024)
	vec.Collect(ch)
	close(ch)

	series := make(map[[3]string]float64)

	for metric := range ch {
		m := &dto.Metric{}
		require.NoError(t, metric.Write(m))

		labels := make(map[string]string, len(m.Label))
		for _, pair := range m.Label {
			labels[pair.GetName()] = pair.GetValue()
		}

		if labels["tenant_id"] != tenantId {
			continue
		}

		var key [3]string
		i := 0
		for _, pair := range m.Label {
			if pair.GetName() == "tenant_id" {
				continue
			}
			key[i] = pair.GetValue()
			i++
		}

		series[key] = m.GetGauge().GetValue()
	}

	return series
}

func TestQueueMetricsPollerLifecycle(t *testing.T) {
	l := zerolog.Nop()
	p := newQueueMetricsPoller(nil, &l)

	tenantId := uuid.New()
	tenantIdStr := tenantId.String()

	// a reported backlog sets the gauges
	p.applyQueueSizes(tenantId, []*sqlcv1.GetQueueSizesRow{
		{Queue: "default", WorkflowName: "my-workflow", Count: 5},
	})
	p.applyMetadataSizes(tenantId, []*sqlcv1.GetQueueSizesByMetadataRow{
		{Queue: "default", Key: "product", Value: "search", Count: 3},
	})

	assert.Equal(t, map[[3]string]float64{{"default", "my-workflow", ""}: 5}, collectGaugeSeries(t, prometheus.TenantQueueSize, tenantIdStr))
	assert.Equal(t, map[[3]string]float64{{"product", "default", "search"}: 3}, collectGaugeSeries(t, prometheus.TenantQueueSizeByMetadata, tenantIdStr))

	// a drained queue reports zero for one poll
	p.applyQueueSizes(tenantId, nil)
	p.applyMetadataSizes(tenantId, nil)

	assert.Equal(t, map[[3]string]float64{{"default", "my-workflow", ""}: 0}, collectGaugeSeries(t, prometheus.TenantQueueSize, tenantIdStr))
	assert.Equal(t, map[[3]string]float64{{"product", "default", "search"}: 0}, collectGaugeSeries(t, prometheus.TenantQueueSizeByMetadata, tenantIdStr))

	// and is deleted on the poll after
	p.applyQueueSizes(tenantId, nil)
	p.applyMetadataSizes(tenantId, nil)

	assert.Empty(t, collectGaugeSeries(t, prometheus.TenantQueueSize, tenantIdStr))
	assert.Empty(t, collectGaugeSeries(t, prometheus.TenantQueueSizeByMetadata, tenantIdStr))
	assert.Empty(t, p.knownQueueSizes)
	assert.Empty(t, p.knownMetadataSizes)
}

func TestQueueMetricsPollerDrainsRemovedTenants(t *testing.T) {
	l := zerolog.Nop()
	p := newQueueMetricsPoller(nil, &l)

	tenantId := uuid.New()
	tenantIdStr := tenantId.String()

	p.applyQueueSizes(tenantId, []*sqlcv1.GetQueueSizesRow{
		{Queue: "default", WorkflowName: "my-workflow", Count: 2},
	})
	p.applyMetadataSizes(tenantId, []*sqlcv1.GetQueueSizesByMetadataRow{
		{Queue: "default", Key: "product", Value: "search", Count: 2},
	})

	// the tenant is no longer on the partition: its series drain to zero, then delete.
	// polling no tenants never touches the task repository (which is nil here).
	p.poll(context.Background(), nil)

	assert.Equal(t, map[[3]string]float64{{"default", "my-workflow", ""}: 0}, collectGaugeSeries(t, prometheus.TenantQueueSize, tenantIdStr))
	assert.Equal(t, map[[3]string]float64{{"product", "default", "search"}: 0}, collectGaugeSeries(t, prometheus.TenantQueueSizeByMetadata, tenantIdStr))

	p.poll(context.Background(), nil)

	assert.Empty(t, collectGaugeSeries(t, prometheus.TenantQueueSize, tenantIdStr))
	assert.Empty(t, collectGaugeSeries(t, prometheus.TenantQueueSizeByMetadata, tenantIdStr))
}
