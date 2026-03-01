/**
 * Jest globalSetup for e2e tests - spawns the shared e2e worker and polls until ready.
 */
import { execSync, spawn, ChildProcess } from 'child_process';
import { existsSync, readFileSync, unlinkSync, writeFileSync } from 'fs';
import * as path from 'path';

const E2E_WORKER_PID_FILE = path.join(process.cwd(), '.e2e-worker.pid');

function killProcessTree(pid: number): void {
  const killTimeoutMs = 10_000;
  try {
    if (process.platform === 'win32') {
      execSync(`taskkill /pid ${pid} /T /F`, { stdio: 'ignore', timeout: killTimeoutMs });
    } else {
      execSync(`pkill -TERM -P ${pid} 2>/dev/null; kill -TERM ${pid} 2>/dev/null; sleep 0.3; kill -9 -${pid} 2>/dev/null; kill -9 ${pid} 2>/dev/null`, {
        stdio: 'ignore',
        timeout: killTimeoutMs,
      });
    }
  } catch {
    // ignore
  }
}

function cleanupOrphanedWorker(): void {
  if (!existsSync(E2E_WORKER_PID_FILE)) return;
  const pid = parseInt(readFileSync(E2E_WORKER_PID_FILE, 'utf8'), 10);
  if (Number.isNaN(pid)) return;
  try {
    process.kill(pid, 0);
  } catch {
    unlinkSync(E2E_WORKER_PID_FILE);
    return;
  }
  killProcessTree(pid);
  unlinkSync(E2E_WORKER_PID_FILE);
}
const HEALTH_PORT = 18001;
const POLL_INTERVAL_MS = 500;
const MAX_WAIT_MS = 60_000;

async function waitForHealth(): Promise<void> {
  const start = Date.now();
  while (Date.now() - start < MAX_WAIT_MS) {
    try {
      const resp = await fetch(`http://localhost:${HEALTH_PORT}/health`, {
        method: 'GET',
      });
      if (resp.ok) return;
    } catch {
      // Not ready yet
    }
    await new Promise((r) => setTimeout(r, POLL_INTERVAL_MS));
  }
  throw new Error(
    `e2e worker failed to become healthy within ${MAX_WAIT_MS}ms (port ${HEALTH_PORT})`
  );
}

export default async function globalSetup(): Promise<void> {
  cleanupOrphanedWorker();

  const workerEnv = {
    ...process.env,
    HATCHET_CLIENT_LOG_LEVEL: 'DEBUG',
    HATCHET_CLIENT_WORKER_HEALTHCHECK_ENABLED: 'true',
    HATCHET_CLIENT_WORKER_HEALTHCHECK_PORT: String(HEALTH_PORT),
  };

  const child = spawn(
    'pnpm',
    [
      'exec',
      'ts-node',
      '-r',
      'tsconfig-paths/register',
      '-P',
      'tsconfig.json',
      'src/v1/examples/e2e-worker.ts',
    ],
    {
      env: workerEnv,
      stdio: ['ignore', 'pipe', 'pipe'],
      cwd: process.cwd(),
      detached: true,
    }
  ) as ChildProcess & { pid: number };

  if (child.pid == null) {
    throw new Error('Failed to spawn e2e worker process');
  }

  child.unref();
  child.stdout?.on('data', (d) => process.stdout.write(d.toString()));
  child.stderr?.on('data', (d) => process.stderr.write(d.toString()));

  await waitForHealth();

  writeFileSync(E2E_WORKER_PID_FILE, String(child.pid), 'utf8');
}
