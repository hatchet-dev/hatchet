import { createNodeTransport } from '@hatchet/clients/transport/node-transport';
import { HatchetClient } from './client';

jest.mock('@hatchet/clients/transport/node-transport', () => {
  const actual = jest.requireActual('@hatchet/clients/transport/node-transport');
  return { ...actual, createNodeTransport: jest.fn(actual.createNodeTransport) };
});

const config = {
  token:
    'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJncnBjX2Jyb2FkY2FzdF9hZGRyZXNzIjoiMTI3LjAuMC4xOjgwODAiLCJzZXJ2ZXJfdXJsIjoiaHR0cDovL2xvY2FsaG9zdDo4MDgwIiwic3ViIjoiNzA3ZDA4NTUtODBhYi00ZTFmLWExNTYtZjFjNDU0NmNiZjUyIn0K.abcdef',
  host_port: '127.0.0.1:1',
  api_url: 'http://127.0.0.1:1',
  tls_config: { tls_strategy: 'none' as const },
  log_level: 'OFF' as const,
};

describe('HatchetClient transport', () => {
  // Reaching `v0` builds the durable listener, whose eviction interval would otherwise keep
  // the process alive after the tests.
  beforeEach(() => jest.useFakeTimers());
  afterEach(() => jest.useRealTimers());

  it('builds one unary transport and shares it with the v0 facade', () => {
    const client = new HatchetClient(config);

    // Touch every facade that issues unary RPCs.
    expect(client.admin).toBeDefined();
    expect(client.events).toBeDefined();
    expect(client.v0.admin).toBeDefined();
    expect(client.v0.event).toBeDefined();

    expect(createNodeTransport).toHaveBeenCalledTimes(1);
  });

  it('hands an injected transport to the v0 facade', () => {
    const transport = {
      unary: jest.fn(),
      stream: jest.fn(),
    };
    const client = new HatchetClient(config, { transport });

    // The facades call the transport lazily; a call through v0 must reach the injected one.
    client.v0.admin.client.putRateLimit({ key: 'k', limit: 1 }).catch(() => {});

    expect(transport.unary).toHaveBeenCalledTimes(1);
    expect(createNodeTransport).not.toHaveBeenCalled();
  });
});
