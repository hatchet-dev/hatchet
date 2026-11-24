package authz

import (
	"context"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"go.yaml.in/yaml/v3"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	// AllCapabilities is what tokens stored without capabilities resolve to, so adding capabilities never narrows them.
	AllCapabilities = "all"

	// ReadOnlyCapability is what read-only tokens stored without capabilities resolve to.
	ReadOnlyCapability = "read-only"

	bearerScheme = "bearerAuth"
)

// TokenCapabilities resolves what a stored token holds; nil means it predates capabilities.
func TokenCapabilities(stored []string, readOnly bool) []string {
	switch {
	case stored != nil:
		return stored
	case readOnly:
		return []string{ReadOnlyCapability}
	default:
		return []string{AllCapabilities}
	}
}

// tokenFile maps token capabilities to the REST operations and gRPC procedures they grant.
//
//	unavailable:
//	  rest: [ApiTokenCreate]
//	capabilities:
//	  all:
//	    rest: ["*"]
//	    grpc: ["*"]
//	  runs:trigger:
//	    rest: [V1WorkflowRunCreate]
//	    grpc: [/v1.AdminService/TriggerWorkflowRun]
type tokenFile struct {
	Unavailable struct {
		REST []string `yaml:"rest"`
	} `yaml:"unavailable"`
	Capabilities map[string]struct {
		Inherits []string `yaml:"inherits"`
		REST     []string `yaml:"rest"`
		GRPC     []string `yaml:"grpc"`
	} `yaml:"capabilities"`
}

// TokenPermissions decides every call a tenant API token makes, over REST and gRPC, from its capabilities.
type TokenPermissions struct {
	rest        grantGraph
	grpc        grantGraph
	unavailable map[string]struct{} // lower-cased REST operation ids
}

func LoadTokenPermissions(yamlBytes []byte, spec *openapi3.T, services ...protoreflect.ServiceDescriptor) (*TokenPermissions, error) {
	if spec == nil {
		return nil, fmt.Errorf("authz: an OpenAPI spec is required to load token permissions")
	}

	var f tokenFile
	if err := yaml.Unmarshal(yamlBytes, &f); err != nil {
		return nil, err
	}

	p := &TokenPermissions{rest: grantGraph{}, grpc: grantGraph{}, unavailable: map[string]struct{}{}}

	for name, c := range f.Capabilities {
		p.rest[name] = &grantDef{inherits: c.Inherits, grants: c.REST}
		p.grpc[name] = &grantDef{inherits: c.Inherits, grants: c.GRPC}
	}

	if err := p.rest.validate(); err != nil {
		return nil, err
	}

	for _, required := range []string{AllCapabilities, ReadOnlyCapability} {
		if !p.rest.defines(required) {
			return nil, fmt.Errorf("capability %s must be defined", required)
		}
	}

	never := func(string) bool { return false }
	if !p.rest.has(AllCapabilities, never) || !p.grpc.has(AllCapabilities, never) {
		return nil, fmt.Errorf("capability %s must grant %q for rest and grpc", AllCapabilities, AllActions)
	}

	granted := p.rest.grantedNames()
	covered := map[string]struct{}{}

	for name := range granted {
		covered[name] = struct{}{}
	}

	for _, name := range f.Unavailable.REST {
		if _, ok := granted[name]; ok {
			return nil, fmt.Errorf("operation %s is both granted and unavailable", name)
		}

		covered[name] = struct{}{}
		p.unavailable[strings.ToLower(name)] = struct{}{}
	}

	scheme := bearerScheme
	if err := checkCoverage("operation", specOperations(spec, &scheme), covered); err != nil {
		return nil, err
	}

	procedures := map[string]struct{}{}
	for _, svc := range services {
		methods := svc.Methods()
		for i := 0; i < methods.Len(); i++ {
			procedures["/"+string(svc.FullName())+"/"+string(methods.Get(i).Name())] = struct{}{}
		}
	}

	if err := checkCoverage("procedure", procedures, p.grpc.grantedNames()); err != nil {
		return nil, err
	}

	return p, nil
}

func (p *TokenPermissions) Handles(req Request) bool {
	t, ok := req.Principal.(TokenPrincipal)
	return ok && t.Kind == TenantAPIToken
}

func (p *TokenPermissions) Decide(_ context.Context, req Request) (Decision, error) {
	t, ok := req.Principal.(TokenPrincipal)
	if !ok {
		return Decision{}, fmt.Errorf("authz: token permissions cannot decide for %T", req.Principal)
	}

	switch a := req.Action.(type) {
	case RESTAction:
		if _, ok := p.unavailable[strings.ToLower(a.OperationID)]; ok {
			return Deny(TokenRestricted), nil
		}

		if a.TenantID == nil || t.TenantID == nil || *a.TenantID != *t.TenantID {
			return Deny(NotMember), nil
		}
	case GRPCAction:
		// Read-only tokens predate capabilities and were never allowed on gRPC, even for reads.
		if t.ReadOnly {
			return Deny(TokenRestricted), nil
		}
	default:
		return Decision{}, fmt.Errorf("authz: token permissions cannot decide %s actions", req.Action.Type())
	}

	if !p.allows(t.Capabilities, req.Action) {
		return Deny(MissingCapability), nil
	}

	return Allow(), nil
}

func (p *TokenPermissions) allows(capabilities []string, a Action) bool {
	graph, matches := p.grpc, func(grant string) bool { return grant == a.Name() }
	if a.Type() == RESTEndpoint {
		graph, matches = p.rest, func(grant string) bool { return strings.EqualFold(grant, a.Name()) }
	}

	for _, c := range capabilities {
		if graph.has(c, matches) {
			return true
		}
	}

	return false
}

func (p *TokenPermissions) Defines(capability string) bool {
	return p.rest.defines(capability)
}

func (p *TokenPermissions) GrantREST(capability string, operationIDs ...string) error {
	return p.rest.grant(capability, operationIDs...)
}
