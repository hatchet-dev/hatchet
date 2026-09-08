package transformers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const testSigningSecretEnc = "ENCRYPTED-SIGNING-SECRET-MUST-NOT-LEAK"

func testServerlessEndpoint() *sqlcv1.V1ServerlessEndpoint {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)

	return &sqlcv1.V1ServerlessEndpoint{
		ID:                    uuid.New(),
		TenantID:              uuid.New(),
		Name:                  "billing-worker",
		Namespace:             uuid.New(),
		Kind:                  sqlcv1.V1ServerlessEndpointKindCLOUDFLAREWORKERS,
		HealthcheckUrl:        "https://example.com/hatchet/health",
		TriggerUrl:            "https://example.com/hatchet/trigger",
		SigningSecretEnc:      testSigningSecretEnc,
		RequestTimeoutSeconds: 60,
		PollIntervalSeconds:   30,
		InlineWaitBudgetMs:    5000,
		Labels:                []byte(`{"region":"us-east-1","tier":2}`),
		Enabled:               true,
		Shard:                 3,
		CreatedAt:             pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt:             pgtype.Timestamptz{Time: now, Valid: true},
	}
}

func TestToV1ServerlessEndpointNeverExposesSecret(t *testing.T) {
	endpoint := testServerlessEndpoint()

	encoded, err := json.Marshal(ToV1ServerlessEndpoint(endpoint))
	require.NoError(t, err)

	assert.NotContains(t, string(encoded), testSigningSecretEnc)
	assert.NotContains(t, string(encoded), "signingSecret")
	assert.NotContains(t, string(encoded), "shard")
}

func TestToV1ServerlessEndpointMapsConfiguration(t *testing.T) {
	endpoint := testServerlessEndpoint()

	result := ToV1ServerlessEndpoint(endpoint)

	assert.Equal(t, endpoint.ID.String(), result.Metadata.Id)
	assert.Equal(t, endpoint.TenantID, result.TenantId)
	assert.Equal(t, endpoint.Namespace, result.Namespace)
	assert.Equal(t, "billing-worker", result.Name)
	assert.Equal(t, "CLOUDFLARE_WORKERS", string(result.Kind))
	assert.Equal(t, endpoint.HealthcheckUrl, result.HealthcheckUrl)
	assert.Equal(t, endpoint.TriggerUrl, result.TriggerUrl)
	assert.Equal(t, int32(60), result.RequestTimeoutSeconds)
	assert.Equal(t, int32(30), result.PollIntervalSeconds)
	assert.Equal(t, int32(5000), result.InlineWaitBudgetMs)
	assert.True(t, result.Enabled)
	assert.Equal(t, map[string]interface{}{"region": "us-east-1", "tier": float64(2)}, result.Labels)
}

func TestToV1ServerlessEndpointStatus(t *testing.T) {
	t.Run("never polled", func(t *testing.T) {
		endpoint := testServerlessEndpoint()

		status := ToV1ServerlessEndpoint(endpoint).Status

		assert.Nil(t, status.Healthy)
		assert.Nil(t, status.Error)
		assert.Nil(t, status.ChangedAt)
		assert.NotNil(t, status.RegisteredActions, "registeredActions must serialize as [] rather than null")
		assert.Empty(t, status.RegisteredActions)
	})

	t.Run("healthy with registered actions", func(t *testing.T) {
		endpoint := testServerlessEndpoint()
		changedAt := time.Date(2026, 9, 7, 13, 0, 0, 0, time.UTC)
		endpoint.Healthy = pgtype.Bool{Bool: true, Valid: true}
		endpoint.StatusChangedAt = pgtype.Timestamptz{Time: changedAt, Valid: true}
		endpoint.RegisteredActions = []string{"ns_billing:charge", "ns_billing:refund"}

		status := ToV1ServerlessEndpoint(endpoint).Status

		require.NotNil(t, status.Healthy)
		assert.True(t, *status.Healthy)
		assert.Nil(t, status.Error)
		require.NotNil(t, status.ChangedAt)
		assert.Equal(t, changedAt, *status.ChangedAt)
		assert.Equal(t, []string{"ns_billing:charge", "ns_billing:refund"}, status.RegisteredActions)
	})

	t.Run("unhealthy with error", func(t *testing.T) {
		endpoint := testServerlessEndpoint()
		endpoint.Healthy = pgtype.Bool{Bool: false, Valid: true}
		endpoint.StatusError = pgtype.Text{String: "healthcheck returned 503", Valid: true}

		status := ToV1ServerlessEndpoint(endpoint).Status

		require.NotNil(t, status.Healthy)
		assert.False(t, *status.Healthy)
		require.NotNil(t, status.Error)
		assert.Equal(t, "healthcheck returned 503", *status.Error)
	})
}

func TestToV1ServerlessEndpointEmptyLabels(t *testing.T) {
	endpoint := testServerlessEndpoint()
	endpoint.Labels = nil

	result := ToV1ServerlessEndpoint(endpoint)

	assert.NotNil(t, result.Labels, "labels must serialize as {} rather than null")
	assert.Empty(t, result.Labels)
}

func TestToV1ServerlessEndpointListPagination(t *testing.T) {
	endpoints := []*sqlcv1.V1ServerlessEndpoint{testServerlessEndpoint(), testServerlessEndpoint()}

	list := ToV1ServerlessEndpointList(endpoints, 7, 2, 2)

	require.NotNil(t, list.Rows)
	assert.Len(t, *list.Rows, 2)
	require.NotNil(t, list.Pagination)
	assert.Equal(t, int64(1), *list.Pagination.CurrentPage)
	assert.Equal(t, int64(2), *list.Pagination.NextPage)
	assert.Equal(t, int64(4), *list.Pagination.NumPages)
}

func TestToV1ServerlessTenantSettings(t *testing.T) {
	tenantId := uuid.New()

	result := ToV1ServerlessTenantSettings(&sqlcv1.V1ServerlessTenant{TenantID: tenantId, ShardCount: 4})

	assert.Equal(t, tenantId, result.TenantId)
	assert.Equal(t, int32(4), result.ShardCount)
}
