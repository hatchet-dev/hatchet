package middleware

import (
	"context"

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
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

	if a.config.Auth.NoAuthEnabled && a.config.Auth.CustomAuthenticator == nil {
		return a.noAuthContext(ctx)
	}

	token, err := auth.AuthFromMD(ctx, "bearer")

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

	span := trace.SpanFromContext(ctx)
	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant.id", Value: tenantId},
	)

	source := analytics.SourceGRPC
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get(analytics.SourceMetadataKey); len(vals) > 0 {
			source = analytics.Source(vals[0])
		}
	}
	ctx = context.WithValue(ctx, analytics.SourceKey, source)

	queriedTenant, err := a.config.V1.Tenant().GetTenantByID(ctx, tenantId)

	if err != nil {
		a.l.Debug().Ctx(ctx).Err(err).Msgf("error getting tenant by id: %s", err)
		return nil, forbidden
	}

	return context.WithValue(ctx, "tenant", queriedTenant), nil
}

func (a *GRPCAuthN) noAuthContext(ctx context.Context) (context.Context, error) {
	forbidden := status.Errorf(codes.Unauthenticated, "invalid auth token")

	tenantId, err := uuid.Parse(a.config.Auth.NoAuthTenantID)

	if err != nil {
		a.l.Error().Ctx(ctx).Err(err).Msgf("no-auth mode: invalid default tenant id: %s", err)
		return nil, forbidden
	}

	queriedTenant, err := a.config.V1.Tenant().GetTenantByID(ctx, tenantId)

	if err != nil {
		a.l.Error().Ctx(ctx).Err(err).Msgf("no-auth mode: could not resolve default tenant: %s", err)
		return nil, forbidden
	}

	ctx = context.WithValue(ctx, analytics.TenantIDKey, tenantId)
	ctx = context.WithValue(ctx, analytics.SourceKey, analytics.SourceGRPC)

	span := trace.SpanFromContext(ctx)
	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: tenantId})

	return context.WithValue(ctx, "tenant", queriedTenant), nil
}
