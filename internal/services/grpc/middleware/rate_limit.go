package middleware

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"golang.org/x/time/rate"

	"github.com/hatchet-dev/hatchet/pkg/analytics"
)

type HatchetApiTokenRateLimiter struct {
	eventsLimiter     *rate.Limiter
	dispatcherLimiter *rate.Limiter
	workflowLimiter   *rate.Limiter
	adminV1Limiter    *rate.Limiter
	otelColLimiter    *rate.Limiter
}

type HatchetRateLimiter struct {
	mu           sync.Mutex
	rateLimiters map[string]*HatchetApiTokenRateLimiter
	rate         rate.Limit
	burst        int
	l            *zerolog.Logger
}

func (rl *HatchetRateLimiter) GetOrCreateTenantRateLimiter(rateLimitToken string) *HatchetApiTokenRateLimiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if _, ok := rl.rateLimiters[rateLimitToken]; !ok {
		rl.rateLimiters[rateLimitToken] = &HatchetApiTokenRateLimiter{
			eventsLimiter:   rate.NewLimiter(rl.rate, rl.burst),
			workflowLimiter: rate.NewLimiter(rl.rate, rl.burst),
			adminV1Limiter:  rate.NewLimiter(rl.rate, rl.burst),
			// 10x the rate for dispatcher and otelcol
			dispatcherLimiter: rate.NewLimiter(rl.rate*10, rl.burst*10),
			otelColLimiter:    rate.NewLimiter(rl.rate*10, rl.burst*10),
		}
	}

	return rl.rateLimiters[rateLimitToken]
}

func NewHatchetRateLimiter(r rate.Limit, b int, l *zerolog.Logger) *HatchetRateLimiter {
	l.Info().Msgf("grpc rate limit set to %v per second with a burst of %v (10X rate for Dispatcher)", r, b)
	return &HatchetRateLimiter{
		rateLimiters: make(map[string]*HatchetApiTokenRateLimiter),
		rate:         r,
		burst:        b,
		l:            l,
	}
}

// Limit is called before each request is processed. It should return an error if rate-limited.
// procedure is the full method name, for example /Dispatcher/Register.
func (r *HatchetRateLimiter) Limit(ctx context.Context, procedure string) error {
	serviceName := procedure

	rateLimitToken, ok := ctx.Value(analytics.APITokenIDKey).(uuid.UUID)

	if !ok || rateLimitToken == uuid.Nil {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("no rate limit token found"))
	}

	switch matchServiceName(serviceName) {
	case "dispatcher":

		if !r.GetOrCreateTenantRateLimiter(rateLimitToken.String()).dispatcherLimiter.Allow() {
			r.l.Info().Ctx(ctx).Msgf("dispatcher rate limit (%v per second) exceeded", r.GetOrCreateTenantRateLimiter(rateLimitToken.String()).dispatcherLimiter.Limit())
			return connect.NewError(connect.CodeResourceExhausted, errors.New("dispatcher rate limit exceeded"))
		}

	case "events":
		if !r.GetOrCreateTenantRateLimiter(rateLimitToken.String()).eventsLimiter.Allow() {
			r.l.Info().Ctx(ctx).Msgf("ingest rate limit (%v per second) exceeded", r.GetOrCreateTenantRateLimiter(rateLimitToken.String()).eventsLimiter.Limit())
			return connect.NewError(connect.CodeResourceExhausted, errors.New("ingest rate limit exceeded"))
		}

	case "workflow":
		if !r.GetOrCreateTenantRateLimiter(rateLimitToken.String()).workflowLimiter.Allow() {
			r.l.Info().Ctx(ctx).Msgf("workflow rate limit (%v per second) exceeded", r.GetOrCreateTenantRateLimiter(rateLimitToken.String()).workflowLimiter.Limit())
			return connect.NewError(connect.CodeResourceExhausted, errors.New("admin rate limit exceeded"))
		}
	case "admin":
		if !r.GetOrCreateTenantRateLimiter(rateLimitToken.String()).adminV1Limiter.Allow() {
			r.l.Info().Ctx(ctx).Msgf("admin rate limit (%v per second) exceeded", r.GetOrCreateTenantRateLimiter(rateLimitToken.String()).adminV1Limiter.Limit())
			return connect.NewError(connect.CodeResourceExhausted, errors.New("admin rate limit exceeded"))
		}

	case "otelcol":
		if !r.GetOrCreateTenantRateLimiter(rateLimitToken.String()).otelColLimiter.Allow() {
			r.l.Info().Ctx(ctx).Msgf("otel collector rate limit (%v per second) exceeded", r.GetOrCreateTenantRateLimiter(rateLimitToken.String()).otelColLimiter.Limit())
			return connect.NewError(connect.CodeResourceExhausted, errors.New("otel collector rate limit exceeded"))
		}

	default:
		return connect.NewError(connect.CodeInternal, fmt.Errorf("service %s not recognized", serviceName))
	}

	return nil
}

func matchServiceName(name string) string {
	switch {
	case strings.HasPrefix(name, "/Dispatcher"):
		return "dispatcher"
	case strings.HasPrefix(name, "/v1.V1Dispatcher"):
		return "dispatcher"
	case strings.HasPrefix(name, "/v1.OperatorService"):
		// operators running outside the engine share the dispatcher bucket
		return "dispatcher"
	case strings.HasPrefix(name, "/EventsService"):
		return "events"
	case strings.HasPrefix(name, "/WorkflowService"):
		return "workflow"
	case strings.HasPrefix(name, "/v1.AdminService"):
		return "admin"
	case strings.HasPrefix(name, "/opentelemetry.proto.collector"):
		return "otelcol"
	default:
		return "unknown"
	}
}
