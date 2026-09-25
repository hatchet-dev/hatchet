package transformers

import (
	"encoding/json"
	"math"

	"github.com/rs/zerolog/log"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// ToV1ServerlessEndpoint transforms a stored endpoint into its API representation. The
// encrypted signing secret and the internal shard assignment are intentionally left out.
func ToV1ServerlessEndpoint(endpoint *sqlcv1.V1ServerlessEndpoint) gen.V1ServerlessEndpoint {
	labels := map[string]interface{}{}

	if len(endpoint.Labels) > 0 {
		// Labels are validated as a JSON object on write, so a failure here is a data bug worth
		// logging rather than failing the whole response over.
		if err := json.Unmarshal(endpoint.Labels, &labels); err != nil {
			log.Error().Err(err).Str("endpoint", endpoint.ID.String()).Msg("failed to unmarshal serverless endpoint labels")
		}
	}

	registeredActions := endpoint.RegisteredActions

	if registeredActions == nil {
		registeredActions = []string{}
	}

	status := gen.V1ServerlessEndpointStatus{
		RegisteredActions: registeredActions,
	}

	if endpoint.Healthy.Valid {
		status.Healthy = &endpoint.Healthy.Bool
	}

	if endpoint.StatusError.Valid {
		status.Error = &endpoint.StatusError.String
	}

	if endpoint.StatusChangedAt.Valid {
		status.ChangedAt = &endpoint.StatusChangedAt.Time
	}

	return gen.V1ServerlessEndpoint{
		Metadata: gen.APIResourceMeta{
			Id:        endpoint.ID.String(),
			CreatedAt: endpoint.CreatedAt.Time,
			UpdatedAt: endpoint.UpdatedAt.Time,
		},
		TenantId:              endpoint.TenantID,
		Name:                  endpoint.Name,
		Namespace:             endpoint.Namespace,
		Kind:                  gen.V1ServerlessEndpointKind(endpoint.Kind),
		HealthcheckUrl:        endpoint.HealthcheckUrl,
		TriggerUrl:            endpoint.TriggerUrl,
		RequestTimeoutSeconds: endpoint.RequestTimeoutSeconds,
		PollIntervalSeconds:   endpoint.PollIntervalSeconds,
		InlineWaitBudgetMs:    endpoint.InlineWaitBudgetMs,
		Labels:                labels,
		Enabled:               endpoint.Enabled,
		Status:                status,
	}
}

func ToV1ServerlessEndpointList(endpoints []*sqlcv1.V1ServerlessEndpoint, total, limit, offset int64) gen.V1ServerlessEndpointList {
	rows := make([]gen.V1ServerlessEndpoint, len(endpoints))

	for i, endpoint := range endpoints {
		rows[i] = ToV1ServerlessEndpoint(endpoint)
	}

	var currentPage, nextPage, totalPages int64

	if limit > 0 {
		currentPage = offset / limit
		nextPage = currentPage + 1
		totalPages = int64(math.Ceil(float64(total) / float64(limit)))
	}

	return gen.V1ServerlessEndpointList{
		Rows: &rows,
		Pagination: &gen.PaginationResponse{
			CurrentPage: &currentPage,
			NextPage:    &nextPage,
			NumPages:    &totalPages,
		},
	}
}

func ToV1ServerlessTenantSettings(tenant *sqlcv1.V1ServerlessTenant) gen.V1ServerlessTenantSettings {
	return gen.V1ServerlessTenantSettings{
		TenantId:   tenant.TenantID,
		ShardCount: tenant.ShardCount,
	}
}
