package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type GRPCAuthN struct {
	config *server.ServerConfig

	l *zerolog.Logger
}

func NewAuthN(config *server.ServerConfig) *GRPCAuthN {
	return &GRPCAuthN{
		config: config,
		l:      config.Logger,
	}
}

// Middleware validates the bearer token in header and returns ctx carrying the tenant, the
// token id and the call source. It runs from the request gate, before any message is read.
func (a *GRPCAuthN) Middleware(ctx context.Context, header http.Header) (context.Context, error) {
	forbidden := connect.NewError(connect.CodeUnauthenticated, errors.New("invalid auth token"))
	token, err := bearerToken(header)

	if err != nil {
		a.l.Debug().Ctx(ctx).Err(err).Msgf("error getting bearer token from request: %s", err)
		return nil, forbidden
	}

	tenantId, tokenUUID, err := a.config.Auth.JWTManager.ValidateTenantToken(ctx, token)

	if err != nil {
		a.l.Debug().Ctx(ctx).Err(err).Msgf("error validating tenant token: %s", err)

		return nil, forbidden
	}

	ctx = context.WithValue(ctx, analytics.APITokenIDKey, tokenUUID)
	ctx = context.WithValue(ctx, analytics.TenantIDKey, tenantId)

	source := analytics.SourceGRPC
	if vals := header.Values(analytics.SourceMetadataKey); len(vals) > 0 {
		source = analytics.Source(vals[0])
	}
	ctx = context.WithValue(ctx, analytics.SourceKey, source)

	queriedTenant, err := a.config.V1.Tenant().GetTenantByID(ctx, tenantId)

	if err != nil {
		a.l.Debug().Ctx(ctx).Err(err).Msgf("error getting tenant by id: %s", err)
		return nil, forbidden
	}

	return context.WithValue(ctx, "tenant", queriedTenant), nil
}

// bearerToken extracts the token from the first authorization header value. The scheme is
// matched case-insensitively.
func bearerToken(header http.Header) (string, error) {
	vals := header.Values("Authorization")
	if len(vals) == 0 {
		return "", errors.New("request unauthenticated with bearer")
	}

	scheme, token, found := strings.Cut(vals[0], " ")
	if !found {
		return "", errors.New("bad authorization string")
	}

	if !strings.EqualFold(scheme, "bearer") {
		return "", errors.New("request unauthenticated with bearer")
	}

	return token, nil
}
