import { DEFAULT_LOGGER } from '@clients/hatchet-client/hatchet-logger';
import { ClientConfig } from '@clients/hatchet-client/client-config';
import { createGrpcTransport } from '@connectrpc/connect-node';
import { createEventsRpc } from '@clients/event/event-client';
import { execFileSync } from 'child_process';
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'fs';
import { createSecureServer, type Http2Session, type SecureServerOptions } from 'http2';
import type { AddressInfo } from 'net';
import { tmpdir } from 'os';
import { join } from 'path';
import type { TLSSocket } from 'tls';
import {
  createNodeTransport,
  MAX_MESSAGE_BYTES,
  nodeTransportOptions,
  tlsSessionOptions,
} from './node-transport';

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
  it('dials the endpoint a gRPC target names, with the scheme the TLS strategy selects', () => {
    expect(options().baseUrl).toBe('http://127.0.0.1:7070');
    expect(options({ host_port: 'dns:///127.0.0.1:7070' }).baseUrl).toBe('http://127.0.0.1:7070');
    expect(options({ host_port: 'ipv6:[::1]:7070' }).baseUrl).toBe('http://[::1]:7070');
    expect(
      options({ host_port: 'engine.example.com:443', tls_config: { tls_strategy: 'tls' } }).baseUrl
    ).toBe('https://engine.example.com:443');
  });

  it('refuses an unsupported target when the transport is first used', async () => {
    const transport = createNodeTransport({ ...baseConfig, host_port: 'unix:/tmp/engine.sock' });
    const events = createEventsRpc(transport);

    await expect(events.putLog({})).rejects.toThrow(/unix domain socket/);
  });

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

/**
 * A disposable PKI for the TLS tests: two CAs, a server certificate from each (both naming
 * `engine.test` and 127.0.0.1) and a client certificate from the first, generated with openssl
 * into a temporary directory.
 */
interface Pki {
  ca: string;
  otherCa: string;
  server: { key: string; cert: string };
  otherServer: { key: string; cert: string };
  client: { key: string; cert: string };
}

function makePki(dir: string): Pki {
  const openssl = (...args: string[]) => execFileSync('openssl', args, { stdio: 'ignore' });
  const keyArgs = ['-newkey', 'ec', '-pkeyopt', 'ec_paramgen_curve:prime256v1', '-nodes'];
  const file = (name: string) => join(dir, name);

  const ca = (name: string) =>
    openssl(
      'req',
      '-x509',
      ...keyArgs,
      '-days',
      '2',
      '-keyout',
      file(`${name}.key`),
      '-out',
      file(`${name}.crt`),
      '-subj',
      `/CN=Hatchet test ${name}`,
      '-addext',
      'basicConstraints=critical,CA:TRUE'
    );
  const leaf = (name: string, issuer: string, extensions: string[]) => {
    writeFileSync(file(`${name}.ext`), `${extensions.join('\n')}\n`);
    openssl(
      'req',
      '-new',
      ...keyArgs,
      '-keyout',
      file(`${name}.key`),
      '-out',
      file(`${name}.csr`),
      '-subj',
      `/CN=${name}`
    );
    openssl(
      'x509',
      '-req',
      '-in',
      file(`${name}.csr`),
      '-CA',
      file(`${issuer}.crt`),
      '-CAkey',
      file(`${issuer}.key`),
      '-CAcreateserial',
      '-out',
      file(`${name}.crt`),
      '-days',
      '2',
      '-extfile',
      file(`${name}.ext`)
    );
    return { key: file(`${name}.key`), cert: file(`${name}.crt`) };
  };

  ca('ca');
  ca('other-ca');
  const serverExtensions = [
    'subjectAltName=DNS:engine.test,IP:127.0.0.1',
    'extendedKeyUsage=serverAuth',
  ];
  return {
    ca: file('ca.crt'),
    otherCa: file('other-ca.crt'),
    server: leaf('server', 'ca', serverExtensions),
    otherServer: leaf('other-server', 'other-ca', serverExtensions),
    client: leaf('client', 'ca', ['extendedKeyUsage=clientAuth']),
  };
}

interface Observed {
  authority: string | undefined;
  servername: string | undefined;
  path: string | undefined;
}

/**
 * A loopback HTTP/2 server answering every gRPC call with an empty message and `grpc-status: 0`,
 * recording what each request carried.
 */
async function serveGrpc(tls: SecureServerOptions) {
  const requests: Observed[] = [];
  const sessions = new Set<Http2Session>();
  const server = createSecureServer(tls);
  server.on('tlsClientError', () => {});
  server.on('session', (session) => {
    sessions.add(session);
    session.on('error', () => {});
    session.on('close', () => sessions.delete(session));
  });
  server.on('stream', (stream, headers) => {
    requests.push({
      authority: headers[':authority'],
      servername: (stream.session?.socket as TLSSocket | undefined)?.servername || undefined,
      path: headers[':path'],
    });
    stream.on('error', () => {});
    stream.resume();
    stream.on('end', () => {
      stream.respond(
        { ':status': 200, 'content-type': 'application/grpc' },
        { waitForTrailers: true }
      );
      stream.on('wantTrailers', () => stream.sendTrailers({ 'grpc-status': '0' }));
      stream.end(Buffer.alloc(5));
    });
  });
  await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));
  const { port } = server.address() as AddressInfo;

  return {
    port,
    requests,
    close: async () => {
      for (const session of sessions) {
        session.destroy();
      }
      await new Promise<void>((resolve) => server.close(() => resolve()));
    },
  };
}

describe('tlsSessionOptions', () => {
  let dir: string;
  const env = { ...process.env };

  beforeAll(() => {
    dir = mkdtempSync(join(tmpdir(), 'hatchet-tls-'));
    writeFileSync(join(dir, 'roots.pem'), 'roots');
    writeFileSync(join(dir, 'env-roots.pem'), 'env roots');
    writeFileSync(join(dir, 'client.pem'), 'client cert');
    writeFileSync(join(dir, 'client-key.pem'), 'client key');
  });

  afterAll(() => rmSync(dir, { recursive: true, force: true }));

  afterEach(() => {
    process.env = { ...env };
  });

  it('trusts ca_file first, then GRPC_DEFAULT_SSL_ROOTS_FILE_PATH, then the platform roots', () => {
    process.env.GRPC_DEFAULT_SSL_ROOTS_FILE_PATH = join(dir, 'env-roots.pem');

    expect(tlsSessionOptions({ tls_strategy: 'tls', ca_file: join(dir, 'roots.pem') }).ca).toEqual(
      Buffer.from('roots')
    );
    expect(tlsSessionOptions({ tls_strategy: 'tls' }).ca).toEqual(Buffer.from('env roots'));

    delete process.env.GRPC_DEFAULT_SSL_ROOTS_FILE_PATH;
    expect(tlsSessionOptions({ tls_strategy: 'tls' }).ca).toBeUndefined();
  });

  it('applies GRPC_SSL_CIPHER_SUITES when set', () => {
    expect(tlsSessionOptions({ tls_strategy: 'tls' }).ciphers).toBeUndefined();

    process.env.GRPC_SSL_CIPHER_SUITES = 'ECDHE-ECDSA-AES128-GCM-SHA256';
    expect(tlsSessionOptions({ tls_strategy: 'tls' }).ciphers).toBe(
      'ECDHE-ECDSA-AES128-GCM-SHA256'
    );
  });
});

describe('createNodeTransport over TLS', () => {
  let dir: string;
  let pki: Pki;
  const env = { ...process.env };

  beforeAll(() => {
    dir = mkdtempSync(join(tmpdir(), 'hatchet-pki-'));
    pki = makePki(dir);
  });

  afterAll(() => rmSync(dir, { recursive: true, force: true }));

  afterEach(() => {
    process.env = { ...env };
  });

  const serverOptions = (identity: { key: string; cert: string }): SecureServerOptions => ({
    key: readFileSync(identity.key),
    cert: readFileSync(identity.cert),
  });

  async function call(port: number, tls_config: ClientConfig['tls_config']) {
    const config = { ...baseConfig, host_port: `127.0.0.1:${port}`, tls_config };
    return createEventsRpc(createNodeTransport(config)).putLog({});
  }

  it('trusts the roots GRPC_DEFAULT_SSL_ROOTS_FILE_PATH names when ca_file is unset', async () => {
    const server = await serveGrpc(serverOptions(pki.server));
    try {
      process.env.GRPC_DEFAULT_SSL_ROOTS_FILE_PATH = pki.ca;

      await expect(call(server.port, { tls_strategy: 'tls' })).resolves.toBeDefined();
      expect(server.requests).toHaveLength(1);
    } finally {
      await server.close();
    }
  });

  it('does not trust a certificate outside the configured roots', async () => {
    const server = await serveGrpc(serverOptions(pki.otherServer));
    try {
      process.env.GRPC_DEFAULT_SSL_ROOTS_FILE_PATH = pki.ca;

      await expect(call(server.port, { tls_strategy: 'tls' })).rejects.toThrow();
      expect(server.requests).toHaveLength(0);
    } finally {
      await server.close();
    }
  });
});
