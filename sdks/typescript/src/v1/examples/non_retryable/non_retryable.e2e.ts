import { makeE2EClient, poll } from '../__e2e__/harness';
import { nonRetryableWorkflow } from './workflow';
import { V1TaskEventType, V1TaskStatus } from '../../../clients/rest/generated/data-contracts';

describe('non-retryable-e2e', () => {
  const hatchet = makeE2EClient();

  it('retries only the retryable failure', async () => {
    const ref = await nonRetryableWorkflow.runNoWait({});

    const details = await poll(
      async () => {
        try {
          return await hatchet.runs.get(ref);
        } catch (e: any) {
          if (e.response?.status === 404) {
            return { run: { status: V1TaskStatus.RUNNING }, taskEvents: [] };
          }
          throw e;
        }
      },
      {
        timeoutMs: 60_000,
        intervalMs: 100,
        label: 'nonRetryableWorkflow terminal',
        shouldStop: (d) =>
          ![V1TaskStatus.QUEUED, V1TaskStatus.RUNNING].includes(d.run.status as any),
      }
    );

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
