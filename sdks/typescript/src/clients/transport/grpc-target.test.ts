import { grpcTargetBaseUrl, parseGrpcTarget } from './grpc-target';

describe('parseGrpcTarget', () => {
  it.each([
    ['host:port', '127.0.0.1:7070', '127.0.0.1', 7070, '127.0.0.1:7070'],
    ['hostname', 'engine.example.com:443', 'engine.example.com', 443, 'engine.example.com:443'],
    ['dns:host:port', 'dns:127.0.0.1:7070', '127.0.0.1', 7070, '127.0.0.1:7070'],
    ['dns:///host:port', 'dns:///127.0.0.1:7070', '127.0.0.1', 7070, '127.0.0.1:7070'],
    [
      'dns:///hostname',
      'dns:///engine.example.com',
      'engine.example.com',
      443,
      'engine.example.com',
    ],
    ['ipv4:address:port', 'ipv4:127.0.0.1:7070', '127.0.0.1', 7070, '127.0.0.1:7070'],
    ['ipv4 address list', 'ipv4:10.0.0.1:7070,10.0.0.2:7070', '10.0.0.1', 7070, '10.0.0.1:7070'],
    ['ipv6:[address]:port', 'ipv6:[::1]:7070', '::1', 7070, '[::1]:7070'],
    ['ipv6 bare address', 'ipv6:::1', '::1', 443, '::1'],
    ['bracketed IPv6 without scheme', '[::1]:7070', '::1', 7070, '[::1]:7070'],
    ['host without port', 'localhost', 'localhost', 443, 'localhost'],
  ])('parses the %s form', (_label, target, host, port, authority) => {
    expect(parseGrpcTarget(target)).toEqual({ host, port, authority });
  });

  it('refuses unix socket targets with a clear error', () => {
    expect(() => parseGrpcTarget('unix:/var/run/hatchet.sock')).toThrow(/unix domain socket/);
    expect(() => parseGrpcTarget('unix:///var/run/hatchet.sock')).toThrow(/unix domain socket/);
  });

  it('refuses a target-specific DNS server', () => {
    expect(() => parseGrpcTarget('dns://8.8.8.8/engine:7070')).toThrow(/DNS server/);
  });

  it('refuses malformed addresses', () => {
    expect(() => parseGrpcTarget('ipv4:engine:7070')).toThrow(/IPv4/);
    expect(() => parseGrpcTarget('ipv6:[engine]:7070')).toThrow(/valid host and port/);
    expect(() => parseGrpcTarget('engine:port')).toThrow(/valid host and port/);
  });
});

describe('grpcTargetBaseUrl', () => {
  it('builds the URL Connect dials, bracketing IPv6 hosts', () => {
    expect(grpcTargetBaseUrl(parseGrpcTarget('dns:///engine:7070'), 'https')).toBe(
      'https://engine:7070'
    );
    expect(grpcTargetBaseUrl(parseGrpcTarget('ipv6:[::1]:7070'), 'http')).toBe('http://[::1]:7070');
    expect(grpcTargetBaseUrl(parseGrpcTarget('engine'), 'https')).toBe('https://engine:443');
  });
});
