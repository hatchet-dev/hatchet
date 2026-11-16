import { isIPv6 as nodeIsIPv6 } from 'node:net';
import { grpcTargetBaseUrl, parseGrpcTarget } from './grpc-target';

/**
 * The literal corpus the IP checks are held to: every form here must be accepted or refused
 * as an `ipv6:` target exactly as Node's `net.isIPv6` decides, since the checks replace it.
 */
const IPV6_CORPUS = [
  '::',
  '::1',
  '0:0:0:0:0:0:0:0',
  '1:2:3:4:5:6:7:8',
  '1:2:3:4:5:6:7::',
  'fe80::1%eth0',
  'fe80::1%1',
  'fe80::1%eth:0',
  '::ffff:10.0.0.1',
  '0:0:0:0:0:ffff:10.0.0.1',
  '1:2:3:4:5:6:10.0.0.1',
  '::10.0.0.1',
  '::ffff:010.0.0.1',
  'fe80::1%',
  'fe80::1%eth0%extra',
  'fe80::1%bad space',
  'fe80::1%eth/0',
  '10.0.0.1::',
  '10.0.0.1::1',
  '1:10.0.0.1::2',
  '10.0.0.1::10.0.0.2',
  '1:10.0.0.1:2',
  '1:::2',
  '1::2::3',
  '1:2:3:4:5:6:7::8',
  '1:2:3:4:5:6:7',
  '12345::1',
  'g::1',
  '',
];

const accepts = (target: string): boolean => {
  try {
    parseGrpcTarget(target);
    return true;
  } catch {
    return false;
  }
};

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

  it.each(IPV6_CORPUS)(
    'decides ipv6:%j as Node does, bare and bracketed with a port',
    (literal) => {
      const expected = nodeIsIPv6(literal);
      expect(accepts(`ipv6:${literal}`)).toBe(expected);
      expect(accepts(`ipv6:[${literal}]:7070`)).toBe(expected);
    }
  );

  it.each([
    'fe80::1%',
    'fe80::1%eth0%extra',
    'fe80::1%bad space',
    'fe80::1%eth/0',
    '10.0.0.1::',
    '10.0.0.1::1',
    '1:10.0.0.1::2',
    '10.0.0.1::10.0.0.2',
  ])('refuses the malformed IPv6 literal %j', (literal) => {
    expect(() => parseGrpcTarget(`ipv6:${literal}`)).toThrow(/IPv6/);
    expect(() => parseGrpcTarget(`ipv6:[${literal}]:7070`)).toThrow(/IPv6/);
  });

  it.each([
    ['fe80::1%eth0', 'fe80::1%eth0'],
    ['[fe80::1%eth0]:7070', 'fe80::1%eth0'],
    ['::ffff:10.0.0.1', '::ffff:10.0.0.1'],
    ['[1:2:3:4:5:6:10.0.0.1]:7070', '1:2:3:4:5:6:10.0.0.1'],
    ['[::]:7070', '::'],
  ])('keeps accepting the valid ipv6:%s form', (address, host) => {
    expect(parseGrpcTarget(`ipv6:${address}`).host).toBe(host);
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
