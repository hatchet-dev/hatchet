// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	grpcMetadata "google.golang.org/grpc/metadata"

	"context"
)

type contextLoader struct {
	Token   string
	extraMD map[string]string
}

func newContextLoader(token string, extraMD map[string]string) *contextLoader {
	return &contextLoader{
		Token:   token,
		extraMD: extraMD,
	}
}

func (c *contextLoader) newContext(ctx context.Context) context.Context {
	pairs := map[string]string{
		"authorization": "Bearer " + c.Token,
	}
	for k, v := range c.extraMD {
		pairs[k] = v
	}
	md := grpcMetadata.New(pairs)
	return grpcMetadata.NewOutgoingContext(ctx, md)
}
