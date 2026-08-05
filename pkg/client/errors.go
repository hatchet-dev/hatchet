// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// PayloadTooLargeErr indicates the client attempted to send a gRPC message that exceeds
// the configured max message size (see Runtime.GRPCMaxMsgSize on the server, and
// grpc.MaxCallSendMsgSize on the client).
type PayloadTooLargeErr struct {
	// PayloadBytes is the exact serialized size, in bytes, of the message that was rejected.
	PayloadBytes int
	details      string
}

func (e *PayloadTooLargeErr) Error() string {
	return fmt.Sprintf(
		"payload too large: attempted to send %d, which exceeds the gRPC max message size configured for this client (%s)",
		e.PayloadBytes, e.details,
	)
}

// wrapIfPayloadTooLarge inspects err for a gRPC RESOURCE_EXHAUSTED status caused by an
// outgoing message that was larger than the configured max send size, and if so wraps it in
// a PayloadTooLargeErr reporting the exact size of the message that was rejected. Any other
// error (including other RESOURCE_EXHAUSTED causes, e.g. tenant quota limits) is returned
// unchanged.
func wrapIfPayloadTooLarge(err error, msg proto.Message) error {
	if err == nil || status.Code(err) != codes.ResourceExhausted {
		return err
	}
	details := status.Convert(err).Message()

	lower := strings.ToLower(details)
	if !strings.Contains(lower, "larger than") && !strings.Contains(lower, "too large") {
		return err
	}

	return &PayloadTooLargeErr{
		PayloadBytes: proto.Size(msg),
		details:      details,
	}
}
