package posthog

import (
	"fmt"

	"github.com/posthog/posthog-go"
)

type PosthogAnalytics struct {
	client *posthog.Client
}

type PosthogAnalyticsOpts struct {
	ApiKey   string
	Endpoint string
}

func NewPosthogAnalytics(opts *PosthogAnalyticsOpts) (*PosthogAnalytics, error) {
	if opts.ApiKey == "" || opts.Endpoint == "" {
		return nil, fmt.Errorf("api key and endpoint are required if posthog is enabled")
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
	}, nil
}

func (p *PosthogAnalytics) Enqueue(event string, userId string, tenantId *string, data map[string]interface{}) {

	var group posthog.Groups

	if tenantId != nil {
		group = posthog.NewGroups().Set("tenant", *tenantId)
	}

	var _ = (*p.client).Enqueue(posthog.Capture{
		DistinctId: userId,
		Event:      event,
		Properties: map[string]interface{}{
			"$set": data,
		},
		Groups: group,
	})
}

func (p *PosthogAnalytics) Tenant(tenantId string, data map[string]interface{}) {
	var _ = (*p.client).Enqueue(posthog.GroupIdentify{
		Type: "tenant",
		Key:  tenantId,
		Properties: map[string]interface{}{
			"$set": data,
		},
	})
}
