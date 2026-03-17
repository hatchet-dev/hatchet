import sleep from '@hatchet/util/sleep';
import { V1TaskStatus } from '@hatchet/clients/rest/generated/data-contracts';
import Hatchet from '@hatchet/index';
import { makeE2EClient, poll, checkDurableEvictionSupport } from '../__e2e__/harness';
import {
  evictableSleep,
  evictableWaitForEvent,
  evictableChildSpawn,
  evictableChildBulkSpawn,
  multipleEviction,
  nonEvictableSleep,
  capacityEvictableSleep,
  LONG_SLEEP_SECONDS,
  EVICTION_TTL_SECONDS,
  EVENT_KEY,
  evictableSleepForGracefulTermination,
} from './workflow';

function getTaskStatuses(details: any): V1TaskStatus[] {
  return (details?.tasks || []).map((t: any) => t.status);
}

function hasEvictedTask(details: any): boolean {
  return (details?.tasks || []).some((t: any) => t.isEvicted === true);
}

function getTaskExternalId(details: any): string | undefined {
  const tasks = details?.tasks || [];
  const [t] = tasks;
  return t?.taskExternalId ?? t?.metadata?.id;
}

describe('durable-eviction-e2e', () => {
  const hatchet = makeE2EClient();
  let evictionSupported = false;

  beforeAll(async () => {
    evictionSupported = await checkDurableEvictionSupport(hatchet);
  });

  function requireEviction() {
    if (!evictionSupported) {
      console.log('Skipping: engine does not support durable eviction');
    }
    return !evictionSupported;
  }

  async function pollUntilStatus(
    runId: string,
    targetStatus: V1TaskStatus,
    maxPollsOverride?: number
  ) {
    const maxPolls = maxPollsOverride || 15;
    const interval = 2000;

    return poll(
      async () => {
        try {
          return await hatchet.runs.get(runId);
        } catch (e: any) {
          if (e?.response?.status === 404) {return undefined;}
          throw e;
        }
      },
      {
        timeoutMs: maxPolls * interval,
        intervalMs: interval,
        shouldStop: (details: any) =>
          details != null && getTaskStatuses(details).includes(targetStatus),
        label: `status=${targetStatus}`,
      }
    );
  }

  async function pollUntilEvicted(runId: string, maxPollsOverride?: number) {
    const maxPolls = maxPollsOverride || 15;
    const interval = 2000;

    return poll(
      async () => {
        try {
          return await hatchet.runs.get(runId);
        } catch (e: any) {
          if (e?.response?.status === 404) {return undefined;}
          throw e;
        }
      },
      {
        timeoutMs: maxPolls * interval,
        intervalMs: interval,
        shouldStop: (details: any) => details != null && hasEvictedTask(details),
        label: 'isEvicted=true',
      }
    );
  }

  it('non-evictable task completes normally', async () => {
    if (requireEviction()) {return;}
    const start = Date.now();
    const result = await nonEvictableSleep.run({});
    const elapsed = (Date.now() - start) / 1000;

    expect(result.status).toBe('completed');
    expect(elapsed).toBeGreaterThanOrEqual(9);
  }, 120_000);

  it('non-evictable task is never evicted past TTL', async () => {
    if (requireEviction()) {return;}
    const ref = await nonEvictableSleep.runNoWait({});
    const runId = await ref.getWorkflowRunId();

    await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    await sleep(7000);
    const details = await hatchet.runs.get(runId);

    expect(hasEvictedTask(details)).toBe(false);

    const result = await ref.output;
    expect(result.status).toBe('completed');
  }, 120_000);

  it('evictable task is evicted after TTL', async () => {
    if (requireEviction()) {return;}
    const ref = await evictableSleep.runNoWait({});
    const runId = await ref.getWorkflowRunId();

    await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    const details = await pollUntilEvicted(runId);

    expect(hasEvictedTask(details)).toBe(true);
  }, 120_000);

  it('evictable task restore re-enqueues the task', async () => {
    if (requireEviction()) {return;}
    const ref = await evictableSleep.runNoWait({});
    const runId = await ref.getWorkflowRunId();

    await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    const details = await pollUntilEvicted(runId);
    const taskId = getTaskExternalId(details);
    expect(taskId).toBeDefined();

    await hatchet.runs.restoreTask(taskId!);

    const restored = await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    const statuses = getTaskStatuses(restored);
    expect(statuses).toContain(V1TaskStatus.RUNNING);
  }, 120_000);

  it('evictable task restore completes', async () => {
    if (requireEviction()) {return;}
    const start = Date.now();
    const ref = await evictableSleep.runNoWait({});
    const runId = await ref.getWorkflowRunId();

    await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    const details = await pollUntilEvicted(runId);
    const taskId = getTaskExternalId(details);
    expect(taskId).toBeDefined();

    await hatchet.runs.restoreTask(taskId!);

    const result = await ref.output;
    const elapsed = (Date.now() - start) / 1000;
    expect(result.status).toBe('completed');
    expect(elapsed).toBeGreaterThanOrEqual(LONG_SLEEP_SECONDS);
  }, 180_000);

  it('evictable wait-for-event is evicted after TTL', async () => {
    if (requireEviction()) {return;}
    const ref = await evictableWaitForEvent.runNoWait({});
    const runId = await ref.getWorkflowRunId();

    await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    const details = await pollUntilEvicted(runId);

    expect(hasEvictedTask(details)).toBe(true);
  }, 120_000);

  it('evictable wait-for-event restore + event completes', async () => {
    if (requireEviction()) {return;}
    const ref = await evictableWaitForEvent.runNoWait({});
    const runId = await ref.getWorkflowRunId();

    await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    const details = await pollUntilEvicted(runId);
    const taskId = getTaskExternalId(details);
    expect(taskId).toBeDefined();

    await hatchet.runs.restoreTask(taskId!);
    await pollUntilStatus(runId, V1TaskStatus.RUNNING);

    await hatchet.events.push(EVENT_KEY, {});

    const result = await ref.output;
    expect(result.status).toBe('completed');
  }, 180_000);

  it('evictable child spawn is evicted after TTL', async () => {
    if (requireEviction()) {return;}
    const ref = await evictableChildSpawn.runNoWait({});
    const runId = await ref.getWorkflowRunId();

    await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    const details = await pollUntilEvicted(runId);

    expect(hasEvictedTask(details)).toBe(true);
  }, 120_000);

  it('evictable child spawn restore completes', async () => {
    if (requireEviction()) {return;}
    const ref = await evictableChildSpawn.runNoWait({});
    const runId = await ref.getWorkflowRunId();

    await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    const details = await pollUntilEvicted(runId);
    const taskId = getTaskExternalId(details);
    expect(taskId).toBeDefined();

    await hatchet.runs.restoreTask(taskId!);

    const result = await ref.output;
    expect(result.status).toBe('completed');
    expect(result.child).toEqual({ child_status: 'completed' });
  }, 180_000);

  it('evictable child spawn restore re-enqueues', async () => {
    if (requireEviction()) {return;}
    const ref = await evictableChildSpawn.runNoWait({});
    const runId = await ref.getWorkflowRunId();

    await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    const details = await pollUntilEvicted(runId);
    const taskId = getTaskExternalId(details);
    expect(taskId).toBeDefined();

    await hatchet.runs.restoreTask(taskId!);

    const restored = await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    const statuses = getTaskStatuses(restored);
    expect(statuses).toContain(V1TaskStatus.RUNNING);
  }, 120_000);

  it('evictable child bulk spawn restore completes', async () => {
    if (requireEviction()) {return;}
    const ref = await evictableChildBulkSpawn.runNoWait({});
    const runId = await ref.getWorkflowRunId();

    let evictionCount = 0;
    for (let i = 0; i < 3; i += 1) {
      await pollUntilStatus(runId, V1TaskStatus.RUNNING);
      const details = await pollUntilEvicted(runId);
      evictionCount += 1;
      const taskId = getTaskExternalId(details)!;
      await hatchet.runs.restoreTask(taskId);
    }

    const result = await ref.output;
    expect(evictionCount).toBe(3);
    expect(result.status).toBe('completed');
    expect(result.child_results).toEqual(
      Array.from({ length: 3 }, (_, i) => ({
        sleepSeconds: (EVICTION_TTL_SECONDS + 5) * (i + 1),
        status: 'completed',
      }))
    );
  }, 300_000);

  it('multiple eviction cycles', async () => {
    if (requireEviction()) {return;}
    const start = Date.now();
    const ref = await multipleEviction.runNoWait({});
    const runId = await ref.getWorkflowRunId();

    // First eviction cycle
    await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    let details = await pollUntilEvicted(runId);
    expect(hasEvictedTask(details)).toBe(true);

    let taskId = getTaskExternalId(details)!;
    await hatchet.runs.restoreTask(taskId);

    // Second eviction cycle
    await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    details = await pollUntilEvicted(runId);
    expect(hasEvictedTask(details)).toBe(true);

    taskId = getTaskExternalId(details)!;
    await hatchet.runs.restoreTask(taskId);

    const result = await ref.output;
    const elapsed = (Date.now() - start) / 1000;
    expect(result.status).toBe('completed');
    expect(elapsed).toBeGreaterThanOrEqual(2 * LONG_SLEEP_SECONDS);
  }, 300_000);

  it('eviction plus replay completes', async () => {
    if (requireEviction()) {return;}
    const ref = await evictableSleep.runNoWait({});
    const runId = await ref.getWorkflowRunId();

    await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    await pollUntilEvicted(runId);

    await hatchet.runs.replay({ ids: [runId] });

    const result = await ref.output;
    expect(result.status).toBe('completed');
  }, 180_000);

  it('cancel after eviction transitions to CANCELLED', async () => {
    if (requireEviction()) {return;}
    const ref = await evictableSleep.runNoWait({});
    const runId = await ref.getWorkflowRunId();

    await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    const details = await pollUntilEvicted(runId);
    expect(hasEvictedTask(details)).toBe(true);

    await hatchet.runs.cancel({ ids: [runId] });

    const cancelled = await pollUntilStatus(runId, V1TaskStatus.CANCELLED, 30);
    const cancelledStatuses = getTaskStatuses(cancelled);
    expect(cancelledStatuses).toContain(V1TaskStatus.CANCELLED);
  }, 120_000);

  it('restore idempotency - double restore completes once', async () => {
    if (requireEviction()) {return;}
    const ref = await evictableSleep.runNoWait({});
    const runId = await ref.getWorkflowRunId();

    await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    const details = await pollUntilEvicted(runId);
    const taskId = getTaskExternalId(details)!;

    await hatchet.runs.restoreTask(taskId);
    await hatchet.runs.restoreTask(taskId);

    const result = await ref.output;
    expect(result.status).toBe('completed');
  }, 180_000);

  it('capacity eviction fires with durable_slots=1 and ttl=undefined', async () => {
    if (requireEviction()) {return;}
    const { spawn } = await import('child_process');

    const workerProc = spawn(
      'pnpm',
      [
        'exec',
        'ts-node',
        '-r',
        'tsconfig-paths/register',
        '-P',
        'tsconfig.json',
        'src/v1/examples/durable_eviction/capacity-worker.ts',
      ],
      {
        cwd: process.cwd(),
        env: {
          ...process.env,
          HATCHET_CLIENT_WORKER_HEALTHCHECK_ENABLED: 'true',
          HATCHET_CLIENT_WORKER_HEALTHCHECK_PORT: '8105',
        },
        stdio: 'pipe',
      }
    );

    workerProc.stdout?.on('data', () => {});
    workerProc.stderr?.on('data', () => {});

    try {
      await poll(
        async () => {
          try {
            const resp = await fetch('http://localhost:8105/health');
            return resp.ok;
          } catch {
            return false;
          }
        },
        {
          timeoutMs: 30_000,
          intervalMs: 1000,
          shouldStop: (healthy) => healthy === true,
          label: 'capacity-worker-health',
        }
      );

      const ref = await capacityEvictableSleep.runNoWait({});
      const runId = await ref.getWorkflowRunId();

      await pollUntilStatus(runId, V1TaskStatus.RUNNING);
      const details = await pollUntilEvicted(runId, 20);

      expect(hasEvictedTask(details)).toBe(true);
    } finally {
      try {
        workerProc.kill('SIGKILL');
      } catch {
        // ignore
      }
    }
  }, 120_000);

  it('capacity eviction restore completes', async () => {
    if (requireEviction()) {return;}
    const { spawn } = await import('child_process');

    const workerProc = spawn(
      'pnpm',
      [
        'exec',
        'ts-node',
        '-r',
        'tsconfig-paths/register',
        '-P',
        'tsconfig.json',
        'src/v1/examples/durable_eviction/capacity-worker.ts',
      ],
      {
        cwd: process.cwd(),
        env: {
          ...process.env,
          HATCHET_CLIENT_WORKER_HEALTHCHECK_ENABLED: 'true',
          HATCHET_CLIENT_WORKER_HEALTHCHECK_PORT: '8106',
        },
        stdio: 'pipe',
      }
    );

    workerProc.stdout?.on('data', () => {});
    workerProc.stderr?.on('data', () => {});

    try {
      await poll(
        async () => {
          try {
            const resp = await fetch('http://localhost:8106/health');
            return resp.ok;
          } catch {
            return false;
          }
        },
        {
          timeoutMs: 30_000,
          intervalMs: 1000,
          shouldStop: (healthy) => healthy === true,
          label: 'capacity-worker-health',
        }
      );

      const ref = await capacityEvictableSleep.runNoWait({});
      const runId = await ref.getWorkflowRunId();

      await pollUntilStatus(runId, V1TaskStatus.RUNNING);
      const details = await pollUntilEvicted(runId, 20);
      const taskId = getTaskExternalId(details)!;

      await hatchet.runs.restoreTask(taskId);

      const result = await ref.output;
      expect(result.status).toBe('completed');
    } finally {
      try {
        workerProc.kill('SIGKILL');
      } catch {
        // ignore
      }
    }
  }, 180_000);

  it('graceful termination evicts waiting runs', async () => {
    if (requireEviction()) {return;}
    const { spawn } = await import('child_process');

    const namespace = 'graceful-termination-evicts-waiting-runs';

    const hatchetWithNamespace = Hatchet.init({
      namespace,
    });

    const workerProc = spawn(
      'pnpm',
      [
        'exec',
        'ts-node',
        '-r',
        'tsconfig-paths/register',
        '-P',
        'tsconfig.json',
        'src/v1/examples/durable_eviction/worker.ts',
      ],
      {
        cwd: process.cwd(),
        env: {
          ...process.env,
          HATCHET_CLIENT_WORKER_HEALTHCHECK_ENABLED: 'true',
          HATCHET_CLIENT_WORKER_HEALTHCHECK_PORT: '8104',
          HATCHET_CLIENT_NAMESPACE: 'graceful-termination-evicts-waiting-runs',
        },
        stdio: 'pipe',
      }
    );

    workerProc.stdout?.on('data', () => {});
    workerProc.stderr?.on('data', () => {});

    try {
      await poll(
        async () => {
          try {
            const resp = await fetch('http://localhost:8104/health');
            return resp.ok;
          } catch {
            return false;
          }
        },
        {
          timeoutMs: 30_000,
          intervalMs: 1000,
          shouldStop: (healthy) => healthy === true,
          label: 'worker-health',
        }
      );

      const ref = await hatchetWithNamespace.admin.runWorkflow(
        evictableSleepForGracefulTermination.name,
        {}
      );

      const runId = await ref.getWorkflowRunId();

      await pollUntilStatus(runId, V1TaskStatus.RUNNING);

      workerProc.kill('SIGTERM');

      const details = await pollUntilEvicted(runId);
      expect(hasEvictedTask(details)).toBe(true);
    } finally {
      try {
        workerProc.kill('SIGKILL');
      } catch {
        // ignore
      }
    }
  }, 120_000);
});
