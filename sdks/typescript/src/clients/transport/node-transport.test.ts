import { DEFAULT_LOGGER } from '@clients/hatchet-client/hatchet-logger';
import { ClientConfig } from '@clients/hatchet-client/client-config';
import { createGrpcTransport } from '@connectrpc/connect-node';
import { MAX_MESSAGE_BYTES, nodeTransportOptions } from './node-transport';

const baseConfig: ClientConfig = {
  token: 'test-token',
  host_port: '127.0.0.1:7070',
  tls_config: { tls_strategy: 'none' },
  api_url: 'http://127.0.0.1:1',
  tenant_id: 'tenant',
  log_level: 'OFF',
  logger: DEFAULT_LOGGER,
};

function options(overrides: Partial<ClientConfig> = {}) {
  return nodeTransportOptions({ ...baseConfig, ...overrides });
}

describe('nodeTransportOptions', () => {
  it('keeps the connection alive with the nice-grpc channel settings', () => {
    // grpc-helpers.ts: keepalive_time_ms 10 s, keepalive_timeout_ms 60 s, permit without calls,
    // client_idle_timeout_ms 60 s.
    expect(options()).toMatchObject({
      pingIntervalMs: 10_000,
      pingTimeoutMs: 60_000,
      pingIdleConnection: true,
      idleConnectionTimeoutMs: 60_000,
    });
  });

  it('defaults both message limits to 4 MiB', () => {
    expect(options()).toMatchObject({
      readMaxBytes: 4 * 1024 * 1024,
      writeMaxBytes: 4 * 1024 * 1024,
    });
  });

  it('passes configured message limits through up to the transport maximum', () => {
    expect(
      options({ grpc_max_recv_message_length: 1024, grpc_max_send_message_length: 2048 })
    ).toMatchObject({ readMaxBytes: 1024, writeMaxBytes: 2048 });
    expect(
      options({
        grpc_max_recv_message_length: MAX_MESSAGE_BYTES,
        grpc_max_send_message_length: MAX_MESSAGE_BYTES,
      })
    ).toMatchObject({ readMaxBytes: MAX_MESSAGE_BYTES, writeMaxBytes: MAX_MESSAGE_BYTES });
  });

  it('clamps message limits above the transport maximum instead of failing the call', () => {
    const over = MAX_MESSAGE_BYTES + 1;
    const built = options({
      grpc_max_recv_message_length: over,
      grpc_max_send_message_length: over,
    });

    expect(built).toMatchObject({
      readMaxBytes: MAX_MESSAGE_BYTES,
      writeMaxBytes: MAX_MESSAGE_BYTES,
    });
    // Connect validates the limits when the transport is created, so this is where an
    // unclamped value would have thrown.
    expect(() => createGrpcTransport(built)).not.toThrow();
  });
});
