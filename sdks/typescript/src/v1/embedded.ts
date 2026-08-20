import { spawn, ChildProcess } from 'child_process';
import type { AxiosRequestConfig } from 'axios';
import type { ClientConfig, HatchetClientOptions } from '@hatchet/clients/hatchet-client';
import { HatchetClient } from './client/client';
import { createHash, randomUUID } from 'crypto';
import { createReadStream, createWriteStream } from 'fs';
import * as fs from 'fs/promises';
import * as os from 'os';
import * as path from 'path';
import { Readable } from 'stream';
import type { ReadableStream } from 'stream/web';
import { Socket } from 'net';
import { pipeline } from 'stream/promises';

const REPO_URL = 'https://github.com/hatchet-dev/hatchet-embedded';
const DEFAULT_READY_TIMEOUT_MS = 300_000;

export interface EmbeddedOptions {
  /**
   * hatchet-embedded release tag to download (defaults to HATCHET_CLIENT_EMBEDDED_VERSION or
   * latest). Tags correspond to the Hatchet engine version baked into the sidecar,
   * so pinning this pins the engine.
   */
  version?: string;
  /** path to an existing sidecar binary, skips the download (or HATCHET_CLIENT_EMBEDDED_BINARY_PATH) */
  binaryPath?: string;
  /**
   * expected sha256 hex digest of the sidecar binary. When set, it replaces the
   * release's checksums.txt as the trust anchor, so a compromised release
   * channel cannot substitute the binary. Pin it together with `version`.
   */
  checksum?: string;
  /** use an existing Postgres instead of the bundled one */
  databaseUrl?: string;
  /** store the bundled Postgres runtime and data under this directory */
  postgresDataDir?: string;
  grpcPort?: number;
  apiPort?: number;
  /** set to false to start only the engine + gRPC, no REST API */
  startApi?: boolean;
  /** set to false to skip running migrations on startup */
  runMigrations?: boolean;
  /** use RabbitMQ instead of the Postgres message queue */
  rabbitmqUrl?: string;
  logLevel?: string;
  readyTimeoutMs?: number;
}

export interface EmbeddedSidecar {
  token: string;
  tenantId: string;
  grpcAddress: string;
  apiUrl: string;
  stop: () => Promise<void>;
}

interface Handshake {
  token: string;
  tenant_id: string;
  grpc_address: string;
  api_url: string;
}

function sidecarAssetName(): string {
  const platform = { darwin: 'darwin', linux: 'linux' }[process.platform as string];
  const arch = { x64: 'amd64', arm64: 'arm64' }[process.arch as string];
  if (!platform || !arch) {
    throw new Error(`hatchet embedded is not supported on ${process.platform}/${process.arch}`);
  }
  return `hatchet-embedded-sidecar_${platform}_${arch}`;
}

async function resolveVersion(version?: string): Promise<string> {
  const requested = version ?? process.env.HATCHET_CLIENT_EMBEDDED_VERSION ?? 'latest';
  if (requested !== 'latest') {
    return requested;
  }
  const res = await fetch(`${REPO_URL}/releases/latest`, { redirect: 'manual' });
  const location = res.headers.get('location');
  const tag = location?.split('/').pop();
  if (!tag || !tag.startsWith('v')) {
    throw new Error(`could not resolve the latest hatchet-embedded release from ${REPO_URL}`);
  }
  return tag;
}

async function expectedChecksum(tag: string, asset: string): Promise<string> {
  const url = `${REPO_URL}/releases/download/${tag}/checksums.txt`;
  const res = await fetch(url);
  if (!res.ok) {
    throw new Error(`could not download ${url}: ${res.status}`);
  }
  for (const line of (await res.text()).split('\n')) {
    const parts = line.split(/\s+/).filter(Boolean);
    if (parts.length === 2 && parts[1] === asset) {
      return parts[0];
    }
  }
  throw new Error(`no checksum for ${asset} in ${url}`);
}

async function sha256File(filePath: string): Promise<string> {
  const digest = createHash('sha256');
  for await (const chunk of createReadStream(filePath)) {
    digest.update(chunk as Buffer);
  }
  return digest.digest('hex');
}

async function ensureSidecarBinary(version?: string, checksum?: string): Promise<string> {
  const tag = await resolveVersion(version);
  const asset = sidecarAssetName();
  const binPath = path.join(os.homedir(), '.hatchet', 'embedded', tag, asset);

  // verified on every start, not just at download; a cached binary that no
  // longer matches the expected checksum is re-downloaded
  const expected = checksum ?? (await expectedChecksum(tag, asset));

  const cached = await fs.access(binPath).then(
    () => true,
    () => false
  );
  if (cached && (await sha256File(binPath)) === expected) {
    return binPath;
  }

  const url = `${REPO_URL}/releases/download/${tag}/${asset}`;
  const res = await fetch(url);
  if (!res.ok || !res.body) {
    throw new Error(`could not download the hatchet embedded sidecar from ${url}: ${res.status}`);
  }

  await fs.mkdir(path.dirname(binPath), { recursive: true });
  // unique temp name per call (not per process) so concurrent downloads of
  // the same version never clobber each other, even within one process; the
  // final rename is atomic and last-writer-wins
  const tmpPath = `${binPath}.${randomUUID()}.download`;
  try {
    await pipeline(
      Readable.fromWeb(res.body as ReadableStream),
      createWriteStream(tmpPath, { mode: 0o755 })
    );

    const actual = await sha256File(tmpPath);
    if (actual !== expected) {
      throw new Error(`checksum mismatch for ${url}: expected ${expected}, got ${actual}`);
    }

    await fs.rename(tmpPath, binPath);
  } finally {
    await fs.rm(tmpPath, { force: true });
  }
  return binPath;
}

async function waitForHandshake(
  child: ChildProcess,
  handshakePath: string,
  timeoutMs: number
): Promise<Handshake> {
  const deadline = Date.now() + timeoutMs;
  let exited: Error | undefined;
  child.once('exit', (code) => {
    exited = new Error(`hatchet embedded sidecar exited with code ${code} before becoming ready`);
  });

  while (Date.now() < deadline) {
    if (exited) {
      throw exited;
    }
    try {
      const handshake = JSON.parse(await fs.readFile(handshakePath, 'utf8')) as Handshake;
      if (handshake.token) {
        return handshake;
      }
    } catch {
      // not ready yet
    }
    await new Promise((resolve) => {
      setTimeout(resolve, 200);
    });
  }

  child.kill();
  throw new Error(`hatchet embedded sidecar did not become ready within ${timeoutMs}ms`);
}

/**
 * Downloads (and caches) the hatchet-embedded sidecar binary, spawns it, and
 * waits until the embedded engine is ready. The sidecar shuts down when this
 * process exits. Use `HatchetEmbedded()` unless you need the raw
 * connection details.
 */
export async function startEmbeddedSidecar(opts: EmbeddedOptions = {}): Promise<EmbeddedSidecar> {
  const binPath =
    opts.binaryPath ??
    process.env.HATCHET_CLIENT_EMBEDDED_BINARY_PATH ??
    (await ensureSidecarBinary(opts.version, opts.checksum));

  const handshakePath = path.join(
    await fs.mkdtemp(path.join(os.tmpdir(), 'hatchet-embedded-')),
    'handshake.json'
  );

  const args = ['-handshake-file', handshakePath];
  if (opts.databaseUrl) {
    args.push('-database-url', opts.databaseUrl);
  }
  if (opts.rabbitmqUrl) {
    args.push('-rabbitmq-url', opts.rabbitmqUrl);
  }
  if (opts.postgresDataDir) {
    args.push('-postgres-data-dir', opts.postgresDataDir);
  }
  if (opts.grpcPort) {
    args.push('-grpc-port', String(opts.grpcPort));
  }
  if (opts.apiPort) {
    args.push('-api-port', String(opts.apiPort));
  }
  if (opts.startApi === false) {
    args.push('-no-api');
  }
  if (opts.runMigrations === false) {
    args.push('-no-migrations');
  }
  if (opts.logLevel) {
    args.push('-log-level', opts.logLevel);
  }

  // the sidecar shuts down when its stdin closes, so it never outlives this
  // process, no matter how this process dies
  const child = spawn(binPath, args, { stdio: ['pipe', 'ignore', 'inherit'] });
  const killChild = () => child.kill();
  process.once('exit', killChild);

  let handshake: Handshake;
  try {
    handshake = await waitForHandshake(
      child,
      handshakePath,
      opts.readyTimeoutMs ?? DEFAULT_READY_TIMEOUT_MS
    );
  } finally {
    await fs.rm(path.dirname(handshakePath), { recursive: true, force: true }).catch(() => {});
  }

  child.unref();
  if (child.stdin instanceof Socket) {
    child.stdin.unref();
  }

  return {
    token: handshake.token,
    tenantId: handshake.tenant_id,
    grpcAddress: handshake.grpc_address,
    apiUrl: handshake.api_url,
    stop: () =>
      new Promise<void>((resolve) => {
        process.removeListener('exit', killChild);
        if (child.exitCode !== null) {
          resolve();
          return;
        }
        const forceKill = setTimeout(() => child.kill('SIGKILL'), 30_000);
        forceKill.unref?.();
        child.once('exit', () => {
          clearTimeout(forceKill);
          resolve();
        });
        child.kill();
      }),
  };
}

export class HatchetEmbeddedClient {
  /**
   * Runs a full Hatchet engine locally via the hatchet-embedded sidecar (downloaded
   * on first use) and returns a client wired to it. By default the sidecar starts a
   * bundled Postgres; pass `databaseUrl` to point it at your own instead.
   * @param embeddedOpts - Options for the embedded engine (version, ports, database, ...).
   * @param config - Optional configuration overrides for the client.
   * @param options - Optional client options.
   * @param axiosConfig - Optional Axios configuration for HTTP requests.
   * @returns A new Hatchet client instance connected to the embedded engine.
   */
  static async init<
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    T extends Record<string, any> = {},
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    U extends Record<string, any> = {},
  >(
    embeddedOpts?: EmbeddedOptions,
    config?: Omit<Partial<ClientConfig>, 'middleware'>,
    options?: HatchetClientOptions,
    axiosConfig?: AxiosRequestConfig
  ): Promise<HatchetClient<T, U>> {
    const sidecar = await startEmbeddedSidecar(embeddedOpts);
    return HatchetClient.init<T, U>(
      {
        token: sidecar.token,
        tenant_id: sidecar.tenantId,
        host_port: sidecar.grpcAddress,
        ...(sidecar.apiUrl ? { api_url: sidecar.apiUrl } : {}),
        tls_config: { tls_strategy: 'none' },
        ...config,
      },
      options,
      axiosConfig
    );
  }
}
