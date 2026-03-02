import { makeE2EClient, poll } from '../__e2e__/harness';
import { timeoutTask, refreshTimeoutTask } from './workflow';
import { V1TaskStatus } from '../../../clients/rest/generated/data-contracts';

describe('timeout-e2e', () => {
  const hatchet = makeE2EClient();

  it('execution timeout should fail the run', async () => {
    const ref = await timeoutTask.runNoWait({ Message: 'hello' });

    const run = await poll(
      async () => {
        try {
          return await hatchet.runs.get(ref);
        } catch (e: any) {
          if (e.response?.status === 404) {
            return { run: { status: V1TaskStatus.RUNNING } } as any;
          }
          throw e;
        }
      },
      {
        timeoutMs: 60_000,
        intervalMs: 100,
        label: 'timeoutTask terminal status',
        shouldStop: (r) =>
          ![V1TaskStatus.QUEUED, V1TaskStatus.RUNNING].includes(r.run.status as any),
      }
    );

    expect([V1TaskStatus.FAILED, V1TaskStatus.CANCELLED]).toContain(run.run.status);
  }, 90_000);

  it('refresh timeout should allow a longer run to succeed', async () => {
    const res = await refreshTimeoutTask.run({ Message: 'hello' });
    expect(res.status).toBe('success');
  }, 90_000);
});
