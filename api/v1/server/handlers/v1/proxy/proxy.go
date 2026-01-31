package proxy

import (
	"context"
	"time"

	"github.com/google/uuid"
	client "github.com/hatchet-dev/hatchet/pkg/client/v1"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type Proxy[in, out any] struct {
	config *server.ServerConfig
	method func(ctx context.Context, cli *client.GRPCClient, input *in) (*out, error)
}

func NewProxy[in, out any](config *server.ServerConfig, method func(ctx context.Context, cli *client.GRPCClient, input *in) (*out, error)) *Proxy[in, out] {
	return &Proxy[in, out]{
		config: config,
		method: method,
	}
}

func (p *Proxy[in, out]) Do(ctx context.Context, tenant *sqlcv1.Tenant, input *in) (*out, error) {
	tenantId := tenant.ID

	expiresAt := time.Now().Add(5 * time.Minute).UTC()

	// generate the API token for the proxy request
	tok, err := p.config.Auth.JWTManager.GenerateTenantToken(ctx, tenantId, "proxy", true, &expiresAt)

	if err != nil {
		return nil, err
	}

	tokenId, err := uuid.Parse(tok.TokenId)

	if err != nil {
		return nil, err
	}

	defer func() {
		deleteCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// delete the API token
		err = p.config.V1.APIToken().DeleteAPIToken(deleteCtx, tenantId, tokenId)

		if err != nil {
			p.config.Logger.Error().Err(err).Msg("failed to delete API token")
		}
	}()

	c, err := p.config.InternalClientFactory.NewGRPCClient(tok.Token)

	if err != nil {
		return nil, err
	}

	// call the client method
	res, err := p.method(client.AuthContext(ctx, tok.Token), c, input)

	if err != nil {
		return nil, err
	}

	return res, nil
}
