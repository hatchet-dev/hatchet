import { readFileSync } from 'fs';
import type { SecureClientSessionOptions } from 'http2';
import {
  compressionGzip,
  createGrpcTransport,
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
 * `none` means plaintext, `tls` verifies the server against `ca_file`, and `mtls` (the default
 * when no strategy is set) additionally presents `cert_file` and `key_file`. `server_name`
 * overrides the hostname used for SNI and certificate verification, which is what lets a
 * client reach an engine through an address its certificate does not name.
 */
export function tlsSessionOptions(tls: ClientConfig['tls_config']): SecureClientSessionOptions {
  const options: SecureClientSessionOptions = {};

  if (tls.ca_file) {
    options.ca = readFileSync(tls.ca_file);
  }

  if (tls.tls_strategy !== 'tls') {
    if (tls.key_file) {
      options.key = readFileSync(tls.key_file);
    }
    if (tls.cert_file) {
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
 */
export function nodeTransportOptions(config: ClientConfig): GrpcTransportOptions {
  const insecure = config.tls_config.tls_strategy === 'none';
  const target = parseGrpcTarget(config.host_port);

  return {
    baseUrl: grpcTargetBaseUrl(target, insecure ? 'http' : 'https'),
    nodeOptions: insecure ? undefined : tlsSessionOptions(config.tls_config),
    interceptors: [createAuthInterceptor(config.token)],
    sendCompression: compressionGzip,
    readMaxBytes: messageLimit(config.grpc_max_recv_message_length),
    writeMaxBytes: messageLimit(config.grpc_max_send_message_length),
    ...SESSION_OPTIONS,
  };
}
