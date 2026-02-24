// Deprecated: This package is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package v1

import (
	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v0Config "github.com/hatchet-dev/hatchet/pkg/config/client"
	"github.com/hatchet-dev/hatchet/pkg/v1/features"
	"github.com/hatchet-dev/hatchet/pkg/v1/worker"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
)

// Deprecated: HatchetClient is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type HatchetClient interface {
	// V0 returns the underlying V0 client for backward compatibility.
	V0() v0Client.Client

	// Worker creates and configures a new worker with the provided options and optional configuration functions.
	// @example
	// ```go
	// worker, err := hatchet.Worker(worker.CreateOpts{
	// 	   Name: "my-worker",
	//   },
	//   v1.WithWorkflows(simple)
	// )
	// ```
	Worker(opts worker.WorkerOpts) (worker.Worker, error)

	// Feature clients

	Metrics() features.MetricsClient
	RateLimits() features.RateLimitsClient
	Runs() features.RunsClient
	Workers() features.WorkersClient
	Workflows() features.WorkflowsClient
	Crons() features.CronsClient
	CEL() features.CELClient
	Schedules() features.SchedulesClient
	Events() v0Client.EventClient
	Filters() features.FiltersClient
	Webhooks() features.WebhooksClient

	// TODO Run, RunNoWait, bulk
}

// v1HatchetClientImpl is the implementation of the HatchetClient interface.
type v1HatchetClientImpl struct {
	v0 v0Client.Client

	metrics    features.MetricsClient
	rateLimits features.RateLimitsClient
	runs       features.RunsClient
	workers    features.WorkersClient
	workflows  features.WorkflowsClient
	cel        features.CELClient
	crons      features.CronsClient
	schedules  features.SchedulesClient
	filters    features.FiltersClient
	webhooks   features.WebhooksClient
}

// Deprecated: NewHatchetClient is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func NewHatchetClient(config ...Config) (HatchetClient, error) {
	cf := &v0Config.ClientConfigFile{}

	v0Opts := []v0Client.ClientOpt{}

	if len(config) > 0 {
		opts := config[0]
		cf = mapConfigToCF(opts)

		if config[0].Logger != nil {
			v0Opts = append(v0Opts, v0Client.WithLogger(config[0].Logger))
		}
	}

	client, err := v0Client.NewFromConfigFile(cf, v0Opts...)
	if err != nil {
		return nil, err
	}

	return &v1HatchetClientImpl{
		v0: client,
	}, nil
}

// Deprecated: Metrics is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (c *v1HatchetClientImpl) Metrics() features.MetricsClient {
	if c.metrics == nil {
		api := c.V0().API()
		tenantId := c.V0().TenantId()
		c.metrics = features.NewMetricsClient(api, &tenantId)
	}

	return c.metrics
}

// Deprecated: V0 is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
//
// V0 returns the underlying V0 client for backward compatibility.
func (c *v1HatchetClientImpl) V0() v0Client.Client {
	return c.v0
}

// Deprecated: Workflow is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
//
// Workflow creates a new workflow declaration with the provided options.
func (c *v1HatchetClientImpl) Workflow(opts create.WorkflowCreateOpts[any]) workflow.WorkflowDeclaration[any, any] {
	return workflow.NewWorkflowDeclaration[any, any](opts, c.v0)
}

// Deprecated: Events is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (c *v1HatchetClientImpl) Events() v0Client.EventClient {
	return c.V0().Event()
}

// Deprecated: Worker is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
//
// Worker creates and configures a new worker with the provided options and optional configuration functions.
func (c *v1HatchetClientImpl) Worker(opts worker.WorkerOpts) (worker.Worker, error) {
	return worker.NewWorker(c.workers, c.v0, opts)
}

// Deprecated: RateLimits is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (c *v1HatchetClientImpl) RateLimits() features.RateLimitsClient {
	if c.rateLimits == nil {
		api := c.V0().API()
		admin := c.V0().Admin()
		tenantId := c.V0().TenantId()
		c.rateLimits = features.NewRateLimitsClient(api, &tenantId, &admin)
	}
	return c.rateLimits
}

// Deprecated: Runs is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (c *v1HatchetClientImpl) Runs() features.RunsClient {
	if c.runs == nil {
		tenantId := c.V0().TenantId()
		c.runs = features.NewRunsClient(c.V0().API(), &tenantId, c.V0())
	}
	return c.runs
}

// Deprecated: Workers is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (c *v1HatchetClientImpl) Workers() features.WorkersClient {
	if c.workers == nil {
		api := c.V0().API()
		tenantId := c.V0().TenantId()
		c.workers = features.NewWorkersClient(api, &tenantId)
	}
	return c.workers
}

// Deprecated: Workflows is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (c *v1HatchetClientImpl) Workflows() features.WorkflowsClient {
	if c.workflows == nil {
		api := c.V0().API()
		tenantId := c.V0().TenantId()
		c.workflows = features.NewWorkflowsClient(api, &tenantId)
	}
	return c.workflows
}

// Deprecated: Crons is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (c *v1HatchetClientImpl) Crons() features.CronsClient {
	if c.crons == nil {
		api := c.V0().API()
		tenantId := c.V0().TenantId()
		c.crons = features.NewCronsClient(api, &tenantId)
	}
	return c.crons
}

// Deprecated: Schedules is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (c *v1HatchetClientImpl) Schedules() features.SchedulesClient {
	if c.schedules == nil {
		api := c.V0().API()
		tenantId := c.V0().TenantId()
		namespace := c.V0().Namespace()

		c.schedules = features.NewSchedulesClient(api, &tenantId, &namespace)
	}
	return c.schedules
}

// Deprecated: Filters is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (c *v1HatchetClientImpl) Filters() features.FiltersClient {
	if c.filters == nil {
		api := c.V0().API()
		tenantID := c.V0().TenantId()
		c.filters = features.NewFiltersClient(api, &tenantID)
	}
	return c.filters
}

// Deprecated: CEL is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (c *v1HatchetClientImpl) CEL() features.CELClient {
	if c.cel == nil {
		api := c.V0().API()
		tenantId := c.V0().TenantId()
		c.cel = features.NewCELClient(api, &tenantId)
	}
	return c.cel
}

// Deprecated: Webhooks is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (c *v1HatchetClientImpl) Webhooks() features.WebhooksClient {
	if c.webhooks == nil {
		api := c.V0().API()
		tenantID := c.V0().TenantId()
		c.webhooks = features.NewWebhooksClient(api, &tenantID)
	}
	return c.webhooks
}
