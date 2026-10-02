// Package streams implements the V1Streams gRPC service: durable, topic-based
// message streams.
package streams

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	v1connect "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1/v1connect"
	"github.com/hatchet-dev/hatchet/internal/services/shared/streams"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

// Service is the durable streams gRPC service. CancelStreamSessions lets the
// engine hang up Subscribe RPCs on shutdown.
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

	// so each publish doesn't query the entitlement
	entitled *expirable.LRU[uuid.UUID, bool]
}

// how long an entitlement change can take to apply
const entitlementCacheTTL = time.Minute

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
		entitled:       expirable.NewLRU[uuid.UUID, bool](10000, nil, entitlementCacheTTL),
	}, nil
}

// CancelStreamSessions hangs up every Subscribe RPC. Safe to call repeatedly.
func (s *ServiceImpl) CancelStreamSessions() {
	s.streamSessions.CancelAll()
}

// Cleanup stops the publisher's workers and the wake subscription.
func (s *ServiceImpl) Cleanup() error {
	s.publisher.stop()
	return s.topicPollers.Close()
}

func (s *ServiceImpl) checkEntitled(ctx context.Context, tenantId uuid.UUID) error {
	enabled, ok := s.entitled.Get(tenantId)

	if !ok {
		var err error

		if enabled, err = s.repo.TenantEntitlement().IsDurableStreamsEnabled(ctx, tenantId); err != nil {
			return err
		}

		s.entitled.Add(tenantId, enabled)
	}

	if !enabled {
		return connect.NewError(connect.CodePermissionDenied, errors.New("durable streams are not enabled for this tenant"))
	}

	return nil
}
