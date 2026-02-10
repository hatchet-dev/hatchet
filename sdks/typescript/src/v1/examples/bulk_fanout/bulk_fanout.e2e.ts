import { makeE2EClient, startWorker, stopWorker } from '../__e2e__/harness';
import { bulkChild, bulkParentWorkflow } from './workflow';

describe('bulk-fanout-e2e', () => {
  const hatchet = makeE2EClient();
  let worker: Awaited<ReturnType<typeof startWorker>> | undefined;

  beforeAll(async () => {
    worker = await startWorker({
      client: hatchet,
      name: 'bulk-fanout-e2e-worker',
      workflows: [bulkChild, bulkParentWorkflow],
      slots: 50,
    });
  });

  afterAll(async () => {
    await stopWorker(worker);
  });

  it('spawns N children and returns all results', async () => {
    const result = await bulkParentWorkflow.run({ n: 12 });
    expect((result as any).spawn.results).toHaveLength(12);
  }, 90_000);
});
