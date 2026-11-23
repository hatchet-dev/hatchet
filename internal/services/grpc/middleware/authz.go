package middleware

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/rs/zerolog"

	grpcauthz "github.com/hatchet-dev/hatchet/internal/services/grpc/authz"
	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
)

type GRPCAuthZ struct {
	authorizer pkgauthz.Authorizer
	l          *zerolog.Logger
}

func NewAuthZ(l *zerolog.Logger) (*GRPCAuthZ, error) {
	authorizer, err := grpcauthz.New()
	if err != nil {
		return nil, err
	}

	return &GRPCAuthZ{authorizer: authorizer, l: l}, nil
}

// Authorize checks the authenticated token's capabilities against the procedure.
func (a *GRPCAuthZ) Authorize(ctx context.Context, procedure string) error {
	principal := pkgauthz.PrincipalFrom(ctx)

	var scope pkgauthz.Scope
	if t, ok := principal.(pkgauthz.TokenPrincipal); ok && t.TenantID != nil {
		scope = pkgauthz.TenantScope(*t.TenantID)
	}

	d, err := a.authorizer.Authorize(ctx, pkgauthz.Request{Principal: principal, Action: pkgauthz.GRPC(procedure), Scope: scope})
	if err != nil {
		a.l.Error().Ctx(ctx).Err(err).Str("procedure", procedure).Msg("could not authorize gRPC call")
		return connect.NewError(connect.CodeInternal, errors.New("An internal error occurred."))
	}

	if d.Allowed {
		return nil
	}

	return connect.NewError(connect.CodePermissionDenied, denialMessage(d.Reason, procedure))
}

func denialMessage(reason pkgauthz.DenyReason, procedure string) error {
	switch reason {
	case pkgauthz.TokenRestricted:
		return errors.New("read-only tokens cannot use gRPC")
	case pkgauthz.MissingCapability:
		return fmt.Errorf("token does not have a capability that allows %s", procedure)
	case pkgauthz.NotMember, pkgauthz.RoleInsufficient, pkgauthz.EmailUnverified, pkgauthz.NotAPIToken:
	}

	return errors.New("not authorized")
}
