import { DEFAULT_LOGGER } from '@clients/hatchet-client/hatchet-logger';
import { ClientConfig } from '@clients/hatchet-client/client-config';
import { nodeTransportOptions } from './node-transport';

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
});
