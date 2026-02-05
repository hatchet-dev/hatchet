import { makeE2EClient, poll, startWorker, stopWorker } from '../__e2e__/harness';
import { timeoutTask, refreshTimeoutTask } from './workflow';
import { V1TaskStatus } from '../../../clients/rest/generated/data-contracts';

describe('timeout-e2e', () => {
  const hatchet = makeE2EClient();

  let worker: Awaited<ReturnType<typeof startWorker>> | undefined;

  beforeAll(async () => {
    worker = await startWorker({
      client: hatchet,
      name: 'timeout-e2e-worker',
      workflows: [timeoutTask, refreshTimeoutTask],
      slots: 10,
    });
  });

  afterAll(async () => {
    await stopWorker(worker);
  });

  it(
    'execution timeout should fail the run',
    async () => {
      const ref = await timeoutTask.runNoWait({ Message: 'hello' });

      const run = await poll(
        async () => hatchet.runs.get(ref),
        {
          timeoutMs: 60_000,
          intervalMs: 1000,
          label: 'timeoutTask terminal status',
          shouldStop: (r) =>
            ![V1TaskStatus.QUEUED, V1TaskStatus.RUNNING].includes(r.run.status as any),
        }
      );

      expect([V1TaskStatus.FAILED, V1TaskStatus.CANCELLED]).toContain(run.run.status);
    },
    90_000
  );

  it(
    'refresh timeout should allow a longer run to succeed',
    async () => {
      const res = await refreshTimeoutTask.run({ Message: 'hello' });
      expect(res.status).toBe('success');
    },
    90_000
  );
});

