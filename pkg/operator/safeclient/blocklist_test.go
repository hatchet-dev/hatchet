//go:build !e2e && !load && !rampup && !integration

package safeclient

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultBlocklistParses(t *testing.T) {
	bl, err := newBlocklist(nil)
	require.NoError(t, err)
	assert.NotEmpty(t, bl.nets)
}

// TestIsBlockedIP_DefaultRanges asserts a representative address inside every default
// blocked range is rejected. safeurl applies exactly this decision to the resolved IP at
// dial time on every attempt, so a hostname that resolves to one of these (incl. a DNS
// rebind on a later attempt) is blocked too.
func TestIsBlockedIP_DefaultRanges(t *testing.T) {
	bl, err := newBlocklist(nil)
	require.NoError(t, err)

	blocked := []string{
		"0.0.0.0",          // this network
		"10.0.0.5",         // RFC 1918
		"100.64.0.1",       // CGNAT
		"127.0.0.1",        // loopback
		"169.254.169.254",  // cloud metadata (link-local)
		"172.16.0.1",       // RFC 1918
		"192.0.0.1",        // IETF protocol assignments
		"192.0.2.1",        // TEST-NET-1
		"192.88.99.1",      // 6to4 relay anycast
		"192.168.1.1",      // RFC 1918
		"198.18.0.1",       // benchmarking
		"198.51.100.1",     // TEST-NET-2
		"203.0.113.1",      // TEST-NET-3
		"224.0.0.1",        // multicast
		"240.0.0.1",        // reserved
		"255.255.255.255",  // broadcast
		"::",               // unspecified
		"::1",              // loopback
		"::ffff:127.0.0.1", // IPv4-mapped IPv6
		"64:ff9b::1",       // NAT64
		"fc00::1",          // unique local
		"fe80::1",          // link-local
		"ff00::1",          // multicast
	}

	for _, s := range blocked {
		ip := net.ParseIP(s)
		require.NotNil(t, ip, "could not parse %q", s)
		assert.True(t, bl.isBlockedIP(ip), "expected %s to be blocked", s)
	}
}

func TestIsBlockedIP_PublicAllowed(t *testing.T) {
	bl, err := newBlocklist(nil)
	require.NoError(t, err)

	allowed := []string{
		"8.8.8.8",
		"1.1.1.1",
		"93.184.216.34",  // a routable public IPv4
		"::ffff:8.8.8.8", // IPv4-mapped form of a PUBLIC IP must stay allowed
		"2606:4700:4700::1111",
	}

	for _, s := range allowed {
		ip := net.ParseIP(s)
		require.NotNil(t, ip, "could not parse %q", s)
		assert.False(t, bl.isBlockedIP(ip), "expected %s to be allowed", s)
	}
}

// TestIsBlockedIP_MappedLoopback guards the subtle IPv4-mapped case: ::ffff:127.0.0.1 must
// be blocked (via the IPv4 loopback range after normalization) even though ::ffff:0:0/96
// is intentionally absent from the denylist.
func TestIsBlockedIP_MappedLoopback(t *testing.T) {
	bl, err := newBlocklist(nil)
	require.NoError(t, err)
	assert.True(t, bl.isBlockedIP(net.ParseIP("::ffff:127.0.0.1")))
}

func TestIsBlockedIP_NilFailsClosed(t *testing.T) {
	bl, err := newBlocklist(nil)
	require.NoError(t, err)
	assert.True(t, bl.isBlockedIP(nil))
}

// TestInfraBlockedCIDRsMechanism uses a synthetic documentation range ONLY (RFC 3849
// 2001:db8::/32, which is NOT in DefaultBlockedCIDRs) to prove the InfraBlockedCIDRs wiring
// adds ranges to the denylist. Real infrastructure ranges must never appear in this repo.
func TestInfraBlockedCIDRsMechanism(t *testing.T) {
	addr := net.ParseIP("2001:db8::dead")

	// Without the infra range, this synthetic address is allowed by the default list.
	def, err := newBlocklist(nil)
	require.NoError(t, err)
	require.False(t, def.isBlockedIP(addr), "2001:db8::/32 must not be in the default list")

	// Adding it via InfraBlockedCIDRs blocks it.
	withInfra, err := newBlocklist([]string{"2001:db8::/32"})
	require.NoError(t, err)
	assert.True(t, withInfra.isBlockedIP(addr))
	// A public IP outside the infra range is still allowed.
	assert.False(t, withInfra.isBlockedIP(net.ParseIP("8.8.8.8")))
}

func TestNewBlocklist_BadCIDR(t *testing.T) {
	_, err := newBlocklist([]string{"not-a-cidr"})
	assert.Error(t, err)
}

func TestParseHostLiteralIP(t *testing.T) {
	tests := []struct {
		host    string
		wantIP  string // empty => expect not-a-literal
		wantLit bool
	}{
		{"127.0.0.1", "127.0.0.1", true},
		{"::1", "::1", true},
		{"2130706433", "127.0.0.1", true},   // decimal
		{"0x7f000001", "127.0.0.1", true},   // hex
		{"017700000001", "127.0.0.1", true}, // octal
		{"example.com", "", false},
		{"my-host", "", false},
		{"", "", false},
	}

	for _, tc := range tests {
		ip, ok := parseHostLiteralIP(tc.host)
		assert.Equal(t, tc.wantLit, ok, "host=%q", tc.host)

		if tc.wantLit {
			require.NotNil(t, ip, "host=%q", tc.host)
			assert.True(t, ip.Equal(net.ParseIP(tc.wantIP)), "host=%q got %s want %s", tc.host, ip, tc.wantIP)
		}
	}
}
