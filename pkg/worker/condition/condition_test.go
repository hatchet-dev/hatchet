//go:build !e2e && !load && !rampup && !integration

package condition_test

import (
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/worker/condition"
	"github.com/stretchr/testify/assert"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

func TestSleepConditionToPB_SleepForFormat(t *testing.T) {
	cases := []struct {
		duration time.Duration
		want     string
	}{
		{0, "0ms"},
		{500 * time.Millisecond, "500ms"},
		{59 * time.Second, "59000ms"},
		{60 * time.Second, "60000ms"},
		{61 * time.Second, "61000ms"},
		{30 * time.Minute, "1800000ms"},
		{time.Hour, "3600000ms"},
		{time.Hour + 10*time.Minute, "4200000ms"},
	}

	for _, tc := range cases {
		t.Run(tc.duration.String(), func(t *testing.T) {
			c := condition.SleepCondition(tc.duration)
			pb := c.ToPB(contracts.Action_CREATE)

			assert.Len(t, pb.SleepConditions, 1)
			assert.Equal(t, tc.want, pb.SleepConditions[0].SleepFor)
		})
	}
}

func TestUserEventConditionToPB_ScopeAndLookback(t *testing.T) {
	since := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)

	c := condition.UserEventCondition("user:create", "input.user_id == 1234",
		condition.WithEventScope("user_id:1234"),
		condition.WithConsiderEventsSince(since),
	)
	pb := c.ToPB(contracts.Action_CREATE)

	assert.Len(t, pb.UserEventConditions, 1)
	cond := pb.UserEventConditions[0]
	assert.Equal(t, "user:create", cond.UserEventKey)
	assert.Equal(t, "input.user_id == 1234", cond.Base.Expression)
	assert.Equal(t, "user_id:1234", cond.GetEventScope())
	assert.Equal(t, since, cond.GetConsiderEventsSince().AsTime())
}

func TestUserEventConditionToPB_NoOpts(t *testing.T) {
	pb := condition.UserEventCondition("user:create", "").ToPB(contracts.Action_CREATE)

	assert.Len(t, pb.UserEventConditions, 1)
	assert.Nil(t, pb.UserEventConditions[0].EventScope)
	assert.Nil(t, pb.UserEventConditions[0].ConsiderEventsSince)
}
