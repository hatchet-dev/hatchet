import { ClientConfigSchema } from './client-config';

function baseConfig() {
  return {
    token: 'token',
    tls_config: {},
    host_port: 'localhost:7070',
    api_url: 'http://localhost:8080',
    tenant_id: 'tenant',
  };
}

describe('ClientConfigSchema cancellation timing', () => {
  it('applies defaults (milliseconds)', () => {
    const cfg = ClientConfigSchema.parse(baseConfig());
    expect(cfg.cancellation_grace_period).toBe(1000);
    expect(cfg.cancellation_warning_threshold).toBe(300);
  });

  it('accepts integer milliseconds', () => {
    const cfg = ClientConfigSchema.parse({
      ...baseConfig(),
      cancellation_grace_period: 2500,
      cancellation_warning_threshold: 400,
    });
    expect(cfg.cancellation_grace_period).toBe(2500);
    expect(cfg.cancellation_warning_threshold).toBe(400);
  });

  it('rejects invalid values', () => {
    expect(() =>
      ClientConfigSchema.parse({
        ...baseConfig(),
        cancellation_grace_period: -1,
      })
    ).toThrow();
    expect(() =>
      ClientConfigSchema.parse({
        ...baseConfig(),
        cancellation_warning_threshold: 0.1,
      })
    ).toThrow();
    expect(() =>
      ClientConfigSchema.parse({
        ...baseConfig(),
        cancellation_warning_threshold: 'nope' as any,
      })
    ).toThrow();
    expect(() =>
      ClientConfigSchema.parse({
        ...baseConfig(),
        cancellation_grace_period: '7s' as any,
      })
    ).toThrow();
  });
});

describe('ClientConfigSchema gRPC max message length', () => {
  it('applies 4MB defaults for both recv and send', () => {
    const cfg = ClientConfigSchema.parse(baseConfig());
    expect(cfg.grpc_max_recv_message_length).toBe(4 * 1024 * 1024);
    expect(cfg.grpc_max_send_message_length).toBe(4 * 1024 * 1024);
  });

  it('accepts custom positive integer values', () => {
    const cfg = ClientConfigSchema.parse({
      ...baseConfig(),
      grpc_max_recv_message_length: 8 * 1024 * 1024,
      grpc_max_send_message_length: 16 * 1024 * 1024,
    });
    expect(cfg.grpc_max_recv_message_length).toBe(8 * 1024 * 1024);
    expect(cfg.grpc_max_send_message_length).toBe(16 * 1024 * 1024);
  });

  it('rejects invalid values', () => {
    expect(() =>
      ClientConfigSchema.parse({
        ...baseConfig(),
        grpc_max_recv_message_length: 0,
      })
    ).toThrow();
    expect(() =>
      ClientConfigSchema.parse({
        ...baseConfig(),
        grpc_max_send_message_length: -1,
      })
    ).toThrow();
    expect(() =>
      ClientConfigSchema.parse({
        ...baseConfig(),
        grpc_max_recv_message_length: 1.5,
      })
    ).toThrow();
    expect(() =>
      ClientConfigSchema.parse({
        ...baseConfig(),
        grpc_max_send_message_length: '4mb' as any,
      })
    ).toThrow();
  });
});
