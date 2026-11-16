package encryption

import (
	"os"
	"strings"
	"testing"
)

func TestSHA1AllowedMatchesFIPSOnlyMode(t *testing.T) {
	only := strings.Contains(os.Getenv("GODEBUG"), "fips140=only")

	if SHA1Allowed() == only {
		t.Fatalf("SHA1Allowed() = %v with GODEBUG=%q", !only, os.Getenv("GODEBUG"))
	}
}
