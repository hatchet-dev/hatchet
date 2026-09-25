import { createConnectTransport } from '@connectrpc/connect-web';
import { grpcTargetBaseUrl, parseGrpcTarget } from './grpc-target';
import { createAuthInterceptor, type Transport } from './transport';

/**
 * How a fetch-based client reaches the engine. `none` speaks plain HTTP; `tls` speaks HTTPS
 * and verifies the server the way the runtime's `fetch` does. `serverName` is accepted so the
 * config mirrors the Node client's `tls_config`, but `fetch` has no way to verify a
 * certificate against a name other than the URL host, so it is only honored by a transport
 * that can (a custom `transport` passed to the client).
 */
export interface FetchTlsConfig {
  strategy: 'none' | 'tls';
  serverName?: string;
}

export interface FetchTransportOptions {
  /** The tenant API token, sent as a bearer token on every call. */
  token: string;
  /**
   * The engine's base URL, for example `https://engine.example.com:7070`: an absolute
   * `http://` or `https://` URL with no username, password, query string or fragment. The RPC
   * paths are appended to it as given, minus trailing slashes. Its scheme must agree with
   * `tls`: `http://` needs `{ strategy: 'none' }`, since the token travels on every call.
   */
  serverUrl?: string;
  /** The engine's `host:port`; used with `tls` to build the base URL when `serverUrl` is unset. */
  hostPort?: string;
  /**
   * Defaults to `tls`. `hostPort` takes its scheme from it (`https://` unless `none`), and a
   * `serverUrl` must agree with it.
   */
  tls?: FetchTlsConfig;
  /** The `fetch` to send requests with. Defaults to the runtime's global `fetch`. */
  fetch?: typeof globalThis.fetch;
}

/**
 * Resolves the engine's base URL from a `serverUrl` or from `hostPort` plus the TLS
 * strategy, the two ways a fetch-based client can be pointed at an engine. `hostPort` takes
 * the gRPC target forms the Node client takes (`host:port`, `dns:///host:port`, `ipv4:`,
 * `ipv6:`), so a token or config shared with a Node client works unchanged. A `serverUrl` is
 * checked against the rules its documentation states, so a plaintext URL cannot slip past
 * the TLS default.
 */
export function resolveServerUrl(
  options: Pick<FetchTransportOptions, 'serverUrl' | 'hostPort' | 'tls'>
): string {
  const strategy = options.tls?.strategy ?? 'tls';

  if (options.serverUrl) {
    return validateServerUrl(options.serverUrl, strategy);
  }

  if (options.hostPort) {
    const scheme = strategy === 'none' ? 'http' : 'https';
    return grpcTargetBaseUrl(parseGrpcTarget(options.hostPort), scheme);
  }

  throw new Error('a fetch transport needs a serverUrl or a hostPort');
}

/*
 * The messages name the field and the rule only: a URL can carry a password, and the token
 * is never part of it, so neither is echoed.
 */
function validateServerUrl(serverUrl: string, strategy: FetchTlsConfig['strategy']): string {
  let url: URL;
  try {
    url = new URL(serverUrl);
  } catch {
    throw new Error('serverUrl must be an absolute http:// or https:// URL');
  }
  if (url.protocol !== 'http:' && url.protocol !== 'https:') {
    throw new Error('serverUrl must be an absolute http:// or https:// URL');
  }
  if (url.username || url.password) {
    throw new Error(
      'serverUrl must not carry a username or password; the client authenticates with its token'
    );
  }
  if (serverUrl.includes('?') || serverUrl.includes('#')) {
    throw new Error(
      'serverUrl must not carry a query string or fragment; the RPC path is appended to it'
    );
  }
  if (url.protocol === 'http:' && strategy !== 'none') {
    throw new Error(
      "serverUrl is http:// but tls.strategy is 'tls' (the default), which would send the token in plaintext; set tls: { strategy: 'none' } to reach an engine over plain HTTP"
    );
  }
  if (url.protocol === 'https:' && strategy === 'none') {
    throw new Error(
      "serverUrl is https:// but tls.strategy is 'none'; use tls: { strategy: 'tls' } or an http:// serverUrl"
    );
  }
  return serverUrl.replace(/\/+$/, '');
}

/**
 * Creates the fetch transport: the Connect protocol with binary protobuf bodies over the
 * runtime's `fetch`, so it runs wherever `fetch` does (Cloudflare Workers, Vercel Functions,
 * Deno, Bun, browsers). The engine answers unary calls over HTTP/1.1 or HTTP/2, whichever
 * the runtime negotiates; nothing here depends on the version. The bearer token is attached
 * to every call.
 */
export function createFetchTransport(options: FetchTransportOptions): Transport {
  return createConnectTransport({
    baseUrl: resolveServerUrl(options),
    useBinaryFormat: true,
    interceptors: [createAuthInterceptor(options.token)],
    fetch: options.fetch,
  });
}
