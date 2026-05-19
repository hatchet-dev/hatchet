/**
 * Jest globalTeardown for e2e tests - kills the shared e2e worker process and its descendants.
 */
import { execSync } from 'child_process';
import { existsSync, readFileSync, unlinkSync } from 'fs';
import * as path from 'path';

const E2E_WORKER_PID_FILE = path.join(process.cwd(), '.e2e-worker.pid');

function killProcessTree(pid: number): void {
  const killTimeoutMs = 10_000;
  try {
    if (process.platform === 'win32') {
      execSync(`taskkill /pid ${pid} /T /F`, { stdio: 'ignore', timeout: killTimeoutMs });
    } else {
      execSync(`pkill -TERM -P ${pid} 2>/dev/null; kill -TERM ${pid} 2>/dev/null; sleep 0.5; kill -9 -${pid} 2>/dev/null; kill -9 ${pid} 2>/dev/null`, {
        stdio: 'ignore',
        timeout: killTimeoutMs,
      });
    }
  } catch {
    // Process may already be dead
  }
}

export default async function globalTeardown(): Promise<void> {
  if (!existsSync(E2E_WORKER_PID_FILE)) return;

  const pid = parseInt(readFileSync(E2E_WORKER_PID_FILE, 'utf8'), 10);
  if (Number.isNaN(pid)) return;

  killProcessTree(pid);
  unlinkSync(E2E_WORKER_PID_FILE);
}
