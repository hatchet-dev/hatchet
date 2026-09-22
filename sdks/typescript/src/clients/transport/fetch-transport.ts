import { createConnectTransport } from '@connectrpc/connect-web';
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
  /** The engine's base URL, for example `https://engine.example.com:7070`. */
  serverUrl?: string;
  /** The engine's `host:port`; used with `tls` to build the base URL when `serverUrl` is unset. */
  hostPort?: string;
  /** Defaults to `tls`. */
  tls?: FetchTlsConfig;
  /** The `fetch` to send requests with. Defaults to the runtime's global `fetch`. */
  fetch?: typeof globalThis.fetch;
}

/**
 * Resolves the engine's base URL from a `serverUrl` or from `hostPort` plus the TLS
 * strategy, the two ways a fetch-based client can be pointed at an engine.
 */
export function resolveServerUrl(
  options: Pick<FetchTransportOptions, 'serverUrl' | 'hostPort' | 'tls'>
): string {
  if (options.serverUrl) {
    return options.serverUrl.replace(/\/+$/, '');
  }

  if (options.hostPort) {
    const scheme = options.tls?.strategy === 'none' ? 'http' : 'https';
    return `${scheme}://${options.hostPort}`;
  }

  throw new Error('a fetch transport needs a serverUrl or a hostPort');
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
