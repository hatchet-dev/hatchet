package client

import (
	"net/http"

	"github.com/hatchet-dev/hatchet/internal/shutdown"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

var _ rest.HttpRequestDoer = new(httpClient)

type httpClient struct {
	wrapped rest.HttpRequestDoer
	shutSig *shutdown.Signaller
}

func newClient(shutsig *shutdown.Signaller) rest.HttpRequestDoer {
	return &httpClient{
		wrapped: http.DefaultClient,
		shutSig: shutsig,
	}
}

func (c *httpClient) Do(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	// Here we derive a new context that is cancelled when a shutdown
	// is signalled.
	ctx, cancel := c.shutSig.WithShutdown(ctx)
	defer cancel()

	return c.wrapped.Do(req.WithContext(ctx))
}
