package authz

import (
	"context"
	"fmt"

	"go.yaml.in/yaml/v3"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// AllCapabilities is the capability existing tokens resolve to, so adding capabilities never narrows them retroactively.
const AllCapabilities = "all"

// grpcFile is the gRPC format: token capabilities grant connect procedure names.
//
//	capabilities:
//	  all:
//	    procedures: ["*"]
//	  worker:
//	    inherits: [runs:trigger]
//	    procedures: [/Dispatcher/Listen]
type grpcFile struct {
	Capabilities map[string]struct {
		Inherits   []string `yaml:"inherits"`
		Procedures []string `yaml:"procedures"`
	} `yaml:"capabilities"`
}

type GRPCPermissions struct {
	capabilities grantGraph
}

// LoadGRPCPermissions requires the file to cover exactly the methods of services and to define AllCapabilities.
func LoadGRPCPermissions(yamlBytes []byte, services ...protoreflect.ServiceDescriptor) (*GRPCPermissions, error) {
	var f grpcFile
	if err := yaml.Unmarshal(yamlBytes, &f); err != nil {
		return nil, err
	}

	caps := grantGraph{}
	for name, c := range f.Capabilities {
		caps[name] = &grantDef{inherits: c.Inherits, grants: c.Procedures}
	}

	if err := caps.validate(); err != nil {
		return nil, err
	}

	if !caps.has(AllCapabilities, func(string) bool { return false }) {
		return nil, fmt.Errorf("capability %s must grant %q", AllCapabilities, AllActions)
	}

	procedures := map[string]struct{}{}
	for _, svc := range services {
		methods := svc.Methods()
		for i := 0; i < methods.Len(); i++ {
			procedures["/"+string(svc.FullName())+"/"+string(methods.Get(i).Name())] = struct{}{}
		}
	}

	if err := checkCoverage("procedure", procedures, caps.grantedNames()); err != nil {
		return nil, err
	}

	return &GRPCPermissions{capabilities: caps}, nil
}

func (p *GRPCPermissions) Type() ActionType { return GRPCEndpoint }

// Decide only admits tenant API tokens: gRPC has no users or roles, just the capabilities on the token.
func (p *GRPCPermissions) Decide(_ context.Context, req Request) (Decision, error) {
	t, ok := req.Principal.(TokenPrincipal)
	if !ok || t.Kind != TenantAPIToken {
		return Deny(NotAPIToken), nil
	}

	if req.Scope.TenantID != nil && (t.TenantID == nil || *t.TenantID != *req.Scope.TenantID) {
		return Deny(NotMember), nil
	}

	if !p.Allows(t.Capabilities, req.Action) {
		return Deny(MissingCapability), nil
	}

	return Allow(), nil
}

// Allows reports whether any of the capabilities grants a.
func (p *GRPCPermissions) Allows(capabilities []string, a Action) bool {
	if a.Type() != GRPCEndpoint {
		return false
	}

	for _, c := range capabilities {
		if p.capabilities.has(c, func(grant string) bool { return grant == a.Name() }) {
			return true
		}
	}

	return false
}

// Defines lets token creation reject capability names the file does not know.
func (p *GRPCPermissions) Defines(capability string) bool {
	_, ok := p.capabilities[capability]
	return ok
}
