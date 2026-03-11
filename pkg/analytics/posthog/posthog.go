package posthog

import (
	"context"
	"fmt"

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
	client *posthog.Client
	l      *zerolog.Logger
}

type PosthogAnalyticsOpts struct {
	ApiKey   string
	Endpoint string
	Logger   *zerolog.Logger
}

func NewPosthogAnalytics(opts *PosthogAnalyticsOpts) (*PosthogAnalytics, error) {
	if opts.ApiKey == "" || opts.Endpoint == "" {
		return nil, fmt.Errorf("api key and endpoint are required if posthog is enabled")
	}
	if opts.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

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

	return &PosthogAnalytics{
		client: &phClient,
		l:      opts.Logger,
	}, nil
}

func (p *PosthogAnalytics) Enqueue(ctx context.Context, resource analytics.Resource, action analytics.Action, userID *uuid.UUID, tenantId *uuid.UUID, resourceId string, properties map[string]interface{}) {
	var tokenID *uuid.UUID
	if userID == nil {
		tokenID = analytics.TokenIDFromContext(ctx)
	}

	event := string(resource) + ":" + string(action)

	props := map[string]interface{}{
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

	if tenantId != nil {
		group = posthog.NewGroups().Set("tenant", *tenantId)
	}

	err := (*p.client).Enqueue(posthog.Capture{
		DistinctId: analytics.DistinctID(userID, tokenID),
		Event:      event,
		Properties: props,
		Groups:     group,
	})

	if err != nil {
		p.l.Error().Err(err).Msg("error enqueuing posthog event")
	}
}

func (p *PosthogAnalytics) Identify(userId uuid.UUID, properties map[string]interface{}) {
	err := (*p.client).Enqueue(posthog.Identify{
		DistinctId: analytics.DistinctID(&userId, nil),
		Properties: map[string]interface{}{
			"$set": properties,
		},
	})

	if err != nil {
		p.l.Error().Err(err).Msg("error enqueuing posthog identify")
	}
}

func (p *PosthogAnalytics) Tenant(tenantId uuid.UUID, data map[string]interface{}) {
	err := (*p.client).Enqueue(posthog.GroupIdentify{
		Type: "tenant",
		Key:  tenantId.String(),
		Properties: map[string]interface{}{
			"$set": data,
		},
	})
	if err != nil {
		p.l.Error().Err(err).Msg("error enqueuing posthog group identify")
	}
}
