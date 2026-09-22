package safeclient

import (
	"net"
	"strconv"
	"strings"
)

// DefaultBlockedCIDRs is the always-applied denylist of internal/reserved IP space. It is
// declared explicitly (rather than relying on safeurl's library defaults) so the SSRF
// policy is auditable in one place and cannot drift with library internals.
//
// NOTE: this list intentionally contains ONLY well-known reserved/private/documentation
// ranges. Real infrastructure CIDRs (VPC, EKS pod/service ranges, internal load
// balancers, etc.) must never be hardcoded here — they are supplied at runtime via
// Config.InfraBlockedCIDRs. The cloud metadata endpoint (169.254.169.254) is already
// covered by the link-local 169.254.0.0/16 range below.
var DefaultBlockedCIDRs = []string{
	"0.0.0.0/8",          // "this network"
	"10.0.0.0/8",         // RFC 1918
	"100.64.0.0/10",      // CGNAT (RFC 6598)
	"127.0.0.0/8",        // loopback
	"169.254.0.0/16",     // link-local incl. cloud metadata (169.254.169.254)
	"172.16.0.0/12",      // RFC 1918
	"192.0.0.0/24",       // IETF protocol assignments
	"192.0.2.0/24",       // TEST-NET-1
	"192.88.99.0/24",     // 6to4 relay anycast
	"192.168.0.0/16",     // RFC 1918
	"198.18.0.0/15",      // benchmarking
	"198.51.100.0/24",    // TEST-NET-2
	"203.0.113.0/24",     // TEST-NET-3
	"224.0.0.0/4",        // multicast
	"240.0.0.0/4",        // reserved
	"255.255.255.255/32", // broadcast
	"::/128",             // unspecified
	"::1/128",            // loopback
	"64:ff9b::/96",       // NAT64
	"fc00::/7",           // unique local
	"fe80::/10",          // link-local
	"ff00::/8",           // multicast

	// NOTE: IPv4-mapped IPv6 (::ffff:0:0/96) is deliberately NOT listed as a CIDR. Go's
	// net.IPNet.Contains normalizes that prefix to 0.0.0.0/0, which would block ALL IPv4
	// traffic (every IPv4 maps to ::ffff:x.y.z.w). Mapped addresses such as
	// ::ffff:127.0.0.1 are still blocked, because net.ParseIP normalizes them to their
	// IPv4 form and they then match the IPv4 ranges above. safeurl's own default denylist
	// omits this prefix for the same reason.
}

// blocklist holds the parsed denylist used for our own synchronous pre-checks. It mirrors
// the ranges handed to safeurl, which performs the authoritative dial-time check against
// the resolved IP. We keep a local copy so IP-literal endpoints (and obfuscated literals)
// can be rejected before any network I/O, and so the decision logic is unit-testable.
type blocklist struct {
	nets []*net.IPNet
}

// newBlocklist parses DefaultBlockedCIDRs plus the caller-supplied extra CIDRs. It returns
// an error if any CIDR string fails to parse.
func newBlocklist(extra []string) (*blocklist, error) {
	all := make([]string, 0, len(DefaultBlockedCIDRs)+len(extra))
	all = append(all, DefaultBlockedCIDRs...)
	all = append(all, extra...)

	nets := make([]*net.IPNet, 0, len(all))

	for _, cidr := range all {
		_, n, err := net.ParseCIDR(cidr)

		if err != nil {
			return nil, err
		}

		nets = append(nets, n)
	}

	return &blocklist{nets: nets}, nil
}

// isBlockedIP reports whether ip falls inside any blocked range. A nil/invalid IP is
// treated as blocked (fail-closed).
func (b *blocklist) isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}

	for _, n := range b.nets {
		if n.Contains(ip) {
			return true
		}
	}

	return false
}

// parseHostLiteralIP attempts to interpret a URL host as an IP literal, including the
// obfuscated forms that are a classic SSRF bypass: decimal (2130706433), hex (0x7f000001),
// and octal (017700000001) encodings of an IPv4 address. It returns (ip, true) if host is
// any kind of IP literal, or (nil, false) if host should be treated as a DNS name (and
// thus left to safeurl's dial-time resolution + check).
func parseHostLiteralIP(host string) (net.IP, bool) {
	host = strings.TrimSpace(host)

	if host == "" {
		return nil, false
	}

	// Standard dotted/colon form (IPv4, IPv6). Bracketed IPv6 hosts arrive here already
	// unbracketed via url.Hostname().
	if ip := net.ParseIP(host); ip != nil {
		return ip, true
	}

	// A dotted host with a non-numeric label is a DNS name, not an obfuscated literal.
	// Only attempt integer interpretation for bare single-token hosts.
	if strings.Contains(host, ".") || strings.Contains(host, ":") {
		return nil, false
	}

	// Try decimal/hex/octal integer forms of a 32-bit IPv4 address. strconv.ParseUint with
	// base 0 honors the 0x (hex) and leading-0 (octal) prefixes.
	if v, err := strconv.ParseUint(host, 0, 64); err == nil && v <= 0xffffffff {
		return net.IPv4(byte(v>>24), byte(v>>16), byte(v>>8), byte(v)), true // #nosec G115 -- v <= 0xffffffff is checked above
	}

	return nil, false
}
