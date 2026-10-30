package encryption

import (
	"crypto/fips140"
	"crypto/sha1" //nolint:gosec // probed only to detect fips140=only, never used for security
	"fmt"
	"runtime/debug"
)

// RequireFIPSBuild fails when a binary built with GOFIPS140 is running with FIPS mode disabled,
// which only happens if GODEBUG=fips140=off overrides the build default at runtime.
func RequireFIPSBuild() error {
	bi, ok := debug.ReadBuildInfo()

	if !ok {
		return nil
	}

	for _, s := range bi.Settings {
		if s.Key == "GOFIPS140" && s.Value != "off" && !fips140.Enabled() {
			return fmt.Errorf("this binary was built with GOFIPS140=%s but FIPS mode is disabled at runtime; remove fips140=off from GODEBUG or run a non-FIPS image", s.Value)
		}
	}

	return nil
}

// SHA1Allowed reports whether this process may compute SHA-1 digests. GODEBUG=fips140=only
// makes crypto/sha1 panic, which rules out protocols whose handshake is specified over
// SHA-1, such as the RFC 6455 websocket upgrade.
func SHA1Allowed() (allowed bool) {
	defer func() {
		if recover() != nil {
			allowed = false
		}
	}()

	sha1.Sum(nil) //nolint:gosec // see the import note

	return true
}
