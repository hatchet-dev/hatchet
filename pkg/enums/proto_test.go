package enums_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/enums"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestProtoRoundTrip(t *testing.T) {
	for _, status := range enums.AllV1ReadableStatusOlapValues() {
		p, err := enums.ToProto[v1.RunStatus](status)
		require.NoError(t, err, status)

		back, err := enums.FromProto[enums.V1ReadableStatusOlap](p)
		require.NoError(t, err, status)
		assert.Equal(t, status, back)
	}
}

func TestFromProtoRejectsValuesTheTargetLacks(t *testing.T) {
	_, err := enums.FromProto[enums.V1ConcurrencyStrategy](v1.ConcurrencyLimitStrategy_DROP_NEWEST)
	assert.Error(t, err)

	_, err = enums.FromProto[enums.V1ReadableStatusOlap](v1.RunStatus(99))
	assert.Error(t, err)
}

func TestFromProtoWithoutValidAcceptsAnyName(t *testing.T) {
	d, err := enums.FromProto[string](v1.RateLimitDuration_MINUTE)
	require.NoError(t, err)
	assert.Equal(t, "MINUTE", d)
}

func TestToProtoRejectsUnknownNames(t *testing.T) {
	_, err := enums.ToProto[v1.ConcurrencyLimitStrategy](enums.V1ConcurrencyStrategyNONE)
	assert.Error(t, err)
}

func TestPtrVariants(t *testing.T) {
	got, err := enums.FromProtoPtr[sqlcv1.StickyStrategy, v1.StickyStrategy](nil)
	require.NoError(t, err)
	assert.Nil(t, got)

	hard := v1.StickyStrategy_HARD
	got, err = enums.FromProtoPtr[sqlcv1.StickyStrategy](&hard)
	require.NoError(t, err)
	assert.Equal(t, sqlcv1.StickyStrategyHARD, *got)

	back, err := enums.ToProtoPtr[v1.StickyStrategy](got)
	require.NoError(t, err)
	assert.Equal(t, hard, *back)

	none, err := enums.ToProtoPtr[v1.StickyStrategy, sqlcv1.StickyStrategy](nil)
	require.NoError(t, err)
	assert.Nil(t, none)
}
