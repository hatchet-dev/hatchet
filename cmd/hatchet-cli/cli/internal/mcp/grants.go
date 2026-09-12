// Package mcp implements the local `hatchet mcp` stdio server that lets AI
// coding agents verify their work against a running Hatchet deployment, plus
// the grant store that controls which CLI profiles the server may use.
package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// GrantWildcard grants every profile, including profiles created in the future.
const GrantWildcard = "*"

// GrantFileName is the grants file stored next to the CLI profile store.
const GrantFileName = "mcp-grants.yaml"

// Grant is a single grant entry. It is an object rather than a bare string so
// that future per-grant options (e.g. a readonly flag) can be added without a
// format change.
type Grant struct {
	Profile string `yaml:"profile"`
}

// Grants is the parsed contents of the grant file.
type Grants struct {
	Entries []Grant `yaml:"grants"`
}

// IsGranted reports whether the named profile is granted for MCP use, either
// directly or via the wildcard entry.
func (g *Grants) IsGranted(name string) bool {
	if g == nil || name == "" {
		return false
	}

	for _, e := range g.Entries {
		if e.Profile == GrantWildcard || e.Profile == name {
			return true
		}
	}

	return false
}

// HasWildcard reports whether the wildcard entry is present.
func (g *Grants) HasWildcard() bool {
	if g == nil {
		return false
	}

	for _, e := range g.Entries {
		if e.Profile == GrantWildcard {
			return true
		}
	}

	return false
}

// Names returns the granted profile names (including the wildcard entry, if
// present), sorted with the wildcard first.
func (g *Grants) Names() []string {
	if g == nil {
		return nil
	}

	names := make([]string, 0, len(g.Entries))
	for _, e := range g.Entries {
		names = append(names, e.Profile)
	}

	sort.Slice(names, func(i, j int) bool {
		if names[i] == GrantWildcard {
			return true
		}
		if names[j] == GrantWildcard {
			return false
		}
		return names[i] < names[j]
	})

	return names
}

// Add adds a grant for name, reporting whether the set changed.
func (g *Grants) Add(name string) bool {
	for _, e := range g.Entries {
		if e.Profile == name {
			return false
		}
	}

	g.Entries = append(g.Entries, Grant{Profile: name})

	return true
}

// Remove removes the grant for name, reporting whether the set changed.
func (g *Grants) Remove(name string) bool {
	for i, e := range g.Entries {
		if e.Profile == name {
			g.Entries = append(g.Entries[:i], g.Entries[i+1:]...)
			return true
		}
	}

	return false
}

// Clear removes every grant.
func (g *Grants) Clear() {
	g.Entries = nil
}

// GrantStore reads and writes the grant file.
type GrantStore struct {
	path string
}

// NewGrantStore returns a store for the grant file inside dir (the directory
// that also holds the CLI profile store, typically ~/.hatchet).
func NewGrantStore(dir string) *GrantStore {
	return &GrantStore{path: filepath.Join(dir, GrantFileName)}
}

// Path returns the location of the grant file.
func (s *GrantStore) Path() string {
	return s.path
}

// Load reads the grant file. A missing file yields an empty grant set.
func (s *GrantStore) Load() (*Grants, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return &Grants{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read grant file %s: %w", s.path, err)
	}

	grants := &Grants{}
	if err := yaml.Unmarshal(data, grants); err != nil {
		return nil, fmt.Errorf("could not parse grant file %s: %w", s.path, err)
	}

	// Drop malformed entries so a hand-edited file cannot grant "".
	kept := grants.Entries[:0]
	for _, e := range grants.Entries {
		if e.Profile != "" {
			kept = append(kept, e)
		}
	}
	grants.Entries = kept

	return grants, nil
}

// Save writes the grant file with owner-only permissions.
func (s *GrantStore) Save(grants *Grants) error {
	if grants == nil {
		grants = &Grants{}
	}

	data, err := yaml.Marshal(grants)
	if err != nil {
		return fmt.Errorf("could not marshal grants: %w", err)
	}

	header := []byte("# Profiles granted for use by 'hatchet mcp serve'. Managed by 'hatchet mcp auth'.\n")

	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return fmt.Errorf("could not create config directory: %w", err)
	}

	if err := os.WriteFile(s.path, append(header, data...), 0600); err != nil {
		return fmt.Errorf("could not write grant file %s: %w", s.path, err)
	}

	return nil
}
