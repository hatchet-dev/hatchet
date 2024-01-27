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
