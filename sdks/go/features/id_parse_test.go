package features_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/sdks/go/features"
)

func TestFiltersClient_InvalidUUIDReturnsError(t *testing.T) {
	client := features.NewFiltersClient(nil, "00000000-0000-0000-0000-000000000000")
	ctx := context.Background()

	t.Run("Get with malformed filter ID", func(t *testing.T) {
		res, err := client.Get(ctx, "invalid-uuid")
		require.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "invalid filter ID")
	})

	t.Run("Delete with malformed filter ID", func(t *testing.T) {
		res, err := client.Delete(ctx, "")
		require.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "invalid filter ID")
	})

	t.Run("Update with malformed filter ID", func(t *testing.T) {
		res, err := client.Update(ctx, "not-a-valid-uuid", rest.V1FilterUpdateJSONRequestBody{})
		require.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "invalid filter ID")
	})
}

func TestRunsClient_InvalidUUIDReturnsError(t *testing.T) {
	client := features.NewRunsClient(nil, "00000000-0000-0000-0000-000000000000", nil)
	ctx := context.Background()

	t.Run("Get with malformed run ID", func(t *testing.T) {
		res, err := client.Get(ctx, "malformed-run-id")
		require.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "invalid run ID")
	})

	t.Run("GetStatus with malformed run ID", func(t *testing.T) {
		res, err := client.GetStatus(ctx, "")
		require.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "invalid run ID")
	})

	t.Run("BranchDurableTask with malformed task external ID", func(t *testing.T) {
		res, err := client.BranchDurableTask(ctx, "bad-external-id", 1, 1)
		require.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "invalid task external ID")
	})

	t.Run("Restore with malformed task external ID", func(t *testing.T) {
		res, err := client.Restore(ctx, "invalid-uuid")
		require.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "invalid task external ID")
	})
}
