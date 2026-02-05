import sleep from '@hatchet/util/sleep';
import { randomUUID } from 'crypto';
import { HatchetClient } from '@hatchet/v1';
import type { BaseWorkflowDeclaration } from '@hatchet/v1';
import { Worker } from '../../client/worker/worker';

export function requireEnv(name: string): string {
  const value = process.env[name];
  if (!value) {
    throw new Error(
      `Missing required environment variable ${name}. ` +
        `E2E tests require a configured Hatchet instance and credentials.`
    );
  }
  return value;
}

export function makeE2EClient(): HatchetClient {
  // ConfigLoader requires a token; this makes the failure message obvious.
  requireEnv('HATCHET_CLIENT_TOKEN');
  return HatchetClient.init();
}

export function makeTestScope(prefix = 'ts_e2e'): string {
  return `${prefix}_${randomUUID()}`;
}

export async function startWorker({
  client,
  name,
  workflows,
  slots = 50,
}: {
  client: HatchetClient;
  name: string;
  workflows: Array<BaseWorkflowDeclaration<any, any>>;
  slots?: number;
}): Promise<Worker> {
  const worker = await client.worker(name, { workflows, slots });
  void worker.start();
  return worker;
}

export async function stopWorker(worker: Worker | undefined) {
  if (!worker) return;
  await worker.stop();
  // give the engine a beat to settle
  await sleep(1500);
}

export async function poll<T>(
  fn: () => Promise<T>,
  {
    timeoutMs = 30_000,
    intervalMs = 1000,
    shouldStop,
    label = 'poll',
  }: {
    timeoutMs?: number;
    intervalMs?: number;
    shouldStop: (value: T) => boolean;
    label?: string;
  }
): Promise<T> {
  const start = Date.now();
  // eslint-disable-next-line no-constant-condition
  while (true) {
    const value = await fn();
    if (shouldStop(value)) return value;
    if (Date.now() - start > timeoutMs) {
      throw new Error(`Timed out waiting for ${label} after ${timeoutMs}ms`);
    }
    await sleep(intervalMs);
  }
}

