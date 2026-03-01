/**
 * Jest globalSetup for e2e tests - spawns the shared e2e worker and polls until ready.
 */
import { spawn, ChildProcess } from 'child_process';
import { writeFileSync } from 'fs';
import * as path from 'path';

const E2E_WORKER_PID_FILE = path.join(process.cwd(), '.e2e-worker.pid');
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
  const workerEnv = {
    ...process.env,
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
