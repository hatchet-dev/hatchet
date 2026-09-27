import { isIPv4, isIPv6 } from 'net';

/**
 * Where a gRPC target points: the host and port to dial and the `:authority` grpc-js sends for
 * it, which is the target's path as written (the first address for the IP schemes).
 */
export interface GrpcTarget {
  host: string;
  port: number;
  authority: string;
}

/** The port grpc-js dials when a target names none. */
const DEFAULT_PORT = 443;

/*
 * grpc-js parses a target as `scheme:` `//authority/` `path`, all but the path optional; a
 * scheme it has no resolver for (which is what `host:port` looks like) makes the whole string a
 * `dns` path.
 */
const URI_REGEX = /^(?:([A-Za-z0-9+.-]+):)?(?:\/\/([^/]*)\/)?(.+)$/;
const RESOLVER_SCHEMES = new Set(['dns', 'ipv4', 'ipv6', 'unix']);

/**
 * Parses the target forms grpc-js accepts for `host_port`, so the unary transport dials the
 * same endpoint the streaming channel does: `host:port`, `dns:host:port`, `dns:///host:port`,
 * `ipv4:address:port` and `ipv6:[address]:port` (the first address of an address list). A
 * missing port is grpc-js's default, 443. `unix:` sockets and a `dns://nameserver/` authority
 * have no HTTP/2 equivalent here and are refused with a clear error.
 */
export function parseGrpcTarget(target: string): GrpcTarget {
  const match = URI_REGEX.exec(target);
  if (!match) {
    throw new Error(`host_port ${JSON.stringify(target)} is not a gRPC target`);
  }

  const [, rawScheme, authority, rawPath] = match;
  const known = rawScheme !== undefined && RESOLVER_SCHEMES.has(rawScheme);
  const scheme = known ? rawScheme : 'dns';
  const path = known ? rawPath : target;

  if (scheme === 'unix') {
    throw new Error(
      `host_port ${JSON.stringify(target)}: unix domain socket targets are not supported by the unary transport; use a host:port target`
    );
  }
  if (known && authority) {
    throw new Error(
      `host_port ${JSON.stringify(target)}: a target-specific DNS server is not supported by the unary transport; use dns:///host:port`
    );
  }

  const address = scheme === 'dns' ? path : path.split(',')[0];
  const hostPort = splitHostPort(address);
  if (!hostPort) {
    throw new Error(`host_port ${JSON.stringify(target)} is not a valid host and port`);
  }
  if (scheme === 'ipv4' && !isIPv4(hostPort.host)) {
    throw new Error(`host_port ${JSON.stringify(target)} is not an IPv4 address`);
  }
  if (scheme === 'ipv6' && !isIPv6(hostPort.host)) {
    throw new Error(`host_port ${JSON.stringify(target)} is not an IPv6 address`);
  }

  return { host: hostPort.host, port: hostPort.port ?? DEFAULT_PORT, authority: address };
}

/** The base URL Connect dials for a target: an IPv6 host is bracketed as a URL requires. */
export function grpcTargetBaseUrl(target: GrpcTarget, protocol: 'http' | 'https'): string {
  const host = target.host.includes(':') ? `[${target.host}]` : target.host;
  return `${protocol}://${host}:${target.port}`;
}

/*
 * grpc-js's host:port split: a bracketed host is IPv6 with an optional `:port`, exactly one
 * colon separates a host from a numeric port, and any other number of colons is a bare IPv6
 * address with no port.
 */
function splitHostPort(address: string): { host: string; port?: number } | null {
  if (address.startsWith('[')) {
    const end = address.indexOf(']');
    if (end === -1) {
      return null;
    }
    const host = address.slice(1, end);
    if (!host.includes(':')) {
      return null;
    }
    const rest = address.slice(end + 1);
    if (rest === '') {
      return { host };
    }
    if (rest.startsWith(':') && /^\d+$/.test(rest.slice(1))) {
      return { host, port: Number(rest.slice(1)) };
    }
    return null;
  }

  const parts = address.split(':');
  if (parts.length === 2) {
    return /^\d+$/.test(parts[1]) ? { host: parts[0], port: Number(parts[1]) } : null;
  }
  return { host: address };
}
