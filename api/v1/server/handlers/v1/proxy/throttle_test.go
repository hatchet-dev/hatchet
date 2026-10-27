package proxy

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
)

func statusWithRetryInfo(t *testing.T, code codes.Code, delay time.Duration) error {
	t.Helper()

	st, err := status.New(code, "throttled").WithDetails(&errdetails.RetryInfo{RetryDelay: durationpb.New(delay)})
	require.NoError(t, err)

	return st.Err()
}

func TestThrottleRetryAfter(t *testing.T) {
	retryInfo := statusWithRetryInfo(t, codes.ResourceExhausted, 3*time.Second)

	for name, tc := range map[string]struct {
		err       error
		want      time.Duration
		throttled bool
	}{
		"resource exhausted with retry info":         {err: retryInfo, want: 3 * time.Second, throttled: true},
		"wrapped resource exhausted with retry info": {err: fmt.Errorf("trigger: %w", retryInfo), want: 3 * time.Second, throttled: true},
		"resource exhausted without retry info":      {err: status.Error(codes.ResourceExhausted, "quota")},
		"retry info on another code":                 {err: statusWithRetryInfo(t, codes.Unavailable, time.Second)},
		"plain error":                                {err: fmt.Errorf("boom")},
		"nil":                                        {},
	} {
		t.Run(name, func(t *testing.T) {
			got, throttled := ThrottleRetryAfter(tc.err)

			require.Equal(t, tc.throttled, throttled)
			require.Equal(t, tc.want, got)
		})
	}
}
