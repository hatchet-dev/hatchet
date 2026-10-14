package encryption

import (
	"crypto/fips140"
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
