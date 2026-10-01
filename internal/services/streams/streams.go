// Package streams implements the V1Streams gRPC service: durable, topic-based
// message streams.
package streams

import (
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	v1connect "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1/v1connect"
	"github.com/hatchet-dev/hatchet/internal/services/shared/streams"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

// Service is the durable-streams gRPC service. It also exposes
// CancelStreamSessions so the engine can hang up long-lived Subscribe RPCs
// during graceful shutdown, matching dispatcher.Dispatcher.
type Service interface {
	v1connect.V1StreamsHandler
	CancelStreamSessions()
	Cleanup() error
}

type ServiceOptFunc func(*ServiceOpts)

type ServiceOpts struct {
	pubsub msgqueue.PubSub
	repov1 v1.Repository
	l      *zerolog.Logger
}

func WithPubSub(pubsub msgqueue.PubSub) ServiceOptFunc {
	return func(opts *ServiceOpts) {
		opts.pubsub = pubsub
	}
}

func WithRepositoryV1(r v1.Repository) ServiceOptFunc {
	return func(opts *ServiceOpts) {
		opts.repov1 = r
	}
}

func WithLogger(l *zerolog.Logger) ServiceOptFunc {
	return func(opts *ServiceOpts) {
		opts.l = l
	}
}

func defaultServiceOpts() *ServiceOpts {
	l := logger.NewDefaultLogger("streams")

	return &ServiceOpts{
		l: &l,
	}
}

type ServiceImpl struct {
	v1connect.UnimplementedV1StreamsHandler

	pubsub msgqueue.PubSub
	repo   v1.Repository
	l      *zerolog.Logger

	publisher      *publishBatcher
	streamSessions *streams.Registry
	topicPollers   *topicPollerRegistry
}

func NewService(fs ...ServiceOptFunc) (Service, error) {
	opts := defaultServiceOpts()

	for _, f := range fs {
		f(opts)
	}

	return &ServiceImpl{
		pubsub:         opts.pubsub,
		repo:           opts.repov1,
		l:              opts.l,
		publisher:      newPublishBatcher(opts.repov1.Streams(), opts.pubsub, opts.l, publishBatchWorkers),
		streamSessions: streams.NewRegistry(),
		topicPollers:   newTopicPollerRegistry(opts.repov1.Streams(), opts.pubsub, opts.l, subscribeTailPollInterval, subscribeIdleHangupTimeout),
	}, nil
}

// CancelStreamSessions hangs up every registered long-lived Subscribe RPC. It is
// safe to call multiple times or with no sessions registered.
func (s *ServiceImpl) CancelStreamSessions() {
	s.streamSessions.CancelAll()
}

// Cleanup stops the publisher's workers and the wake subscription.
func (s *ServiceImpl) Cleanup() error {
	s.publisher.stop()
	return s.topicPollers.Close()
}
