package usagetelemetry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/posthog/posthog-go"
	"github.com/rs/zerolog"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

const reportInterval = time.Hour

const DefaultPosthogEndpoint = "https://us.i.posthog.com"

var DefaultPosthogApiKey = ""

const eventName = "oss_instance_telemetry"

type UsageTelemetry interface {
	Start(ctx context.Context)
	Shutdown()
	Active() bool
	SendFeedback(ctx context.Context, message, email string) error
}

type runCountReader interface {
	ListLastHourRunCountsByStatus(ctx context.Context) (map[string]int64, error)
}

type DefaultUsageTelemetry struct {
	enabled       bool
	logger        *zerolog.Logger
	securityCheck v1.SecurityCheckRepository
	usageMetrics  v1.UsageMetricsRepository
	olap          runCountReader
	client        posthog.Client
	shutdownOnce  sync.Once
}

type Opts struct {
	Enabled bool
	Logger  *zerolog.Logger
}

func KeyConfigured() bool {
	return DefaultPosthogApiKey != ""
}

func NewUsageTelemetry(opts *Opts, securityCheck v1.SecurityCheckRepository, usageMetrics v1.UsageMetricsRepository, olap runCountReader) (UsageTelemetry, error) {
	if DefaultPosthogApiKey == "" {
		return &noOpUsageTelemetry{}, nil
	}

	client, err := posthog.NewWithConfig(DefaultPosthogApiKey, posthog.Config{
		Endpoint: DefaultPosthogEndpoint,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create posthog client for usage telemetry: %w", err)
	}

	return &DefaultUsageTelemetry{
		enabled:       opts.Enabled,
		logger:        opts.Logger,
		securityCheck: securityCheck,
		usageMetrics:  usageMetrics,
		olap:          olap,
		client:        client,
	}, nil
}

func (t *DefaultUsageTelemetry) Start(ctx context.Context) {
	if !t.enabled {
		return
	}

	t.report(ctx)

	ticker := time.NewTicker(reportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.report(ctx)
		}
	}
}

func (t *DefaultUsageTelemetry) Active() bool {
	return t.enabled && KeyConfigured()
}

func (t *DefaultUsageTelemetry) Shutdown() {
	t.shutdownOnce.Do(func() {
		_ = t.client.Close()
	})
}

func (t *DefaultUsageTelemetry) SendFeedback(ctx context.Context, message, email string) error {
	ident, err := t.securityCheck.GetIdent()
	if err != nil {
		return err
	}

	props := posthog.NewProperties().
		Set("message", message)

	if email != "" {
		props.Set("email", email)
	}

	return t.client.Enqueue(posthog.Capture{
		DistinctId: ident,
		Event:      "oss_feedback_submitted",
		Properties: props,
	})
}

func (t *DefaultUsageTelemetry) report(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			t.logger.Debug().Msgf("panic in usage telemetry report: %v", r)
		}
	}()

	ident, err := t.securityCheck.GetIdent()
	if err != nil {
		t.logger.Debug().Err(err).Msg("usage telemetry: could not read ident, skipping")
		return
	}

	metrics, err := t.usageMetrics.GetUsageMetrics(ctx)
	if err != nil {
		t.logger.Debug().Err(err).Msg("usage telemetry: could not gather metrics, skipping")
		return
	}

	props := posthog.NewProperties().
		Set("tenant_count", metrics.TenantCount).
		Set("user_count", metrics.UserCount).
		Set("workflow_count", metrics.WorkflowCount).
		Set("worker_count", metrics.WorkerCount)

	for lang, count := range metrics.WorkersByLanguage {
		props.Set("workers_"+lang, count)
	}

	if t.olap != nil {
		if runCounts, err := t.olap.ListLastHourRunCountsByStatus(ctx); err != nil {
			t.logger.Debug().Err(err).Msg("usage telemetry: could not gather last-hour run counts")
		} else {
			var totalRuns int64
			for status, count := range runCounts {
				props.Set("runs_last_hour_"+status, count)
				totalRuns += count
			}
			props.Set("runs_last_hour_total", totalRuns)
		}
	}

	if err := t.client.Enqueue(posthog.Capture{
		DistinctId: ident,
		Event:      eventName,
		Properties: props,
	}); err != nil {
		t.logger.Debug().Err(err).Msg("usage telemetry: could not enqueue event")
	}
}

type noOpUsageTelemetry struct{}

func (n *noOpUsageTelemetry) Start(ctx context.Context) {}
func (n *noOpUsageTelemetry) Shutdown()                 {}
func (n *noOpUsageTelemetry) Active() bool              { return false }
func (n *noOpUsageTelemetry) SendFeedback(ctx context.Context, message, email string) error {
	return nil
}
