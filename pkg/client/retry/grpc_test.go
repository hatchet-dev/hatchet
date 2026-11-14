package retry

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestGRPCRetryableCodesExcludeFailedPrecondition(t *testing.T) {
	t.Parallel()

	retryable := grpcRetryableCodes()
	assert.Contains(t, retryable, codes.ResourceExhausted)
	assert.Contains(t, retryable, codes.DeadlineExceeded)
	assert.Contains(t, retryable, codes.Internal)
	assert.Contains(t, retryable, codes.Unavailable)
	assert.NotContains(t, retryable, codes.FailedPrecondition)
}

func TestGRPCFullJitterBackoff(t *testing.T) {
	t.Parallel()

	backoff := grpcFullJitterBackoff(func(maxExclusive int64) int64 {
		return maxExclusive - 1
	})

	assert.Equal(t, 5*time.Second, backoff(context.Background(), 1))
	assert.Equal(t, 10*time.Second, backoff(context.Background(), 2))
	assert.Equal(t, grpcMaxDelay, backoff(context.Background(), 99))
}

func TestGRPCBackoffUsesServerRetryDelay(t *testing.T) {
	t.Parallel()

	backoff := grpcFullJitterBackoff(func(maxExclusive int64) int64 {
		return maxExclusive - 1
	})

	serverDelay := new(time.Duration)
	ctx := context.WithValue(context.Background(), serverRetryDelayKey{}, serverDelay)

	assert.Equal(t, 5*time.Second, backoff(ctx, 1), "no RetryInfo keeps the jittered backoff")

	*serverDelay = 300 * time.Millisecond
	assert.Equal(t, 300*time.Millisecond, backoff(ctx, 1))

	*serverDelay = 10 * time.Minute
	assert.Equal(t, 5*time.Second, grpcMaxServerRetryDelay)
	assert.Equal(t, grpcMaxServerRetryDelay, backoff(ctx, 1), "a longer server delay is capped")
	assert.Equal(t, grpcMaxServerRetryDelay, backoff(ctx, 3), "the cap does not grow with the attempt")
	assert.Equal(t, grpcMaxServerRetryDelay, backoff(ctx, 99))

	*serverDelay = 0
	assert.Equal(t, 20*time.Second, backoff(ctx, 3), "a zero delay keeps the jittered backoff")
}

// scriptedHealthServer fails Check with the scripted errors in order, then succeeds, and records
// when each attempt arrived.
type scriptedHealthServer struct {
	healthpb.UnimplementedHealthServer

	mu       sync.Mutex
	errs     []error
	attempts []time.Time
}

func (s *scriptedHealthServer) Check(context.Context, *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.attempts = append(s.attempts, time.Now())
	if len(s.errs) > 0 {
		err := s.errs[0]
		s.errs = s.errs[1:]
		return nil, err
	}

	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

func (s *scriptedHealthServer) gaps() []time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()

	gaps := make([]time.Duration, 0, len(s.attempts))
	for i := 1; i < len(s.attempts); i++ {
		gaps = append(gaps, s.attempts[i].Sub(s.attempts[i-1]))
	}

	return gaps
}

// fixedJitter makes the client's own backoff a constant, short and distinct from the server delays.
const fixedJitter = 50 * time.Millisecond

func dialScripted(t *testing.T, errs ...error) (healthpb.HealthClient, *scriptedHealthServer) {
	t.Helper()

	srv := &scriptedHealthServer{errs: errs}
	lis := bufconn.Listen(1 << 20)
	gs := grpc.NewServer()
	healthpb.RegisterHealthServer(gs, srv)
	go func() { _ = gs.Serve(lis) }()
	t.Cleanup(gs.Stop)

	l := zerolog.Nop()
	opts := append([]grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
	}, grpcDialOptions(&l, func(int64) int64 { return int64(fixedJitter) })...)

	conn, err := grpc.NewClient("passthrough:///bufnet", opts...)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	return healthpb.NewHealthClient(conn), srv
}

func withRetryInfo(t *testing.T, code codes.Code, delay time.Duration) error {
	t.Helper()

	st, err := status.New(code, "throttled").WithDetails(&errdetails.RetryInfo{RetryDelay: durationpb.New(delay)})
	require.NoError(t, err)

	return st.Err()
}

func TestGRPCUnaryRetryWaitsServerRetryDelay(t *testing.T) {
	t.Parallel()

	const serverDelay = 300 * time.Millisecond
	client, srv := dialScripted(t, withRetryInfo(t, codes.ResourceExhausted, serverDelay))

	_, err := client.Check(context.Background(), &healthpb.HealthCheckRequest{})
	require.NoError(t, err)

	gaps := srv.gaps()
	require.Len(t, gaps, 1)
	assert.GreaterOrEqual(t, gaps[0], serverDelay)
	assert.Less(t, gaps[0], serverDelay+5*time.Second)
}

func TestGRPCUnaryRetryWithoutRetryInfoKeepsBackoff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		errs []error
	}{
		{name: "resource exhausted", errs: []error{status.Error(codes.ResourceExhausted, "quota")}},
		{name: "unavailable", errs: []error{status.Error(codes.Unavailable, "down")}},
		{name: "internal", errs: []error{status.Error(codes.Internal, "boom")}},
		{name: "deadline exceeded", errs: []error{status.Error(codes.DeadlineExceeded, "slow")}},
		{
			// The delay applies to the attempt that carried it, not to later attempts without one.
			// It is shorter than fixedJitter, so a carried-over delay would fail the lower bound.
			name: "retry info then none",
			errs: []error{withRetryInfo(t, codes.ResourceExhausted, 10*time.Millisecond), status.Error(codes.Unavailable, "down")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, srv := dialScripted(t, tt.errs...)

			_, err := client.Check(context.Background(), &healthpb.HealthCheckRequest{})
			require.NoError(t, err)

			gaps := srv.gaps()
			require.Len(t, gaps, len(tt.errs))
			last := gaps[len(gaps)-1]
			assert.GreaterOrEqual(t, last, fixedJitter)
			assert.Less(t, last, fixedJitter+5*time.Second)
		})
	}
}

func TestGRPCUnaryRetryStopsAfterMaxAttempts(t *testing.T) {
	t.Parallel()

	errs := make([]error, grpcMaxRetries)
	for i := range errs {
		errs[i] = withRetryInfo(t, codes.ResourceExhausted, 10*time.Millisecond)
	}
	client, srv := dialScripted(t, errs...)

	_, err := client.Check(context.Background(), &healthpb.HealthCheckRequest{})
	assert.Equal(t, codes.ResourceExhausted, status.Code(err))
	assert.Len(t, srv.gaps(), grpcMaxRetries-1)
}

func TestGRPCUnaryRetryInfoOnNonRetryableCodeIsNotRetried(t *testing.T) {
	t.Parallel()

	client, srv := dialScripted(t, withRetryInfo(t, codes.FailedPrecondition, 10*time.Millisecond))

	_, err := client.Check(context.Background(), &healthpb.HealthCheckRequest{})
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	assert.Empty(t, srv.gaps())
}
