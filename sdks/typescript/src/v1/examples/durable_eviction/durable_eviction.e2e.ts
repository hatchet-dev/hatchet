import sleep from '@hatchet/util/sleep';
import { V1TaskStatus } from '@hatchet/clients/rest/generated/data-contracts';
import { makeE2EClient, poll } from '../__e2e__/harness';
import {
  evictableSleep,
  evictableWaitForEvent,
  evictableChildSpawn,
  multipleEviction,
  nonEvictableSleep,
  LONG_SLEEP_SECONDS,
  EVENT_KEY,
} from './workflow';

function getTaskStatuses(details: any): V1TaskStatus[] {
  return (details?.tasks || []).map((t: any) => t.status);
}

function getTaskExternalId(details: any): string | undefined {
  const tasks = details?.tasks || [];
  const t = tasks[0];
  return t?.taskExternalId ?? t?.metadata?.id;
}

describe('durable-eviction-e2e', () => {
  const hatchet = makeE2EClient();

  async function pollUntilStatus(
    runId: string,
    targetStatus: V1TaskStatus,
    maxPollsOverride?: number
  ) {
    const maxPolls = maxPollsOverride || 15;
    const interval = 2000;

    return poll(() => hatchet.runs.get(runId), {
      timeoutMs: maxPolls * interval,
      intervalMs: interval,
      shouldStop: (details: any) => getTaskStatuses(details).includes(targetStatus),
      label: `status=${targetStatus}`,
    });
  }

  it('non-evictable task completes normally', async () => {
    const start = Date.now();
    const result = await nonEvictableSleep.run({});
    const elapsed = (Date.now() - start) / 1000;

    expect(result.status).toBe('completed');
    expect(elapsed).toBeGreaterThanOrEqual(9);
  }, 120_000);

  it('non-evictable task is never evicted past TTL', async () => {
    const ref = await nonEvictableSleep.runNoWait({});
    const runId = await ref.getWorkflowRunId();

    await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    await sleep(7000);
    const details = await hatchet.runs.get(runId);
    const statuses = getTaskStatuses(details);

    expect(statuses).not.toContain(V1TaskStatus.EVICTED);

    const result = await ref.output;
    expect(result.status).toBe('completed');
  }, 120_000);

  it('evictable task is evicted after TTL', async () => {
    const ref = await evictableSleep.runNoWait({});
    const runId = await ref.getWorkflowRunId();

    await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    const details = await pollUntilStatus(runId, V1TaskStatus.EVICTED);
    const statuses = getTaskStatuses(details);

    expect(statuses).toContain(V1TaskStatus.EVICTED);
  }, 120_000);

  it('evictable task restore re-enqueues the task', async () => {
    const ref = await evictableSleep.runNoWait({});
    const runId = await ref.getWorkflowRunId();

    await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    const details = await pollUntilStatus(runId, V1TaskStatus.EVICTED);
    const taskId = getTaskExternalId(details);
    expect(taskId).toBeDefined();

    await hatchet.runs.restoreTask(taskId!);

    const restored = await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    const statuses = getTaskStatuses(restored);
    expect(statuses).toContain(V1TaskStatus.RUNNING);
  }, 120_000);

  it('evictable task restore completes', async () => {
    const start = Date.now();
    const ref = await evictableSleep.runNoWait({});
    const runId = await ref.getWorkflowRunId();

    await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    const details = await pollUntilStatus(runId, V1TaskStatus.EVICTED);
    const taskId = getTaskExternalId(details);
    expect(taskId).toBeDefined();

    await hatchet.runs.restoreTask(taskId!);

    const result = await ref.output;
    const elapsed = (Date.now() - start) / 1000;
    expect(result.status).toBe('completed');
    expect(elapsed).toBeGreaterThanOrEqual(LONG_SLEEP_SECONDS);
  }, 180_000);

  it('evictable wait-for-event is evicted after TTL', async () => {
    const ref = await evictableWaitForEvent.runNoWait({});
    const runId = await ref.getWorkflowRunId();

    await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    const details = await pollUntilStatus(runId, V1TaskStatus.EVICTED);
    const statuses = getTaskStatuses(details);

    expect(statuses).toContain(V1TaskStatus.EVICTED);
  }, 120_000);

  it('evictable wait-for-event restore + event completes', async () => {
    const ref = await evictableWaitForEvent.runNoWait({});
    const runId = await ref.getWorkflowRunId();

    await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    const details = await pollUntilStatus(runId, V1TaskStatus.EVICTED);
    const taskId = getTaskExternalId(details);
    expect(taskId).toBeDefined();

    await hatchet.runs.restoreTask(taskId!);
    await pollUntilStatus(runId, V1TaskStatus.RUNNING);

    await hatchet.events.push(EVENT_KEY, {});

    const result = await ref.output;
    expect(result.status).toBe('completed');
  }, 180_000);

  it('evictable child spawn is evicted after TTL', async () => {
    const ref = await evictableChildSpawn.runNoWait({});
    const runId = await ref.getWorkflowRunId();

    await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    const details = await pollUntilStatus(runId, V1TaskStatus.EVICTED);
    const statuses = getTaskStatuses(details);

    expect(statuses).toContain(V1TaskStatus.EVICTED);
  }, 120_000);

  it('evictable child spawn restore completes', async () => {
    const ref = await evictableChildSpawn.runNoWait({});
    const runId = await ref.getWorkflowRunId();

    await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    const details = await pollUntilStatus(runId, V1TaskStatus.EVICTED);
    const taskId = getTaskExternalId(details);
    expect(taskId).toBeDefined();

    await hatchet.runs.restoreTask(taskId!);

    const result = await ref.output;
    expect(result.status).toBe('completed');
    expect(result.child).toEqual({ child_status: 'completed' });
  }, 180_000);

  it('multiple eviction cycles', async () => {
    const start = Date.now();
    const ref = await multipleEviction.runNoWait({});
    const runId = await ref.getWorkflowRunId();

    // First eviction cycle
    await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    let details = await pollUntilStatus(runId, V1TaskStatus.EVICTED);
    let statuses = getTaskStatuses(details);
    expect(statuses).toContain(V1TaskStatus.EVICTED);

    let taskId = getTaskExternalId(details)!;
    await hatchet.runs.restoreTask(taskId);

    // Second eviction cycle
    await pollUntilStatus(runId, V1TaskStatus.RUNNING);
    details = await pollUntilStatus(runId, V1TaskStatus.EVICTED);
    statuses = getTaskStatuses(details);
    expect(statuses).toContain(V1TaskStatus.EVICTED);

    taskId = getTaskExternalId(details)!;
    await hatchet.runs.restoreTask(taskId);

    const result = await ref.output;
    const elapsed = (Date.now() - start) / 1000;
    expect(result.status).toBe('completed');
    expect(elapsed).toBeGreaterThanOrEqual(2 * LONG_SLEEP_SECONDS);
  }, 300_000);
});
