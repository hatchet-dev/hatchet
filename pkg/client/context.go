// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	grpcMetadata "google.golang.org/grpc/metadata"

	"context"
)

type contextLoader struct {
	// The token
	Token string
}

func newContextLoader(token string) *contextLoader {
	return &contextLoader{
		Token: token,
	}
}

func (c *contextLoader) newContext(ctx context.Context) context.Context {
	md := grpcMetadata.New(map[string]string{
		"authorization": "Bearer " + c.Token,
	})

	return grpcMetadata.NewOutgoingContext(ctx, md)
}
