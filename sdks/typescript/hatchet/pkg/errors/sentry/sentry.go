package sentry

import (
	"context"
	"fmt"

	"github.com/getsentry/sentry-go"
)

type SentryAlerter struct {
	client *sentry.Client
}

func noIntegrations(ints []sentry.Integration) []sentry.Integration {
	return []sentry.Integration{}
}

type SentryAlerterOpts struct {
	DSN         string
	Environment string
	SampleRate  float64
}

func NewSentryAlerter(opts *SentryAlerterOpts) (*SentryAlerter, error) {
	sentryClient, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:              opts.DSN,
		AttachStacktrace: true,
		Integrations:     noIntegrations,
		Environment:      opts.Environment,
		SampleRate:       opts.SampleRate,
	})

	if err != nil {
		return nil, err
	}

	return &SentryAlerter{
		client: sentryClient,
	}, nil
}

func (s *SentryAlerter) SendAlert(ctx context.Context, err error, data map[string]interface{}) {
	if data == nil {
		data = make(map[string]interface{})
	}

	scope := sentry.NewScope()

	for key, val := range data {
		scope.SetTag(key, fmt.Sprintf("%v", val))
	}

	s.client.CaptureException(
		err,
		&sentry.EventHint{
			Data: data,
		},
		scope,
	)
}
