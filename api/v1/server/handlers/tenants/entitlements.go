package tenants

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

func toAPITenantEntitlements(entitlements v1.TenantEntitlements) gen.TenantEntitlements {
	return gen.TenantEntitlements{
		AuditLogs:                       entitlements.AuditLogs,
		PrometheusMetrics:               entitlements.PrometheusMetrics,
		StrictAdditionalMetadataFilters: entitlements.StrictAdditionalMetadataFilters,
		DurableStreams:                  entitlements.DurableStreams,
		ServerlessOperator:              entitlements.ServerlessOperator,
		SandboxOperator:                 entitlements.SandboxOperator,
	}
}
