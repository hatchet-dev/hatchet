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
