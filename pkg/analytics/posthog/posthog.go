package posthog

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/posthog/posthog-go"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/analytics"
)

type zerologAdapter struct {
	l *zerolog.Logger
}

func (z *zerologAdapter) Debugf(format string, args ...interface{}) {
	z.l.Debug().Msgf(format, args...)
}

func (z *zerologAdapter) Logf(format string, args ...interface{}) {
	z.l.Info().Msgf(format, args...)
}

func (z *zerologAdapter) Warnf(format string, args ...interface{}) {
	z.l.Warn().Msgf(format, args...)
}

func (z *zerologAdapter) Errorf(format string, args ...interface{}) {
	z.l.Error().Msgf(format, args...)
}

type PosthogAnalytics struct {
	client     *posthog.Client
	l          *zerolog.Logger
	aggregator *analytics.Aggregator
}

type PosthogAnalyticsOpts struct {
	ApiKey           string
	Endpoint         string
	Logger           *zerolog.Logger
	AggregateEnabled bool
	FlushInterval    time.Duration
	MaxKeys          int64
}

func NewPosthogAnalytics(opts *PosthogAnalyticsOpts) (*PosthogAnalytics, error) {
	if opts.ApiKey == "" || opts.Endpoint == "" {
		return nil, fmt.Errorf("api key and endpoint are required if posthog is enabled")
	}
	if opts.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	flushInterval := opts.FlushInterval

	phClient, err := posthog.NewWithConfig(
		opts.ApiKey,
		posthog.Config{
			Endpoint: opts.Endpoint,
			Logger:   &zerologAdapter{l: opts.Logger},
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create posthog client: %w", err)
	}

	p := &PosthogAnalytics{
		client: &phClient,
		l:      opts.Logger,
	}
	p.aggregator = analytics.NewAggregator(opts.Logger, opts.AggregateEnabled, flushInterval, opts.MaxKeys, p.flushCount)
	return p, nil
}

func (p *PosthogAnalytics) Enqueue(ctx context.Context, resource analytics.Resource, action analytics.Action, resourceId string, properties analytics.Properties) {
	userID := analytics.UserIDFromContext(ctx)
	tenantID := analytics.TenantIDFromContext(ctx)
	tokenID := analytics.TokenIDFromContext(ctx)

	event := string(resource) + ":" + string(action)

	props := analytics.Properties{
		"resource_id": resourceId,
	}
	if userID != nil {
		props["user_id"] = userID.String()
	}
	if tokenID != nil {
		props["token_id"] = tokenID.String()
	}
	for k, v := range properties {
		props[k] = v
	}

	var group posthog.Groups

	if tenantID != nil {
		group = posthog.NewGroups().Set("tenant", *tenantID)
	}

	err := (*p.client).Enqueue(posthog.Capture{
		DistinctId: analytics.DistinctID(userID, tokenID, tenantID),
		Event:      event,
		Properties: posthog.Properties(props),
		Groups:     group,
	})

	if err != nil {
		p.l.Error().Err(err).Str("event", event).Msg("error enqueuing posthog event")
	}
}

func (p *PosthogAnalytics) Count(ctx context.Context, resource analytics.Resource, action analytics.Action, props ...analytics.Properties) {
	tenantID := analytics.TenantIDFromContext(ctx)
	tokenID := analytics.TokenIDFromContext(ctx)

	var tid uuid.UUID
	if tenantID != nil {
		tid = *tenantID
	}

	p.aggregator.Count(resource, action, tid, tokenID, 1, props...)
}

func (p *PosthogAnalytics) flushCount(resource analytics.Resource, action analytics.Action, tenantID uuid.UUID, tokenID *uuid.UUID, count int64, properties analytics.Properties) {
	merged := analytics.Properties{"count": count}
	for k, v := range properties {
		merged[k] = v
	}
	ctx := context.Background()
	ctx = context.WithValue(ctx, analytics.TenantIDKey, tenantID)
	if tokenID != nil {
		ctx = context.WithValue(ctx, analytics.APITokenIDKey, *tokenID)
	}
	p.Enqueue(ctx, resource, action, "", merged)
}

func (p *PosthogAnalytics) Start() {
	p.aggregator.Start()
}

func (p *PosthogAnalytics) Identify(userId uuid.UUID, properties analytics.Properties) {
	err := (*p.client).Enqueue(posthog.Identify{
		DistinctId: analytics.DistinctID(&userId, nil, nil),
		Properties: posthog.Properties{
			"$set": properties,
		},
	})

	if err != nil {
		p.l.Error().Err(err).Str("user_id", userId.String()).Msg("error enqueuing posthog identify")
	}
}

func (p *PosthogAnalytics) Tenant(tenantId uuid.UUID, data analytics.Properties) {
	err := (*p.client).Enqueue(posthog.GroupIdentify{
		Type: "tenant",
		Key:  tenantId.String(),
		Properties: posthog.Properties{
			"$set": data,
		},
	})
	if err != nil {
		p.l.Error().Err(err).Str("tenant_id", tenantId.String()).Msg("error enqueuing posthog group identify")
	}
}

func (p *PosthogAnalytics) Close() error {
	p.aggregator.Shutdown()
	return (*p.client).Close()
}
