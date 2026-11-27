package proxy

import (
	"errors"
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Throttle reports whether err is, or wraps, a gRPC ResourceExhausted status that carries a
// google.rpc.RetryInfo detail, and if so how long the caller should wait before retrying and the
// status's own message.
//
// The engine attaches RetryInfo only to transient throttling. A ResourceExhausted status without
// it is a tenant quota the caller cannot resolve by retrying, so the two must not be conflated.
func Throttle(err error) (retryAfter time.Duration, message string, ok bool) {
	var withStatus interface{ GRPCStatus() *status.Status }

	if !errors.As(err, &withStatus) {
		return 0, "", false
	}

	st := withStatus.GRPCStatus()

	if st.Code() != codes.ResourceExhausted {
		return 0, "", false
	}

	for _, detail := range st.Details() {
		if info, isRetryInfo := detail.(*errdetails.RetryInfo); isRetryInfo {
			return info.GetRetryDelay().AsDuration(), st.Message(), true
		}
	}

	return 0, "", false
}
