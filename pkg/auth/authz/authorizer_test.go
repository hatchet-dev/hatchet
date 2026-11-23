package authz

import (
	"context"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts"
)

const testPermissions = `
roles:
  ALL:
    permissions: ["*"]
  ADMIN:
    inherits: [VIEWER]
    permissions: [Write]
  VIEWER:
    permissions: [Read]
`

type staticRoles struct {
	role string
	ok   bool
}

func (s staticRoles) ResolveRole(context.Context, Principal, Action) (string, bool, error) {
	return s.role, s.ok, nil
}

func testSpec(operationIDs ...string) *openapi3.T {
	paths := openapi3.NewPaths()
	for _, id := range operationIDs {
		paths.Set("/"+id, &openapi3.PathItem{Get: &openapi3.Operation{OperationID: id}})
	}

	return &openapi3.T{Paths: paths}
}

func loadTestREST(t *testing.T) *RESTPermissions {
	t.Helper()

	pm, err := LoadRESTPermissions([]byte(testPermissions), testSpec("Read", "Write"))
	require.NoError(t, err)

	return pm
}

func newTestAuthorizer(t *testing.T, role string, opts ...Option) Authorizer {
	t.Helper()

	pm := loadTestREST(t)

	a, err := New(append([]Option{WithPermissions(RoleBased(pm, staticRoles{role: role, ok: true}, func(Request) bool { return true }))}, opts...)...)
	require.NoError(t, err)

	return a
}

func testRequest(action string) Request {
	return Request{Principal: UserPrincipal{UserID: uuid.New()}, Action: REST(action, nil)}
}

func TestRESTPermissions(t *testing.T) {
	pm := loadTestREST(t)

	require.True(t, pm.HasPermission("ADMIN", REST("Read", nil)))
	require.True(t, pm.HasPermission("ADMIN", REST("write", nil)))
	require.False(t, pm.HasPermission("VIEWER", REST("Write", nil)))
	require.True(t, pm.HasPermission("ALL", REST("AnythingAtAll", nil)))
	require.False(t, pm.HasPermission("UNKNOWN", REST("Read", nil)))
	require.False(t, pm.HasPermission("ALL", GRPC("Read")), "a REST file never answers for another action type")
}

func TestRESTPermissionsMustMatchSpec(t *testing.T) {
	_, err := LoadRESTPermissions([]byte(testPermissions), testSpec("Read"))
	require.Error(t, err)

	_, err = LoadRESTPermissions([]byte(testPermissions), testSpec("Read", "Write", "Delete"))
	require.Error(t, err)

	_, err = LoadRESTPermissions([]byte(testPermissions), nil)
	require.Error(t, err)
}

func TestRoleCyclesAreRejected(t *testing.T) {
	_, err := LoadRESTPermissions([]byte(`
roles:
  A:
    inherits: [B]
  B:
    inherits: [A]
`), testSpec())
	require.Error(t, err)
}

func TestCapToRole(t *testing.T) {
	pm := loadTestREST(t)
	a := newTestAuthorizer(t, "ALL", WithPolicies(CapToRole(pm, "VIEWER", EmailUnverified, func(p Principal) bool {
		u, ok := p.(UserPrincipal)
		return ok && !u.EmailVerified
	})))

	d, err := a.Authorize(context.Background(), Request{Principal: UserPrincipal{}, Action: REST("Write", nil)})
	require.NoError(t, err)
	require.Equal(t, Deny(EmailUnverified), d)

	d, err = a.Authorize(context.Background(), Request{Principal: UserPrincipal{}, Action: REST("Read", nil)})
	require.NoError(t, err)
	require.True(t, d.Allowed)

	d, err = a.Authorize(context.Background(), Request{Principal: UserPrincipal{EmailVerified: true}, Action: REST("Write", nil)})
	require.NoError(t, err)
	require.True(t, d.Allowed, "the cap only applies to matching principals")
}

func TestGrantRequiresExistingRole(t *testing.T) {
	pm := loadTestREST(t)

	require.NoError(t, pm.Grant("VIEWER", "Extra"))
	require.True(t, pm.HasPermission("ADMIN", REST("Extra", nil)))
	require.Error(t, pm.Grant("MISSING", "Extra"))
}

// bearerSpec serves Read and Write to bearer tokens and Login to cookies only.
func bearerSpec() *openapi3.T {
	spec := testSpec("Read", "Write", "Mint")
	spec.Security = openapi3.SecurityRequirements{{"bearerAuth": []string{}}, {"cookieAuth": []string{}}}

	cookieOnly := openapi3.SecurityRequirements{{"cookieAuth": []string{}}}
	spec.Paths.Set("/Login", &openapi3.PathItem{Post: &openapi3.Operation{OperationID: "Login", Security: &cookieOnly}})

	return spec
}

func loadTestTokens(t *testing.T) (*TokenPermissions, string) {
	t.Helper()

	svc := contracts.File_events_proto.Services().Get(0)

	var procedures []string
	for i := 0; i < svc.Methods().Len(); i++ {
		procedures = append(procedures, `"/`+string(svc.FullName())+"/"+string(svc.Methods().Get(i).Name())+`"`)
	}

	file := func(caps string) []byte {
		return []byte("unavailable:\n  rest: [Mint]\ncapabilities:\n" + caps)
	}

	valid := `  all:
    rest: ["*"]
    grpc: ["*"]
  read-only:
    rest: [Read]
  events:
    inherits: [read-only]
    rest: [Write]
    grpc: [` + strings.Join(procedures, ", ") + `]
`

	pm, err := LoadTokenPermissions(file(valid), bearerSpec(), svc)
	require.NoError(t, err)

	_, err = LoadTokenPermissions(file("  all:\n    rest: [\"*\"]\n    grpc: [\"*\"]\n  read-only: {}\n"), bearerSpec(), svc)
	require.Error(t, err, "every bearer operation and procedure must be listed explicitly somewhere")

	_, err = LoadTokenPermissions(file(strings.Replace(valid, "all:", "everything:", 1)), bearerSpec(), svc)
	require.Error(t, err, "the all carveout must exist")

	_, err = LoadTokenPermissions(file(strings.Replace(valid, "rest: [Read]", "rest: [Read, Mint]", 1)), bearerSpec(), svc)
	require.Error(t, err, "an operation cannot be both granted and unavailable")

	return pm, strings.Trim(procedures[0], `"`)
}

func TestTokenCapabilitiesResolveLegacyTokens(t *testing.T) {
	require.Equal(t, []string{AllCapabilities}, TokenCapabilities(nil, false))
	require.Equal(t, []string{ReadOnlyCapability}, TokenCapabilities(nil, true))
	require.Equal(t, []string{"events"}, TokenCapabilities([]string{"events"}, false))
}

func TestTokenPermissions(t *testing.T) {
	pm, procedure := loadTestTokens(t)

	a, err := New(WithPermissions(pm))
	require.NoError(t, err)

	tenant, other := uuid.New(), uuid.New()
	token := func(readOnly bool, caps ...string) TokenPrincipal {
		return TokenPrincipal{Kind: TenantAPIToken, TenantID: &tenant, ReadOnly: readOnly, Capabilities: caps}
	}

	cases := []struct {
		name string
		req  Request
		want Decision
	}{
		{"rest grant", Request{Principal: token(false, "events"), Action: REST("write", &tenant)}, Allow()},
		{"inherited rest grant", Request{Principal: token(false, "events"), Action: REST("Read", &tenant)}, Allow()},
		{"rest without capability", Request{Principal: token(false, "read-only"), Action: REST("Write", &tenant)}, Deny(MissingCapability)},
		{"all reaches unknown operations", Request{Principal: token(false, "all"), Action: REST("Extension", &tenant)}, Allow()},
		{"unavailable beats all", Request{Principal: token(false, "all"), Action: REST("Mint", &tenant)}, Deny(TokenRestricted)},
		{"another tenant", Request{Principal: token(false, "all"), Action: REST("Read", &other)}, Deny(NotMember)},
		{"no tenant", Request{Principal: token(false, "all"), Action: REST("Read", nil)}, Deny(NotMember)},
		{"grpc grant", Request{Principal: token(false, "events"), Action: GRPC(procedure)}, Allow()},
		{"grpc without capability", Request{Principal: token(false, "events"), Action: GRPC("/v1.Other/Method")}, Deny(MissingCapability)},
		{"any held capability is enough", Request{Principal: token(false, "unknown", "all"), Action: GRPC("/v1.Other/Method")}, Allow()},
		{"read-only stays off grpc", Request{Principal: token(true, "all"), Action: GRPC(procedure)}, Deny(TokenRestricted)},
	}

	for _, c := range cases {
		d, err := a.Authorize(context.Background(), c.req)
		require.NoError(t, err, c.name)
		require.Equal(t, c.want, d, c.name)
	}

	require.True(t, pm.Defines("events"))
	require.False(t, pm.Defines("unknown"))

	require.NoError(t, pm.GrantREST("read-only", "Extension"))
	d, err := a.Authorize(context.Background(), Request{Principal: token(false, "events"), Action: REST("Extension", &tenant)})
	require.NoError(t, err)
	require.True(t, d.Allowed, "grants reach capabilities that inherit them")
}

func TestPermissionsAreChosenPerRequest(t *testing.T) {
	tokens, _ := loadTestTokens(t)
	tenant := uuid.New()

	a, err := New(WithPermissions(tokens, RoleBased(loadTestREST(t), staticRoles{role: "VIEWER", ok: true}, func(r Request) bool {
		_, isUser := r.Principal.(UserPrincipal)
		return isUser
	})))
	require.NoError(t, err)

	d, err := a.Authorize(context.Background(), Request{Principal: UserPrincipal{}, Action: REST("Write", &tenant)})
	require.NoError(t, err)
	require.Equal(t, Deny(RoleInsufficient), d, "users are decided by roles")

	d, err = a.Authorize(context.Background(), Request{Principal: TokenPrincipal{Kind: TenantAPIToken, TenantID: &tenant, Capabilities: []string{"events"}}, Action: REST("Write", &tenant)})
	require.NoError(t, err)
	require.True(t, d.Allowed, "tenant API tokens are decided by capabilities")

	_, err = a.Authorize(context.Background(), Request{Principal: TokenPrincipal{Kind: ManagementToken}, Action: REST("Read", &tenant)})
	require.Error(t, err, "a request nothing handles is an error, not a denial")
}

func TestAuthorizeChecksRole(t *testing.T) {
	a := newTestAuthorizer(t, "VIEWER")

	d, err := a.Authorize(context.Background(), testRequest("Read"))
	require.NoError(t, err)
	require.True(t, d.Allowed)

	d, err = a.Authorize(context.Background(), testRequest("Write"))
	require.NoError(t, err)
	require.Equal(t, Deny(RoleInsufficient), d)
}

func TestAuthorizeWithoutPrincipalErrors(t *testing.T) {
	a := newTestAuthorizer(t, "VIEWER")

	_, err := a.Authorize(context.Background(), Request{Action: REST("Read", nil)})
	require.Error(t, err)
}

type denyAll struct{}

func (denyAll) Evaluate(context.Context, Request) (Decision, error) {
	return Deny(EmailUnverified), nil
}

func TestPolicyDenialWinsOverRole(t *testing.T) {
	a := newTestAuthorizer(t, "ALL", WithPolicies(denyAll{}))

	d, err := a.Authorize(context.Background(), testRequest("Read"))
	require.NoError(t, err)
	require.Equal(t, Deny(EmailUnverified), d)
}

func TestFetchMemoizesWithinRequest(t *testing.T) {
	ctx := WithRequestCache(context.Background())
	key := Key{Kind: "row", ID: "a"}
	loads := 0

	load := func(_ context.Context, missing []Key) (map[Key]string, error) {
		loads++
		return map[Key]string{key: "row"}, nil
	}

	for range 2 {
		rows, err := Fetch(ctx, []Key{key}, load)
		require.NoError(t, err)
		require.Equal(t, "row", rows[key])
	}

	require.Equal(t, 1, loads)

	_, err := Fetch(context.Background(), []Key{key}, load)
	require.NoError(t, err)
	require.Equal(t, 2, loads, "no cache without WithRequestCache")
}

func TestFetchDoesNotCacheMisses(t *testing.T) {
	ctx := WithRequestCache(context.Background())
	key := Key{Kind: "row", ID: "a"}
	loads := 0

	load := func(_ context.Context, missing []Key) (map[Key]string, error) {
		loads++
		return map[Key]string{}, nil
	}

	for range 2 {
		_, err := Fetch(ctx, []Key{key}, load)
		require.NoError(t, err)
	}

	require.Equal(t, 2, loads)
}
