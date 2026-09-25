import { Code, ConnectError } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';
import { grpcTargetBaseUrl, parseGrpcTarget } from './grpc-target';
import { createAuthInterceptor, DEFAULT_MAX_MESSAGE_BYTES, type Transport } from './transport';

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
  /**
   * The largest response accepted, in bytes after the runtime's HTTP decompression. Defaults
   * to 4 MiB, the Node transport's `grpc_max_recv_message_length` default.
   */
  maxReceiveMessageBytes?: number;
  /**
   * The largest request sent, in bytes. Defaults to 4 MiB, the Node transport's
   * `grpc_max_send_message_length` default.
   */
  maxSendMessageBytes?: number;
}

/** The limits `limitMessageSizes` applies, in bytes. */
export interface MessageSizeLimits {
  receive: number;
  send: number;
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
 * to every call, and requests and responses are held to the same message size limits as the
 * Node transport's.
 */
export function createFetchTransport(options: FetchTransportOptions): Transport {
  const send: typeof globalThis.fetch =
    options.fetch ?? ((input, init) => globalThis.fetch(input, init));
  const limits: MessageSizeLimits = {
    receive: options.maxReceiveMessageBytes ?? DEFAULT_MAX_MESSAGE_BYTES,
    send: options.maxSendMessageBytes ?? DEFAULT_MAX_MESSAGE_BYTES,
  };

  return createConnectTransport({
    baseUrl: resolveServerUrl(options),
    useBinaryFormat: true,
    interceptors: [createAuthInterceptor(options.token)],
    fetch: limitMessageSizes(send, limits),
  });
}

/**
 * Wraps a `fetch` with the message size limits the Node transport enforces through Connect's
 * `readMaxBytes` and `writeMaxBytes`, which the fetch transport has no option for. A request
 * body over the send limit is refused before anything is sent, and a response body that grows
 * past the receive limit is cancelled as it streams, counted after the runtime's HTTP
 * decompression so a small compressed body cannot expand past it; the limit covers Connect
 * error bodies as well as messages. Both surface as a `ConnectError` with
 * `Code.ResourceExhausted`, as they do on the Node transport.
 */
export function limitMessageSizes(
  send: typeof globalThis.fetch,
  limits: MessageSizeLimits
): typeof globalThis.fetch {
  return async (input, init) => {
    const size = bodySize(init?.body);
    if (size > limits.send) {
      throw new ConnectError(
        `message size ${size} is larger than configured maxSendMessageBytes ${limits.send}`,
        Code.ResourceExhausted
      );
    }

    const response = await send(input, init);
    if (!response.body) {
      return response;
    }
    return new Response(limitBody(response.body, limits.receive), {
      status: response.status,
      statusText: response.statusText,
      headers: response.headers,
    });
  };
}

/** The size of the bodies Connect sends; a body of another kind is not measured. */
function bodySize(body: RequestInit['body']): number {
  if (typeof body === 'string') {
    return new TextEncoder().encode(body).byteLength;
  }
  if (body instanceof ArrayBuffer || ArrayBuffer.isView(body)) {
    return body.byteLength;
  }
  if (body instanceof Blob) {
    return body.size;
  }
  return 0;
}

function limitBody(body: ReadableStream<Uint8Array>, max: number): ReadableStream<Uint8Array> {
  const reader = body.getReader();
  let received = 0;

  return new ReadableStream<Uint8Array>({
    async pull(controller) {
      const { done, value } = await reader.read();
      if (done) {
        controller.close();
        return;
      }
      received += value.byteLength;
      if (received > max) {
        const error = new ConnectError(
          `response body is larger than configured maxReceiveMessageBytes ${max}`,
          Code.ResourceExhausted
        );
        await reader.cancel(error).catch(() => undefined);
        controller.error(error);
        return;
      }
      controller.enqueue(value);
    },
    cancel(reason) {
      return reader.cancel(reason);
    },
  });
}
