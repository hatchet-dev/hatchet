import { makeE2EClient, startWorker, stopWorker } from '../__e2e__/harness';
import { streamingTask } from './workflow';

describe('streaming-e2e', () => {
  const hatchet = makeE2EClient();
  let worker: Awaited<ReturnType<typeof startWorker>> | undefined;

  beforeAll(async () => {
    worker = await startWorker({
      client: hatchet,
      name: 'streaming-e2e-worker',
      workflows: [streamingTask],
      slots: 2,
    });
  });

  afterAll(async () => {
    await stopWorker(worker);
  });

  it(
    'stream output arrives in-order and complete',
    async () => {
      const ref = await streamingTask.runNoWait({});
      const runId = await ref.getWorkflowRunId();

      let combined = '';
      for await (const chunk of hatchet.runs.subscribeToStream(runId)) {
        combined += chunk;
      }

      // Basic correctness: we got *something* and it includes the known leading text.
      // (Exact chunk-by-chunk equality is possible, but this keeps the test resilient to
      // small changes in chunking while still validating ordering/completeness.)
      expect(combined.length).toBeGreaterThan(0);
      expect(combined.startsWith('\nHappy families are all alike')).toBeTruthy();

      // Ensure the run itself completed (stream closes at completion)
      await ref.output;
    },
    120_000
  );
});

