package middleware

import (
	"context"
	"strings"
	"sync"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"golang.org/x/time/rate"
)

type HatchetApiTokenRateLimiter struct {
	eventsLimiter     *rate.Limiter
	dispatcherLimiter *rate.Limiter
	workflowLimiter   *rate.Limiter
	adminV1Limiter    *rate.Limiter
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
			// 10x the rate for dispatcher
			dispatcherLimiter: rate.NewLimiter(rl.rate*10, rl.burst*10),
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
func (r *HatchetRateLimiter) Limit(ctx context.Context) error {
	serviceName, ok := ctx.Value(grpcServiceName).(string)
	if !ok {
		return status.Errorf(codes.Internal, "no server in context")
	}

	rateLimitToken := ctx.Value("rate_limit_token").(string)

	if rateLimitToken == "" {
		return status.Errorf(codes.Unauthenticated, "no rate limit token found")
	}

	switch matchServiceName(serviceName) {
	case "dispatcher":

		if !r.GetOrCreateTenantRateLimiter(rateLimitToken).dispatcherLimiter.Allow() {
			r.l.Info().Msgf("dispatcher rate limit (%v per second) exceeded", r.GetOrCreateTenantRateLimiter(rateLimitToken).dispatcherLimiter.Limit())
			return status.Errorf(codes.ResourceExhausted, "dispatcher rate limit exceeded")
		}

	case "events":
		if !r.GetOrCreateTenantRateLimiter(rateLimitToken).eventsLimiter.Allow() {
			r.l.Info().Msgf("ingest rate limit (%v per second) exceeded", r.GetOrCreateTenantRateLimiter(rateLimitToken).eventsLimiter.Limit())
			return status.Errorf(codes.ResourceExhausted, "ingest rate limit exceeded")
		}

	case "workflow":
		if !r.GetOrCreateTenantRateLimiter(rateLimitToken).workflowLimiter.Allow() {
			r.l.Info().Msgf("workflow rate limit (%v per second) exceeded", r.GetOrCreateTenantRateLimiter(rateLimitToken).workflowLimiter.Limit())
			return status.Errorf(codes.ResourceExhausted, "admin rate limit exceeded")
		}
	case "admin":
		if !r.GetOrCreateTenantRateLimiter(rateLimitToken).adminV1Limiter.Allow() {
			r.l.Info().Msgf("admin rate limit (%v per second) exceeded", r.GetOrCreateTenantRateLimiter(rateLimitToken).adminV1Limiter.Limit())
			return status.Errorf(codes.ResourceExhausted, "admin rate limit exceeded")
		}

	default:
		return status.Errorf(codes.Internal, "service %s not recognized", serviceName)
	}

	return nil
}

type contextKey string

const grpcServiceName = contextKey("grpcServiceName")

func AttachServerNameInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	ctx = context.WithValue(ctx, grpcServiceName, info.FullMethod)

	return handler(ctx, req)
}

func ServerNameStreamingInterceptor(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {

	ctx := context.WithValue(ss.Context(), grpcServiceName, info.FullMethod)

	wrappedStream := &wrappedServerStream{
		ServerStream: ss,
		ctx:          ctx,
	}

	return handler(srv, wrappedStream)
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

func matchServiceName(name string) string {
	switch {
	case strings.HasPrefix(name, "/Dispatcher"):
		return "dispatcher"
	case strings.HasPrefix(name, "/EventsService"):
		return "events"
	case strings.HasPrefix(name, "/WorkflowService"):
		return "workflow"
	case strings.HasPrefix(name, "/AdminService"):
		return "admin"
	default:
		return "unknown"
	}
}
