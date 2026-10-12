import { makeE2EClient, poll } from '../__e2e__/harness';
import { retryAfterExponentialBackoff, retryAfterUpstreamDelay } from './workflow';
import { V1TaskStatus } from '../../../clients/rest/generated/data-contracts';

describe('retry-after-e2e', () => {
  const hatchet = makeE2EClient();

  it('waits the requested delay before retrying', async () => {
    const start = Date.now();

    const result = await retryAfterUpstreamDelay.run({ failingAttempts: 2 });

    expect(result.attempt).toBe(2);
    expect(Date.now() - start).toBeGreaterThanOrEqual(4_000);
  }, 60_000);

  it('fails once the retries are exhausted', async () => {
    const ref = await retryAfterExponentialBackoff.runNoWait({ failingAttempts: 100 });

    const details = await poll(
      async () => {
        try {
          return await hatchet.runs.get(ref);
        } catch (e: any) {
          if (e.response?.status === 404) {
            return { run: { status: V1TaskStatus.RUNNING }, tasks: [] };
          }
          throw e;
        }
      },
      {
        timeoutMs: 60_000,
        intervalMs: 500,
        label: 'retryAfterExponentialBackoff terminal',
        shouldStop: (d) =>
          ![V1TaskStatus.QUEUED, V1TaskStatus.RUNNING].includes(d.run.status as any) &&
          (d.tasks[0]?.retryCount ?? 0) >= 3,
      }
    );

    expect(details.run.status).toBe(V1TaskStatus.FAILED);

    expect(details.tasks[0].retryCount).toBe(3);
  }, 90_000);
});
