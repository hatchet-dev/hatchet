import { makeE2EClient, poll, startWorker, stopWorker } from '../__e2e__/harness';
import { nonRetryableWorkflow } from './workflow';
import { V1TaskEventType, V1TaskStatus } from '../../../clients/rest/generated/data-contracts';

describe('non-retryable-e2e', () => {
  const hatchet = makeE2EClient();
  let worker: Awaited<ReturnType<typeof startWorker>> | undefined;

  beforeAll(async () => {
    worker = await startWorker({
      client: hatchet,
      name: 'non-retryable-e2e-worker',
      workflows: [nonRetryableWorkflow],
      slots: 10,
    });
  });

  afterAll(async () => {
    await stopWorker(worker);
  });

  it('retries only the retryable failure', async () => {
    const ref = await nonRetryableWorkflow.runNoWait({});

    const details = await poll(async () => hatchet.runs.get(ref), {
      timeoutMs: 60_000,
      intervalMs: 1000,
      label: 'nonRetryableWorkflow terminal',
      shouldStop: (d) => ![V1TaskStatus.QUEUED, V1TaskStatus.RUNNING].includes(d.run.status as any),
    });

    expect(details.run.status).toBe(V1TaskStatus.FAILED);

    const retrying = details.taskEvents.filter(
      (e: { eventType: V1TaskEventType }) => e.eventType === V1TaskEventType.RETRYING
    );
    expect(retrying.length).toBe(1);

    const failed = details.taskEvents.filter(
      (e: { eventType: V1TaskEventType }) => e.eventType === V1TaskEventType.FAILED
    );
    // python expects 3 FAILED events (two initial failures + one retry failure)
    expect(failed.length).toBeGreaterThanOrEqual(3);
  }, 90_000);
});
