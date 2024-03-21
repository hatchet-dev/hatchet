package middleware

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/internal/config/server"
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

func (a *GRPCAuthN) Middleware(ctx context.Context) (context.Context, error) {
	forbidden := status.Errorf(codes.Unauthenticated, "invalid auth token")
	token, err := auth.AuthFromMD(ctx, "bearer")

	if err != nil {
		a.l.Debug().Err(err).Msgf("error getting bearer token from request: %s", err)
		return nil, forbidden
	}

	tenantId, err := a.config.Auth.JWTManager.ValidateTenantToken(token)

	if err != nil {
		a.l.Debug().Err(err).Msgf("error validating tenant token: %s", err)

		return nil, forbidden
	}

	// get the tenant id
	queriedTenant, err := a.config.EngineRepository.Tenant().GetTenantByID(tenantId)

	if err != nil {
		a.l.Debug().Err(err).Msgf("error getting tenant by id: %s", err)
		return nil, forbidden
	}

	return context.WithValue(ctx, "tenant", queriedTenant), nil
}
