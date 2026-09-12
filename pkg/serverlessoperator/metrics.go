package serverlessoperator

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// metricVectors are registered once per process on the default registry. Every series carries
// the link label (grpc or engine) so the in-engine and out-of-process modes share dashboards.
type metricVectors struct {
	unitsOwned          *prometheus.GaugeVec
	endpointsOwned      *prometheus.GaugeVec
	registrationsOpen   *prometheus.GaugeVec
	leaseClaims         *prometheus.CounterVec
	leaseSheds          *prometheus.CounterVec
	rebalanceDuration   *prometheus.HistogramVec
	healthchecks        *prometheus.CounterVec
	healthcheckDuration *prometheus.HistogramVec
	endpointsUnhealthy  *prometheus.GaugeVec
	tenantsWithoutToken *prometheus.GaugeVec
	deliveries          *prometheus.CounterVec
	deliveryDuration    *prometheus.HistogramVec
	routingMisses       *prometheus.CounterVec
	wsConnectionsOpen   *prometheus.GaugeVec
	evictions           *prometheus.CounterVec
}

var (
	vectorsOnce sync.Once
	vectors     *metricVectors
)

func registerVectors() *metricVectors {
	vectorsOnce.Do(func() {
		link := []string{"link"}

		vectors = &metricVectors{
			unitsOwned:          promauto.NewGaugeVec(prometheus.GaugeOpts{Name: "hatchet_serverless_units_owned", Help: "Lease units owned by this process"}, link),
			endpointsOwned:      promauto.NewGaugeVec(prometheus.GaugeOpts{Name: "hatchet_serverless_endpoints_owned", Help: "Endpoints polled by this process"}, link),
			registrationsOpen:   promauto.NewGaugeVec(prometheus.GaugeOpts{Name: "hatchet_serverless_registrations_open", Help: "Open engine registrations"}, link),
			leaseClaims:         promauto.NewCounterVec(prometheus.CounterOpts{Name: "hatchet_serverless_lease_claims_total", Help: "Lease units claimed"}, link),
			leaseSheds:          promauto.NewCounterVec(prometheus.CounterOpts{Name: "hatchet_serverless_lease_sheds_total", Help: "Lease units shed"}, link),
			rebalanceDuration:   promauto.NewHistogramVec(prometheus.HistogramOpts{Name: "hatchet_serverless_rebalance_duration_seconds", Help: "Rebalance tick duration"}, link),
			healthchecks:        promauto.NewCounterVec(prometheus.CounterOpts{Name: "hatchet_serverless_healthchecks_total", Help: "Healthcheck polls by result"}, []string{"link", "result"}),
			healthcheckDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{Name: "hatchet_serverless_healthcheck_duration_seconds", Help: "Healthcheck poll duration"}, link),
			endpointsUnhealthy:  promauto.NewGaugeVec(prometheus.GaugeOpts{Name: "hatchet_serverless_endpoints_unhealthy", Help: "Owned endpoints currently unhealthy"}, link),
			tenantsWithoutToken: promauto.NewGaugeVec(prometheus.GaugeOpts{Name: "hatchet_serverless_tenants_without_token", Help: "Served tenants with no token"}, link),
			deliveries:          promauto.NewCounterVec(prometheus.CounterOpts{Name: "hatchet_serverless_deliveries_total", Help: "Task deliveries by result"}, []string{"link", "result"}),
			deliveryDuration:    promauto.NewHistogramVec(prometheus.HistogramOpts{Name: "hatchet_serverless_delivery_duration_seconds", Help: "Task delivery duration"}, link),
			routingMisses:       promauto.NewCounterVec(prometheus.CounterOpts{Name: "hatchet_serverless_routing_misses_total", Help: "Actions whose namespace had no endpoint"}, link),
			wsConnectionsOpen:   promauto.NewGaugeVec(prometheus.GaugeOpts{Name: "hatchet_serverless_ws_connections_open", Help: "Open durable websockets"}, link),
			evictions:           promauto.NewCounterVec(prometheus.CounterOpts{Name: "hatchet_serverless_evictions_total", Help: "Durable evictions by source"}, []string{"link", "source"}),
		}
	})

	return vectors
}

// metrics is the per-run view of the vectors, curried on the link label.
type metrics struct {
	v    *metricVectors
	link string
}

func newMetrics(link string) *metrics {
	return &metrics{v: registerVectors(), link: link}
}

func (m *metrics) setOwned(units, endpoints int) {
	m.v.unitsOwned.WithLabelValues(m.link).Set(float64(units))
	m.v.endpointsOwned.WithLabelValues(m.link).Set(float64(endpoints))
}

func (m *metrics) setRegistrationsOpen(n int) {
	m.v.registrationsOpen.WithLabelValues(m.link).Set(float64(n))
}

func (m *metrics) claimed(n int) {
	m.v.leaseClaims.WithLabelValues(m.link).Add(float64(n))
}

func (m *metrics) shed(n int) {
	m.v.leaseSheds.WithLabelValues(m.link).Add(float64(n))
}

func (m *metrics) rebalanced(d time.Duration) {
	m.v.rebalanceDuration.WithLabelValues(m.link).Observe(d.Seconds())
}

func (m *metrics) healthcheck(result string, d time.Duration) {
	m.v.healthchecks.WithLabelValues(m.link, result).Inc()
	m.v.healthcheckDuration.WithLabelValues(m.link).Observe(d.Seconds())
}

func (m *metrics) setEndpointsUnhealthy(n int) {
	m.v.endpointsUnhealthy.WithLabelValues(m.link).Set(float64(n))
}

func (m *metrics) setTenantsWithoutToken(n int) {
	m.v.tenantsWithoutToken.WithLabelValues(m.link).Set(float64(n))
}

func (m *metrics) delivered(result string, d time.Duration) {
	m.v.deliveries.WithLabelValues(m.link, result).Inc()
	m.v.deliveryDuration.WithLabelValues(m.link).Observe(d.Seconds())
}

func (m *metrics) wsOpened() {
	m.v.wsConnectionsOpen.WithLabelValues(m.link).Inc()
}

func (m *metrics) wsClosed() {
	m.v.wsConnectionsOpen.WithLabelValues(m.link).Dec()
}

func (m *metrics) evicted(source string) {
	m.v.evictions.WithLabelValues(m.link, source).Inc()
}

func (m *metrics) routingMiss() {
	m.v.routingMisses.WithLabelValues(m.link).Inc()
}
