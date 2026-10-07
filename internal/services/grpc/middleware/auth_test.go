package middleware

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

type readonlyRepo struct {
	fakeRepository
	readOnly bool
}
type readonlyTokens struct {
	v1.APITokenRepository
	readOnly bool
}

func (r readonlyRepo) APIToken() v1.APITokenRepository { return readonlyTokens{readOnly: r.readOnly} }
func (r readonlyTokens) GetAPITokenById(_ context.Context, id uuid.UUID) (*sqlcv1.APIToken, error) {
	return &sqlcv1.APIToken{ID: id, ReadOnly: r.readOnly}, nil
}
func TestReadOnlyTokenAuthentication(t *testing.T) {
	for _, readOnly := range []bool{false, true} {
		t.Run(map[bool]string{true: "read-only", false: "unrestricted"}[readOnly], func(t *testing.T) {
			l := zerolog.Nop()
			config := &server.ServerConfig{Layer: &database.Layer{V1: readonlyRepo{readOnly: readOnly}}}
			config.Logger = &l
			config.Auth.JWTManager = &fakeJWTManager{}
			ctx, err := NewAuthN(config).Middleware(context.Background(), http.Header{"Authorization": []string{"Bearer " + testToken}})
			if readOnly {
				require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
				require.Nil(t, ctx)
			} else {
				require.NoError(t, err)
				require.Equal(t, testTenantID, ctx.Value("tenant").(*sqlcv1.Tenant).ID)
			}
		})
	}
}
