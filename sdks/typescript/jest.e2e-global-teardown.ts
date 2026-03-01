/**
 * Jest globalTeardown for e2e tests - kills the shared e2e worker process.
 */
import { existsSync, readFileSync, unlinkSync } from 'fs';
import * as path from 'path';

const E2E_WORKER_PID_FILE = path.join(process.cwd(), '.e2e-worker.pid');

async function killProcess(pid: number): Promise<void> {
  // Kill the entire process group (pnpm + ts-node child) spawned with detached: true
  try {
    process.kill(-pid, 'SIGTERM');
  } catch {
    // fallback: kill just the pid
    try {
      process.kill(pid, 'SIGKILL');
    } catch {
      // Process may already be dead
    }
  }
}

export default async function globalTeardown(): Promise<void> {
  if (!existsSync(E2E_WORKER_PID_FILE)) return;

  const pid = parseInt(readFileSync(E2E_WORKER_PID_FILE, 'utf8'), 10);
  if (Number.isNaN(pid)) return;

  await killProcess(pid);
  unlinkSync(E2E_WORKER_PID_FILE);
}
