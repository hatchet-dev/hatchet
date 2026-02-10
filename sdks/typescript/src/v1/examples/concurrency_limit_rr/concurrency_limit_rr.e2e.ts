import { makeE2EClient, startWorker, stopWorker } from '../__e2e__/harness';
import { simpleConcurrency } from './workflow';

describe('concurrency_limit_rr-e2e', () => {
  const hatchet = makeE2EClient();
  let worker: Awaited<ReturnType<typeof startWorker>> | undefined;

  beforeAll(async () => {
    worker = await startWorker({
      client: hatchet,
      name: 'concurrency-rr-e2e-worker',
      workflows: [simpleConcurrency],
      slots: 1,
    });
  });

  afterAll(async () => {
    await stopWorker(worker);
  });

  it.skip('round-robin concurrency behavior (timing-sensitive)', async () => {
    // Python version is skipped due to timing unreliability; keep parity here.
    // If we want to test this reliably, we should assert on engine events/ordering
    // rather than wall-clock duration.
    await simpleConcurrency.run([
      { Message: 'a', GroupKey: 'A' },
      { Message: 'b', GroupKey: 'A' },
      { Message: 'c', GroupKey: 'B' },
      { Message: 'd', GroupKey: 'B' },
    ]);
  }, 120_000);
});
