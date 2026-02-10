import { makeE2EClient, poll, startWorker, stopWorker } from '../__e2e__/harness';
import { ERROR_TEXT, failureWorkflow } from './workflow';
import { V1TaskStatus } from '../../../clients/rest/generated/data-contracts';

describe('on-failure-e2e', () => {
  const hatchet = makeE2EClient();
  let worker: Awaited<ReturnType<typeof startWorker>> | undefined;

  beforeAll(async () => {
    worker = await startWorker({
      client: hatchet,
      name: 'on-failure-e2e-worker',
      workflows: [failureWorkflow],
      slots: 10,
    });
  });

  afterAll(async () => {
    await stopWorker(worker);
  });

  xit('runs on_failure task after failure', async () => {
    const ref = await failureWorkflow.runNoWait({});

    await expect(ref.output).rejects.toEqual(
      expect.arrayContaining([expect.stringContaining(ERROR_TEXT)])
    );

    const details = await poll(async () => hatchet.runs.get(ref), {
      timeoutMs: 120_000,
      intervalMs: 1000,
      label: 'onFailure run details',
      shouldStop: (d) =>
        ![V1TaskStatus.QUEUED, V1TaskStatus.RUNNING].includes(d.run.status as any) &&
        (d.tasks || []).some((t) => `${t.displayName}`.includes('on_failure')),
    });

    expect(details.tasks.length).toBeGreaterThanOrEqual(2);
    expect(details.run.status).toBe(V1TaskStatus.FAILED);

    const completed = details.tasks.filter((t) => t.status === V1TaskStatus.COMPLETED);
    const failed = details.tasks.filter((t) => t.status === V1TaskStatus.FAILED);
    expect(completed.length).toBeGreaterThanOrEqual(1);
    expect(failed.length).toBeGreaterThanOrEqual(1);

    expect(completed.some((t) => t.displayName.includes('on_failure'))).toBeTruthy();
    expect(failed.some((t) => t.displayName.includes('step1'))).toBeTruthy();
  }, 180_000);
});
