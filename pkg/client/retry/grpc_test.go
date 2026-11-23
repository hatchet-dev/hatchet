package retry

import (
	"context"
	"encoding/json"
	"net"
	"os"
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

	zeroDelay := context.WithValue(context.Background(), serverRetryDelayKey{}, new(time.Duration))
	assert.Equal(t, 20*time.Second, backoff(zeroDelay, 3), "a server delay of zero keeps the jittered backoff")
}

// TestGRPCThrottledCallScenarios runs the scenarios every SDK's tests share. The retry interceptor
// waits on real timers and some scenarios wait 12 seconds, so instead of making a call this steps
// through the interceptor's loop with the two functions the client gives it: retriable decides
// whether an attempt is retried, and the backoff, called with the retry's 1-based attempt number
// before each retry, reads the delay that unaryRetry stores in the context slot.
func TestGRPCThrottledCallScenarios(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../../../api-contracts/sdk-conformance/throttled_calls.json")
	require.NoError(t, err)

	var file struct {
		Jitter    float64 `json:"jitter"`
		Scenarios []struct {
			Name     string  `json:"name"`
			DelaysMs []int64 `json:"delays_ms"`
			WaitsMs  []int64 `json:"waits_ms"`
			Succeeds bool    `json:"succeeds"`
		} `json:"scenarios"`
	}
	require.NoError(t, json.Unmarshal(raw, &file))
	require.NotEmpty(t, file.Scenarios)
	require.Positive(t, file.Jitter)

	// The random source held at its lowest and at its highest value.
	randoms := []struct {
		name   string
		jitter func(maxExclusive int64) int64
		extend float64
	}{
		{"shortest waits", func(int64) int64 { return 0 }, 1},
		{"longest waits", func(maxExclusive int64) int64 { return maxExclusive - 1 }, 1 + file.Jitter},
	}

	for _, sc := range file.Scenarios {
		for _, random := range randoms {
			t.Run(sc.Name+", "+random.name, func(t *testing.T) {
				t.Parallel()

				backoff := grpcFullJitterBackoff(random.jitter)
				serverDelay := new(time.Duration)
				ctx := context.WithValue(context.Background(), serverRetryDelayKey{}, serverDelay)

				waits := []time.Duration{}
				succeeded := false
				for attempt := uint(0); attempt < grpcMaxRetries; attempt++ {
					if attempt > 0 {
						waits = append(waits, backoff(ctx, attempt))
					}
					if int(attempt) == len(sc.DelaysMs) {
						succeeded = true
						break
					}

					err := throttled(t, time.Duration(sc.DelaysMs[attempt])*time.Millisecond)
					*serverDelay = retryInfoDelay(err)
					if !retriableWithinServerDelayCap(err) {
						break
					}
				}

				require.Len(t, waits, len(sc.WaitsMs))
				for i, ms := range sc.WaitsMs {
					want := float64(time.Duration(ms)*time.Millisecond) * random.extend
					assert.InDelta(t, want, float64(waits[i]), float64(time.Microsecond), "wait %d", i)
				}
				assert.Equal(t, sc.Succeeds, succeeded)
			})
		}
	}
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

func throttled(t *testing.T, delay time.Duration) error {
	t.Helper()

	st, err := status.New(codes.ResourceExhausted, "throttled").WithDetails(&errdetails.RetryInfo{RetryDelay: durationpb.New(delay)})
	require.NoError(t, err)

	return st.Err()
}

// The retry interceptor's backoff sees only the call's context, so these run calls against a real
// server: the delay of a failed attempt has to reach the backoff, and only the next one.
func TestGRPCUnaryRetryWaitsServerRetryDelay(t *testing.T) {
	t.Parallel()

	// The client's own backoff is this constant, between the two server delays below.
	const ownBackoff = 50 * time.Millisecond
	const serverDelay = 300 * time.Millisecond

	dial := func(t *testing.T, errs ...error) (*grpc.ClientConn, *scriptedHealthServer) {
		t.Helper()

		srv := &scriptedHealthServer{errs: errs}
		lis := bufconn.Listen(1 << 20)
		// Methods of services not registered here, such as the dispatcher's, reach Check too.
		gs := grpc.NewServer(grpc.UnknownServiceHandler(func(_ any, stream grpc.ServerStream) error {
			req := &healthpb.HealthCheckRequest{}
			if err := stream.RecvMsg(req); err != nil {
				return err
			}
			resp, err := srv.Check(stream.Context(), req)
			if err != nil {
				return err
			}
			return stream.SendMsg(resp)
		}))
		healthpb.RegisterHealthServer(gs, srv)
		go func() { _ = gs.Serve(lis) }()
		t.Cleanup(gs.Stop)

		l := zerolog.Nop()
		opts := append([]grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		}, grpcDialOptions(&l, func(int64) int64 { return int64(ownBackoff) })...)

		conn, err := grpc.NewClient("passthrough:///bufnet", opts...)
		require.NoError(t, err)
		t.Cleanup(func() { _ = conn.Close() })

		return conn, srv
	}

	t.Run("waits the delay", func(t *testing.T) {
		t.Parallel()

		conn, srv := dial(t,
			throttled(t, serverDelay),
			throttled(t, time.Millisecond),
			status.Error(codes.Unavailable, "down"),
		)

		_, err := healthpb.NewHealthClient(conn).Check(context.Background(), &healthpb.HealthCheckRequest{})
		require.NoError(t, err)

		srv.mu.Lock()
		defer srv.mu.Unlock()

		require.Len(t, srv.attempts, 4)
		assert.GreaterOrEqual(t, srv.attempts[1].Sub(srv.attempts[0]), serverDelay, "waits the server's delay, not its own backoff")
		assert.GreaterOrEqual(t, srv.attempts[3].Sub(srv.attempts[2]), ownBackoff, "a delay does not carry over to an attempt without one")
	})

	t.Run("a delay over the cap fails the call", func(t *testing.T) {
		t.Parallel()

		conn, srv := dial(t, throttled(t, 10*time.Minute))

		_, err := healthpb.NewHealthClient(conn).Check(context.Background(), &healthpb.HealthCheckRequest{})
		assert.Equal(t, codes.ResourceExhausted, status.Code(err))
		assert.Equal(t, 10*time.Minute, retryInfoDelay(err), "the caller can read the server's delay")

		srv.mu.Lock()
		defer srv.mu.Unlock()

		assert.Len(t, srv.attempts, 1)
	})

	t.Run("dispatcher calls ignore the delay", func(t *testing.T) {
		t.Parallel()

		conn, srv := dial(t, throttled(t, 10*time.Minute), throttled(t, 10*time.Minute))

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := conn.Invoke(ctx, "/Dispatcher/SendStepActionEvent", &healthpb.HealthCheckRequest{}, &healthpb.HealthCheckResponse{})
		require.NoError(t, err)

		srv.mu.Lock()
		defer srv.mu.Unlock()

		require.Len(t, srv.attempts, 3)
		assert.GreaterOrEqual(t, srv.attempts[1].Sub(srv.attempts[0]), ownBackoff, "retried on its own backoff")
		assert.GreaterOrEqual(t, srv.attempts[2].Sub(srv.attempts[1]), ownBackoff, "retried on its own backoff")
	})
}
