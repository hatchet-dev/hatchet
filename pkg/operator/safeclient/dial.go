package safeclient

import (
	"context"
	"fmt"
	"net"
	"slices"
	"strconv"
)

// Dialer is the network seam a Sender exposes for protocols that are not plain HTTP, such
// as the serverless operator's durable websocket. It has the shape of net.Dialer.DialContext
// so it drops into websocket and other libraries' NetDialContext hooks.
type Dialer interface {
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

// InsecureDestinations reports whether the Sender was built with the development-only
// policy off, so callers layering other protocols on DialContext (the durable websocket
// relay) can relax their own scheme checks in step.
func (s *Sender) InsecureDestinations() bool {
	return s.insecure
}

// DialContext opens a TCP connection to addr (host:port) under the same SSRF policy the
// HTTP path enforces: the port must be allowed, the host is resolved fresh on every call,
// every resolved address is checked against the blocklist, IPv6 is skipped unless enabled,
// and the connection is made to the validated IP, never to the name, so a DNS answer cannot
// change between the check and the connect. Under InsecureDestinations the policy is off and
// a plain dial is made.
//
// TLS is not negotiated here; the caller layers it on top with the original host name so
// certificate verification is unaffected by dialing the IP.
func (s *Sender) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	if network != "tcp" && network != "tcp4" && network != "tcp6" {
		return nil, fmt.Errorf("%w: network %q", ErrBlockedDestination, network)
	}

	dialer := &net.Dialer{Timeout: s.connectTimeout}

	if s.insecure {
		return dialer.DialContext(ctx, network, addr)
	}

	host, portStr, err := net.SplitHostPort(addr)

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBlockedDestination, err)
	}

	port, err := strconv.Atoi(portStr)

	if err != nil || !slices.Contains(s.allowedPorts, port) {
		s.recordBlocked(host, reasonPort, ErrBadPort)
		return nil, fmt.Errorf("%w: got %q", ErrBadPort, portStr)
	}

	ips, err := s.resolveAllowed(ctx, host)

	if err != nil {
		return nil, err
	}

	var lastErr error

	for _, ip := range ips {
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), portStr))

		if err == nil {
			return conn, nil
		}

		lastErr = err

		if ctx.Err() != nil {
			break
		}
	}

	return nil, s.publicError(host, lastErr)
}

// resolveAllowed resolves host and returns its addresses that pass the policy. A host with
// any blocked address is rejected as a whole: a name that mixes public and private answers
// is the DNS rebinding shape, not a legitimate endpoint. The blocked address is logged, not
// returned: the error reaches the tenant.
func (s *Sender) resolveAllowed(ctx context.Context, host string) ([]net.IP, error) {
	var ips []net.IP

	if ip, ok := parseHostLiteralIP(host); ok {
		ips = []net.IP{ip}
	} else {
		addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)

		if err != nil {
			return nil, s.publicError(host, err)
		}

		for _, a := range addrs {
			ips = append(ips, a.IP)
		}
	}

	allowed := make([]net.IP, 0, len(ips))

	for _, ip := range ips {
		if ip.To4() == nil && !s.enableIPv6 {
			continue
		}

		if s.blocklist.isBlockedIP(ip) {
			s.recordBlocked(host, reasonDestination, fmt.Errorf("%w: %s resolves to %s", ErrBlockedDestination, host, ip))

			return nil, fmt.Errorf("%w: %s resolves to a blocked address", ErrBlockedDestination, host)
		}

		allowed = append(allowed, ip)
	}

	if len(allowed) == 0 {
		err := fmt.Errorf("%w: %s has no allowed address", ErrBlockedDestination, host)
		s.recordBlocked(host, reasonDestination, err)

		return nil, err
	}

	return allowed, nil
}
