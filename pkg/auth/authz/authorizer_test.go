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

func (s staticRoles) ResolveRole(context.Context, Principal, Scope) (string, bool, error) {
	return s.role, s.ok, nil
}

type ownedWorkers struct {
	owner uuid.UUID
	calls int
}

func (ownedWorkers) Kind() Kind { return KindWorker }

func (w *ownedWorkers) Check(_ context.Context, _ Request, rs []Resource) (Decisions, error) {
	w.calls++

	out := make(Decisions, len(rs))
	for _, r := range rs {
		if r.(WorkerResource).OperatorID == w.owner {
			out[r.Key()] = Allow()
		} else {
			out[r.Key()] = Deny(NotOwned)
		}
	}

	return out, nil
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

	a, err := New(append([]Option{WithPermissions(RoleBased(pm, staticRoles{role: role, ok: true}))}, opts...)...)
	require.NoError(t, err)

	return a
}

func testRequest(action string) Request {
	return Request{Principal: UserPrincipal{UserID: uuid.New()}, Action: REST(action)}
}

func TestRESTPermissions(t *testing.T) {
	pm := loadTestREST(t)

	require.True(t, pm.HasPermission("ADMIN", REST("Read")))
	require.True(t, pm.HasPermission("ADMIN", REST("write")))
	require.False(t, pm.HasPermission("VIEWER", REST("Write")))
	require.True(t, pm.HasPermission("ALL", REST("AnythingAtAll")))
	require.False(t, pm.HasPermission("UNKNOWN", REST("Read")))
	require.False(t, pm.HasPermission("ALL", Capability("Read")), "a REST file never answers for another action type")
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
	_, err := LoadCapabilityPermissions([]byte(`
roles:
  A:
    inherits: [B]
  B:
    inherits: [A]
`))
	require.Error(t, err)
}

func TestGrantRequiresExistingRole(t *testing.T) {
	pm := loadTestREST(t)

	require.NoError(t, pm.Grant("VIEWER", "Extra"))
	require.True(t, pm.HasPermission("ADMIN", REST("Extra")))
	require.Error(t, pm.Grant("MISSING", "Extra"))
}

func TestCapabilityPermissions(t *testing.T) {
	pm, err := LoadCapabilityPermissions([]byte(`
capabilities: [tenant.payloads.read]
roles:
  ADMIN:
    capabilities: [tenant.payloads.read]
  VIEWER: {}
`))
	require.NoError(t, err)
	require.True(t, pm.HasPermission("ADMIN", Capability("tenant.payloads.read")))
	require.False(t, pm.HasPermission("VIEWER", Capability("tenant.payloads.read")))

	_, err = LoadCapabilityPermissions([]byte(`
capabilities: [tenant.payloads.read]
roles:
  ADMIN:
    capabilities: [tenant.payload.read]
`))
	require.Error(t, err, "undeclared capability")
}

func loadTestGRPC(t *testing.T) (*GRPCPermissions, string) {
	t.Helper()

	svc := contracts.File_events_proto.Services().Get(0)

	var procedures []string
	for i := 0; i < svc.Methods().Len(); i++ {
		procedures = append(procedures, `"/`+string(svc.FullName())+"/"+string(svc.Methods().Get(i).Name())+`"`)
	}

	pm, err := LoadGRPCPermissions([]byte("capabilities:\n  events:\n    procedures: ["+strings.Join(procedures, ", ")+"]\n  all:\n    procedures: [\"*\"]\n"), svc)
	require.NoError(t, err)

	_, err = LoadGRPCPermissions([]byte("capabilities:\n  all:\n    procedures: [\"*\"]\n"), svc)
	require.Error(t, err, "every procedure must be listed explicitly somewhere")

	_, err = LoadGRPCPermissions([]byte("capabilities:\n  events:\n    procedures: ["+strings.Join(procedures, ", ")+"]\n"), svc)
	require.Error(t, err, "the all carveout must exist")

	return pm, strings.Trim(procedures[0], `"`)
}

func TestGRPCPermissions(t *testing.T) {
	pm, first := loadTestGRPC(t)

	require.True(t, pm.Allows([]string{"events"}, GRPC(first)))
	require.False(t, pm.Allows([]string{"events"}, GRPC("/v1.Other/Method")))
	require.True(t, pm.Allows([]string{"unknown", "all"}, GRPC("/v1.Other/Method")), "any held capability is enough")
	require.False(t, pm.Allows(nil, GRPC(first)))
	require.False(t, pm.Allows([]string{"all"}, REST(first)), "a gRPC file never answers for another action type")
	require.True(t, pm.Defines("events"))
	require.False(t, pm.Defines("unknown"))
}

func TestGRPCOnlyAdmitsTenantAPITokens(t *testing.T) {
	pm, first := loadTestGRPC(t)

	a, err := New(WithPermissions(pm))
	require.NoError(t, err)

	tenant, other := uuid.New(), uuid.New()
	token := TokenPrincipal{Kind: TenantAPIToken, TenantID: &tenant, Capabilities: []string{"events"}}

	cases := []struct {
		name string
		req  Request
		want Decision
	}{
		{"token with capability", Request{Principal: token, Action: GRPC(first), Scope: TenantScope(tenant)}, Allow()},
		{"token without capability", Request{Principal: token, Action: GRPC("/v1.Other/Method"), Scope: TenantScope(tenant)}, Deny(MissingCapability)},
		{"token for another tenant", Request{Principal: token, Action: GRPC(first), Scope: TenantScope(other)}, Deny(NotMember)},
		{"user", Request{Principal: UserPrincipal{UserID: uuid.New()}, Action: GRPC(first)}, Deny(NotAPIToken)},
		{"management token", Request{Principal: TokenPrincipal{Kind: ManagementToken, Capabilities: []string{"all"}}, Action: GRPC(first)}, Deny(NotAPIToken)},
	}

	for _, c := range cases {
		d, err := a.Authorize(context.Background(), c.req)
		require.NoError(t, err, c.name)
		require.Equal(t, c.want, d, c.name)
	}
}

func TestAuthorizeWithoutPermissionsForActionTypeErrors(t *testing.T) {
	a := newTestAuthorizer(t, "ALL")

	_, err := a.Authorize(context.Background(), Request{Principal: UserPrincipal{}, Action: GRPC("/v1.Dispatcher/Listen")})
	require.Error(t, err)
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

	_, err := a.Authorize(context.Background(), Request{Action: REST("Read")})
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

func TestAuthorizeResources(t *testing.T) {
	owner := uuid.New()
	workers := &ownedWorkers{owner: owner}
	a := newTestAuthorizer(t, "VIEWER", WithResolver(workers))

	mine := WorkerResource{OperatorID: owner, WorkerID: uuid.New()}
	theirs := WorkerResource{OperatorID: uuid.New(), WorkerID: uuid.New()}

	ds, err := a.AuthorizeResources(context.Background(), testRequest("Read"), []Resource{mine, theirs})
	require.NoError(t, err)
	require.True(t, ds[mine.Key()].Allowed)
	require.Equal(t, Deny(NotOwned), ds[theirs.Key()])
	require.Equal(t, 1, workers.calls, "one batched call per kind")

	key, d := ds.All()
	require.Equal(t, theirs.Key(), key)
	require.False(t, d.Allowed)
}

func TestAuthorizeResourcesDeniedActionSkipsResolvers(t *testing.T) {
	workers := &ownedWorkers{}
	a := newTestAuthorizer(t, "VIEWER", WithResolver(workers))
	r := WorkerResource{WorkerID: uuid.New()}

	ds, err := a.AuthorizeResources(context.Background(), testRequest("Write"), []Resource{r})
	require.NoError(t, err)
	require.Equal(t, Deny(RoleInsufficient), ds[r.Key()])
	require.Zero(t, workers.calls)
}

func TestAuthorizeResourcesRejectsDuplicatesAndUnknownKinds(t *testing.T) {
	a := newTestAuthorizer(t, "VIEWER", WithResolver(&ownedWorkers{}))
	w := WorkerResource{WorkerID: uuid.New()}

	_, err := a.AuthorizeResources(context.Background(), testRequest("Read"), []Resource{w, w})
	require.Error(t, err)

	_, err = a.AuthorizeResource(context.Background(), testRequest("Read"), OperatorResource{OperatorID: uuid.New()})
	require.Error(t, err)
}

func TestFetchMemoizesWithinRequest(t *testing.T) {
	ctx := WithRequestCache(context.Background())
	scope := TenantScope(uuid.New())
	key := Key{Kind: KindTenant, ID: "a"}
	loads := 0

	load := func(_ context.Context, missing []Key) (map[Key]string, error) {
		loads++
		return map[Key]string{key: "row"}, nil
	}

	for range 2 {
		rows, err := Fetch(ctx, scope, []Key{key}, load)
		require.NoError(t, err)
		require.Equal(t, "row", rows[key])
	}

	require.Equal(t, 1, loads)

	_, err := Fetch(context.Background(), scope, []Key{key}, load)
	require.NoError(t, err)
	require.Equal(t, 2, loads, "no cache without WithRequestCache")
}

func TestFetchDoesNotCacheMisses(t *testing.T) {
	ctx := WithRequestCache(context.Background())
	key := Key{Kind: KindTenant, ID: "a"}
	loads := 0

	load := func(_ context.Context, missing []Key) (map[Key]string, error) {
		loads++
		return map[Key]string{}, nil
	}

	for range 2 {
		_, err := Fetch(ctx, NoScope, []Key{key}, load)
		require.NoError(t, err)
	}

	require.Equal(t, 2, loads)
}
