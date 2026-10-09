package proxy

import (
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ThrottleRetryAfter reports whether err is a gRPC ResourceExhausted status that carries a
// google.rpc.RetryInfo detail, and if so how long the caller should wait before retrying.
//
// The engine attaches RetryInfo only to transient throttling. A ResourceExhausted status without
// it is a tenant quota the caller cannot resolve by retrying, so the two must not be conflated.
func ThrottleRetryAfter(err error) (time.Duration, bool) {
	st, ok := status.FromError(err)

	if !ok || st.Code() != codes.ResourceExhausted {
		return 0, false
	}

	for _, detail := range st.Details() {
		if info, ok := detail.(*errdetails.RetryInfo); ok {
			return info.GetRetryDelay().AsDuration(), true
		}
	}

	return 0, false
}
