// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"

	"google.golang.org/grpc"

	sharedcontracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// authedConn adds the client's credentials to every call, so a generated stub can use it directly.
type authedConn struct {
	conn *grpc.ClientConn
	ctx  *contextLoader
}

func (a *authedConn) Invoke(ctx context.Context, method string, args, reply any, opts ...grpc.CallOption) error {
	return a.conn.Invoke(a.ctx.newContext(ctx), method, args, reply, opts...)
}

func (a *authedConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return a.conn.NewStream(a.ctx.newContext(ctx), desc, method, opts...)
}

func newStreams(conn *grpc.ClientConn, opts *sharedClientOpts) sharedcontracts.V1StreamsClient {
	return sharedcontracts.NewV1StreamsClient(&authedConn{conn: conn, ctx: opts.ctxLoader})
}
