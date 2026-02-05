import { makeE2EClient, startWorker, stopWorker } from '../__e2e__/harness';
import { priority, priorityWf } from './workflow';
import { Priority } from '@hatchet/v1';

describe('priority-e2e', () => {
  const hatchet = makeE2EClient();
  let worker: Awaited<ReturnType<typeof startWorker>> | undefined;

  beforeAll(async () => {
    worker = await startWorker({
      client: hatchet,
      name: 'priority-e2e-worker',
      workflows: [priority, priorityWf],
      slots: 1,
    });
  });

  afterAll(async () => {
    await stopWorker(worker);
  });

  it(
    'task sees its configured default priority (unless overridden)',
    async () => {
      const res = await priority.run({});
      expect(res.priority).toBe(Priority.MEDIUM);
    },
    60_000
  );
});

