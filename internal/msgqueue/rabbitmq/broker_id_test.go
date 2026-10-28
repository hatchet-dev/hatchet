//go:build integration

package rabbitmq

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
)

func TestBrokerIDReportsClusterName(t *testing.T) {
	cleanup, tq, err := New(WithURL("amqp://user:password@localhost:5672/"))
	require.NoError(t, err)

	defer func() { _ = cleanup() }()

	clusterName, _ := tq.pubChannels.getConnection().Properties["cluster_name"].(string)
	require.NotEmpty(t, clusterName, "broker should report a cluster_name")

	require.Equal(t, clusterName, tq.BrokerID())
	require.Equal(t, clusterName, msgqueue.BrokerID(tq))
}
