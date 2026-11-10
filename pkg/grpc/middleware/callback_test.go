package middleware

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/pkg/constants"
)

func TestTriggerResourceCallback(t *testing.T) {
	type reported struct {
		resourceID    any
		resourceType  any
		correlationID any
	}

	var got []reported

	ctx := WithCallback(context.Background(), func(ctx context.Context) {
		got = append(got, reported{
			resourceID:    ctx.Value(constants.ResourceIdKey),
			resourceType:  ctx.Value(constants.ResourceTypeKey),
			correlationID: ctx.Value(constants.CorrelationIdKey),
		})
	})

	correlationID := "corr-1"
	TriggerResourceCallback(ctx, constants.ResourceTypeEvent, "event-1", &correlationID)
	TriggerResourceCallback(ctx, constants.ResourceTypeEvent, "event-2", nil)

	assert.Equal(t, []reported{
		{resourceID: "event-1", resourceType: constants.ResourceTypeEvent, correlationID: "corr-1"},
		{resourceID: "event-2", resourceType: constants.ResourceTypeEvent, correlationID: nil},
	}, got)
}
