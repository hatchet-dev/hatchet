package posthog

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/posthog/posthog-go"
	"github.com/rs/zerolog"
)

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

func (p *PosthogAnalytics) Enqueue(event string, userId string, tenantId *uuid.UUID, set map[string]interface{}, metadata map[string]interface{}) {

	var group posthog.Groups

	if tenantId != nil {
		group = posthog.NewGroups().Set("tenant", *tenantId)
	}

	err := (*p.client).Enqueue(posthog.Capture{
		DistinctId: userId,
		Event:      event,
		Properties: map[string]interface{}{
			"$set":      set,
			"$metadata": metadata,
		},
		Groups: group,
	})

	if err != nil {
		p.l.Error().Err(err).Msg("error enqueuing posthog event")
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
