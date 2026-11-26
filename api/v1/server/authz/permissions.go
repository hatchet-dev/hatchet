package authz

import (
	"sync"

	"github.com/hatchet-dev/hatchet/internal/permissions"
	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/enums"
)

func LoadPermissions(config *server.ServerConfig) (*pkgauthz.RESTPermissions, *pkgauthz.TokenPermissions, error) {
	perms, err := permissions.Users()
	if err != nil {
		return nil, nil, err
	}

	// Extension operations are not in the OSS spec, so they are granted after users.yaml is validated
	// against it. Users() returns a fresh copy each call, so these grants stay local to this server.
	if err := perms.Grant(string(enums.TenantMemberRoleVIEWER), config.Auth.AllowedOperations...); err != nil {
		return nil, nil, err
	}

	if err := perms.Grant(string(enums.TenantMemberRoleMEMBER), config.Auth.AllowedWriteOperations...); err != nil {
		return nil, nil, err
	}

	tokens, err := permissions.Tokens()
	if err != nil {
		return nil, nil, err
	}

	// The extension's operations are its infrastructure, so they get their own capabilities; read-only
	// inherits infra:read and "all" already grants every operation. Tokens() returns a fresh copy, so the
	// gRPC authorizer never sees these REST grants.
	if err := tokens.GrantREST("infra:read", config.Auth.AllowedOperations...); err != nil {
		return nil, nil, err
	}

	if err := tokens.GrantREST("infra:write", config.Auth.AllowedWriteOperations...); err != nil {
		return nil, nil, err
	}

	return perms, tokens, nil
}

// CachedPermissions defers loading to the first request because the service constructors cannot return an error.
func CachedPermissions(config *server.ServerConfig) func() (*pkgauthz.RESTPermissions, *pkgauthz.TokenPermissions, error) {
	type loaded struct {
		perms  *pkgauthz.RESTPermissions
		tokens *pkgauthz.TokenPermissions
	}

	load := sync.OnceValues(func() (loaded, error) {
		perms, tokens, err := LoadPermissions(config)
		return loaded{perms: perms, tokens: tokens}, err
	})

	return func() (*pkgauthz.RESTPermissions, *pkgauthz.TokenPermissions, error) {
		l, err := load()
		return l.perms, l.tokens, err
	}
}
