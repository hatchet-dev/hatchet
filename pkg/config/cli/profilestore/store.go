// Package profilestore is the shared implementation of the Hatchet CLI's
// profile store (~/.hatchet/profiles.yaml by default).
//
// The hatchet CLI uses it for every profile operation, and external processes
// (for example an embedded engine registering itself as a discoverable
// profile) use the same package so that all writers agree on the file
// location, the lock protocol, and the on-disk format. Writes preserve
// comments, key ordering, and fields the store does not know about, so a
// profile entry may carry extra metadata that the CLI ignores.
package profilestore

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"

	"github.com/hatchet-dev/hatchet/pkg/config/cli"
	"github.com/hatchet-dev/hatchet/pkg/config/loader/loaderutils"
)

// DefaultProfileFileName is the profile file name used when none is
// configured via HATCHET_CLI_PROFILE_FILE_NAME or the profileFileName key in
// ~/.hatchet/config.yaml.
const DefaultProfileFileName = "profiles.yaml"

// hatchetDirName is the per-user configuration directory under $HOME.
const hatchetDirName = ".hatchet"

// cliConfigFileName is the CLI config file (read by NewDefaultStore for the
// profileFileName override).
const cliConfigFileName = "config.yaml"

// Store reads and writes the profile file in a fixed directory. Reads go
// through viper (lenient and case-insensitive, so unknown fields are ignored
// and key casing does not matter); writes take the shared config.lock,
// surgically edit the yaml document, and replace the file atomically with
// 0600 permissions.
//
// A Store is safe for concurrent use within a process, and the file lock
// serializes writers across processes.
type Store struct {
	dir      string
	fileName string

	mu sync.RWMutex // protects v
	v  *viper.Viper
}

// NewStore opens the profile store in dir (the ~/.hatchet directory), using
// fileName as the profile file name ("" means profiles.yaml). A missing
// directory or file is fine: reads see an empty store and the first write
// creates both. An unreadable or malformed file is an error.
func NewStore(dir, fileName string) (*Store, error) {
	if dir == "" {
		return nil, fmt.Errorf("profile store directory is required")
	}

	if fileName == "" {
		fileName = DefaultProfileFileName
	}

	s := &Store{dir: dir, fileName: fileName}

	if err := s.load(); err != nil {
		return nil, err
	}

	return s, nil
}

// NewDefaultStore opens the profile store at the location the CLI uses:
// ~/.hatchet, with the profile file name taken from the
// HATCHET_CLI_PROFILE_FILE_NAME environment variable or the profileFileName
// key in ~/.hatchet/config.yaml, defaulting to profiles.yaml.
func NewDefaultStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not determine the home directory: %w", err)
	}

	dir := filepath.Join(home, hatchetDirName)

	var files [][]byte

	if data, err := os.ReadFile(filepath.Join(dir, cliConfigFileName)); err == nil {
		files = append(files, data)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("could not read cli config file: %w", err)
	}

	conf := &cli.CLIConfig{}
	if _, err := loaderutils.LoadConfigFromViper(cli.BindAllEnv, conf, files...); err != nil {
		return nil, fmt.Errorf("could not load cli config file: %w", err)
	}

	return NewStore(dir, conf.ProfileFileName)
}

// Dir returns the directory containing the profile file (and its lock).
func (s *Store) Dir() string {
	return s.dir
}

// Path returns the path of the profile file.
func (s *Store) Path() string {
	return filepath.Join(s.dir, s.fileName)
}

// load parses the profile file into a fresh viper instance. The file is also
// unmarshalled into cli.ProfileFile so malformed values (for example a quoted
// expiresAt that cannot decode into time.Time) surface as an error here
// instead of corrupting later operations.
func (s *Store) load() error {
	var files [][]byte

	if data, err := os.ReadFile(s.Path()); err == nil {
		files = append(files, data)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("could not read profiles file: %w", err)
	}

	v, err := loaderutils.LoadConfigFromViper(func(v *viper.Viper) {}, &cli.ProfileFile{}, files...)
	if err != nil {
		return err
	}

	v.SetConfigFile(s.Path())
	v.SetConfigType("yaml")

	s.v = v

	return nil
}

// GetProfiles returns all profiles from the config. Profile names are
// reported lowercased (viper normalizes map keys).
func (s *Store) GetProfiles() map[string]cli.Profile {
	profiles := make(map[string]cli.Profile)

	s.mu.RLock()
	defer s.mu.RUnlock()

	profilesMap := s.v.GetStringMap("profiles")
	for name := range profilesMap {
		tlsStrategy := s.v.GetString(fmt.Sprintf("profiles.%s.tlsStrategy", name))
		if tlsStrategy == "" {
			tlsStrategy = "tls"
		}
		profile := cli.Profile{
			TenantId:     s.v.GetString(fmt.Sprintf("profiles.%s.tenantId", name)),
			Name:         s.v.GetString(fmt.Sprintf("profiles.%s.name", name)),
			Token:        s.v.GetString(fmt.Sprintf("profiles.%s.token", name)),
			ExpiresAt:    s.v.GetTime(fmt.Sprintf("profiles.%s.expiresAt", name)),
			ApiServerURL: s.v.GetString(fmt.Sprintf("profiles.%s.apiServerURL", name)),
			GrpcHostPort: s.v.GetString(fmt.Sprintf("profiles.%s.grpcHostPort", name)),
			TLSStrategy:  tlsStrategy,
		}
		profiles[name] = profile
	}

	return profiles
}

// GetProfile returns a specific profile by name (case-insensitive).
func (s *Store) GetProfile(name string) (*cli.Profile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("profiles.%s", name)
	if !s.v.IsSet(key) {
		return nil, fmt.Errorf("profile '%s' not found", name)
	}

	tlsStrategy := s.v.GetString(fmt.Sprintf("%s.tlsStrategy", key))
	if tlsStrategy == "" {
		tlsStrategy = "tls"
	}

	profile := &cli.Profile{
		TenantId:     s.v.GetString(fmt.Sprintf("%s.tenantId", key)),
		Name:         s.v.GetString(fmt.Sprintf("%s.name", key)),
		Token:        s.v.GetString(fmt.Sprintf("%s.token", key)),
		ExpiresAt:    s.v.GetTime(fmt.Sprintf("%s.expiresAt", key)),
		ApiServerURL: s.v.GetString(fmt.Sprintf("%s.apiServerURL", key)),
		GrpcHostPort: s.v.GetString(fmt.Sprintf("%s.grpcHostPort", key)),
		TLSStrategy:  tlsStrategy,
	}

	return profile, nil
}

// ListProfiles returns a sorted list of all profile names.
func (s *Store) ListProfiles() []string {
	profiles := s.GetProfiles()
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetDefaultProfile returns the name of the default profile, or empty string
// if none is set.
func (s *Store) GetDefaultProfile() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.v.GetString("defaultProfile")
}

// AddProfile adds a new profile to the config (or fully overwrites the named
// profile's known fields). All profile fields are required; TLSStrategy
// defaults to "tls" when empty (the default is written back to profile).
// Unknown fields already stored under the same profile entry are preserved.
func (s *Store) AddProfile(name string, profile *cli.Profile) error {
	return s.UpsertProfile(name, profile, nil)
}

// UpsertProfile is AddProfile plus extraFields: additional entries written
// under the same profile mapping (keys are lowercased). The CLI ignores
// fields it does not know, so extraFields can carry caller metadata; values
// go through the yaml encoder, meaning time.Time values become unquoted yaml
// timestamps just like expiresAt.
func (s *Store) UpsertProfile(name string, profile *cli.Profile, extraFields map[string]any) error {
	// Validate required fields
	if profile.TenantId == "" {
		return fmt.Errorf("tenantId is required")
	}
	if profile.Name == "" {
		return fmt.Errorf("profile name is required")
	}
	if profile.Token == "" {
		return fmt.Errorf("token is required")
	}
	if profile.ExpiresAt.IsZero() {
		return fmt.Errorf("expiresAt is required")
	}
	if profile.ApiServerURL == "" {
		return fmt.Errorf("apiServerURL is required")
	}
	if profile.GrpcHostPort == "" {
		return fmt.Errorf("grpcHostPort is required")
	}

	// Set default TLS strategy if not provided
	if profile.TLSStrategy == "" {
		profile.TLSStrategy = "tls"
	}

	// Validate TLS strategy
	if profile.TLSStrategy != "tls" && profile.TLSStrategy != "none" {
		return fmt.Errorf("tlsStrategy must be either 'tls' or 'none'")
	}

	return s.mutate(func(d *document) (bool, error) {
		profiles, err := mappingValueOrCreate(d.root, "profiles")
		if err != nil {
			return false, err
		}

		entry, err := mappingValueOrCreate(profiles, strings.ToLower(name))
		if err != nil {
			return false, err
		}

		fields := []struct {
			key   string
			value any
		}{
			{"tenantid", profile.TenantId},
			{"name", profile.Name},
			{"token", profile.Token},
			{"expiresat", profile.ExpiresAt},
			{"apiserverurl", profile.ApiServerURL},
			{"grpchostport", profile.GrpcHostPort},
			{"tlsstrategy", profile.TLSStrategy},
		}

		for _, f := range fields {
			node, err := yamlNodeFor(f.value)
			if err != nil {
				return false, err
			}
			setMappingValue(entry, f.key, node)
		}

		extraKeys := make([]string, 0, len(extraFields))
		for key := range extraFields {
			extraKeys = append(extraKeys, key)
		}
		sort.Strings(extraKeys)

		for _, key := range extraKeys {
			node, err := yamlNodeFor(extraFields[key])
			if err != nil {
				return false, err
			}
			setMappingValue(entry, strings.ToLower(key), node)
		}

		// Force block style on the mappings we touched: an empty
		// "profiles: {}" left behind by removing the last profile parses as
		// flow style, and flow style would make the encoder quote the
		// timestamp values, which the CLI cannot decode into time.Time.
		d.root.Style = 0
		profiles.Style = 0
		entry.Style = 0

		return true, nil
	})
}

// UpdateProfile updates an existing profile with new values. Only non-empty
// fields are updated.
func (s *Store) UpdateProfile(name string, profile *cli.Profile) error {
	if profile.TLSStrategy != "" && profile.TLSStrategy != "tls" && profile.TLSStrategy != "none" {
		return fmt.Errorf("tlsStrategy must be either 'tls' or 'none'")
	}

	return s.mutate(func(d *document) (bool, error) {
		profiles := mappingValue(d.root, "profiles")

		entry := mappingValue(profiles, name)
		if entry == nil || entry.Kind != yaml.MappingNode {
			return false, fmt.Errorf("profile '%s' not found", name)
		}

		fields := []struct {
			key   string
			value any
			set   bool
		}{
			{"tenantid", profile.TenantId, profile.TenantId != ""},
			{"name", profile.Name, profile.Name != ""},
			{"token", profile.Token, profile.Token != ""},
			{"expiresat", profile.ExpiresAt, !profile.ExpiresAt.IsZero()},
			{"apiserverurl", profile.ApiServerURL, profile.ApiServerURL != ""},
			{"grpchostport", profile.GrpcHostPort, profile.GrpcHostPort != ""},
			{"tlsstrategy", profile.TLSStrategy, profile.TLSStrategy != ""},
		}

		for _, f := range fields {
			if !f.set {
				continue
			}
			node, err := yamlNodeFor(f.value)
			if err != nil {
				return false, err
			}
			setMappingValue(entry, f.key, node)
		}

		d.root.Style = 0
		profiles.Style = 0
		entry.Style = 0

		return true, nil
	})
}

// RemoveProfile removes a profile from the config. If the removed profile was
// the default profile, the default is cleared.
func (s *Store) RemoveProfile(name string) error {
	return s.mutate(func(d *document) (bool, error) {
		profiles := mappingValue(d.root, "profiles")

		if mappingValue(profiles, name) == nil {
			return false, fmt.Errorf("profile '%s' not found", name)
		}

		deleteMappingKey(profiles, name)

		if def := mappingValue(d.root, "defaultProfile"); def != nil && def.Kind == yaml.ScalarNode && def.Value == name {
			deleteMappingKey(d.root, "defaultProfile")
		}

		return true, nil
	})
}

// RemoveProfileIfTokenMatches removes the named profile only when its stored
// token equals token, reporting whether an entry was removed. A missing file,
// profile, or non-matching token is not an error, and the default-profile
// setting is never touched: this is the primitive for a registration that
// must not delete a newer instance's entry (last writer wins).
func (s *Store) RemoveProfileIfTokenMatches(name, token string) (bool, error) {
	if _, err := os.Stat(s.Path()); os.IsNotExist(err) {
		return false, nil
	}

	removed := false

	err := s.mutate(func(d *document) (bool, error) {
		profiles := mappingValue(d.root, "profiles")
		if profiles == nil || profiles.Kind != yaml.MappingNode {
			return false, nil
		}

		entry := mappingValue(profiles, name)
		if entry == nil || entry.Kind != yaml.MappingNode {
			return false, nil
		}

		entryToken := mappingValue(entry, "token")
		if entryToken == nil || entryToken.Value != token {
			return false, nil
		}

		deleteMappingKey(profiles, name)
		removed = true

		return true, nil
	})

	return removed, err
}

// SetDefaultProfile sets the default profile. The profile must exist.
func (s *Store) SetDefaultProfile(name string) error {
	return s.mutate(func(d *document) (bool, error) {
		profiles := mappingValue(d.root, "profiles")

		if mappingValue(profiles, name) == nil {
			return false, fmt.Errorf("profile '%s' not found", name)
		}

		node, err := yamlNodeFor(name)
		if err != nil {
			return false, err
		}

		setMappingValue(d.root, "defaultprofile", node)
		d.root.Style = 0

		return true, nil
	})
}

// SetDefaultProfileIfUnset makes name the default profile when none is
// configured and reports whether it did. The check and the write happen under
// the same file lock against the freshly parsed file, so concurrent callers
// cannot both observe an unset default and both report success.
func (s *Store) SetDefaultProfileIfUnset(name string) (bool, error) {
	set := false

	err := s.mutate(func(d *document) (bool, error) {
		if v := mappingValue(d.root, "defaultProfile"); v != nil && v.Value != "" {
			return false, nil
		}

		profiles := mappingValue(d.root, "profiles")

		if mappingValue(profiles, name) == nil {
			return false, fmt.Errorf("profile '%s' not found", name)
		}

		node, err := yamlNodeFor(name)
		if err != nil {
			return false, err
		}

		setMappingValue(d.root, "defaultprofile", node)
		d.root.Style = 0
		set = true

		return true, nil
	})

	return set, err
}

// ClearDefaultProfile clears the default profile setting. Clearing an unset
// default is a no-op.
func (s *Store) ClearDefaultProfile() error {
	return s.mutate(func(d *document) (bool, error) {
		if mappingValue(d.root, "defaultProfile") == nil {
			return false, nil
		}

		deleteMappingKey(d.root, "defaultProfile")

		return true, nil
	})
}

// mutate runs fn on the parsed profile file under the cross-process file lock
// and writes the result back atomically when fn reports a change, then
// reloads the in-memory view. The profile directory is created if absent.
func (s *Store) mutate(fn func(d *document) (bool, error)) error {
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return fmt.Errorf("could not create %s: %w", s.dir, err)
	}

	unlock, err := acquireLock(s.dir)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	d, err := loadDocument(s.Path())
	if err != nil {
		return err
	}

	changed, err := fn(d)
	if err != nil {
		return err
	}

	if !changed {
		return nil
	}

	if err := d.write(s.Path()); err != nil {
		return err
	}

	return s.load()
}
