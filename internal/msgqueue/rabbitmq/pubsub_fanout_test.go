package rabbitmq

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
)

func TestIsFanout(t *testing.T) {
	assert.False(t, isFanout(msgqueue.SchedulerPartitionTopic("p")))

	for _, topic := range []msgqueue.Topic{
		msgqueue.TenantTopic(uuid.New()),
		msgqueue.StreamWakeTopic(),
		msgqueue.CronRefreshTopic(),
		msgqueue.TaskControllerWakeTopic(),
	} {
		assert.True(t, isFanout(topic), topic.Name())
	}
}
