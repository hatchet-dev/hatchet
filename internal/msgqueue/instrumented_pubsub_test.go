package msgqueue

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	prommetrics "github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
)

type capturingPubSub struct {
	received *Message
	handler  MsgHandler
}

func (c *capturingPubSub) Pub(ctx context.Context, topic Topic, msg *Message) error {
	c.received = msg
	return nil
}

func (c *capturingPubSub) Sub(topic Topic, handler MsgHandler) (func() error, error) {
	c.handler = handler
	return func() error { return nil }, nil
}

func (c *capturingPubSub) IsReady() bool {
	return true
}

func histogramSampleCount(t *testing.T, vec *prometheus.HistogramVec, labelValues ...string) uint64 {
	t.Helper()

	obs, err := vec.GetMetricWithLabelValues(labelValues...)
	require.NoError(t, err)

	m := &dto.Metric{}
	require.NoError(t, obs.(prometheus.Metric).Write(m))

	return m.GetHistogram().GetSampleCount()
}

func TestInstrumentedPubSubStampsPublishedAtWithoutMutatingCaller(t *testing.T) {
	inner := &capturingPubSub{}
	ps := NewInstrumentedPubSub(inner, "test-stamp")

	caller := &Message{ID: "msg-1"}
	err := ps.Pub(context.Background(), SchedulerPartitionTopic("p"), caller)
	require.NoError(t, err)

	assert.True(t, caller.PublishedAt.IsZero(), "caller message must not be mutated")
	require.NotNil(t, inner.received)
	assert.False(t, inner.received.PublishedAt.IsZero(), "inner backend must receive a stamped copy")
	assert.NotSame(t, caller, inner.received)
}

func TestInstrumentedPubSubTransitObservesOnlyStampedMessages(t *testing.T) {
	kind := "test-transit"
	topicKind := string(TopicKindSchedulerPartition)

	inner := &capturingPubSub{}
	ps := NewInstrumentedPubSub(inner, kind)

	var handled int
	_, err := ps.Sub(SchedulerPartitionTopic("p"), func(msg *Message) error {
		handled++
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, inner.handler)

	before := histogramSampleCount(t, prommetrics.PubSubTransit, kind, topicKind)

	require.NoError(t, inner.handler(&Message{ID: "unstamped"}))
	assert.EqualValues(t, before, histogramSampleCount(t, prommetrics.PubSubTransit, kind, topicKind))

	require.NoError(t, inner.handler(&Message{ID: "stamped", PublishedAt: time.Now().Add(-5 * time.Millisecond)}))
	assert.EqualValues(t, before+1, histogramSampleCount(t, prommetrics.PubSubTransit, kind, topicKind))
	assert.Equal(t, 2, handled)
}

func TestInstrumentedPubSubPublishDurationResultLabels(t *testing.T) {
	topicKind := string(TopicKindSchedulerPartition)

	beforeOK := histogramSampleCount(t, prommetrics.PubSubPublishDuration, "test-pub-ok", topicKind, "ok")
	okInner := &capturingPubSub{}
	err := NewInstrumentedPubSub(okInner, "test-pub-ok").Pub(context.Background(), SchedulerPartitionTopic("p"), &Message{ID: "ok"})
	require.NoError(t, err)
	assert.EqualValues(t, beforeOK+1, histogramSampleCount(t, prommetrics.PubSubPublishDuration, "test-pub-ok", topicKind, "ok"))

	beforeErr := histogramSampleCount(t, prommetrics.PubSubPublishDuration, "test-pub-error", topicKind, "error")
	err = NewInstrumentedPubSub(&erroringPubSub{}, "test-pub-error").Pub(context.Background(), SchedulerPartitionTopic("p"), &Message{ID: "err"})
	assert.Error(t, err)
	assert.EqualValues(t, beforeErr+1, histogramSampleCount(t, prommetrics.PubSubPublishDuration, "test-pub-error", topicKind, "error"))
}
