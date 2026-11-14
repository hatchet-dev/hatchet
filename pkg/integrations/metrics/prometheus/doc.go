// Package prometheus registers the Hatchet Prometheus collectors (global and
// tenant-labeled task metrics, pub/sub, database pool), gates tenant series on
// entitlement, and serves the registry over HTTP.
package prometheus
