package serverlessoperator

import "time"

// Config holds the knobs shared by the out-of-process binary and the in-engine mode. Zero
// values are replaced by the defaults below; the binary binds SERVERLESS_OPERATOR_* onto it.
type Config struct {
	// OperatorName is the operator name the sessions register as.
	OperatorName string

	// LinkName labels metrics with how the core reaches the engine: grpc or engine.
	LinkName string

	DefaultSlots int32
	DurableSlots int32

	// LeaseTTL is how long a process row stays live after a heartbeat.
	LeaseTTL          time.Duration
	HeartbeatInterval time.Duration
	RebalanceInterval time.Duration

	// ShedHysteresis is the fraction above fair share a process tolerates before shedding.
	ShedHysteresis float64

	// DrainTimeout bounds how long in-flight deliveries are awaited when a unit is lost or the
	// process stops.
	DrainTimeout time.Duration

	// RoutingRefreshInterval is the incremental (updated_at based) routing cache refresh
	// cadence; RoutingFullReloadInterval is how often the cache is reconciled against the
	// tenant's endpoint ids and versions (spread by up to ten percent per tenant), which is
	// what drops hard-deleted endpoints.
	RoutingRefreshInterval    time.Duration
	RoutingFullReloadInterval time.Duration

	// HealthcheckTimeout bounds one healthcheck request; HealthcheckConcurrency caps
	// healthchecks process-wide and HealthcheckTenantConcurrency caps them per tenant inside
	// that limit, so one tenant's slow endpoints cannot hold every slot.
	HealthcheckTimeout           time.Duration
	HealthcheckConcurrency       int
	HealthcheckTenantConcurrency int

	// HealthcheckApplyTimeout bounds the application of one changed catalog: the workflow
	// puts, the action delta and the registered_actions write.
	HealthcheckApplyTimeout time.Duration

	// MaxWorkflowsPerEndpoint and MaxActionsPerEndpoint cap what one healthcheck may
	// advertise; a catalog over either cap is refused and the endpoint marked with the error.
	MaxWorkflowsPerEndpoint int
	MaxActionsPerEndpoint   int

	// MaintenanceConcurrency is how many tenants a maintenance pass refreshes at once.
	MaintenanceConcurrency int

	// LeaseMaxClaimPerTick caps how many lease units one rebalance tick claims; the process's
	// share of the claimable units is taken up to this many, in statements of the leaser's
	// claim batch.
	LeaseMaxClaimPerTick int32

	// WSMaxFrameBytes and WSPingInterval configure the durable websocket relay (a later
	// phase); they are carried here so the binary's env binding is complete.
	WSMaxFrameBytes int64
	WSPingInterval  time.Duration

	// HealthPort serves /healthz, /readyz and /metrics. Zero disables the server.
	HealthPort int

	// Relay resource limits (durable websocket).

	// WSMaxUpgradeHeaderBytes bounds an endpoint's websocket upgrade response head, which is
	// parsed before WSMaxFrameBytes applies.
	WSMaxUpgradeHeaderBytes int64

	// WSMaxQueuedBytes bounds the encoded frames one relay retains for an endpoint that reads
	// slower than the engine answers; crossing it closes the socket with backpressure.
	WSMaxQueuedBytes int64
}

const (
	DefaultOperatorName                       = "serverless"
	DefaultLinkName                           = "grpc"
	DefaultDefaultSlots                 int32 = 10000
	DefaultDurableSlots                 int32 = 10000
	DefaultLeaseTTL                           = 15 * time.Second
	DefaultHeartbeatInterval                  = 5 * time.Second
	DefaultRebalanceInterval                  = 5 * time.Second
	DefaultShedHysteresis                     = 0.2
	DefaultDrainTimeout                       = 60 * time.Second
	DefaultRoutingRefreshInterval             = 10 * time.Second
	DefaultRoutingFullReloadInterval          = 60 * time.Second
	DefaultHealthcheckTimeout                 = 10 * time.Second
	DefaultHealthcheckConcurrency             = 256
	DefaultHealthcheckTenantConcurrency       = 32
	DefaultHealthcheckApplyTimeout            = 60 * time.Second
	DefaultMaxWorkflowsPerEndpoint            = 200
	DefaultMaxActionsPerEndpoint              = 500
	DefaultMaintenanceConcurrency             = 8
	DefaultLeaseMaxClaimPerTick         int32 = 1024
	DefaultWSMaxFrameBytes              int64 = 4 * 1024 * 1024
	DefaultWSPingInterval                     = 15 * time.Second
	DefaultHealthPort                         = 8080

	// Relay resource limit defaults.
	DefaultWSMaxUpgradeHeaderBytes int64 = 64 * 1024
	DefaultWSMaxQueuedBytes        int64 = 16 * 1024 * 1024

	// processSweepInterval and processExpiryCutoff drive the expired process row sweep every
	// process runs.
	processSweepInterval = 10 * time.Minute
	processExpiryCutoff  = time.Hour

	// workerLabelProcess is the worker label carrying the owning process id.
	workerLabelProcess = "hatchet-serverless-process"
)

// DefaultConfig returns the plan's defaults.
func DefaultConfig() Config {
	return Config{
		OperatorName:                 DefaultOperatorName,
		LinkName:                     DefaultLinkName,
		DefaultSlots:                 DefaultDefaultSlots,
		DurableSlots:                 DefaultDurableSlots,
		LeaseTTL:                     DefaultLeaseTTL,
		HeartbeatInterval:            DefaultHeartbeatInterval,
		RebalanceInterval:            DefaultRebalanceInterval,
		ShedHysteresis:               DefaultShedHysteresis,
		DrainTimeout:                 DefaultDrainTimeout,
		RoutingRefreshInterval:       DefaultRoutingRefreshInterval,
		RoutingFullReloadInterval:    DefaultRoutingFullReloadInterval,
		HealthcheckTimeout:           DefaultHealthcheckTimeout,
		HealthcheckConcurrency:       DefaultHealthcheckConcurrency,
		HealthcheckTenantConcurrency: DefaultHealthcheckTenantConcurrency,
		HealthcheckApplyTimeout:      DefaultHealthcheckApplyTimeout,
		MaxWorkflowsPerEndpoint:      DefaultMaxWorkflowsPerEndpoint,
		MaxActionsPerEndpoint:        DefaultMaxActionsPerEndpoint,
		MaintenanceConcurrency:       DefaultMaintenanceConcurrency,
		LeaseMaxClaimPerTick:         DefaultLeaseMaxClaimPerTick,
		WSMaxFrameBytes:              DefaultWSMaxFrameBytes,
		WSPingInterval:               DefaultWSPingInterval,
		HealthPort:                   DefaultHealthPort,
		WSMaxUpgradeHeaderBytes:      DefaultWSMaxUpgradeHeaderBytes,
		WSMaxQueuedBytes:             DefaultWSMaxQueuedBytes,
	}
}

// withDefaults fills zero fields from DefaultConfig. HealthPort is left alone so zero keeps
// meaning "no server", which tests rely on.
func (c Config) withDefaults() Config {
	d := DefaultConfig()

	if c.OperatorName == "" {
		c.OperatorName = d.OperatorName
	}

	if c.LinkName == "" {
		c.LinkName = d.LinkName
	}

	if c.DefaultSlots <= 0 {
		c.DefaultSlots = d.DefaultSlots
	}

	if c.DurableSlots <= 0 {
		c.DurableSlots = d.DurableSlots
	}

	if c.LeaseTTL <= 0 {
		c.LeaseTTL = d.LeaseTTL
	}

	if c.HeartbeatInterval <= 0 {
		c.HeartbeatInterval = d.HeartbeatInterval
	}

	if c.RebalanceInterval <= 0 {
		c.RebalanceInterval = d.RebalanceInterval
	}

	if c.ShedHysteresis <= 0 {
		c.ShedHysteresis = d.ShedHysteresis
	}

	if c.DrainTimeout <= 0 {
		c.DrainTimeout = d.DrainTimeout
	}

	if c.RoutingRefreshInterval <= 0 {
		c.RoutingRefreshInterval = d.RoutingRefreshInterval
	}

	if c.RoutingFullReloadInterval <= 0 {
		c.RoutingFullReloadInterval = d.RoutingFullReloadInterval
	}

	if c.HealthcheckTimeout <= 0 {
		c.HealthcheckTimeout = d.HealthcheckTimeout
	}

	if c.HealthcheckConcurrency <= 0 {
		c.HealthcheckConcurrency = d.HealthcheckConcurrency
	}

	if c.HealthcheckTenantConcurrency <= 0 {
		c.HealthcheckTenantConcurrency = d.HealthcheckTenantConcurrency
	}

	if c.HealthcheckTenantConcurrency > c.HealthcheckConcurrency {
		c.HealthcheckTenantConcurrency = c.HealthcheckConcurrency
	}

	if c.HealthcheckApplyTimeout <= 0 {
		c.HealthcheckApplyTimeout = d.HealthcheckApplyTimeout
	}

	if c.MaxWorkflowsPerEndpoint <= 0 {
		c.MaxWorkflowsPerEndpoint = d.MaxWorkflowsPerEndpoint
	}

	if c.MaxActionsPerEndpoint <= 0 {
		c.MaxActionsPerEndpoint = d.MaxActionsPerEndpoint
	}

	if c.MaintenanceConcurrency <= 0 {
		c.MaintenanceConcurrency = d.MaintenanceConcurrency
	}

	if c.LeaseMaxClaimPerTick <= 0 {
		c.LeaseMaxClaimPerTick = d.LeaseMaxClaimPerTick
	}

	if c.WSMaxFrameBytes <= 0 {
		c.WSMaxFrameBytes = d.WSMaxFrameBytes
	}

	if c.WSPingInterval <= 0 {
		c.WSPingInterval = d.WSPingInterval
	}

	if c.WSMaxUpgradeHeaderBytes <= 0 {
		c.WSMaxUpgradeHeaderBytes = d.WSMaxUpgradeHeaderBytes
	}

	if c.WSMaxQueuedBytes <= 0 {
		c.WSMaxQueuedBytes = d.WSMaxQueuedBytes
	}

	return c
}
