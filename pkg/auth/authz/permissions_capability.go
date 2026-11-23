package authz

import (
	"fmt"

	"go.yaml.in/yaml/v3"
)

// capabilityFile is the capability format: capabilities have no external spec, so the file declares them.
//
//	capabilities:
//	  - tenant.payloads.read
//	roles:
//	  ADMIN:
//	    capabilities: [tenant.payloads.read]
type capabilityFile struct {
	Capabilities []string `yaml:"capabilities"`
	Roles        map[string]struct {
		Inherits     []string `yaml:"inherits"`
		Capabilities []string `yaml:"capabilities"`
	} `yaml:"roles"`
}

type CapabilityPermissions struct {
	roles grantGraph
}

// LoadCapabilityPermissions rejects grants of capabilities the file does not declare, which catches typos.
func LoadCapabilityPermissions(yamlBytes []byte) (*CapabilityPermissions, error) {
	var f capabilityFile
	if err := yaml.Unmarshal(yamlBytes, &f); err != nil {
		return nil, err
	}

	declared := make(map[string]struct{}, len(f.Capabilities))
	for _, c := range f.Capabilities {
		if c == AllActions {
			return nil, fmt.Errorf("capability %q cannot be declared", AllActions)
		}

		declared[c] = struct{}{}
	}

	roles := grantGraph{}
	for name, r := range f.Roles {
		roles[name] = &grantDef{inherits: r.Inherits, grants: r.Capabilities}
	}

	if err := roles.validate(); err != nil {
		return nil, err
	}

	for c := range roles.grantedNames() {
		if _, ok := declared[c]; !ok {
			return nil, fmt.Errorf("capability %s is granted but not declared", c)
		}
	}

	return &CapabilityPermissions{roles: roles}, nil
}

func (p *CapabilityPermissions) Type() ActionType { return CapabilityCheck }

func (p *CapabilityPermissions) HasPermission(role string, a Action) bool {
	if a.Type() != CapabilityCheck {
		return false
	}

	return p.roles.has(role, func(grant string) bool { return grant == a.Name() })
}
