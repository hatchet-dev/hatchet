import sleep from '@hatchet/util/sleep';
import { V1TaskStatus } from '@hatchet/clients/rest/generated/data-contracts';
import { makeE2EClient, startWorker, stopWorker } from '../__e2e__/harness';
import { concurrencyCancelNewestWorkflow } from './workflow';

describe('concurrency-cancel-newest-e2e', () => {
  const hatchet = makeE2EClient();
  let worker: Awaited<ReturnType<typeof startWorker>> | undefined;

  beforeAll(async () => {
    worker = await startWorker({
      client: hatchet,
      name: 'concurrency-cancel-newest-e2e-worker',
      workflows: [concurrencyCancelNewestWorkflow],
      slots: 10,
    });
  });

  afterAll(async () => {
    await stopWorker(worker);
  });

  it('cancels newest runs when concurrency limit reached', async () => {
    const testRunId = crypto.randomUUID();

    const toRun = await concurrencyCancelNewestWorkflow.runNoWait(
      { group: 'A' },
      { additionalMetadata: { test_run_id: testRunId } }
    );

    await sleep(1000);

    const toCancel = await Promise.all(
      Array.from({ length: 10 }, () =>
        concurrencyCancelNewestWorkflow.runNoWait(
          { group: 'A' },
          { additionalMetadata: { test_run_id: testRunId } }
        )
      )
    );

    await toRun.output;

    for (const ref of toCancel) {
      try {
        await ref.output;
      } catch {
        // Expected for cancelled runs
      }
    }

    await sleep(5000);

    const details = (await hatchet.runs.get(toRun)) as { run?: { status?: string } };
    expect(details.run?.status).toBe(V1TaskStatus.COMPLETED);

    const runId = await toRun.getWorkflowRunId();
    const allRuns =
      (
        await hatchet.runs.list({
          additionalMetadata: { test_run_id: testRunId },
          onlyTasks: false,
        } as any)
      ).rows || [];

    const otherRuns = allRuns.filter((r: any) => r.metadata?.id !== runId);
    expect(otherRuns.every((r: any) => r.status === V1TaskStatus.CANCELLED)).toBe(true);
  }, 90_000);
});
