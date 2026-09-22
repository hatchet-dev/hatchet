import { readFileSync } from 'fs';
import type { SecureClientSessionOptions } from 'http2';
import {
  compressionGzip,
  createGrpcTransport,
  Http2SessionManager,
  type GrpcTransportOptions,
} from '@connectrpc/connect-node';
import type { ClientConfig } from '@clients/hatchet-client/client-config';
import { grpcTargetBaseUrl, parseGrpcTarget } from './grpc-target';
import { createAuthInterceptor, type Transport } from './transport';

const DEFAULT_MAX_MESSAGE_BYTES = 4 * 1024 * 1024;

/**
 * The largest message size Connect can enforce (a 32-bit length prefix). The config schema
 * accepts any positive integer, as grpc-js does, so a larger configured limit is clamped to it
 * rather than refused at the first call.
 */
export const MAX_MESSAGE_BYTES = 0xffffffff;

function messageLimit(configured: number | undefined): number {
  return Math.min(configured ?? DEFAULT_MAX_MESSAGE_BYTES, MAX_MESSAGE_BYTES);
}

/**
 * The connection settings the `nice-grpc` channel uses (`grpc.keepalive_time_ms`,
 * `grpc.keepalive_timeout_ms`, `grpc.keepalive_permit_without_calls` and
 * `grpc.client_idle_timeout_ms` in `util/grpc-helpers.ts`), so a unary connection is kept alive,
 * given up on and idled exactly as a streaming one.
 */
export const SESSION_OPTIONS = {
  pingIntervalMs: 10 * 1000,
  pingTimeoutMs: 60 * 1000,
  pingIdleConnection: true,
  idleConnectionTimeoutMs: 60 * 1000,
} as const;

/**
 * Builds the TLS session options for the engine connection from the client's `tls_config`:
 * `none` means plaintext, `tls` verifies the server against the trusted roots, and `mtls` (the
 * default when no strategy is set) additionally presents `cert_file` and `key_file`.
 * `server_name` overrides the hostname used for SNI and certificate verification, which is what
 * lets a client reach an engine through an address its certificate does not name.
 *
 * The roots and cipher suites follow grpc-js, which the streaming channel uses: `ca_file`, else
 * the bundle `GRPC_DEFAULT_SSL_ROOTS_FILE_PATH` names, else the platform roots; and
 * `GRPC_SSL_CIPHER_SUITES` when set.
 */
export function tlsSessionOptions(tls: ClientConfig['tls_config']): SecureClientSessionOptions {
  // grpc-js verifies the peer regardless of NODE_TLS_REJECT_UNAUTHORIZED, which is Node's
  // process-wide switch; set explicitly, the switch cannot weaken this connection either.
  const options: SecureClientSessionOptions = { rejectUnauthorized: true };

  const rootsFile = tls.ca_file || process.env.GRPC_DEFAULT_SSL_ROOTS_FILE_PATH;
  if (rootsFile) {
    options.ca = readFileSync(rootsFile);
  }

  const ciphers = process.env.GRPC_SSL_CIPHER_SUITES;
  if (ciphers) {
    options.ciphers = ciphers;
  }

  if (tls.tls_strategy !== 'tls') {
    // grpc-js refuses half a client identity with these messages rather than connecting without
    // one, which an endpoint that does not require client certificates would silently accept.
    if (tls.key_file && !tls.cert_file) {
      throw new Error('Private key must be given with accompanying certificate chain');
    }
    if (tls.cert_file && !tls.key_file) {
      throw new Error('Certificate chain must be given with accompanying private key');
    }
    if (tls.key_file && tls.cert_file) {
      options.key = readFileSync(tls.key_file);
      options.cert = readFileSync(tls.cert_file);
    }
  }

  if (tls.server_name) {
    options.servername = tls.server_name;
  }

  return options;
}

/**
 * Creates the Node transport: the gRPC protocol over HTTP/2 (`createGrpcTransport` speaks
 * nothing else), the same wire the SDK's `nice-grpc` clients speak, so an engine sees no
 * difference between the two. The bearer token from the config is attached to every call and
 * the connection is kept alive with pings the way the `nice-grpc` channel is.
 *
 * The transport is built on the first call so that constructing a client reads no TLS files;
 * a client whose unary RPCs are never used never touches them.
 */
export function createNodeTransport(config: ClientConfig): Transport {
  let transport: Transport | undefined;

  const get = () => {
    if (!transport) {
      transport = createGrpcTransport(nodeTransportOptions(config));
    }
    return transport;
  };

  return {
    unary: (...args) => get().unary(...args),
    stream: (...args) => get().stream(...args),
  };
}

/**
 * The options `createNodeTransport` builds its transport from, derived from the client config.
 * The session manager is built here from `nodeOptions` and the session settings; Connect takes
 * the connection from it and leaves those transport-level copies as the record of what it uses.
 */
export function nodeTransportOptions(config: ClientConfig): GrpcTransportOptions {
  const insecure = config.tls_config.tls_strategy === 'none';
  const target = parseGrpcTarget(config.host_port);
  const baseUrl = grpcTargetBaseUrl(target, insecure ? 'http' : 'https');
  const nodeOptions = insecure ? undefined : tlsSessionOptions(config.tls_config);

  return {
    baseUrl,
    nodeOptions,
    sessionManager: createSessionManager(baseUrl, target.authority, nodeOptions),
    interceptors: [createAuthInterceptor(config.token)],
    sendCompression: compressionGzip,
    readMaxBytes: messageLimit(config.grpc_max_recv_message_length),
    writeMaxBytes: messageLimit(config.grpc_max_send_message_length),
    ...SESSION_OPTIONS,
  };
}

type SessionManager = NonNullable<GrpcTransportOptions['sessionManager']>;

/**
 * The HTTP/2 session for the transport, sending `:authority` as grpc-js does: the target as
 * written, so `server_name` changes only SNI and certificate verification. Node otherwise
 * derives the authority from `servername` when it is set, which would also change the virtual
 * host a proxy in front of the engine routes on.
 */
function createSessionManager(
  baseUrl: string,
  authority: string,
  nodeOptions: SecureClientSessionOptions | undefined
): SessionManager {
  const manager = new Http2SessionManager(baseUrl, SESSION_OPTIONS, nodeOptions);

  return {
    authority: manager.authority,
    request: (method, path, headers, options) =>
      manager.request(method, path, { ':authority': authority, ...headers }, options),
    notifyResponseByteRead: (stream) => manager.notifyResponseByteRead(stream),
  };
}
