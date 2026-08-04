package msgqueue

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
)

// instrumentedPubSub decorates a PubSub with publish-duration and transit-latency
// Prometheus histograms. It never mutates the caller's *Message.
type instrumentedPubSub struct {
	inner PubSub
	kind  string
}

// NewInstrumentedPubSub wraps a PubSub so that Pub stamps PublishedAt and
// observes publisher-side duration, and Sub observes transit latency for
// stamped messages. kind is the backend label ("rabbitmq"|"postgres"|"nats").
func NewInstrumentedPubSub(inner PubSub, kind string) PubSub {
	return &instrumentedPubSub{inner: inner, kind: kind}
}

func (i *instrumentedPubSub) Pub(ctx context.Context, topic Topic, msg *Message) error {
	start := time.Now()

	cp := msg.Clone()
	cp.PublishedAt = start
	err := i.inner.Pub(ctx, topic, cp)
	elapsed := time.Since(start).Seconds()

	result := "ok"
	if err != nil {
		result = "error"
	}

	prometheus.PubSubPublishDuration.WithLabelValues(i.kind, string(topic.Kind()), result).Observe(elapsed)

	return err
}

func (i *instrumentedPubSub) Sub(topic Topic, handler MsgHandler) (func() error, error) {
	return i.inner.Sub(topic, func(msg *Message) error {
		if !msg.PublishedAt.IsZero() {
			transit := max(0, time.Since(msg.PublishedAt).Seconds())
			prometheus.PubSubTransit.WithLabelValues(i.kind, string(topic.Kind())).Observe(transit)
		}

		return handler(msg)
	})
}

func (i *instrumentedPubSub) IsReady() bool {
	return i.inner.IsReady()
}
