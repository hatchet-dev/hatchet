package mcp

import (
	"fmt"
	"sort"
	"strings"

	cliconfig "github.com/hatchet-dev/hatchet/pkg/config/cli"
)

// EmbeddedProfileName is the reserved profile name for the embedded instance.
// Newer embedded engines register themselves in the profile store under this
// name; a stored profile with this name is never treated as a regular profile.
// It is only usable through detection, which verifies the engine is live and
// reports embedded: true, and grants it implicitly. A registration whose
// engine is dead is treated as absent (see detectEmbedded).
const EmbeddedProfileName = "embedded"

// ProfileSource provides read access to the CLI profile store.
type ProfileSource struct {
	// Profiles returns all configured profiles keyed by name.
	Profiles func() map[string]cliconfig.Profile

	// DefaultProfile returns the configured default profile name ("" if unset).
	DefaultProfile func() string
}

// resolvedProfile is the outcome of profile resolution for a tool call.
type resolvedProfile struct {
	// Name of the profile the call executes against.
	Name string

	// Profile holds the connection configuration.
	Profile *cliconfig.Profile

	// Embedded is true when this is the implicitly granted embedded instance.
	Embedded bool
}

// authError builds the deny-by-default error for an ungranted profile. It is
// phrased so the agent can relay the fix to the user, and never reveals
// ungranted profile names other than the one the caller itself supplied.
func authError(requested string, grants *Grants) error {
	return fmt.Errorf(
		"profile %q is not authorized for MCP use; ask the user to run `hatchet mcp auth` to grant access. %s",
		requested,
		grantedSummary(grants),
	)
}

func grantedSummary(grants *Grants) string {
	names := grants.Names()
	if len(names) == 0 {
		return "No profiles are currently granted."
	}

	return fmt.Sprintf("Granted profiles: %s.", strings.Join(names, ", "))
}

// resolveProfile picks the profile for a tool call.
//
// With an explicit name: the name must be granted (or be the detected embedded
// instance, which is implicitly granted). Whether an ungranted name exists is
// not revealed.
//
// Without a name: the CLI default profile (or the sole configured profile) is
// used if granted, otherwise a detected embedded instance, otherwise an error
// listing only granted profiles.
//
// The reserved "embedded" name never resolves through the profile store
// directly: the store's entry (the engine's self-registration) only counts
// when detection has verified it live, in which case it arrives here as the
// detection result.
func resolveProfile(requested string, source ProfileSource, grants *Grants, embedded *EmbeddedDetection) (*resolvedProfile, error) {
	profiles := regularProfiles(source)

	if requested != "" {
		if profile, ok := profiles[strings.ToLower(requested)]; ok && grants.IsGranted(strings.ToLower(requested)) {
			return &resolvedProfile{Name: strings.ToLower(requested), Profile: &profile}, nil
		}

		if requested == EmbeddedProfileName {
			if embedded != nil && embedded.Usable() {
				return &resolvedProfile{Name: EmbeddedProfileName, Profile: embedded.Profile, Embedded: true}, nil
			}
			if embedded != nil && embedded.Detected {
				return nil, fmt.Errorf("an embedded Hatchet instance was detected at %s but no client token is available: %s", embedded.APIURL, embedded.Note)
			}
			if embedded != nil && embedded.Note != "" {
				return nil, fmt.Errorf("no running embedded Hatchet instance was detected (%s); %s", embedded.Note, grantedSummary(grants))
			}
			return nil, fmt.Errorf("no running embedded Hatchet instance was detected; %s", grantedSummary(grants))
		}

		// Same error whether the profile is ungranted or does not exist, so an
		// agent cannot probe for the existence of ungranted profiles.
		return nil, authError(requested, grants)
	}

	if defaultName := effectiveDefault(profiles, source.DefaultProfile()); defaultName != "" && grants.IsGranted(defaultName) {
		profile := profiles[defaultName]
		return &resolvedProfile{Name: defaultName, Profile: &profile}, nil
	}

	if embedded != nil && embedded.Usable() {
		return &resolvedProfile{Name: EmbeddedProfileName, Profile: embedded.Profile, Embedded: true}, nil
	}

	if names := grantedProfileNames(profiles, grants); len(names) > 0 {
		return nil, fmt.Errorf(
			"no default profile is granted for MCP use%s; pass one of the granted profiles explicitly: %s",
			embeddedHint(embedded),
			strings.Join(names, ", "),
		)
	}

	return nil, fmt.Errorf("no profiles are granted for MCP use%s; ask the user to run `hatchet mcp auth` to grant access", embeddedHint(embedded))
}

// embeddedHint qualifies "no embedded instance" errors: a detected-but-
// tokenless instance, or a stale registration that was skipped, is called out
// rather than reported as absent.
func embeddedHint(embedded *EmbeddedDetection) string {
	if embedded != nil && embedded.Detected && !embedded.Usable() {
		return fmt.Sprintf(" (an embedded instance was detected at %s but no client token is available: %s)", embedded.APIURL, embedded.Note)
	}

	if embedded != nil && !embedded.Detected && embedded.Note != "" {
		return fmt.Sprintf(" and no embedded instance was detected (%s)", embedded.Note)
	}

	return " and no embedded instance was detected"
}

// regularProfiles returns the configured profiles without the reserved
// embedded registration entry: that entry is only usable via detection, so a
// dead engine's leftover registration is treated as absent everywhere else
// (never selected as a default or sole profile, never listed as granted).
func regularProfiles(source ProfileSource) map[string]cliconfig.Profile {
	profiles := source.Profiles()
	if _, ok := profiles[EmbeddedProfileName]; !ok {
		return profiles
	}

	filtered := make(map[string]cliconfig.Profile, len(profiles)-1)
	for name, profile := range profiles {
		if name != EmbeddedProfileName {
			filtered[name] = profile
		}
	}

	return filtered
}

// effectiveDefault mirrors CLI profile selection: the configured default if it
// still exists, otherwise the sole configured profile.
func effectiveDefault(profiles map[string]cliconfig.Profile, defaultName string) string {
	if defaultName != "" {
		if _, ok := profiles[defaultName]; ok {
			return defaultName
		}
	}

	if len(profiles) == 1 {
		for name := range profiles {
			return name
		}
	}

	return ""
}

// grantedProfileNames returns the names of configured profiles that are
// granted, sorted. Ungranted names are never included.
func grantedProfileNames(profiles map[string]cliconfig.Profile, grants *Grants) []string {
	names := make([]string, 0, len(profiles))

	for name := range profiles {
		if grants.IsGranted(name) {
			names = append(names, name)
		}
	}

	sort.Strings(names)

	return names
}
