package mcp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	cliconfig "github.com/hatchet-dev/hatchet/pkg/config/cli"
)

// EmbeddedProfileName is the reserved profile name for the embedded instance.
// Embedded engines register themselves in the profile store under this name;
// a stored profile with this name is never treated as a regular profile. It
// is only usable through detection, which verifies the engine is live and
// reports embedded: true, and grants it implicitly. A registration whose
// engine is dead is treated as absent (see detectEmbedded).
const EmbeddedProfileName = "embedded"

// embeddedProbeTimeout bounds the live check of the embedded registration.
const embeddedProbeTimeout = 3 * time.Second

// staleEmbeddedNote is the constant message carried for a registration that
// failed live verification. It deliberately contains none of the stored
// registration's contents: the URL of a rejected registration can carry
// internal hosts, paths, or credentials and must not reach the agent.
const staleEmbeddedNote = "a stale embedded registration was ignored; start the embedded engine again or remove the stale profile entry"

// ProfileSource provides read access to the CLI profile store.
type ProfileSource struct {
	// Profiles returns all configured profiles keyed by name.
	Profiles func() map[string]cliconfig.Profile

	// DefaultProfile returns the configured default profile name ("" if unset).
	DefaultProfile func() string
}

// EmbeddedDetection is the outcome of looking for a running embedded instance
// via its profile-store registration.
type EmbeddedDetection struct {
	// Profile is the live-verified connection profile of the embedded
	// instance, nil when none was found.
	Profile *cliconfig.Profile

	// Note explains a stale registration that was skipped.
	Note string
}

// Detected reports whether a live embedded instance was found.
func (d *EmbeddedDetection) Detected() bool {
	return d != nil && d.Profile != nil
}

// detectEmbedded looks for a running embedded Hatchet instance.
//
// Embedded engines register themselves in the profile store under the
// reserved "embedded" name on ready and remove the entry on graceful
// shutdown. The registration only counts when both of its stored destinations
// (API URL and gRPC address) are loopback and the engine behind the API URL is
// live and its /api/v1/meta reports embedded: true, so a hand-made "embedded"
// profile pointing at a non-embedded or non-local deployment can never sneak
// past the grant checks.
//
// A registration failing any of these checks (e.g. left behind by a crash) is
// skipped with a constant note, never auto-deleted: the note is carried so
// engine_status and resolution errors can surface it, but the entry is only
// ever removed by the engine that wrote it (or overwritten by the next one).
func detectEmbedded(ctx context.Context, source ProfileSource) *EmbeddedDetection {
	if source.Profiles == nil {
		return &EmbeddedDetection{}
	}

	registered, ok := source.Profiles()[EmbeddedProfileName]
	if !ok {
		return &EmbeddedDetection{}
	}

	if registered.Token != "" &&
		isLoopbackURL(registered.ApiServerURL) &&
		isLoopbackHostPort(registered.GrpcHostPort) &&
		isEmbeddedAPI(ctx, registered.ApiServerURL) {
		profile := registered
		profile.Name = EmbeddedProfileName

		return &EmbeddedDetection{Profile: &profile}
	}

	return &EmbeddedDetection{Note: staleEmbeddedNote}
}

// isLoopbackURL reports whether rawURL is an http(s) URL whose host is a
// loopback destination. An embedded engine always registers a loopback API
// URL, so anything else is not an embedded registration.
func isLoopbackURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}

	return isLoopbackHost(parsed.Hostname())
}

// isLoopbackHostPort reports whether hostPort is a host:port pair whose host
// is a loopback destination.
func isLoopbackHostPort(hostPort string) bool {
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return false
	}

	return isLoopbackHost(host)
}

// isLoopbackHost reports whether host names the local machine: "localhost" or
// a literal loopback IP. Other hostnames are rejected without resolving them,
// so DNS can never turn a non-local registration into an implicit grant.
func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip := net.ParseIP(host)

	return ip != nil && ip.IsLoopback()
}

// isEmbeddedAPI reports whether apiURL is a live Hatchet API server running in
// embedded mode, via the unauthenticated /api/v1/meta endpoint. The probe uses
// a dedicated HTTP client with redirects and proxies disabled, so the answer
// always comes directly from the already-validated loopback destination.
func isEmbeddedAPI(ctx context.Context, apiURL string) bool {
	probeCtx, cancel := context.WithTimeout(ctx, embeddedProbeTimeout)
	defer cancel()

	probeClient := &http.Client{
		Timeout: embeddedProbeTimeout,
		Transport: &http.Transport{
			// Never route the loopback probe through a proxy.
			Proxy: nil,
		},
		// Do not follow redirects: a redirect answer fails the 200 check
		// below, so only a direct response from the probed destination counts.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	client, err := rest.NewClientWithResponses(apiURL, rest.WithHTTPClient(probeClient))
	if err != nil {
		return false
	}

	resp, err := client.MetadataGetWithResponse(probeCtx)
	if err != nil || resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return false
	}

	return resp.JSON200.Embedded != nil && *resp.JSON200.Embedded
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
			if embedded.Detected() {
				return &resolvedProfile{Name: EmbeddedProfileName, Profile: embedded.Profile, Embedded: true}, nil
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

	if embedded.Detected() {
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

// embeddedHint qualifies "no embedded instance" errors: a stale registration
// that was skipped is called out rather than reported as absent.
func embeddedHint(embedded *EmbeddedDetection) string {
	if embedded != nil && embedded.Note != "" {
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
